package api

import (
	"encoding/json"
	"time"

	"github.com/gin-gonic/gin"
	"simplec2/pkg/logger"
	"simplec2/teamserver/websocket"
)

// serveWs godoc
// @Summary Establish WebSocket connection
// @Description Establishes a WebSocket connection for real-time event updates.
// @Tags websocket
// @Produce  json
// @Param token query string true "JWT token for authentication"
// @Success 101 "Switching Protocols"
// @Failure 401 {object} gin.H{"error": string} "Unauthorized"
// @Router /ws [get]
// serveWs handles websocket requests from the peer.
// It acts as an adapter between the Gin context and the standard http.ResponseWriter and http.Request
// expected by the websocket handler.
func (a *API) serveWs(c *gin.Context) {
	// Get username from context (set by AuthMiddleware)
	username, _ := c.Get("username")

	// Broadcast CLIENT_CONNECTED event via WebSocket
	event := struct {
		Type    string      `json:"type"`
		Payload interface{} `json:"payload"`
	}{
		Type: "CLIENT_CONNECTED",
		Payload: map[string]interface{}{
			"username": username,
			"remote_addr": c.ClientIP(),
			"user_agent": c.Request.UserAgent(),
			"timestamp": time.Now(),
		},
	}
	eventBytes, err := json.Marshal(event)
	if err != nil {
		logger.Errorf("Error marshalling CLIENT_CONNECTED event: %v", err)
	} else {
		if a.Hub != nil {
			a.Hub.Broadcast(eventBytes)
			logger.Debugf("Broadcasted CLIENT_CONNECTED event for user %v", username)
		}
	}

	websocket.ServeWs(a.Hub, c.Writer, c.Request)
}
