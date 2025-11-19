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

// UploadInitRequest defines the structure for the initial upload request body.
type UploadInitRequest struct {
	FileName string `json:"filename" binding:"required"`
}

// UploadCompleteRequest defines the structure for the upload completion request body.
type UploadCompleteRequest struct {
	UploadID string `json:"upload_id" binding:"required"`
	FileName string `json:"filename" binding:"required"`
}

// UploadInit initializes a new chunked upload.
func (a *API) UploadInit(c *gin.Context) {
	var req UploadInitRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Filename is required", err.Error()))
		return
	}

	uploadID := uuid.New().String()
	tmpDir := filepath.Join(a.Config.UploadsDir, "tmp", uploadID)

	if err := os.MkdirAll(tmpDir, 0755); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to create temporary upload directory", err.Error()))
		return
	}

	Respond(c, http.StatusOK, NewSuccessResponse(gin.H{"upload_id": uploadID}, nil))
}

// UploadChunk handles a single file chunk.
func (a *API) UploadChunk(c *gin.Context) {
	uploadID := c.GetHeader("X-Upload-ID")
	chunkNumberStr := c.GetHeader("X-Chunk-Number")

	if uploadID == "" || chunkNumberStr == "" {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "X-Upload-ID and X-Chunk-Number headers are required", ""))
		return
	}

	tmpDir := filepath.Join(a.Config.UploadsDir, "tmp", uploadID)
	// Basic security check to prevent path traversal
	if !strings.HasPrefix(filepath.Clean(tmpDir), filepath.Clean(filepath.Join(a.Config.UploadsDir, "tmp"))) {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid upload ID", ""))
		return
	}

	chunkPath := filepath.Join(tmpDir, "chunk_"+chunkNumberStr)
	file, err := os.Create(chunkPath)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to create chunk file", err.Error()))
		return
	}
	defer file.Close()

	if _, err := io.Copy(file, c.Request.Body); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to write chunk data", err.Error()))
		return
	}

	c.Status(http.StatusOK)
}

// UploadComplete finalizes the chunked upload, merging chunks into a single file.
func (a *API) UploadComplete(c *gin.Context) {
	var req UploadCompleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "UploadID and FileName are required", err.Error()))
		return
	}

	tmpDir := filepath.Join(a.Config.UploadsDir, "tmp", req.UploadID)
	// Basic security check
	if !strings.HasPrefix(filepath.Clean(tmpDir), filepath.Clean(filepath.Join(a.Config.UploadsDir, "tmp"))) {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid upload ID", ""))
		return
	}

	entries, err := os.ReadDir(tmpDir)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to read temporary upload directory", err.Error()))
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
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to create final file", err.Error()))
		return
	}
	defer destFile.Close()

	// Merge chunks
	for _, entry := range entries {
		chunkPath := filepath.Join(tmpDir, entry.Name())
		chunkFile, err := os.Open(chunkPath)
		if err != nil {
			Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to open chunk file", err.Error()))
			return
		}
		if _, err := io.Copy(destFile, chunkFile); err != nil {
			chunkFile.Close()
			Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to merge chunk file", err.Error()))
			return
		}
		chunkFile.Close()
	}

	// Clean up temporary directory
	if err := os.RemoveAll(tmpDir); err != nil {
		// Log this error but don't fail the request, as the file has been successfully created.
		fmt.Printf("Warning: failed to remove temporary upload directory %s: %v\n", tmpDir, err)
	}

	Respond(c, http.StatusOK, NewSuccessResponse(gin.H{"filepath": finalPath}, nil))
}

// DownloadLootFile godoc
// @Summary Download a loot file
// @Description Downloads a file that was collected from a beacon and stored in the loot directory.
// @Tags files
// @Produce  octet-stream
// @Param filename path string true "The name of the file to download"
// @Success 200 {file} binary "File content"
// @Failure 400 {object} gin.H{"error": string} "Bad request (e.g., invalid filename)"
// @Failure 403 {object} gin.H{"error": string} "Access denied (e.g., path traversal attempt, trying to download a directory)"
// @Failure 404 {object} gin.H{"error": string} "File not found"
// @Failure 500 {object} gin.H{"error": string} "Internal server error"
// @Router /files/loot/{filename} [get]
// DownloadLootFile handles the API request to download a loot file.
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
