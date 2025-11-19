package api

import (
	"encoding/json"
	"net/http"
	"simplec2/pkg/logger"
	"simplec2/teamserver/service"

	"github.com/gin-gonic/gin"
)

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

	var _ service.ListenerService // 强制使用service包

	listener, err := a.ListenerService.CreateListener(c.Request.Context(), req.Name, req.Type, req.Config)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast LISTENER_STARTED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "LISTENER_STARTED",
		Payload: listener,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling LISTENER_STARTED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted LISTENER_STARTED event for %s", req.Name)
		}
	}

	c.JSON(http.StatusCreated, gin.H{"data": listener})
}

func (a *API) GetListeners(c *gin.Context) {
	listeners, err := a.ListenerService.ListListeners(c.Request.Context())
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": listeners})
}

func (a *API) DeleteListener(c *gin.Context) {
	listenerName := c.Param("name")

	// Get listener info before deletion for event broadcasting
	listener, err := a.ListenerService.GetListener(c.Request.Context(), listenerName)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Listener not found"})
		return
	}

	err = a.ListenerService.DeleteListener(c.Request.Context(), listenerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	// Broadcast LISTENER_STOPPED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "LISTENER_STOPPED",
		Payload: listener,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling LISTENER_STOPPED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted LISTENER_STOPPED event for %s", listenerName)
		}
	}

	c.Status(http.StatusNoContent)
}
