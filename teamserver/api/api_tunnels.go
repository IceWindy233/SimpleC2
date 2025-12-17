package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

// StartTunnelRequest defines the request body for starting a tunnel.
type StartTunnelRequest struct {
	BeaconID string `json:"beacon_id" binding:"required"`
	Target   string `json:"target" binding:"required"` // host:port
}

// StartTunnel initiates a new port forwarding tunnel.
func (a *API) StartTunnel(c *gin.Context) {
	var req StartTunnelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Invalid request body", err.Error()))
		return
	}

	// Ideally, get the current operator's ID from context/session
	operatorID := "operator" // Placeholder

	tunnel, err := a.PortFwdService.StartNewTunnel(c.Request.Context(), req.BeaconID, req.Target, operatorID)
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to start tunnel", err.Error()))
		return
	}

	Respond(c, http.StatusOK, NewSuccessResponse(tunnel, nil))
}

// CloseTunnel closes an active tunnel.
func (a *API) CloseTunnel(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")
	if tunnelID == "" {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Tunnel ID is required", ""))
		return
	}

	if err := a.PortFwdService.CloseTunnel(c.Request.Context(), tunnelID); err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to close tunnel", err.Error()))
		return
	}

	Respond(c, http.StatusOK, NewSuccessResponse(gin.H{"message": "Tunnel closed successfully"}, nil))
}

// GetTunnel retrieves details of a specific tunnel.
func (a *API) GetTunnel(c *gin.Context) {
	tunnelID := c.Param("tunnel_id")
	if tunnelID == "" {
		Respond(c, http.StatusBadRequest, NewErrorResponse(http.StatusBadRequest, "Tunnel ID is required", ""))
		return
	}

	tunnel, err := a.PortFwdService.GetTunnel(c.Request.Context(), tunnelID)
	if err != nil {
		Respond(c, http.StatusNotFound, NewErrorResponse(http.StatusNotFound, "Tunnel not found", err.Error()))
		return
	}

	// Convert service.Tunnel to a response friendly format if needed, 
	// for now returning the struct directly assuming it's JSON serializable
	Respond(c, http.StatusOK, NewSuccessResponse(tunnel, nil))
}

// ListTunnels retrieves all active tunnels.
func (a *API) ListTunnels(c *gin.Context) {
	tunnels, err := a.PortFwdService.ListTunnels(c.Request.Context())
	if err != nil {
		Respond(c, http.StatusInternalServerError, NewErrorResponse(http.StatusInternalServerError, "Failed to list tunnels", err.Error()))
		return
	}
	Respond(c, http.StatusOK, NewSuccessResponse(tunnels, nil))
}
