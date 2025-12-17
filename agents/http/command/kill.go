package command

import (
	"fmt"
	"os"
	"strconv"
)

// CommandIDKill Kill 命令 ID
const CommandIDKill uint32 = 14

// KillCommand implements the kill command execution.
type KillCommand struct{}

func init() {
	Register(&KillCommand{})
}

func (c *KillCommand) ID() uint32 {
	return CommandIDKill
}

func (c *KillCommand) Name() string {
	return "kill"
}

func (c *KillCommand) Execute(task *Task) ([]byte, error) {
	// task.Arguments is expected to be the PID as a string
	pidStr := string(task.Arguments)
	pid, err := strconv.Atoi(pidStr)
	if err != nil {
		return nil, fmt.Errorf("invalid PID provided: %s", pidStr)
	}

	process, err := os.FindProcess(pid)
	if err != nil {
		return nil, fmt.Errorf("failed to find process with PID %d: %v", pid, err)
	}

	// Attempt to kill the process
	err = process.Kill()
	if err != nil {
		return nil, fmt.Errorf("failed to kill process with PID %d: %v", pid, err)
	}

	return []byte(fmt.Sprintf("Successfully killed process with PID %d", pid)), nil
}
