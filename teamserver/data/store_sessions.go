package data

import (
	"time"
)

// CreateSession creates a new session in the database.
func (s *GormStore) CreateSession(session *Session) error {
	return s.DB.Create(session).Error
}

// GetSession retrieves a session by token hash.
func (s *GormStore) GetSession(tokenHash string) (*Session, error) {
	var session Session
	if err := s.DB.Where("token_hash = ? AND is_active = ?", tokenHash, true).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// UpdateSession updates a session in the database.
func (s *GormStore) UpdateSession(session *Session) error {
	return s.DB.Save(session).Error
}

// DeleteSession marks a session as inactive.
func (s *GormStore) DeleteSession(tokenHash string) error {
	return s.DB.Model(&Session{}).Where("token_hash = ?", tokenHash).Update("is_active", false).Error
}

// GetActiveSessions retrieves all active sessions.
func (s *GormStore) GetActiveSessions() ([]Session, error) {
	var sessions []Session
	if err := s.DB.Where("is_active = ? AND expires_at > ?", true, time.Now()).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

// DeleteExpiredSessions removes all expired sessions from the database.
func (s *GormStore) DeleteExpiredSessions() (int64, error) {
	result := s.DB.Where("expires_at < ? OR is_active = ?", time.Now(), false).Delete(&Session{})
	return result.RowsAffected, result.Error
}

// CleanupExpiredSessions periodically removes expired sessions.
func (s *GormStore) CleanupExpiredSessions(interval time.Duration) {
	ticker := time.NewTicker(interval)
	go func() {
		for range ticker.C {
			count, err := s.DeleteExpiredSessions()
			if err != nil {
				// Log error but don't stop the cleanup routine
				continue
			}
			if count > 0 {
				// Log the number of cleaned up sessions
			}
		}
	}()
}
