package api

import (
	"github.com/gin-gonic/gin"
	"simplec2/teamserver/websocket"
)

// serveWs handles websocket requests from the peer.
// It acts as an adapter between the Gin context and the standard http.ResponseWriter and http.Request
// expected by the websocket handler.
func (a *API) serveWs(c *gin.Context) {
	websocket.ServeWs(a.Hub, c.Writer, c.Request)
}
