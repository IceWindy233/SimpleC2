package command

import (
	"log"
	"os"
)

// CommandIDExit Exit 命令 ID
const CommandIDExit uint32 = 4

// ExitCommand 实现退出命令
type ExitCommand struct{}

func init() {
	Register(&ExitCommand{})
}

func (c *ExitCommand) ID() uint32 {
	return CommandIDExit
}

func (c *ExitCommand) Name() string {
	return "exit"
}

func (c *ExitCommand) Execute(task *Task) ([]byte, error) {
	log.Println("Received exit command. Terminating.")
	os.Exit(0)
	return nil, nil // 永远不会执行到这里
}
