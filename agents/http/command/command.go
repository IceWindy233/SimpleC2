package command

// Task 任务结构，与 TeamServer 通信时使用
type Task struct {
	TaskID    string `json:"task_id"`
	CommandID uint32 `json:"command_id"`
	Arguments []byte `json:"arguments"`
}

// CommandHandler 命令处理器接口
// 每个命令实现此接口以注册到全局注册表
type CommandHandler interface {
	// ID 返回命令的唯一标识符
	ID() uint32
	// Name 返回命令的可读名称
	Name() string
	// Execute 执行命令并返回输出
	Execute(task *Task) ([]byte, error)
}

// 全局命令注册表
var registry = make(map[uint32]CommandHandler)

// Register 注册一个命令处理器
func Register(handler CommandHandler) {
	registry[handler.ID()] = handler
}

// Get 根据命令ID获取对应的处理器
func Get(id uint32) (CommandHandler, bool) {
	h, ok := registry[id]
	return h, ok
}

// GetAll 返回所有已注册的命令
func GetAll() map[uint32]CommandHandler {
	return registry
}
