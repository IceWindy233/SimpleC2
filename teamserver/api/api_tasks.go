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
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Task not found", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(task, nil))
}

// GetTasksForBeacon handles the API request to retrieve all tasks for a specific beacon.
func (a *API) GetTasksForBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")
	status := c.Query("status")

	tasks, err := a.TaskService.GetTasksByBeaconID(c.Request.Context(), beaconID, status)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Tasks not found for beacon", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(tasks, nil))
}

// CreateTaskRequest defines the structure for the task creation API request body.
type CreateTaskRequest struct {
	Command   string `json:"command" binding:"required"`
	Arguments string `json:"arguments"`
	Source    string `json:"source"`
}

// CreateTaskForBeacon handles the API request to create a new task for a beacon.
func (a *API) CreateTaskForBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	task, err := a.TaskService.CreateTask(c.Request.Context(), beaconID, req.Command, req.Arguments, req.Source)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Failed to create task", err.Error()))
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

	Respond(c, http.StatusCreated, NewSuccessResponse(task, nil))
}

// CancelTask handles the API request to cancel a queued task.
func (a *API) CancelTask(c *gin.Context) {
	taskID := c.Param("task_id")

	// Get task info before cancellation for event broadcasting
	task, err := a.TaskService.GetTask(c.Request.Context(), taskID)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Task not found", err.Error()))
		return
	}

	// Only allow cancellation of queued tasks
	if task.Status != "queued" {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Only queued tasks can be canceled", ""))
		return
	}

	// Update task status to canceled
	task.Status = "canceled"
	if err := a.TaskService.UpdateTask(c.Request.Context(), task); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to cancel task", err.Error()))
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
