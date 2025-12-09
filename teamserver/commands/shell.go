package commands

import (
	"simplec2/teamserver/data"
)

// CommandIDShell Shell 命令 ID
const CommandIDShell uint32 = 1

// ShellConverter Shell 命令转换器
type ShellConverter struct{}

func init() {
	Register(&ShellConverter{})
}

func (c *ShellConverter) Name() string {
	return "shell"
}

func (c *ShellConverter) CommandID() uint32 {
	return CommandIDShell
}

func (c *ShellConverter) Convert(task *data.Task) ([]byte, error) {
	// Shell 命令直接使用参数文本
	return []byte(task.Arguments), nil
}
