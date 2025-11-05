package data

import (
	"fmt"
	"log"
	"os"
	"path/filepath"

	"gorm.io/driver/postgres"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"simplec2/pkg/config"
)

// DataStore defines the interface for all database operations.
type DataStore interface {
	// Beacon methods
	GetBeacons() ([]Beacon, error)
	GetBeacon(beaconID string) (*Beacon, error)
	CreateBeacon(beacon *Beacon) error
	UpdateBeacon(beacon *Beacon) error
	DeleteBeacon(beaconID string) error

	// Task methods
	GetTask(taskID string) (*Task, error)
	GetTasksByBeaconID(beaconID string) ([]Task, error)
	CreateTask(task *Task) error
	UpdateTask(task *Task) error

	// Listener methods
	GetListeners() ([]Listener, error)
	GetListener(name string) (*Listener, error)
	CreateListener(listener *Listener) error
	DeleteListener(name string) error
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

	log.Println("Running database migrations...")
	if err := db.AutoMigrate(&Beacon{}, &Task{}, &Listener{}); err != nil {
		return nil, fmt.Errorf("failed to auto-migrate database: %w", err)
	}

	log.Println("Database connection successful and schema migrated.")
	return &GormStore{DB: db}, nil
}

// --- Beacon Methods ---

func (s *GormStore) GetBeacons() ([]Beacon, error) {
	var beacons []Beacon
	err := s.DB.Find(&beacons).Error
	return beacons, err
}

func (s *GormStore) GetBeacon(beaconID string) (*Beacon, error) {
	var beacon Beacon
	err := s.DB.Where("beacon_id = ?", beaconID).First(&beacon).Error
	return &beacon, err
}

func (s *GormStore) CreateBeacon(beacon *Beacon) error {
	return s.DB.Create(beacon).Error
}

func (s *GormStore) UpdateBeacon(beacon *Beacon) error {
	return s.DB.Save(beacon).Error
}

func (s *GormStore) DeleteBeacon(beaconID string) error {
	return s.DB.Where("beacon_id = ?", beaconID).Delete(&Beacon{}).Error
}

// --- Task Methods ---

func (s *GormStore) GetTask(taskID string) (*Task, error) {
	var task Task
	err := s.DB.Where("task_id = ?", taskID).First(&task).Error
	return &task, err
}

func (s *GormStore) GetTasksByBeaconID(beaconID string) ([]Task, error) {
	var tasks []Task
	err := s.DB.Where("beacon_id = ?", beaconID).Find(&tasks).Error
	return tasks, err
}

func (s *GormStore) CreateTask(task *Task) error {
	return s.DB.Create(task).Error
}

func (s*GormStore) UpdateTask(task *Task) error {
    return s.DB.Save(task).Error
}

// --- Listener Methods ---

func (s *GormStore) GetListeners() ([]Listener, error) {
	var listeners []Listener
	err := s.DB.Find(&listeners).Error
	return listeners, err
}

func (s *GormStore) GetListener(name string) (*Listener, error) {
	var listener Listener
	err := s.DB.Where("name = ?", name).First(&listener).Error
	return &listener, err
}

func (s *GormStore) CreateListener(listener *Listener) error {
	return s.DB.Create(listener).Error
}

func (s *GormStore) DeleteListener(name string) error {
	return s.DB.Where("name = ?", name).Delete(&Listener{}).Error
}