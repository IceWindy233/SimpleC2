package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
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
