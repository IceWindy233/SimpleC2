package websocket

import (
	"sync"
	"testing"
	"time"
)

// TestHubConcurrent tests the hub's concurrent safety
func TestHubConcurrent(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	// Number of concurrent operations
	const numClients = 100
	const numBroadcasts = 50

	var wg sync.WaitGroup

	// Launch goroutines to register clients concurrently
	wg.Add(numClients)
	for i := 0; i < numClients; i++ {
		go func(id int) {
			defer wg.Done()

			// Create a mock client
			client := &Client{
				hub:  hub,
				conn: nil, // Mock connection for testing
				send: make(chan []byte, 256),
			}

			// Register the client
			hub.register <- client

			// Simulate some work
			time.Sleep(time.Millisecond)

			// Unregister the client
			hub.unregister <- client
		}(i)
	}

	// Launch goroutines to broadcast messages concurrently
	wg.Add(numBroadcasts)
	for i := 0; i < numBroadcasts; i++ {
		go func(id int) {
			defer wg.Done()

			// Broadcast a message
			message := []byte("test message")
			hub.Broadcast(message)

			// Simulate some work
			time.Sleep(time.Millisecond)
		}(i)
	}

	// Wait for all goroutines to complete
	wg.Wait()

	// Give some time for the hub to process remaining messages
	time.Sleep(100 * time.Millisecond)

	// Verify the hub is still functional
	if hub.clients == nil {
		t.Fatal("Hub clients map is nil")
	}

	// The test passes if no panic occurred
	t.Log("Test completed successfully without panic")
}

// TestHubStressTest performs a more intensive stress test
func TestHubStressTest(t *testing.T) {
	hub := NewHub()
	go hub.Run()

	const numRoutines = 200
	const operationsPerRoutine = 100

	var wg sync.WaitGroup

	for i := 0; i < numRoutines; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			for j := 0; j < operationsPerRoutine; j++ {
				// Alternate between registering clients and broadcasting
				if j%3 == 0 {
					// Register a client
					client := &Client{
						hub:  hub,
						conn: nil,
						send: make(chan []byte, 256),
					}
					hub.register <- client

					// Keep the client around for a bit
					time.Sleep(time.Microsecond * 100)

					// Unregister the client
					hub.unregister <- client
				} else {
					// Broadcast a message
					hub.Broadcast([]byte("stress test"))
				}
			}
		}(i)
	}

	wg.Wait()

	// Give time for cleanup
	time.Sleep(200 * time.Millisecond)

	// Log final client count
	t.Logf("Final client count: %d", hub.clients.Len())
	t.Log("Stress test completed successfully without panic")
}
