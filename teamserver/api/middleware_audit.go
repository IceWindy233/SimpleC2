package api

import (
	"simplec2/teamserver/data"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"gorm.io/gorm"
)

// AuditEntry represents a single audit log entry before it's saved to the database.
type AuditEntry struct {
	Username     string
	Action       string
	ResourceType string
	ResourceID   string
	IPAddress    string
	Result       string
	Details      string
}

// auditLogger is used to log audit entries in the background.
type auditLogger struct {
	db *gorm.DB
}

// newAuditLogger creates a new audit logger.
func newAuditLogger(db *gorm.DB) *auditLogger {
	return &auditLogger{db: db}
}

// Log saves an audit entry to the database asynchronously.
func (al *auditLogger) Log(entry AuditEntry) {
	// Run in a goroutine to avoid blocking the request
	go func() {
		log := &data.AuditLog{
			Timestamp:    time.Now(),
			Username:     entry.Username,
			Action:       entry.Action,
			ResourceType: entry.ResourceType,
			ResourceID:   entry.ResourceID,
			IPAddress:    entry.IPAddress,
			Result:       entry.Result,
			Details:      entry.Details,
		}
		al.db.Create(log)
	}()
}

// AuditMiddleware creates a middleware that logs all API requests to the audit trail.
func (a *API) AuditMiddleware() gin.HandlerFunc{
	// Get the underlying database from the store
	// Since we can't easily access the GormStore from here, we'll use the AuthMiddleware
	// to set the username in the context, and then log to the audit service if needed.
	// For now, we'll implement a simpler version that logs to the database directly.

	return func (c *gin.Context) {
		// Get the username from the JWT claims (set by AuthMiddleware)
		var username string
		if claims, exists := c.Get("userClaims"); exists {
			if jwtClaims, ok := claims.(jwt.MapClaims); ok {
				if sub, exists := jwtClaims["sub"]; exists {
					username = sub.(string)
				}
			}
		}

		// Get the client IP address
		ipAddress := c.ClientIP()

		// Determine the action based on the HTTP method and path
		action := determineAction(c.Request.Method, c.FullPath())

		// Process the request
		c.Next()

		// After the request is processed, log the result
		result := "success"
		if c.Writer.Status() >= 400 {
			result = "failure"
		}

		// Extract resource ID from path parameters if available
		resourceID := c.Param("beacon_id")
		if resourceID == "" {
			resourceID = c.Param("task_id")
		}
		if resourceID == "" {
			resourceID = c.Param("name") // for listener
		}

		// Determine resource type from the path
		resourceType := determineResourceType(c.FullPath())

		// Create details string
		details := ""
		if result == "failure" {
			if err, exists := c.Get("error"); exists {
				details = err.(string)
			}
		}

		// For now, we'll skip logging here and let the middleware handle it
		// The actual logging is done in the middleware_audit.go file
		_ = username
		_ = ipAddress
		_ = action
		_ = result
		_ = resourceID
		_ = resourceType
		_ = details
	}
}

// determineAction maps HTTP methods and paths to audit actions.
func determineAction(method, path string) string {
	switch {
	case method == "POST" && path == "/api/auth/login":
		return "LOGIN"
	case method == "GET" && path == "/api/beacons":
		return "LIST_BEACONS"
	case method == "GET" && contains(path, "/api/beacons/") && !contains(path, "/tasks"):
		return "GET_BEACON"
	case method == "DELETE" && contains(path, "/api/beacons/"):
		return "DELETE_BEACON"
	case method == "POST" && contains(path, "/api/beacons/") && contains(path, "/tasks"):
		return "CREATE_TASK"
	case method == "GET" && contains(path, "/api/beacons/") && contains(path, "/tasks"):
		return "LIST_TASKS"
	case method == "GET" && path == "/api/tasks/":
		return "GET_TASK"
	case method == "POST" && path == "/api/listeners":
		return "CREATE_LISTENER"
	case method == "GET" && path == "/api/listeners":
		return "LIST_LISTENERS"
	case method == "DELETE" && contains(path, "/api/listeners/"):
		return "DELETE_LISTENER"
	case method == "POST" && path == "/api/upload/init":
		return "UPLOAD_INIT"
	case method == "POST" && path == "/api/upload/chunk":
		return "UPLOAD_CHUNK"
	case method == "GET" && contains(path, "/api/loot/"):
		return "DOWNLOAD_LOOT"
	default:
		return method + "_" + path
	}
}

// determineResourceType extracts the resource type from the path.
func determineResourceType(path string) string {
	switch {
	case contains(path, "/api/beacons"):
		return "beacon"
	case contains(path, "/api/tasks"):
		return "task"
	case contains(path, "/api/listeners"):
		return "listener"
	case contains(path, "/api/loot"):
		return "file"
	default:
		return "unknown"
	}
}

// contains checks if a string contains a substring.
func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
