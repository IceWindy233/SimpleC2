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
	BeaconID      string    `gorm:"uniqueIndex;not null" json:"BeaconID"`
	SessionKey    []byte    `json:"-"`
	Listener      string    `json:"Listener"`
	RemoteAddr    string    `json:"RemoteAddr"`
	Status        string    `gorm:"default:'active'" json:"Status"`
	FirstSeen     time.Time `json:"FirstSeen"`
	LastSeen      time.Time `json:"LastSeen"`
	Sleep         int       `json:"Sleep"`
	Jitter        int       `json:"Jitter"`

	// Metadata from the beacon
	OS              string `json:"OS"`
	Arch            string `json:"Arch"`
	Username        string `json:"Username"`
	Hostname        string `json:"Hostname"`
	InternalIP      string `json:"InternalIP"`
	ProcessName     string `json:"ProcessName"`
	PID             int32  `json:"PID"`
	IsHighIntegrity bool   `json:"IsHighIntegrity"`
	Note            string `json:"Note"` // User notes for the beacon
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
	Source    string // e.g., "console", "ui", "api"
}

// Listener represents a listener configuration in the database.
type Listener struct {
	gorm.Model
	Name   string `gorm:"uniqueIndex;not null"`
	Type   string // e.g., "http", "dns"
	Config string `gorm:"type:text"` // Store listener-specific config as a JSON string

	// Runtime status (not persisted)
	Active bool `gorm:"-" json:"active"`
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

// IssuedCertificate tracks certificates issued to listeners for revocation purposes.
type IssuedCertificate struct {
	gorm.Model
	SerialNumber string     `gorm:"uniqueIndex;not null"` // Certificate Serial Number (decimal string)
	CommonName   string     `gorm:"index"`
	ListenerName string     `gorm:"index"`
	Revoked      bool       `gorm:"default:false;index"`
	RevokedAt    *time.Time
}
