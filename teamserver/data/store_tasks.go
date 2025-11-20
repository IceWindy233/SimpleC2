package data

// --- Task Methods ---

func (s *GormStore) GetTask(taskID string) (*Task, error) {
	var task Task
	err := s.DB.Where("task_id = ?", taskID).First(&task).Error
	return &task, err
}

func (s *GormStore) GetTasksByBeaconID(beaconID string, status string) ([]Task, error) {
	var tasks []Task
	db := s.DB.Where("beacon_id = ?", beaconID)
	if status != "" {
		db = db.Where("status = ?", status)
	}
	err := db.Find(&tasks).Error
	return tasks, err
}

func (s *GormStore) CreateTask(task *Task) error {
	return s.DB.Create(task).Error
}

func (s *GormStore) UpdateTask(task *Task) error {
	return s.DB.Save(task).Error
}
