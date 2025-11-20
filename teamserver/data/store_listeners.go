package data

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
