package events

import (
	"log"
	"sync"
)

// Dispatcher handles event publishing and subscription.
type Dispatcher struct {
	subscribers map[EventType][]chan Event
	mu          sync.RWMutex
}

// NewDispatcher creates a new event dispatcher.
func NewDispatcher() *Dispatcher {
	return &Dispatcher{
		subscribers: make(map[EventType][]chan Event),
	}
}

// Subscribe registers a handler for a specific event type.
func (d *Dispatcher) Subscribe(eventType EventType, handler func(Event)) {
	d.mu.Lock()
	defer d.mu.Unlock()

	ch := make(chan Event, 100)
	d.subscribers[eventType] = append(d.subscribers[eventType], ch)

	go func() {
		for event := range ch {
			handler(event)
		}
	}()
}

// Publish broadcasts an event to all subscribed handlers.
func (d *Dispatcher) Publish(event Event) {
	d.mu.RLock()
	defer d.mu.RUnlock()

	if subs, ok := d.subscribers[event.Type]; ok {
		for _, ch := range subs {
			select {
			case ch <- event:
			default:
				// Channel full, drop oldest event
				<-ch
				ch <- event
			}
		}
	}
}

// UnsubscribeAll closes all subscription channels.
func (d *Dispatcher) UnsubscribeAll() {
	d.mu.Lock()
	defer d.mu.Unlock()

	for _, channels := range d.subscribers {
		for _, ch := range channels {
			close(ch)
		}
	}
	d.subscribers = make(map[EventType][]chan Event)
}

// PublishAsync publishes an event asynchronously without blocking.
func (d *Dispatcher) PublishAsync(event Event) {
	go func() {
		d.Publish(event)
	}()
}

// PublishToWebsocket publishes an event to websocket clients.
func (d *Dispatcher) PublishToWebsocket(event Event, hub interface{}) {
	// Convert event to JSON
	// TODO: Implement this when integrating with websocket hub
	log.Printf("Event published: %s - %v\n", event.Type, event.Payload)
}
