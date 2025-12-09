package commands

import (
	"simplec2/teamserver/data"
)

// CommandIDExit Exit 命令 ID
const CommandIDExit uint32 = 4

// ExitConverter Exit 命令转换器
type ExitConverter struct{}

func init() {
	Register(&ExitConverter{})
}

func (c *ExitConverter) Name() string {
	return "exit"
}

func (c *ExitConverter) CommandID() uint32 {
	return CommandIDExit
}

func (c *ExitConverter) Convert(task *data.Task) ([]byte, error) {
	// Exit 命令不需要参数
	return nil, nil
}
