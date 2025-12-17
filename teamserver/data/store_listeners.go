package data

import (
	"time"
)

// --- Listener Methods ---

func (s *GormStore) GetListeners(page int, limit int) ([]Listener, int64, error) {
	var listeners []Listener
	var total int64
	db := s.DB.Model(&Listener{})

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (page - 1) * limit
	err = db.Limit(limit).Offset(offset).Find(&listeners).Error
	return listeners, total, err
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

// --- Certificate Methods ---

func (s *GormStore) CreateIssuedCertificate(cert *IssuedCertificate) error {
	return s.DB.Create(cert).Error
}

func (s *GormStore) RevokeCertificatesByListener(listenerName string) error {
	now := time.Now()
	result := s.DB.Model(&IssuedCertificate{}).Where("listener_name = ?", listenerName).
		Updates(map[string]interface{}{"revoked": true, "revoked_at": &now})
	return result.Error
}

func (s *GormStore) IsCertificateRevoked(serialNumber string) (bool, error) {
	var cert IssuedCertificate
	err := s.DB.Where("serial_number = ?", serialNumber).First(&cert).Error
	if err != nil {
		// Fail Closed: If record not found or DB error, assume revoked/invalid.
		// We return nil for error to signal "check completed, result is revoked".
		// You could log the specific error here if needed.
		return true, nil 
	}
	return cert.Revoked, nil
}
