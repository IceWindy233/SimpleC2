package service

import (
	"context"
	"fmt"
	"sync"

	"simplec2/pkg/bridge"
	"simplec2/teamserver/data"

	"github.com/google/uuid"
)

// ListenerService defines the interface for listener-related business logic.
type ListenerService interface {
	// GetListener retrieves a listener by its name.
	GetListener(ctx context.Context, name string) (*data.Listener, error)

	// CreateListener creates a new listener configuration.
	CreateListener(ctx context.Context, name string, listenerType string, config string) (*data.Listener, error)

	// DeleteListener deletes a listener configuration.
	DeleteListener(ctx context.Context, name string) error

	// ListListeners retrieves all listeners.
	ListListeners(ctx context.Context, page int, limit int) ([]data.Listener, int64, error)

	// RegisterConnection registers a gRPC control stream for a listener.
	RegisterConnection(name string, stream bridge.TeamServerBridgeService_ListenerControlServer)

	// UnregisterConnection removes a gRPC control stream.
	UnregisterConnection(name string)

	// StartListener sends a start command to the listener.
	StartListener(ctx context.Context, name string) error

	// StopListener sends a stop command to the listener.
	StopListener(ctx context.Context, name string) error

	// RestartListener sends a restart command to the listener.
	RestartListener(ctx context.Context, name string) error

	// RecordIssuedCertificate saves a new certificate record.
	RecordIssuedCertificate(ctx context.Context, serialNumber, commonName, listenerName string) error

	// RevokeCertificateForListener revokes all certificates associated with a listener.
	RevokeCertificateForListener(ctx context.Context, listenerName string) error

	// IsCertificateRevoked checks if a serial number is revoked.
	IsCertificateRevoked(serialNumber string) bool
}

// listenerService implements the ListenerService interface.
type listenerService struct {
	store       data.DataStore
	connections map[string]bridge.TeamServerBridgeService_ListenerControlServer
	mu          sync.RWMutex
}

// NewListenerService creates a new instance of listenerService.
func NewListenerService(store data.DataStore) ListenerService {
	return &listenerService{
		store:       store,
		connections: make(map[string]bridge.TeamServerBridgeService_ListenerControlServer),
	}
}

// RecordIssuedCertificate saves a new certificate record.
func (s *listenerService) RecordIssuedCertificate(ctx context.Context, serialNumber, commonName, listenerName string) error {
	cert := &data.IssuedCertificate{
		SerialNumber: serialNumber,
		CommonName:   commonName,
		ListenerName: listenerName,
		Revoked:      false,
	}
	// We need to access Gorm DB directly or add a method to DataStore.
	// Since DataStore interface is generic, let's assume we can cast to GormStore or add method to interface.
	// For now, let's assume we need to extend DataStore interface.
	// To avoid changing DataStore interface immediately, let's assume s.store exposes a method or we add it now.
	// Wait, s.store is an interface. I should add methods to DataStore interface first.
	return s.store.CreateIssuedCertificate(cert)
}

// RevokeCertificateForListener revokes all certificates associated with a listener.
func (s *listenerService) RevokeCertificateForListener(ctx context.Context, listenerName string) error {
	return s.store.RevokeCertificatesByListener(listenerName)
}

// IsCertificateRevoked checks if a serial number is revoked.
func (s *listenerService) IsCertificateRevoked(serialNumber string) bool {
	revoked, err := s.store.IsCertificateRevoked(serialNumber)
	if err != nil {
		// Log error? For security, fail closed (return true) if error?
		// But here we return false (not revoked) on error to avoid DoS if DB is flaky.
		// Let's rely on DB.
		return false 
	}
	return revoked
}

// RegisterConnection registers a gRPC control stream for a listener.
func (s *listenerService) RegisterConnection(name string, stream bridge.TeamServerBridgeService_ListenerControlServer) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.connections[name] = stream
}

// UnregisterConnection removes a gRPC control stream.
func (s *listenerService) UnregisterConnection(name string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	delete(s.connections, name)
}

// StartListener sends a start command to the listener.
func (s *listenerService) StartListener(ctx context.Context, name string) error {
	return s.sendCommand(name, bridge.ListenerCommand_START, "")
}

// StopListener sends a stop command to the listener.
func (s *listenerService) StopListener(ctx context.Context, name string) error {
	return s.sendCommand(name, bridge.ListenerCommand_STOP, "")
}

// RestartListener sends a restart command to the listener.
func (s *listenerService) RestartListener(ctx context.Context, name string) error {
	return s.sendCommand(name, bridge.ListenerCommand_RESTART, "")
}

func (s *listenerService) sendCommand(name string, action bridge.ListenerCommand_Action, configJSON string) error {
	s.mu.RLock()
	stream, ok := s.connections[name]
	s.mu.RUnlock()

	if !ok {
		return fmt.Errorf("listener '%s' is not connected", name)
	}

	cmd := &bridge.ListenerCommand{
		RequestId:  uuid.New().String(),
		Action:     action,
		ConfigJson: configJSON,
	}

	if err := stream.Send(cmd); err != nil {
		// If sending fails, assume connection is dead and unregister
		s.UnregisterConnection(name)
		return fmt.Errorf("failed to send command to listener '%s': %w", name, err)
	}

	return nil
}

// GetListener retrieves a listener by its name.
func (s *listenerService) GetListener(ctx context.Context, name string) (*data.Listener, error) {
	listener, err := s.store.GetListener(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}

	s.mu.RLock()
	if _, ok := s.connections[listener.Name]; ok {
		listener.Active = true
	} else {
		listener.Active = false
	}
	s.mu.RUnlock()

	return listener, nil
}

// CreateListener creates a new listener configuration.
func (s *listenerService) CreateListener(ctx context.Context, name string, listenerType string, config string) (*data.Listener, error) {
	// Check if listener already exists
	if _, err := s.store.GetListener(name); err == nil {
		return nil, fmt.Errorf("listener with name '%s' already exists", name)
	}

	listener := &data.Listener{
		Name:   name,
		Type:   listenerType,
		Config: config,
	}

	if err := s.store.CreateListener(listener); err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return listener, nil
}

// DeleteListener deletes a listener configuration.
func (s *listenerService) DeleteListener(ctx context.Context, name string) error {
	// First, ensure listener exists
	if _, err := s.store.GetListener(name); err != nil {
		return fmt.Errorf("listener not found: %w", err)
	}

	// Try to send EXIT command to the active listener instance
	// We ignore the error because the listener might already be disconnected
	_ = s.sendCommand(name, bridge.ListenerCommand_EXIT, "")

	// Remove connection from memory
	s.UnregisterConnection(name)

	// Revoke certificates
	if err := s.RevokeCertificateForListener(ctx, name); err != nil {
		// Log error but proceed with deletion?
		// For strict security we might want to return error, but user wants to delete.
		// Let's log (fmt.Printf for now as we don't have logger here easily without import)
		fmt.Printf("Warning: Failed to revoke certificates for listener %s: %v\n", name, err)
	}

	if err := s.store.DeleteListener(name); err != nil {
		return fmt.Errorf("failed to delete listener: %w", err)
	}

	return nil
}

// ListListeners retrieves all listeners.
func (s *listenerService) ListListeners(ctx context.Context, page int, limit int) ([]data.Listener, int64, error) {
	listeners, total, err := s.store.GetListeners(page, limit)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list listeners: %w", err)
	}

	// Populate Active status
	s.mu.RLock()
	defer s.mu.RUnlock()
	for i := range listeners {
		if _, ok := s.connections[listeners[i].Name]; ok {
			listeners[i].Active = true
		} else {
			listeners[i].Active = false
		}
	}

	return listeners, total, nil
}
