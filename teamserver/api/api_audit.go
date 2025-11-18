package api

import (
	"net/http"
	"strconv"
	"simplec2/teamserver/service"

	"github.com/gin-gonic/gin"
)

// AuditLogResponse represents the response format for audit logs.
type AuditLogResponse struct {
	ID           uint      `json:"id"`
	CreatedAt    string    `json:"timestamp"`
	Username     string    `json:"username"`
	Action       string    `json:"action"`
	ResourceType string    `json:"resource_type"`
	ResourceID   string    `json:"resource_id"`
	IPAddress    string    `json:"ip_address"`
	Result       string    `json:"result"`
	Details      string    `json:"details"`
}

// AuditLogsResponse represents the paginated response for audit logs.
type AuditLogsResponse struct {
	Data []AuditLogResponse `json:"data"`
	Meta PaginationMeta       `json:"meta"`
}

type PaginationMeta struct {
	Page       int   `json:"page"`
	Limit      int   `json:"limit"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

// GetAuditLogs handles the API request to retrieve audit logs with optional filtering.
// Query parameters:
// - page: Page number (default: 1)
// - limit: Items per page (default: 20)
// - username: Filter by username
// - action: Filter by action type
// - resource_type: Filter by resource type (beacon, task, listener)
// - resource_id: Filter by resource ID
// - result: Filter by result (success, failure)
// - start_date: Filter start date
// - end_date: Filter end date
func (a *API) GetAuditLogs(c *gin.Context) {
	// Parse pagination parameters
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 20
	}

	// Parse query parameters
	query := &service.AuditQuery{
		Page:         page,
		Limit:        limit,
		Username:     c.Query("username"),
		Action:       c.Query("action"),
		ResourceType: c.Query("resource_type"),
		ResourceID:   c.Query("resource_id"),
		Result:       c.Query("result"),
		StartDate:    c.Query("start_date"),
		EndDate:      c.Query("end_date"),
	}

	// Call the service to get audit logs
	logs, total, err := a.AuditService.GetAuditLogs(c.Request.Context(), query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"error": "Failed to retrieve audit logs",
		})
		return
	}

	// Convert to response format
	response := AuditLogsResponse{
		Data: make([]AuditLogResponse, len(logs)),
		Meta: PaginationMeta{
			Page:       query.Page,
			Limit:      query.Limit,
			Total:      total,
			TotalPages: calculateTotalPages(total, query.Limit),
		},
	}

	for i, log := range logs {
		response.Data[i] = AuditLogResponse{
			ID:           log.ID,
			CreatedAt:    log.Timestamp.Format("2006-01-02T15:04:05Z"),
			Username:     log.Username,
			Action:       log.Action,
			ResourceType: log.ResourceType,
			ResourceID:   log.ResourceID,
			IPAddress:    log.IPAddress,
			Result:       log.Result,
			Details:      log.Details,
		}
	}

	c.JSON(http.StatusOK, response)
}

// Helper function to calculate total pages
func calculateTotalPages(total int64, limit int) int {
	if total == 0 {
		return 0
	}
	pages := int(total) / limit
	if int(total)%limit > 0 {
		pages++
	}
	return pages
}
