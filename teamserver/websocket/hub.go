package websocket

import "simplec2/pkg/safe"

// Hub maintains the set of active clients and broadcasts messages to them.
type Hub struct {
	// Registered clients.
	clients *safe.Map

	// Inbound messages from the clients.
	broadcast chan []byte

	// Register requests from the clients.
	register chan *Client

	// Unregister requests from clients.
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		broadcast:  make(chan []byte),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		clients:    safe.NewMap(),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case client := <-h.register:
			h.clients.Store(client, true)
		case client := <-h.unregister:
			if _, ok := h.clients.Load(client); ok {
				h.clients.Delete(client)
				close(client.send)
			}
		case message := <-h.broadcast:
			// First, collect all clients to send to
			var clientsToSend []*Client
			h.clients.Range(func(key, value interface{}) bool {
				client := key.(*Client)
				clientsToSend = append(clientsToSend, client)
				return true
			})

			// Send to clients, track those that failed
			var failedClients []*Client
			for _, client := range clientsToSend {
				select {
				case client.send <- message:
					// Success
				default:
					// Failed, mark for cleanup
					failedClients = append(failedClients, client)
					close(client.send)
				}
			}

			// Cleanup failed clients (outside of Range to avoid deadlock)
			for _, client := range failedClients {
				h.clients.Delete(client)
			}
		}
	}
}

// Broadcast sends a message to all connected clients.
func (h *Hub) Broadcast(message []byte) {
	h.broadcast <- message
}
