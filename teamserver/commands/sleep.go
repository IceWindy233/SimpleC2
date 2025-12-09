package commands

import (
	"simplec2/pkg/logger"
	"simplec2/teamserver/data"
)

// CommandIDSleep Sleep 命令 ID
const CommandIDSleep uint32 = 5

// SleepConverter Sleep 命令转换器
type SleepConverter struct{}

func init() {
	Register(&SleepConverter{})
}

func (c *SleepConverter) Name() string {
	return "sleep"
}

func (c *SleepConverter) CommandID() uint32 {
	return CommandIDSleep
}

func (c *SleepConverter) Convert(task *data.Task) ([]byte, error) {
	logger.Infof("Processing sleep task %s: Arguments=%q (len=%d)",
		task.TaskID, task.Arguments, len(task.Arguments))

	if task.Arguments != "" {
		return []byte(task.Arguments), nil
	}
	// 默认 sleep 值
	return []byte("5"), nil
}
