package data

// --- Listener Methods ---

func (s *GormStore) GetListeners() ([]Listener, error) {
	var listeners []Listener
	err := s.DB.Find(&listeners).Error
	return listeners, err
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
