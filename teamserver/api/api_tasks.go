package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"simplec2/teamserver/data"
)

// GetTask handles the API request to retrieve a single task by its ID.
func (a *API) GetTask(c *gin.Context) {
	taskID := c.Param("task_id")
	task, err := a.Store.GetTask(taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

// GetTasksForBeacon handles the API request to retrieve all tasks for a specific beacon.
func (a *API) GetTasksForBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")
	// First, check if beacon exists to return a 404 if not
	if _, err := a.Store.GetBeacon(beaconID); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	tasks, err := a.Store.GetTasksByBeaconID(beaconID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to retrieve tasks for beacon"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": tasks})
}

// CreateTaskRequest defines the structure for the task creation API request body.
type CreateTaskRequest struct {
	Command   string `json:"command" binding:"required"`
	Arguments string `json:"arguments"`
}

// CreateTaskForBeacon handles the API request to create a new task for a beacon.
func (a *API) CreateTaskForBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := a.Store.GetBeacon(beaconID)
	if err != nil {
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

	log.Printf("Creating task - Command: %s, Arguments: %q, Length: %d", req.Command, req.Arguments, len(req.Arguments))

	if err := a.Store.CreateTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}
