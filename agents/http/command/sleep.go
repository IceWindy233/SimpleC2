package command

import (
	"encoding/json"
	"fmt"
	"log"
	"strconv"
	"time"
)

// CommandIDSleep Sleep 命令 ID
const CommandIDSleep uint32 = 5

// SleepInterval 全局 sleep 间隔，供 main.go 使用
var SleepInterval = 5 * time.Second

// SleepCommand 实现 sleep 命令
type SleepCommand struct{}

func init() {
	Register(&SleepCommand{})
}

func (c *SleepCommand) ID() uint32 {
	return CommandIDSleep
}

func (c *SleepCommand) Name() string {
	return "sleep"
}

func (c *SleepCommand) Execute(task *Task) ([]byte, error) {
	var newSleep int32

	log.Printf("Sleep task received. TaskID: %s, Arguments length: %d, Arguments raw: %q",
		task.TaskID, len(task.Arguments), string(task.Arguments))

	if len(task.Arguments) == 0 {
		return nil, fmt.Errorf("empty sleep arguments")
	}

	// 尝试解析为 JSON 数字或字符串
	var sleepValue interface{}
	if err := json.Unmarshal(task.Arguments, &sleepValue); err != nil {
		return nil, fmt.Errorf("invalid JSON: %v", err)
	}

	// 处理数字或字符串格式
	switch v := sleepValue.(type) {
	case float64: // JSON 数字: 30
		newSleep = int32(v)
	case string: // JSON 字符串: "30"
		parsed, err := parseInt32(v)
		if err != nil {
			return nil, fmt.Errorf("invalid sleep value: %s", v)
		}
		newSleep = parsed
	default:
		return nil, fmt.Errorf("unsupported sleep argument type: %T", v)
	}

	// 验证范围
	if newSleep < 1 || newSleep > 3600 {
		return nil, fmt.Errorf("sleep value must be between 1 and 3600 seconds, got %d", newSleep)
	}

	SleepInterval = time.Duration(newSleep) * time.Second
	log.Printf("Updated check-in interval to %s", SleepInterval)
	return []byte(fmt.Sprintf("Sleep interval set to %d seconds", newSleep)), nil
}

// parseInt32 解析字符串为 int32
func parseInt32(s string) (int32, error) {
	result, err := strconv.ParseInt(s, 10, 32)
	if err != nil {
		return 0, err
	}
	return int32(result), nil
}
