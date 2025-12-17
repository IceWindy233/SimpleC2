package command

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"os/exec"
	"runtime"
	"strconv"
	"strings"
)

// CommandIDPs Ps 命令 ID
const CommandIDPs uint32 = 13

// Process defines the structure for a process entry.
type Process struct {
	PID        int    `json:"pid"`
	ParentPID  int    `json:"parent_pid,omitempty"` // For Linux/macOS
	Name       string `json:"name"`
	Executable string `json:"executable,omitempty"` // For Windows
	User       string `json:"user,omitempty"`
	Status     string `json:"status,omitempty"` // For Linux/macOS
	CPU        string `json:"cpu,omitempty"`    // For Linux/macOS
	Memory     string `json:"memory,omitempty"` // For Linux/macOS
	// Add more fields as needed
}

// PsCommand implements the ps command execution.
type PsCommand struct{}

func init() {
	Register(&PsCommand{})
}

func (c *PsCommand) ID() uint32 {
	return CommandIDPs
}

func (c *PsCommand) Name() string {
	return "ps"
}

func (c *PsCommand) Execute(task *Task) ([]byte, error) {
	var processes []Process
	var err error

	switch runtime.GOOS {
	case "windows":
		processes, err = getWindowsProcesses()
	case "linux", "darwin":
		processes, err = getUnixProcesses()
	default:
		err = fmt.Errorf("unsupported operating system: %s", runtime.GOOS)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to get process list: %v", err)
	}

	data, err := json.MarshalIndent(processes, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal process list: %v", err)
	}
	return data, nil
}

func getWindowsProcesses() ([]Process, error) {
	cmd := exec.Command("tasklist", "/FO", "CSV", "/NH")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	reader := csv.NewReader(&out)
	records, err := reader.ReadAll()
	if err != nil {
		return nil, err
	}

	processes := make([]Process, 0, len(records))
	for _, rec := range records {
		if len(rec) < 5 { // ImageName,PID,SessionName,Session#,MemUsage
			continue
		}
		pid, _ := strconv.Atoi(rec[1])
		processes = append(processes, Process{
			PID:  pid,
			Name: strings.Trim(rec[0], `"`),
			// Executable: rec[0], // Image name
			// User:       "",       // tasklist CSV doesn't easily expose user
		})
	}
	return processes, nil
}

func getUnixProcesses() ([]Process, error) {
	// ps -eo pid,ppid,user,comm,pcpu,pmem,stat,args
	// pid: process ID
	// ppid: parent process ID
	// user: user name
	// comm: command name (usually executable name)
	// pcpu: %cpu
	// pmem: %mem
	// stat: process state
	// args: command with arguments
	cmd := exec.Command("ps", "-eo", "pid,user,comm,pcpu,pmem,stat,args")
	var out bytes.Buffer
	cmd.Stdout = &out
	err := cmd.Run()
	if err != nil {
		return nil, err
	}

	lines := strings.Split(out.String(), "\n")
	processes := make([]Process, 0, len(lines))
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "PID") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 7 {
			continue
		}

		pid, _ := strconv.Atoi(fields[0])
		cpu := fields[3]
		mem := fields[4]
		status := fields[5]
		
		// The command and arguments can contain spaces, so combine the rest
		// command is fields[2], but args starts from fields[6]
		name := fields[2]
		fullCommand := strings.Join(fields[6:], " ")


		processes = append(processes, Process{
			PID:    pid,
			User:   fields[1],
			Name:   name,
			Status: status,
			CPU:    cpu,
			Memory: mem,
			Executable: fullCommand, // Store full command line in Executable for now
		})
	}
	return processes, nil
}
