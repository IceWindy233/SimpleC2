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

// BeaconQuery defines parameters for querying beacons.
type BeaconQuery struct {
	Page   int
	Limit  int
	Search string
	Status string
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

// Session represents a user session for tracking login state.
type Session struct {
	ID        uint      `gorm:"primarykey"`
	CreatedAt time.Time `gorm:"not null;index"`
	UpdatedAt time.Time `gorm:"not null;index"`
	ExpiresAt time.Time `gorm:"not null;index"` // Session expiration time

	UserID   string `gorm:"not null;index"` // User identifier (username from JWT)
	TokenHash string `gorm:"not null;uniqueIndex"` // JWT token hash for validation
	IPAddress string `gorm:"not null;index"` // Client IP address
	UserAgent string // Client user agent
	IsActive  bool   `gorm:"default:true;index"` // Whether the session is active
}
