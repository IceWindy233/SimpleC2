package api

import (
	"encoding/json"
	"math"
	"net/http"

	"simplec2/pkg/logger"
	"simplec2/teamserver/service"
	"strconv"

	"github.com/gin-gonic/gin"
)

// GetBeacons handles the API request to list all beacons.
func (a *API) GetBeacons(c *gin.Context) {
	// Parse and validate query parameters
	page, err := strconv.Atoi(c.DefaultQuery("page", "1"))
	if err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid 'page' parameter", "must be an integer"))
		return
	}
	limit, err := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid 'limit' parameter", "must be an integer"))
		return
	}
	search := c.Query("search")
	status := c.Query("status")

	query := &service.ListQuery{
		Page:   page,
		Limit:  limit,
		Search: search,
		Status: status,
	}

	beacons, total, err := a.BeaconService.ListBeacons(c.Request.Context(), query)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to retrieve beacons", err.Error()))
		return
	}

	totalPages := int(math.Ceil(float64(total) / float64(limit)))

	meta := gin.H{
		"page":        page,
		"limit":       limit,
		"total":       total,
		"total_pages": totalPages,
	}
	Respond(c, http.StatusOK, NewSuccessResponse(beacons, meta))
}

// GetBeacon handles the API request to retrieve a single beacon by its ID.
func (a *API) GetBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")
	beacon, err := a.BeaconService.GetBeacon(c.Request.Context(), beaconID)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Beacon not found", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(beacon, nil))
}

// DeleteBeacon handles the API request to soft delete a beacon and task it to exit.
func (a *API) DeleteBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	// Get beacon info before deletion for event broadcasting
	beacon, err := a.BeaconService.GetBeacon(c.Request.Context(), beaconID)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Beacon not found", err.Error()))
		return
	}

	err = a.BeaconService.DeleteBeacon(c.Request.Context(), beaconID)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to delete beacon", err.Error()))
		return
	}

	// Broadcast BEACON_DELETED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type:    "BEACON_DELETED",
		Payload: beacon,
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling BEACON_DELETED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted BEACON_DELETED event for %s", beaconID)
		}
	}

	c.Status(http.StatusNoContent)
}

// UpdateBeaconRequest defines the request body for updating a beacon.
type UpdateBeaconRequest struct {
	Note string `json:"note"`
}

// UpdateBeacon handles the API request to update a beacon's metadata.
func (a *API) UpdateBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")
	var req UpdateBeaconRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	updates := map[string]interface{}{
		"note": req.Note,
	}

	err := a.BeaconService.UpdateBeaconMetadata(c.Request.Context(), beaconID, updates)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to update beacon", err.Error()))
		return
	}

	// Fetch updated beacon to broadcast change
	beacon, err := a.BeaconService.GetBeacon(c.Request.Context(), beaconID)
	if err == nil {
		event := struct {
			Type    string      `json:"type"`
			Payload interface{} `json:"payload"`
		}{
			Type:    "BEACON_METADATA_UPDATED",
			Payload: beacon,
		}
		if eventBytes, err := json.Marshal(event); err == nil && a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
		}
	}

	Respond(c, http.StatusOK, NewSuccessResponse(beacon, nil))
}
