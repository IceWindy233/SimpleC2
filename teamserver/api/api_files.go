package api

import (
	"net/http"
	"os"
	"path/filepath"

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
	filename := filepath.Base(c.Param("filename"))
	filePath := filepath.Join(a.Config.LootDir, filename)

	if _, err := os.Stat(filePath); os.IsNotExist(err) {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found"})
		return
	}

	c.File(filePath)
}
