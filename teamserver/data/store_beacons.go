package data

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
