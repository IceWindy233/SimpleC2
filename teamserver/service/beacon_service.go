package service

import (
	"context"
	"fmt"

	"simplec2/pkg/bridge"
	"simplec2/teamserver/data"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

// BeaconService defines the interface for beacon-related business logic.
type BeaconService interface {
	// RegisterBeacon creates a new beacon record when a beacon first checks in.
	RegisterBeacon(ctx context.Context, metadata *bridge.BeaconMetadata, listener string) (*data.Beacon, error)

	// DeleteBeacon soft deletes a beacon and creates an exit task.
	DeleteBeacon(ctx context.Context, beaconID string) error

	// UpdateBeaconLastSeen updates the LastSeen timestamp for a beacon.
	UpdateBeaconLastSeen(ctx context.Context, beaconID string) error

	// GetBeacon retrieves a beacon by its ID.
	GetBeacon(ctx context.Context, beaconID string) (*data.Beacon, error)

	// ListBeacons retrieves all beacons with optional filtering and pagination.
	ListBeacons(ctx context.Context, query *ListQuery) ([]data.Beacon, int64, error)

	// SetBeaconSleep updates the sleep interval for a beacon.
	SetBeaconSleep(ctx context.Context, beaconID string, sleep int) error
}

// ListQuery defines parameters for paginated and filtered queries.
type ListQuery struct {
	Page   int    `form:"page,default=1"`   // Page number (1-based)
	Limit  int    `form:"limit,default=20"` // Items per page
	Search string `form:"search"`           // Optional search/filter term
	Status string `form:"status"`           // Optional status filter
}

// beaconService implements the BeaconService interface.
type beaconService struct {
	store data.DataStore
}

// NewBeaconService creates a new instance of beaconService.
func NewBeaconService(store data.DataStore) BeaconService {
	return &beaconService{
		store: store,
	}
}

// RegisterBeacon creates a new beacon record when a beacon first checks in.
func (s *beaconService) RegisterBeacon(ctx context.Context, metadata *bridge.BeaconMetadata, listener string) (*data.Beacon, error) {
	beacon := &data.Beacon{
		BeaconID:        metadata.BeaconId,
		Listener:        listener,
		Status:          "active",
		OS:              metadata.Os,
		Arch:            metadata.Arch,
		Username:        metadata.Username,
		Hostname:        metadata.Hostname,
		InternalIP:      metadata.InternalIp,
		ProcessName:     metadata.ProcessName,
		PID:             metadata.Pid,
		IsHighIntegrity: metadata.IsHighIntegrity,
	}

	if err := s.store.CreateBeacon(beacon); err != nil {
		return nil, fmt.Errorf("failed to create beacon: %w", err)
	}

	return beacon, nil
}

// DeleteBeacon soft deletes a beacon and creates an exit task.
func (s *beaconService) DeleteBeacon(ctx context.Context, beaconID string) error {
	// Get the underlying GormStore to use transactions
	gormStore, ok := s.store.(*data.GormStore)
	if !ok {
		return fmt.Errorf("invalid data store type")
	}

	// Wrap the operations in a transaction to ensure atomicity
	err := gormStore.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Ensure beacon exists before proceeding (within the transaction)
		var beacon data.Beacon
		if err := tx.Where("beacon_id = ?", beaconID).First(&beacon).Error; err != nil {
			return err // Beacon not found, will cause rollback
		}

		// 2. Create an exit task for the beacon
		exitTask := data.Task{
			TaskID:    "task-exit-" + uuid.New().String(),
			BeaconID:  beaconID,
			Command:   "exit",
			Arguments: "",
			Status:    "queued",
		}
		if err := tx.Create(&exitTask).Error; err != nil {
			return err // Task creation failed, will cause rollback
		}

		// 3. Soft-delete the beacon
		if err := tx.Where("beacon_id = ?", beaconID).Delete(&data.Beacon{}).Error; err != nil {
			return err // Deletion failed, will cause rollback
		}

		// Return nil to commit the transaction
		return nil
	})

	if err != nil {
		return fmt.Errorf("failed to delete beacon: %w", err)
	}

	return nil
}

// UpdateBeaconLastSeen updates the LastSeen timestamp for a beacon.
func (s *beaconService) UpdateBeaconLastSeen(ctx context.Context, beaconID string) error {
	// Get the beacon first
	beacon, err := s.store.GetBeacon(beaconID)
	if err != nil {
		return fmt.Errorf("beacon not found: %w", err)
	}

	// Update the last seen time
	if err := s.store.UpdateBeacon(beacon); err != nil {
		return fmt.Errorf("failed to update beacon last seen: %w", err)
	}
	return nil
}

// GetBeacon retrieves a beacon by its ID.
func (s *beaconService) GetBeacon(ctx context.Context, beaconID string) (*data.Beacon, error) {
	beacon, err := s.store.GetBeacon(beaconID)
	if err != nil {
		return nil, fmt.Errorf("failed to get beacon: %w", err)
	}
	return beacon, nil
}

// ListBeacons retrieves all beacons with optional filtering and pagination.
func (s *beaconService) ListBeacons(ctx context.Context, query *ListQuery) ([]data.Beacon, int64, error) {
	storeQuery := &data.BeaconQuery{
		Page:   query.Page,
		Limit:  query.Limit,
		Search: query.Search,
		Status: query.Status,
	}
	beacons, total, err := s.store.GetBeacons(storeQuery)
	if err != nil {
		return nil, 0, fmt.Errorf("failed to list beacons: %w", err)
	}
	return beacons, total, nil
}

// SetBeaconSleep updates the sleep interval for a beacon.
func (s *beaconService) SetBeaconSleep(ctx context.Context, beaconID string, sleep int) error {
	// Get the beacon first to ensure it exists
	beacon, err := s.store.GetBeacon(beaconID)
	if err != nil {
		return fmt.Errorf("beacon not found: %w", err)
	}

	// Update the sleep value
	beacon.Sleep = sleep
	if err := s.store.UpdateBeacon(beacon); err != nil {
		return fmt.Errorf("failed to update beacon sleep: %w", err)
	}

	return nil
}
