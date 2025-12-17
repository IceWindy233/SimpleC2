package command

import (
	"encoding/json"
	"fmt"
	"net"
	"os"
	"os/user"
	"runtime"
)

// CommandIDSysInfo SysInfo 命令 ID
const CommandIDSysInfo uint32 = 12

// SysInfo 定义了系统信息结构
type SysInfo struct {
	Hostname        string `json:"hostname"`
	OS              string `json:"os"`
	Arch            string `json:"arch"`
	Username        string `json:"username"`
	InternalIP      string `json:"internal_ip"`
	NumCPU          int    `json:"num_cpu"`
	GoVersion       string `json:"go_version"`
	CurrentCMD      string `json:"current_cmd"`
	IsHighIntegrity bool   `json:"is_high_integrity"` // Placeholder for future implementation
}

// SysInfoCommand 实现 sysinfo 命令执行
type SysInfoCommand struct{}

func init() {
	Register(&SysInfoCommand{})
}

func (c *SysInfoCommand) ID() uint32 {
	return CommandIDSysInfo
}

func (c *SysInfoCommand) Name() string {
	return "sysinfo"
}

func (c *SysInfoCommand) Execute(task *Task) ([]byte, error) {
	hostname, _ := os.Hostname()
	sysInfo := SysInfo{
		Hostname:        hostname,
		OS:              runtime.GOOS,
		Arch:            runtime.GOARCH,
		Username:        getUsername(),
		InternalIP:      getInternalIP(),
		NumCPU:          runtime.NumCPU(),
		GoVersion:       runtime.Version(),
		CurrentCMD:      os.Args[0],
		IsHighIntegrity: false, // TODO: Implement actual integrity check
	}

	data, err := json.MarshalIndent(sysInfo, "", "  ")
	if err != nil {
		return nil, fmt.Errorf("failed to marshal sysinfo: %v", err)
	}
	return data, nil
}

// getUsername 获取当前用户名
func getUsername() string {
	currentUser, err := user.Current()
	if err != nil {
		return "unknown"
	}
	return currentUser.Username
}

// getInternalIP 获取内部 IP 地址
func getInternalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}
	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}
	return "127.0.0.1"
}
