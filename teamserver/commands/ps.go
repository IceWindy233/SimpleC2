package commands

import (
	"simplec2/teamserver/data"
)

// CommandIDPs Ps 命令 ID
const CommandIDPs uint32 = 13

// PsCommand implements the CommandConverter interface for the ps command.
type PsCommand struct{}

func init() {
	Register(&PsCommand{})
}

func (c *PsCommand) Name() string {
	return "ps"
}

func (c *PsCommand) CommandID() uint32 {
	return CommandIDPs
}

func (c *PsCommand) Convert(task *data.Task) ([]byte, error) {
	// Ps command does not require any specific arguments,
	// so we return nil.
	return nil, nil
}
