package commands

import (
	"fmt"
	"strconv"
	"simplec2/teamserver/data"
)

// CommandIDKill Kill 命令 ID
const CommandIDKill uint32 = 14

// KillCommand implements the CommandConverter interface for the kill command.
type KillCommand struct{}

func init() {
	Register(&KillCommand{})
}

func (c *KillCommand) Name() string {
	return "kill"
}

func (c *KillCommand) CommandID() uint32 {
	return CommandIDKill
}

func (c *KillCommand) Convert(task *data.Task) ([]byte, error) {
	// The task.Arguments from the TeamServer will be the PID as a string.
	// We just pass it through to the agent.
	if task.Arguments == "" {
		return nil, fmt.Errorf("kill command requires a PID argument")
	}
	
	// Validate if it's a valid integer PID
	_, err := strconv.Atoi(task.Arguments)
	if err != nil {
		return nil, fmt.Errorf("invalid PID argument for kill command: %v", err)
	}

	return []byte(task.Arguments), nil
}
