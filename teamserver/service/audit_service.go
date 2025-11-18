package service

import (
	"context"
	"fmt"
	"simplec2/teamserver/data"
)

// AuditQuery defines parameters for querying audit logs with filtering and pagination.
type AuditQuery struct {
	Page         int    // Page number (1-based)
	Limit        int    // Items per page
	Username     string // Filter by username
	Action       string // Filter by action type
	ResourceType string // Filter by resource type
	ResourceID   string // Filter by resource ID
	Result       string // Filter by result (success/failure)
	StartDate    string // Filter start date (ISO format)
	EndDate      string // Filter end date (ISO format)
}

// AuditService defines the interface for audit log operations.
type AuditService interface {
	// CreateAuditLog creates a new audit log entry.
	CreateAuditLog(ctx context.Context, log *data.AuditLog) error

	// GetAuditLogs retrieves audit logs with optional filtering and pagination.
	GetAuditLogs(ctx context.Context, query *AuditQuery) ([]data.AuditLog, int64, error)
}

// auditService implements the AuditService interface.
type auditService struct {
	store data.DataStore
}

// NewAuditService creates a new audit service instance.
func NewAuditService(store data.DataStore) AuditService {
	return &auditService{
		store: store,
	}
}

// CreateAuditLog creates a new audit log entry.
func (s *auditService) CreateAuditLog(ctx context.Context, log *data.AuditLog) error {
	gormStore, ok := s.store.(*data.GormStore)
	if !ok {
		return fmt.Errorf("invalid data store type")
	}

	if err := gormStore.DB.WithContext(ctx).Create(log).Error; err != nil {
		return fmt.Errorf("failed to create audit log: %w", err)
	}

	return nil
}

// GetAuditLogs retrieves audit logs with optional filtering and pagination.
func (s *auditService) GetAuditLogs(ctx context.Context, query *AuditQuery) ([]data.AuditLog, int64, error) {
	gormStore, ok := s.store.(*data.GormStore)
	if !ok {
		return nil, 0, fmt.Errorf("invalid data store type")
	}

	// Build the query with filters
	db := gormStore.DB.WithContext(ctx).Model(&data.AuditLog{})

	// Apply filters
	if query.Username != "" {
		db = db.Where("username = ?", query.Username)
	}
	if query.Action != "" {
		db = db.Where("action = ?", query.Action)
	}
	if query.ResourceType != "" {
		db = db.Where("resource_type = ?", query.ResourceType)
	}
	if query.ResourceID != "" {
		db = db.Where("resource_id = ?", query.ResourceID)
	}
	if query.Result != "" {
		db = db.Where("result = ?", query.Result)
	}
	if query.StartDate != "" {
		db = db.Where("timestamp >= ?", query.StartDate)
	}
	if query.EndDate != "" {
		db = db.Where("timestamp <= ?", query.EndDate)
	}

	// Get total count
	var total int64
	if err := db.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count audit logs: %w", err)
	}

	// Apply pagination
	offset := (query.Page - 1) * query.Limit
	db = db.Offset(offset).Limit(query.Limit)

	// Order by timestamp descending
	db = db.Order("timestamp DESC")

	// Execute query
	var logs []data.AuditLog
	if err := db.Find(&logs).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to retrieve audit logs: %w", err)
	}

	return logs, total, nil
}
