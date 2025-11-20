package api

import (
	"encoding/json"
	"math"
	"net/http"
	"simplec2/pkg/logger"
	"strconv"

	"github.com/gin-gonic/gin"
)

// CreateListenerRequest defines the structure for the listener creation API request body.
type CreateListenerRequest struct {
	Name   string `json:"name" binding:"required"`
	Type   string `json:"type" binding:"required"`
	Config string `json:"config"`
}

// CreateListener godoc
// @Summary Create a new listener
// @Description Creates and starts a new listener with the specified configuration.
// @Tags listeners
// @Accept  json
// @Produce  json
// @Param listener body CreateListenerRequest true "Listener creation request"
// @Success 201 {object} gin.H{"data": data.Listener}
// @Failure 400 {object} gin.H{"error": string} "Invalid request body"
// @Failure 500 {object} gin.H{"error": string} "Internal server error"
// @Router /listeners [post]
func (a *API) CreateListener(c *gin.Context) {
	var req CreateListenerRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	listener, err := a.ListenerService.CreateListener(c.Request.Context(), req.Name, req.Type, req.Config)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to create listener", err.Error()))
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

	Respond(c, http.StatusCreated, NewSuccessResponse(listener, nil))
}

// GetListeners godoc
// @Summary Get all listeners
// @Description Retrieves a list of all active listeners.
// @Tags listeners
// @Produce  json
// @Success 200 {object} StandardResponse{data=[]data.Listener}
// @Failure 500 {object} StandardResponse
// @Router /listeners [get]
func (a *API) GetListeners(c *gin.Context) {
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid 'page' parameter", "must be an integer"))
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "10"))
	if err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid 'limit' parameter", "must be an integer"))
		return
	}

	listeners, total, err := a.ListenerService.ListListeners(c.Request.Context(), page, limit)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to retrieve listeners", err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}
	Respond(c, http.StatusOK, NewSuccessResponse(listeners, meta))
}

// DeleteListener godoc
// @Summary Delete a listener
// @Description Stops and deletes a listener by its name.
// @Tags listeners
// @Produce  json
// @Param name path string true "The name of the listener to delete"
// @Success 204 "No Content"
// @Failure 404 {object} StandardResponse
// @Failure 500 {object} StandardResponse
// @Router /listeners/{name} [delete]
func (a *API) DeleteListener(c *gin.Context) {
	listenerName := c.Param("name")

	// Get listener info before deletion for event broadcasting
	listener, err := a.ListenerService.GetListener(c.Request.Context(), listenerName)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Listener not found", err.Error()))
		return
	}

	err = a.ListenerService.DeleteListener(c.Request.Context(), listenerName)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to delete listener", err.Error()))
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
