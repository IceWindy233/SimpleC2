package api

import (
	"net/http"
	"os"
	"path/filepath"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// --- File Handlers ---

func (a *API) UploadFile(c *gin.Context) {
	file, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File not provided"})
		return
	}

	filename := uuid.New().String() + "_" + filepath.Base(file.Filename)
	dst := filepath.Join(a.Config.UploadsDir, filename)

	if err := os.MkdirAll(a.Config.UploadsDir, 0755); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create uploads directory"})
		return
	}

	if err := c.SaveUploadedFile(file, dst); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to save file"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"filepath": dst})
}

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
