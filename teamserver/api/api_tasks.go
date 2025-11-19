package api

import (
	"encoding/json"
	"net/http"
	"simplec2/pkg/logger"
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

	// Broadcast TASK_QUEUED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "TASK_QUEUED",
		Payload: task,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling TASK_QUEUED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted TASK_QUEUED event for %s", task.TaskID)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}

// CancelTask handles the API request to cancel a queued task.
func (a *API) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")

	// Get task info before cancellation for event broadcasting
	task, err := a.TaskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Task not found"})
		return
	}

	// Only allow cancellation of queued tasks
	if task.Status != "queued" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Only queued tasks can be canceled"})
		return
	}

	// Update task status to canceled
	task.Status = "canceled"
	if err := a.TaskService.UpdateTask(c.Request.Context(), task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cancel task"})
		return
	}

	// Broadcast TASK_CANCELED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "TASK_CANCELED",
		Payload: task,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling TASK_CANCELED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted TASK_CANCELED event for %s", taskID)
		}
	}

	c.Status(http.StatusNoContent)
}
