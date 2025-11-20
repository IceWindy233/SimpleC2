package data

import (
	"time"
)

// --- Beacon Methods ---

func (s *GormStore) GetBeacons(query *BeaconQuery) ([]Beacon, int64, error) {
	var beacons []Beacon
	var total int64
	db := s.DB.Model(&Beacon{})

	if query.Search != "" {
		db = db.Where("hostname LIKE ? OR username LIKE ? OR internal_ip LIKE ?", "%"+query.Search+"%", "%"+query.Search+"%", "%"+query.Search+"%")
	}
	if query.Status == "active" {
		// Active means seen in the last 30 seconds
		cutoff := time.Now().Add(-30 * time.Second)
		db = db.Where("last_seen >= ?", cutoff)
	} else if query.Status == "inactive" {
		// Inactive means not seen in the last 30 seconds
		cutoff := time.Now().Add(-30 * time.Second)
		db = db.Where("last_seen < ?", cutoff)
	} else if query.Status != "" {
		// Fallback for other statuses if any
		db = db.Where("status = ?", query.Status)
	}

	err := db.Count(&total).Error
	if err != nil {
		return nil, 0, err
	}

	offset := (query.Page - 1) * query.Limit
	err = db.Limit(query.Limit).Offset(offset).Find(&beacons).Error
	return beacons, total, err
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
