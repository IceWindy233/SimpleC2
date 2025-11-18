package service

import (
	"crypto/sha256"
	"encoding/hex"
	"simplec2/teamserver/data"
	"time"
)

// SessionService handles session management operations.
type SessionService struct {
	store data.DataStore
}

// NewSessionService creates a new session service.
func NewSessionService(store data.DataStore) *SessionService {
	return &SessionService{store: store}
}

// CreateSession creates a new session for a user.
func (s *SessionService) CreateSession(userID, token, ipAddress, userAgent string, duration time.Duration) (*data.Session, error) {
	tokenHash := hashToken(token)

	session := &data.Session{
		UserID:    userID,
		TokenHash: tokenHash,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		ExpiresAt: time.Now().Add(duration),
		IsActive:  true,
	}

	if err := s.store.CreateSession(session); err != nil {
		return nil, err
	}

	return session, nil
}

// ValidateSession checks if a session is valid and not expired.
func (s *SessionService) ValidateSession(token string) (*data.Session, bool) {
	tokenHash := hashToken(token)

	session, err := s.store.GetSession(tokenHash)
	if err != nil {
		return nil, false
	}

	// Check if session is expired
	if time.Now().After(session.ExpiresAt) {
		// Mark as inactive
		session.IsActive = false
		s.store.UpdateSession(session)
		return nil, false
	}

	return session, true
}

// InvalidateSession invalidates a session (logout).
func (s *SessionService) InvalidateSession(token string) error {
	tokenHash := hashToken(token)
	return s.store.DeleteSession(tokenHash)
}

// GetActiveSessions returns all active sessions.
func (s *SessionService) GetActiveSessions() ([]*data.Session, error) {
	sessions, err := s.store.GetActiveSessions()
	if err != nil {
		return nil, err
	}

	result := make([]*data.Session, len(sessions))
	for i := range sessions {
		result[i] = &sessions[i]
	}

	return result, nil
}

// CleanupExpiredSessions removes all expired sessions.
func (s *SessionService) CleanupExpiredSessions() (int64, error) {
	return s.store.DeleteExpiredSessions()
}

// StartCleanupRoutine starts a background routine to cleanup expired sessions.
func (s *SessionService) StartCleanupRoutine(interval time.Duration) {
	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for range ticker.C {
			count, err := s.CleanupExpiredSessions()
			if err != nil {
				continue
			}
			if count > 0 {
			}
		}
	}()
}

// hashToken creates a SHA256 hash of the token for storage.
func hashToken(token string) string {
	hash := sha256.Sum256([]byte(token))
	return hex.EncodeToString(hash[:])
}
