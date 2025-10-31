package api

import (
	"log"
	"net/http"
	"os"
	"path/filepath"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"simplec2/teamserver/data"
)

// GetBeacons handles the API request to list all beacons.
func (a *API) GetBeacons(c *gin.Context) {
	var beacons []data.Beacon
	result := data.DB.Find(&beacons)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": beacons})
}

// GetBeacon handles the API request to retrieve a single beacon by its ID.
func (a *API) GetBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")
	var beacon data.Beacon

	if err := data.DB.Where("beacon_id = ?", beaconID).First(&beacon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": beacon})
}

// DeleteBeacon handles the API request to soft delete a beacon and task it to exit.
func (a *API) DeleteBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	var beacon data.Beacon
	if err := data.DB.Where("beacon_id = ?", beaconID).First(&beacon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	exitTask := data.Task{
		TaskID:    uuid.New().String(),
		BeaconID:  beaconID,
		Command:   "exit",
		Arguments: "",
		Status:    "queued",
	}

	if err := data.DB.Create(&exitTask).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create exit task for beacon"})
		return
	}

	if err := data.DB.Model(&beacon).Update("status", "exiting").Error; err != nil {
		log.Printf("Error updating beacon status to exiting: %v", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update beacon status"})
		return
	}

	result := data.DB.Delete(&beacon)
	if result.Error != nil {
		log.Printf("Error soft-deleting beacon %s: %v", beaconID, result.Error)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to soft-delete beacon"})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTaskRequest defines the structure for the task creation API request body.
type CreateTaskRequest struct {
	Command   string `json:"command" binding:"required"`
	Arguments string `json:"arguments"`
}

// CreateBeaconTask handles the API request to create a new task for a beacon.
func (a *API) CreateBeaconTask(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var beacon data.Beacon
	if err := data.DB.Where("beacon_id = ?", beaconID).First(&beacon).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	task := data.Task{
		TaskID:    uuid.New().String(),
		BeaconID:  beaconID,
		Command:   req.Command,
		Arguments: req.Arguments,
		Status:    "queued",
	}

	if err := data.DB.Create(&task).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}

// GetTask handles the API request to retrieve a single task by its ID.
func (a *API) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")
	var task data.Task

	if err := data.DB.Where("task_id = ?", taskID).First(&task).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": task})
}

// --- Listener Handlers ---

type CreateListenerRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Config string `json:"config"`
}

func (a *API) CreateListener(c *gin.Context) {
	var req CreateListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	listener := data.Listener{
		Name:   req.Name,
		Type:   req.Type,
		Config: req.Config,
	}

	if err := data.DB.Create(&listener).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listener"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": listener})
}

func (a *API) GetListeners(c *gin.Context) {
	var listeners []data.Listener
	result := data.DB.Find(&listeners)
	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": result.Error.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": listeners})
}

func (a *API) DeleteListener(c *gin.Context) {
	listenerName := c.Param("name")
	result := data.DB.Where("name = ?", listenerName).Delete(&data.Listener{})

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete listener"})
		return
	}

	if result.RowsAffected == 0 {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listener not found"})
		return
	}

	c.Status(http.StatusNoContent)
}

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