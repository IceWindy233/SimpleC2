package api

import (
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"gorm.io/gorm"
	"simplec2/teamserver/data"
)

// GetBeacons handles the API request to list all beacons.
func (a *API) GetBeacons(c *gin.Context) {
	beacons, err := a.Store.GetBeacons()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": beacons})
}

// GetBeacon handles the API request to retrieve a single beacon by its ID.
func (a *API) GetBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")
	beacon, err := a.Store.GetBeacon(beaconID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}
	c.JSON(http.StatusOK, gin.H{"data": beacon})
}

// DeleteBeacon handles the API request to soft delete a beacon and task it to exit.
func (a *API) DeleteBeacon(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	// The store is a GormStore, so we can get the DB object from it.
	gormStore, ok := a.Store.(*data.GormStore)
	if !ok {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Internal server error: invalid data store type"})
		return
	}

	// Wrap the operations in a transaction to ensure atomicity.
	err := gormStore.DB.Transaction(func(tx *gorm.DB) error {
		// 1. Ensure beacon exists before proceeding (within the transaction).
		var beacon data.Beacon
		if err := tx.Where("beacon_id = ?", beaconID).First(&beacon).Error; err != nil {
			return err // Beacon not found, will cause rollback.
		}

		// 2. Create an exit task for the beacon.
		exitTask := data.Task{
			TaskID:    uuid.New().String(),
			BeaconID:  beaconID,
			Command:   "exit",
			Arguments: "",
			Status:    "queued",
		}
		if err := tx.Create(&exitTask).Error; err != nil {
			return err // Task creation failed, will cause rollback.
		}

		// 3. Soft-delete the beacon.
		if err := tx.Where("beacon_id = ?", beaconID).Delete(&data.Beacon{}).Error; err != nil {
			return err // Deletion failed, will cause rollback.
		}

		// Return nil to commit the transaction.
		return nil
	})

	if err != nil {
		// Check if the error is because the beacon was not found.
		if err == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
			return
		}
		// For any other transaction error.
		log.Printf("Error in delete beacon transaction for %s: %v", beaconID, err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete beacon and create exit task"})
		return
	}

	c.Status(http.StatusNoContent)
}

// CreateTaskRequest defines the structure for the task creation API request body.
type CreateTaskRequest struct {
	Command   string `json:"command" binding:"required"`
	Arguments string `json:"arguments"`
}

// CreateBeaconTask handles the API request to create a new task for a beacon.
func (a *API) CreateBeaconTask(c *gin.Context) {
	beaconID := c.Param("beacon_id")

	var req CreateTaskRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	_, err := a.Store.GetBeacon(beaconID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Beacon not found"})
		return
	}

	task := data.Task{
		TaskID:    uuid.New().String(),
		BeaconID:  beaconID,
		Command:   req.Command,
		Arguments: req.Arguments,
		Status:    "queued",
	}

	log.Printf("Creating task - Command: %s, Arguments: %q, Length: %d", req.Command, req.Arguments, len(req.Arguments))

	if err := a.Store.CreateTask(&task); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create task"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": task})
}
