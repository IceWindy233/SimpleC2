package api

import (
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- Chunked File Upload Handlers ---

type UploadInitRequest struct {
	FileName string `json:"filename" binding:"required"`
}

// UploadInit initializes a new chunked upload.
func (a *API) UploadInit(c *gin.Context) {
	var req UploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Filename is required"})
		return
	}

	uploadID := uuid.New().String()
	tmpDir := filepath.Join(a.Config.UploadsDir, "tmp", uploadID)

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create temporary upload directory"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"upload_id": uploadID})
}

// UploadChunk handles a single file chunk.
func (a *API) UploadChunk(c *gin.Context) {
	uploadID := c.GetHeader("X-Upload-ID")
	chunkNumberStr := c.GetHeader("X-Chunk-Number")

	if uploadID == "" || chunkNumberStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "X-Upload-ID and X-Chunk-Number headers are required"})
		return
	}

	tmpDir := filepath.Join(a.Config.UploadsDir, "tmp", uploadID)
	// Basic security check to prevent path traversal
	if !strings.HasPrefix(filepath.Clean(tmpDir), filepath.Clean(filepath.Join(a.Config.UploadsDir, "tmp"))) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload ID"})
		return
	}

	chunkPath := filepath.Join(tmpDir, "chunk_"+chunkNumberStr)
	file, err := os.Create(chunkPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create chunk file"})
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, c.Request.Body); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to write chunk data"})
		return
	}

	c.Status(http.StatusOK)
}

type UploadCompleteRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
	FileName string `json:"filename" binding:"required"`
}

// UploadComplete finalizes the chunked upload, merging chunks into a single file.
func (a *API) UploadComplete(c *gin.Context) {
	var req UploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "UploadID and FileName are required"})
		return
	}

	tmpDir := filepath.Join(a.Config.UploadsDir, "tmp", req.UploadID)
	// Basic security check
	if !strings.HasPrefix(filepath.Clean(tmpDir), filepath.Clean(filepath.Join(a.Config.UploadsDir, "tmp"))) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid upload ID"})
		return
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to read temporary upload directory"})
		return
	}

	// Sort chunks by number
	sort.Slice(entries, func(i, j int) bool {
		numA, _ := strconv.Atoi(strings.TrimPrefix(entries[i].Name(), "chunk_"))
		numB, _ := strconv.Atoi(strings.TrimPrefix(entries[j].Name(), "chunk_"))
		return numA < numB
	})

	// Create final destination file
	finalFileName := fmt.Sprintf("%s_%s", uuid.New().String(), filepath.Base(req.FileName))
	finalPath := filepath.Join(a.Config.UploadsDir, finalFileName)

	destFile, err := os.Create(finalPath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create final file"})
		return
	}
	defer destFile.Close()

	// Merge chunks
	for _, entry := range entries {
		chunkPath := filepath.Join(tmpDir, entry.Name())
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to open chunk file"})
			return
		}
		if _, err := io.Copy(destFile, chunkFile); err != nil {
			chunkFile.Close()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to merge chunk file"})
			return
		}
		chunkFile.Close()
	}

	// Clean up temporary directory
	if err := os.RemoveAll(tmpDir); err != nil {
		// Log this error but don't fail the request, as the file has been successfully created.
		fmt.Printf("Warning: failed to remove temporary upload directory %s: %v\n", tmpDir, err)
	}

	c.JSON(http.StatusOK, gin.H{"filepath": finalPath})
}


// --- Loot Download Handler ---

func (a *API) DownloadLootFile(c *gin.Context) {
	// 1. Get and sanitize filename to prevent path traversal.
	filename := filepath.Base(c.Param("filename"))
	if strings.Contains(filename, "..") || strings.Contains(filename, "/") || strings.Contains(filename, "\\") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid filename"})
		return
	}

	// 2. Construct the full, cleaned path.
	filePath := filepath.Clean(filepath.Join(a.Config.LootDir, filename))

	// 3. Security Check: Ensure the final path is within the intended loot directory.
	absLootDir, err := filepath.Abs(a.Config.LootDir)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: could not resolve loot directory"})
		return
	}
	absFilePath, err := filepath.Abs(filePath)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: could not resolve file path"})
		return
	}

	if !strings.HasPrefix(absFilePath, absLootDir) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: file is outside of the loot directory"})
		return
	}

	// 4. Check if the file exists and is not a directory.
	fileInfo, err := os.Stat(absFilePath)
	if os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: could not stat file"})
		return
	}
	if fileInfo.IsDir() {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied: cannot download a directory"})
		return
	}

	// 5. Serve the file with secure headers.
	c.Header("Content-Disposition", "attachment; filename=\""+filename+"\"")
	c.Header("X-Content-Type-Options", "nosniff")
	c.File(absFilePath)
}
