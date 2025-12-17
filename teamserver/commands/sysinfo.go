package commands

import (
	"simplec2/teamserver/data"
)

// CommandIDSysInfo SysInfo 命令 ID
const CommandIDSysInfo uint32 = 12

// SysInfoCommand implements the CommandConverter interface for the sysinfo command.
type SysInfoCommand struct{}

func init() {
	Register(&SysInfoCommand{})
}

func (c *SysInfoCommand) Name() string {
	return "sysinfo"
}

func (c *SysInfoCommand) CommandID() uint32 {
	return CommandIDSysInfo
}

func (c *SysInfoCommand) Convert(task *data.Task) ([]byte, error) {
	// Sysinfo command does not require any specific arguments,
	// so we return nil.
	return nil, nil
}
