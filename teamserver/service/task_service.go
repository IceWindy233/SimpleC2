package service

import (
	"context"
	"fmt"

	"simplec2/teamserver/data"

	"github.com/google/uuid"
)

// TaskService defines the interface for task-related business logic.
type TaskService interface {
	// GetTask retrieves a task by its ID.
	GetTask(ctx context.Context, taskID string) (*data.Task, error)

	// GetTasksByBeaconID retrieves all tasks for a specific beacon.
	GetTasksByBeaconID(ctx context.Context, beaconID string, status string) ([]data.Task, error)

	// CreateTask creates a new task for a beacon.
	CreateTask(ctx context.Context, beaconID string, command string, arguments string, source string) (*data.Task, error)

	// UpdateTask updates a task.
	UpdateTask(ctx context.Context, task *data.Task) error
}

// taskService implements the TaskService interface.
type taskService struct {
	store data.DataStore
}

// NewTaskService creates a new instance of taskService.
func NewTaskService(store data.DataStore) TaskService {
	return &taskService{
		store: store,
	}
}

// GetTask retrieves a task by its ID.
func (s *taskService) GetTask(ctx context.Context, taskID string) (*data.Task, error) {
	task, err := s.store.GetTask(taskID)
	if err != nil {
		return nil, fmt.Errorf("failed to get task: %w", err)
	}
	return task, nil
}

// GetTasksByBeaconID retrieves all tasks for a specific beacon.
func (s *taskService) GetTasksByBeaconID(ctx context.Context, beaconID string, status string) ([]data.Task, error) {
	// First, ensure beacon exists
	if _, err := s.store.GetBeacon(beaconID); err != nil {
		return nil, fmt.Errorf("beacon not found: %w", err)
	}

	tasks, err := s.store.GetTasksByBeaconID(beaconID, status)
	if err != nil {
		return nil, fmt.Errorf("failed to get tasks for beacon: %w", err)
	}
	return tasks, nil
}

// CreateTask creates a new task for a beacon.
func (s *taskService) CreateTask(ctx context.Context, beaconID string, command string, arguments string, source string) (*data.Task, error) {
	// First, ensure beacon exists
	if _, err := s.store.GetBeacon(beaconID); err != nil {
		return nil, fmt.Errorf("beacon not found: %w", err)
	}

	task := &data.Task{
		TaskID:    uuid.New().String(),
		BeaconID:  beaconID,
		Command:   command,
		Arguments: arguments,
		Status:    "queued",
		Source:    source,
	}

	if err := s.store.CreateTask(task); err != nil {
		return nil, fmt.Errorf("failed to create task: %w", err)
	}

	return task, nil
}

// UpdateTask updates a task.
func (s *taskService) UpdateTask(ctx context.Context, task *data.Task) error {
	if err := s.store.UpdateTask(task); err != nil {
		return fmt.Errorf("failed to update task: %w", err)
	}
	return nil
}
