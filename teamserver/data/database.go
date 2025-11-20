package data

import (
	"fmt"
	"os"
	"path/filepath"

	"simplec2/pkg/config"
	"simplec2/pkg/logger"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
)

// DataStore defines the interface for all database operations.
type DataStore interface {
	// Beacon methods
	GetBeacons(query *BeaconQuery) ([]Beacon, int64, error)
	GetBeacon(beaconID string) (*Beacon, error)
	CreateBeacon(beacon *Beacon) error
	UpdateBeacon(beacon *Beacon) error
	DeleteBeacon(beaconID string) error

	// Task methods
	GetTask(taskID string) (*Task, error)
	GetTasksByBeaconID(beaconID string, status string) ([]Task, error)
	CreateTask(task *Task) error
	UpdateTask(task *Task) error

	// Listener methods
	GetListeners(page int, limit int) ([]Listener, int64, error)
	GetListener(name string) (*Listener, error)
	CreateListener(listener *Listener) error
	DeleteListener(name string) error

	// Session methods
	CreateSession(session *Session) error
	GetSession(tokenHash string) (*Session, error)
	UpdateSession(session *Session) error
	DeleteSession(tokenHash string) error
	GetActiveSessions() ([]Session, error)
	DeleteExpiredSessions() (int64, error)
}

// GormStore is a generic implementation of DataStore using GORM.
type GormStore struct {
	DB *gorm.DB
}

// NewDataStore is a factory function that returns a DataStore implementation
// based on the provided configuration.
func NewDataStore(cfg config.DatabaseConfig) (DataStore, error) {
	var db *gorm.DB
	var err error

	switch cfg.Type {
	case "postgres":
		db, err = gorm.Open(postgres.Open(cfg.DSN), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to postgres: %w", err)
		}
	case "sqlite":
		// Ensure the directory for the database file exists.
		if err := os.MkdirAll(filepath.Dir(cfg.Path), 0755); err != nil {
			return nil, fmt.Errorf("failed to create database directory: %w", err)
		}
		db, err = gorm.Open(sqlite.Open(cfg.Path), &gorm.Config{})
		if err != nil {
			return nil, fmt.Errorf("failed to connect to sqlite: %w", err)
		}
	default:
		return nil, fmt.Errorf("unsupported database type: %s", cfg.Type)
	}

	logger.Info("Running database migrations...")
	if err := db.AutoMigrate(&Beacon{}, &Task{}, &Listener{}, &Session{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	logger.Info("Database connection successful and schema migrated.")
	return &GormStore{DB: db}, nil
}
