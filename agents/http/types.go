package main

// Task 任务结构，从 TeamServer 接收的任务
type Task struct {
	TaskID    string `json:"task_id"`
	CommandID uint32 `json:"command_id"`
	Arguments []byte `json:"arguments"`
}

// CheckInResponse 检入响应结构
type CheckInResponse struct {
	Tasks    []*Task `json:"tasks"`
	NewSleep int32   `json:"new_sleep,omitempty"`
}

// StagingResponse 阶段响应结构
type StagingResponse struct {
	AssignedBeaconID string `json:"assigned_beacon_id"`
}

// BeaconMetadata beacon 元数据
type BeaconMetadata struct {
	PID             int    `json:"pid"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	Username        string `json:"username"`
	Hostname        string `json:"hostname"`
	InternalIP      string `json:"internal_ip"`
	ProcessName     string `json:"process_name"`
	IsHighIntegrity bool   `json:"is_high_integrity"`
}

// FileInfo 文件信息（已移至 command/file.go，此处保留用于兼容）
type FileInfo struct {
	Name        string `json:"name"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	LastModTime string `json:"last_mod_time"`
}
