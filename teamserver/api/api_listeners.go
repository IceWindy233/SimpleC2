package api

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"simplec2/teamserver/data"
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

	listener := data.Listener{
		Name:   req.Name,
		Type:   req.Type,
		Config: req.Config,
	}

	if err := a.Store.CreateListener(&listener); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create listener"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": listener})
}

func (a *API) GetListeners(c *gin.Context) {
	listeners, err := a.Store.GetListeners()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": listeners})
}

func (a *API) DeleteListener(c *gin.Context) {
	listenerName := c.Param("name")
	if err := a.Store.DeleteListener(listenerName); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete listener"})
		return
	}
	c.Status(http.StatusNoContent)
}
