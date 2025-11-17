package api

import (
	"net/http"
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
	err := a.ListenerService.DeleteListener(c.Request.Context(), listenerName)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.Status(http.StatusNoContent)
}
