package commands

import (
	"fmt"

	"simplec2/teamserver/data"
)

// CommandConverter 将数据库任务转换为 gRPC 任务参数
// 每个命令实现此接口以注册到全局注册表
type CommandConverter interface {
	// Name 返回命令名称（与数据库中 Task.Command 匹配）
	Name() string
	// CommandID 返回命令 ID（发送给 beacon）
	CommandID() uint32
	// Convert 将数据库任务转换为 beacon 可执行的参数
	Convert(task *data.Task) ([]byte, error)
}

// 全局命令注册表（命令名 -> 转换器）
var registry = make(map[string]CommandConverter)

// Register 注册命令转换器
func Register(converter CommandConverter) {
	registry[converter.Name()] = converter
}

// Get 根据命令名获取转换器
func Get(name string) (CommandConverter, bool) {
	c, ok := registry[name]
	return c, ok
}

// GetAll 返回所有已注册的命令
func GetAll() map[string]CommandConverter {
	return registry
}

// ErrUnknownCommand 未知命令错误
func ErrUnknownCommand(name string) error {
	return fmt.Errorf("unknown command: %s", name)
}
