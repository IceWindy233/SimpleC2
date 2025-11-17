package events

import (
	"time"
)

// EventType represents the type of event that occurred.
type EventType string

// Event types for the SimpleC2 system.
const (
	// Beacon events
	BeaconRegistered     EventType = "BEACON_REGISTERED"
	BeaconCheckin                  = "BEACON_CHECKIN"
	BeaconMetadataUpdated          = "BEACON_METADATA_UPDATED"
	BeaconDeleted                  = "BEACON_DELETED"
	BeaconExited                   = "BEACON_EXITED"

	// Task events
	TaskQueued     EventType = "TASK_QUEUED"
	TaskDispatched EventType = "TASK_DISPATCHED"
	TaskCompleted  EventType = "TASK_COMPLETED"
	TaskFailed     EventType = "TASK_FAILED"
	TaskCanceled   EventType = "TASK_CANCELED"

	// File events
	FileDownloadStarted   EventType = "FILE_DOWNLOAD_STARTED"
	FileDownloadCompleted EventType = "FILE_DOWNLOAD_COMPLETED"
	FileUploadCompleted   EventType = "FILE_UPLOAD_COMPLETED"

	// Listener events
	ListenerStarted EventType = "LISTENER_STARTED"
	ListenerStopped EventType = "LISTENER_STOPPED"

	// Client events
	ClientConnected     EventType = "CLIENT_CONNECTED"
	ClientAuthenticated EventType = "CLIENT_AUTHENTICATED"
)

// Event represents a system event that can be published and subscribed to.
type Event struct {
	Type      EventType   `json:"type"`
	Timestamp time.Time   `json:"timestamp"`
	Payload   interface{} `json:"payload"`
}

// NewEvent creates a new event with the current timestamp.
func NewEvent(eventType EventType, payload interface{}) Event {
	return Event{
		Type:      eventType,
		Timestamp: time.Now(),
		Payload:   payload,
	}
}
