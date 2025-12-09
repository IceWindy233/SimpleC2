package command

import (
	"os/exec"
	"runtime"
)

// CommandIDShell Shell 命令 ID
const CommandIDShell uint32 = 1

// ShellCommand 实现 shell 命令执行
type ShellCommand struct{}

func init() {
	Register(&ShellCommand{})
}

func (c *ShellCommand) ID() uint32 {
	return CommandIDShell
}

func (c *ShellCommand) Name() string {
	return "shell"
}

func (c *ShellCommand) Execute(task *Task) ([]byte, error) {
	command := string(task.Arguments)
	return executeShellCommand(command)
}

// executeShellCommand 根据操作系统执行 shell 命令
func executeShellCommand(command string) ([]byte, error) {
	var cmd *exec.Cmd
	if runtime.GOOS == "windows" {
		cmd = exec.Command("cmd", "/C", command)
	} else {
		cmd = exec.Command("/bin/sh", "-c", command)
	}
	return cmd.CombinedOutput()
}
