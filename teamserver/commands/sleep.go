package commands

import (
	"encoding/json"
	"fmt"
	"strconv"
	"strings"

	"simplec2/teamserver/data"
)

// CommandIDSleep Sleep 命令 ID (与 agent 保持一致)
const CommandIDSleep uint32 = 5

// SleepArgs 定义 sleep 命令的参数结构，与 agent 保持一致
type SleepArgs struct {
	Sleep  int32 `json:"sleep"`
	Jitter int32 `json:"jitter"` // Percentage, 0-99
}

// SleepCommand 实现 sleep 命令的转换器
type SleepCommand struct{}

func init() {
	Register(&SleepCommand{})
}

func (c *SleepCommand) Name() string {
	return "sleep"
}

func (c *SleepCommand) CommandID() uint32 {
	return CommandIDSleep
}

func (c *SleepCommand) Convert(task *data.Task) ([]byte, error) {
	var sleep int32 = 0
	var jitter int32 = 0 // Default jitter

	if task.Arguments == "" {
		return nil, fmt.Errorf("sleep command requires arguments: <seconds> [jitter_percent]")
	}

	parts := strings.Fields(task.Arguments)
	if len(parts) > 0 {
		parsedSleep, err := strconv.ParseInt(parts[0], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid sleep seconds: %v", err)
		}
		sleep = int32(parsedSleep)
	}

	if len(parts) > 1 {
		parsedJitter, err := strconv.ParseInt(parts[1], 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid jitter percentage: %v", err)
		}
		jitter = int32(parsedJitter)
	}

	// Validate sleep and jitter before sending
	if sleep < 1 || sleep > 3600 {
		return nil, fmt.Errorf("sleep value must be between 1 and 3600 seconds, got %d", sleep)
	}
	if jitter < 0 || jitter > 99 {
		return nil, fmt.Errorf("jitter value must be between 0 and 99 percent, got %d", jitter)
	}

	args := SleepArgs{
		Sleep:  sleep,
		Jitter: jitter,
	}

	jsonArgs, err := json.Marshal(args)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sleep arguments: %v", err)
	}

	return jsonArgs, nil
}
