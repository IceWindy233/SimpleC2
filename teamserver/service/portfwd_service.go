package service

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"simplec2/pkg/bridge"
	"simplec2/pkg/safe"
	"github.com/google/uuid"
)

// TunnelStatus represents the current status of a tunnel.
type TunnelStatus string

const (
	TunnelStatusPending TunnelStatus = "pending"
	TunnelStatusActive  TunnelStatus = "active"
	TunnelStatusClosed  TunnelStatus = "closed"
	TunnelStatusError   TunnelStatus = "error"
)

// Tunnel represents an active port forwarding tunnel.
type Tunnel struct {
	ID           string
	BeaconID     string
	Target       string // Format: host:port
	OperatorID   string // The operator who initiated the tunnel
	Status       TunnelStatus
	CreatedAt    time.Time
	LastActivity time.Time

	// Channels for data flow
	// Inbound: Data from Agent to Operator
	InboundData chan *bridge.TunnelMessage
	// Outbound: Data from Operator to Agent
	OutboundData chan *bridge.TunnelMessage

	// Context for managing the tunnel lifecycle
	Ctx    context.Context
	Cancel context.CancelFunc
}

// PortFwdService defines the interface for managing port forwarding tunnels.
type PortFwdService interface {
	// StartNewTunnel initiates a new tunnel request for a beacon.
	StartNewTunnel(ctx context.Context, beaconID, target, operatorID string) (*Tunnel, error)
	// GetTunnel retrieves an active tunnel by its ID.
	GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error)
	// CloseTunnel closes an active tunnel and cleans up resources.
	CloseTunnel(ctx context.Context, tunnelID string) error
	// PushOutboundData queues data to be sent to the agent via the specified tunnel.
	PushOutboundData(ctx context.Context, tunnelID string, data []byte, isFin bool) error
	// GetInboundData retrieves data received from the agent for a specified tunnel.
	GetInboundData(ctx context.Context, tunnelID string) ([]*bridge.TunnelMessage, error)
	// ProcessAgentOutgoingMessages processes messages from the agent during check-in.
	ProcessAgentOutgoingMessages(ctx context.Context, beaconID string, messages []*bridge.TunnelMessage)
	// GetAgentIncomingMessages retrieves messages to be sent to a specific agent during check-in.
	GetAgentIncomingMessages(ctx context.Context, beaconID string) []*bridge.TunnelMessage
	// ListTunnels retrieves all tunnels.
	ListTunnels(ctx context.Context) ([]*Tunnel, error)
}

// InMemoryPortFwdService implements PortFwdService using in-memory storage.
type InMemoryPortFwdService struct {
	// Map: BeaconID -> Map: TunnelID -> Tunnel
	// Using non-generic safe.Map, so values will be interface{} and require type assertions.
	tunnels *safe.Map 
	mu      sync.Mutex // Protects top-level map operations
}

// NewInMemoryPortFwdService creates a new InMemoryPortFwdService.
func NewInMemoryPortFwdService() *InMemoryPortFwdService {
	return &InMemoryPortFwdService{
		tunnels: safe.NewMap(),
	}
}

// ListTunnels retrieves all tunnels.
func (s *InMemoryPortFwdService) ListTunnels(ctx context.Context) ([]*Tunnel, error) {
	var tunnels []*Tunnel
	s.tunnels.Range(func(key, value interface{}) bool {
		beaconTunnels := value.(*safe.Map)
		beaconTunnels.Range(func(key, value interface{}) bool {
			tunnel := value.(*Tunnel)
			tunnels = append(tunnels, tunnel)
			return true
		})
		return true
	})
	return tunnels, nil
}


// StartNewTunnel initiates a new tunnel request for a beacon.
func (s *InMemoryPortFwdService) StartNewTunnel(ctx context.Context, beaconID, target, operatorID string) (*Tunnel, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	tunnelID := uuid.New().String()
	newTunnelCtx, cancel := context.WithCancel(ctx)

	tunnel := &Tunnel{
		ID:           tunnelID,
		BeaconID:     beaconID,
		Target:       target,
		OperatorID:   operatorID,
		Status:       TunnelStatusPending, // Will become active once agent confirms
		CreatedAt:    time.Now(),
		LastActivity: time.Now(),
		InboundData:  make(chan *bridge.TunnelMessage, 100),
		OutboundData: make(chan *bridge.TunnelMessage, 100),
		Ctx:          newTunnelCtx,
		Cancel:       cancel,
	}

	// Load or create the inner map for this beaconID
	beaconTunnelsVal, _ := s.tunnels.LoadOrStore(beaconID, safe.NewMap())
	beaconTunnels := beaconTunnelsVal.(*safe.Map) // Type assertion

	beaconTunnels.Store(tunnelID, tunnel)

	// Queue a START command for the agent
	startMsg := &bridge.TunnelMessage{
		TunnelId:    tunnelID,
		Target:      target,
		CommandType: bridge.TunnelMessage_START,
	}
	s.queueOutboundMessage(beaconID, startMsg)

	log.Printf("Started new tunnel %s for beacon %s to target %s by operator %s", tunnelID, beaconID, target, operatorID)
	return tunnel, nil
}

// GetTunnel retrieves an active tunnel by its ID.
func (s *InMemoryPortFwdService) GetTunnel(ctx context.Context, tunnelID string) (*Tunnel, error) {
	var foundTunnel *Tunnel
	s.tunnels.Range(func(key, value interface{}) bool {
		beaconTunnels := value.(*safe.Map) // Type assertion
		beaconTunnels.Range(func(key, value interface{}) bool {
			tunnel := value.(*Tunnel) // Type assertion
			if tunnel.ID == tunnelID {
				foundTunnel = tunnel
				return false // Stop iterating inner map
			}
			return true
		})
		return foundTunnel == nil // Continue iterating top-level map if not found
	})

	if foundTunnel == nil {
		return nil, fmt.Errorf("tunnel %s not found", tunnelID)
	}
	return foundTunnel, nil
}

// CloseTunnel closes an active tunnel and cleans up resources.
func (s *InMemoryPortFwdService) CloseTunnel(ctx context.Context, tunnelID string) error {
	tunnel, err := s.GetTunnel(ctx, tunnelID)
	if err != nil {
		return err
	}

	tunnel.Cancel() // Signal tunnel goroutines to stop
	tunnel.Status = TunnelStatusClosed
	tunnel.LastActivity = time.Now()

	// Queue a STOP command for the agent
	stopMsg := &bridge.TunnelMessage{
		TunnelId:    tunnelID,
		IsFin:       true,
		CommandType: bridge.TunnelMessage_STOP,
	}
	s.queueOutboundMessage(tunnel.BeaconID, stopMsg)

	log.Printf("Closed tunnel %s for beacon %s", tunnelID, tunnel.BeaconID)
	return nil
}

// PushOutboundData queues data to be sent to the agent via the specified tunnel.
func (s *InMemoryPortFwdService) PushOutboundData(ctx context.Context, tunnelID string, data []byte, isFin bool) error {
	tunnel, err := s.GetTunnel(ctx, tunnelID)
	if err != nil {
		return err
	}

	if tunnel.Status != TunnelStatusActive {
		return fmt.Errorf("tunnel %s is not active", tunnelID)
	}

	msg := &bridge.TunnelMessage{
		TunnelId:    tunnelID,
		Data:        data,
		IsFin:       isFin,
		CommandType: bridge.TunnelMessage_DATA,
	}

	select {
	case tunnel.OutboundData <- msg:
		tunnel.LastActivity = time.Now()
		return nil
	case <-time.After(1 * time.Second): // Prevent blocking indefinitely
		return fmt.Errorf("failed to queue outbound data for tunnel %s: outbound channel full", tunnelID)
	}
}

// GetInboundData retrieves data received from the agent for a specified tunnel.
func (s *InMemoryPortFwdService) GetInboundData(ctx context.Context, tunnelID string) ([]*bridge.TunnelMessage, error) {
	tunnel, err := s.GetTunnel(ctx, tunnelID)
	if err != nil {
		return nil, err
	}

	var messages []*bridge.TunnelMessage
	for {
		select {
		case msg := <-tunnel.InboundData:
			messages = append(messages, msg)
		default:
			return messages, nil
		}
	}
}

// ProcessAgentOutgoingMessages processes messages from the agent during check-in.
func (s *InMemoryPortFwdService) ProcessAgentOutgoingMessages(ctx context.Context, beaconID string, messages []*bridge.TunnelMessage) {
	for _, msg := range messages {
		tunnel, err := s.GetTunnel(ctx, msg.TunnelId)
		if err != nil {
			log.Printf("Agent %s sent message for non-existent tunnel %s: %v", beaconID, msg.TunnelId, err)
			continue
		}
		
		tunnel.LastActivity = time.Now()

		switch msg.CommandType {
		case bridge.TunnelMessage_START:
			// Agent confirms tunnel started
			if msg.IsError {
				tunnel.Status = TunnelStatusError
				log.Printf("Agent %s reported error starting tunnel %s: %s", beaconID, msg.TunnelId, msg.ErrorMsg)
			} else {
				tunnel.Status = TunnelStatusActive
				log.Printf("Agent %s confirmed tunnel %s active to %s", beaconID, msg.TunnelId, tunnel.Target)
			}
		case bridge.TunnelMessage_DATA:
			select {
			case tunnel.InboundData <- msg:
				// Data successfully queued for operator
			case <-time.After(100 * time.Millisecond):
				log.Printf("Failed to queue inbound data for tunnel %s (from agent %s): inbound channel full", msg.TunnelId, beaconID)
			}
		case bridge.TunnelMessage_STOP:
			log.Printf("Agent %s signaled tunnel %s stop. Reason: %s", beaconID, msg.TunnelId, msg.ErrorMsg)
			s.CloseTunnel(ctx, msg.TunnelId) // Clean up on TeamServer side
		default:
			log.Printf("Agent %s sent unknown tunnel message type: %v for tunnel %s", beaconID, msg.TunnelId, msg.CommandType)
		}
	}
}

// GetAgentIncomingMessages retrieves messages to be sent to a specific agent during check-in.
func (s *InMemoryPortFwdService) GetAgentIncomingMessages(ctx context.Context, beaconID string) []*bridge.TunnelMessage {
	var messages []*bridge.TunnelMessage
	
	beaconTunnelsVal, ok := s.tunnels.Load(beaconID)
	if !ok {
		return messages
	}
	beaconTunnels := beaconTunnelsVal.(*safe.Map) // Type assertion

	beaconTunnels.Range(func(key, value interface{}) bool {
		tunnel := value.(*Tunnel) // Type assertion
		select {
		case msg := <-tunnel.OutboundData:
			messages = append(messages, msg)
		default:
			// No data in this tunnel's outbound channel
		}
		return true
	})
	return messages
}

// queueOutboundMessage is a helper to queue messages to an agent's tunnel.
func (s *InMemoryPortFwdService) queueOutboundMessage(beaconID string, msg *bridge.TunnelMessage) {
	beaconTunnelsVal, ok := s.tunnels.Load(beaconID)
	if !ok {
		log.Printf("Attempted to queue outbound message for non-existent beacon %s", beaconID)
		return
	}
	beaconTunnels := beaconTunnelsVal.(*safe.Map) // Type assertion
	
	tunnelVal, ok := beaconTunnels.Load(msg.TunnelId)
	if !ok {
		log.Printf("Attempted to queue outbound message for non-existent tunnel %s on beacon %s", msg.TunnelId, beaconID)
		return
	}
	tunnel := tunnelVal.(*Tunnel) // Type assertion

	select {
	case tunnel.OutboundData <- msg:
		// Message sent to queue
	case <-time.After(1 * time.Second):
		log.Printf("Warning: Failed to queue outbound tunnel message for tunnel %s on beacon %s, queue full", msg.TunnelId, beaconID)
	}
}
