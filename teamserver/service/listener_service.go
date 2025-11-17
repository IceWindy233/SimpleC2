package service

import (
	"context"
	"fmt"

	"simplec2/teamserver/data"
)

// ListenerService defines the interface for listener-related business logic.
type ListenerService interface {
	// GetListener retrieves a listener by its name.
	GetListener(ctx context.Context, name string) (*data.Listener, error)

	// CreateListener creates a new listener configuration.
	CreateListener(ctx context.Context, name string, listenerType string, config string) (*data.Listener, error)

	// DeleteListener deletes a listener configuration.
	DeleteListener(ctx context.Context, name string) error

	// ListListeners retrieves all listeners.
	ListListeners(ctx context.Context) ([]data.Listener, error)
}

// listenerService implements the ListenerService interface.
type listenerService struct {
	store data.DataStore
}

// NewListenerService creates a new instance of listenerService.
func NewListenerService(store data.DataStore) ListenerService {
	return &listenerService{
		store: store,
	}
}

// GetListener retrieves a listener by its name.
func (s *listenerService) GetListener(ctx context.Context, name string) (*data.Listener, error) {
	listener, err := s.store.GetListener(name)
	if err != nil {
		return nil, fmt.Errorf("failed to get listener: %w", err)
	}
	return listener, nil
}

// CreateListener creates a new listener configuration.
func (s *listenerService) CreateListener(ctx context.Context, name string, listenerType string, config string) (*data.Listener, error) {
	// Check if listener already exists
	if _, err := s.store.GetListener(name); err == nil {
		return nil, fmt.Errorf("listener with name '%s' already exists", name)
	}

	listener := &data.Listener{
		Name:   name,
		Type:   listenerType,
		Config: config,
	}

	if err := s.store.CreateListener(listener); err != nil {
		return nil, fmt.Errorf("failed to create listener: %w", err)
	}

	return listener, nil
}

// DeleteListener deletes a listener configuration.
func (s *listenerService) DeleteListener(ctx context.Context, name string) error {
	// First, ensure listener exists
	if _, err := s.store.GetListener(name); err != nil {
		return fmt.Errorf("listener not found: %w", err)
	}

	if err := s.store.DeleteListener(name); err != nil {
		return fmt.Errorf("failed to delete listener: %w", err)
	}

	return nil
}

// ListListeners retrieves all listeners.
func (s *listenerService) ListListeners(ctx context.Context) ([]data.Listener, error) {
	listeners, err := s.store.GetListeners()
	if err != nil {
		return nil, fmt.Errorf("failed to list listeners: %w", err)
	}
	return listeners, nil
}
