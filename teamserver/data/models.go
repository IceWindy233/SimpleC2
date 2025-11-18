package data

import (
	"time"

	"gorm.io/gorm"
)

// Beacon represents a registered implant in the database.
type Beacon struct {
	ID            uint           `gorm:"primarykey"`
	CreatedAt     time.Time
	UpdatedAt     time.Time
	DeletedAt     gorm.DeletedAt `gorm:"index"`

	// Beacon-specific fields
	BeaconID      string    `gorm:"uniqueIndex;not null"`
	SessionKey    []byte
	Listener      string
	RemoteAddr    string
	Status        string `gorm:"default:'active'"` // e.g., "active", "inactive", "lost", "exiting"
	FirstSeen     time.Time
	LastSeen      time.Time
	Sleep         int

	// Metadata from the beacon
	OS              string
	Arch            string
	Username        string
	Hostname        string
	InternalIP      string
	ProcessName     string
	PID             int32
	IsHighIntegrity bool
}

// Task represents a command to be executed by a beacon.
type Task struct {
	gorm.Model
	TaskID    string `gorm:"uniqueIndex;not null"`
	BeaconID  string `gorm:"index"`
	Command   string
	Arguments string
	Status    string // e.g., "queued", "dispatched", "completed", "error"
	Output    string
}

// Listener represents a listener configuration in the database.
type Listener struct {
	gorm.Model
	Name   string `gorm:"uniqueIndex;not null"`
	Type   string // e.g., "http", "dns"
	Config string `gorm:"type:text"` // Store listener-specific config as a JSON string
}

// AuditLog represents an audit trail entry for tracking user actions.
type AuditLog struct {
	ID           uint      `gorm:"primarykey"`
	Timestamp    time.Time `gorm:"not null;index"`
	Username     string    `gorm:"not null;index"` // Who performed the action
	Action       string    `gorm:"not null;index"` // Action type: DELETE_BEACON, CREATE_TASK, etc.
	ResourceType string    // Resource type: beacon, task, listener, etc.
	ResourceID   string    // Resource identifier
	IPAddress    string    // Client IP address
	Result       string    // success or failure
	Details      string    `gorm:"type:text"` // Additional details about the action
}

