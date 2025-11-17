package api

import (
	"net/http"
	"simplec2/teamserver/service"

	"github.com/gin-gonic/gin"
)

// GetTask handles the API request to retrieve a single task by its ID.
func (a *API) GetTask(c *gin.Context) {
	var _ service.TaskService // 强制使用service包

	taskID := c.Param("task_id")
	task, err := a.TaskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": task})
}

// GetTasksForBeacon handles the API request to retrieve all tasks for a specific beacon.
func (a *API) GetTasksForBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	tasks, err := a.TaskService.GetTasksByBeaconID(c.Request.Context(), beaconID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
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

	task, err := a.TaskService.CreateTask(c.Request.Context(), beaconID, req.Command, req.Arguments)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}
