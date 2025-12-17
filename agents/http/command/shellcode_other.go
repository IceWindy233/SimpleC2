//go:build !windows

package command

import (
	"fmt"
)

// CommandIDShellcode Shellcode 命令 ID
const CommandIDShellcode uint32 = 15

// ShellcodeCommand implements the shellcode execution command.
type ShellcodeCommand struct{}

func init() {
	Register(&ShellcodeCommand{})
}

func (c *ShellcodeCommand) ID() uint32 {
	return CommandIDShellcode
}

func (c *ShellcodeCommand) Name() string {
	return "shellcode"
}

func (c *ShellcodeCommand) Execute(task *Task) ([]byte, error) {
	return nil, fmt.Errorf("shellcode execution is only supported on Windows")
}
