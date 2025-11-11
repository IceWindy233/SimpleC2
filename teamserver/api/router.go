package api

import (
	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"simplec2/pkg/config"
	"simplec2/teamserver/data"
	"simplec2/teamserver/websocket"
)

// API holds the configuration and dependencies for the API handlers.
type API struct {
	Config *config.TeamServerConfig
	Store  data.DataStore
	Hub    *websocket.Hub
}

// NewRouter sets up the API routes and returns the Gin engine.
func NewRouter(cfg *config.TeamServerConfig, store data.DataStore, hub *websocket.Hub) *gin.Engine {
	router := gin.Default()

	// Add CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // For development; in production, lock this down.
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization")
	router.Use(cors.New(corsConfig))

	api := &API{Config: cfg, Store: store, Hub: hub}

	// 获取 JWT 签名密钥
	jwtSecret := config.GetJWTSecret(cfg.Auth.JWTSecret)

	// Public group for authentication
	auth := router.Group("/api/auth")
	{
		auth.POST("/login", api.Login())
	}

	// Protected group for C2 operations
	protected := router.Group("/api")
	protected.Use(AuthMiddleware(jwtSecret))
	{
		// WebSocket endpoint
		protected.GET("/ws", api.serveWs)

		// Beacon management
		protected.GET("/beacons", api.GetBeacons)
		protected.GET("/beacons/:beacon_id", api.GetBeacon)
		protected.DELETE("/beacons/:beacon_id", api.DeleteBeacon)

		// Task management
		protected.POST("/beacons/:beacon_id/tasks", api.CreateBeaconTask)
		protected.GET("/tasks/:task_id", api.GetTask)

		// Listener management
		protected.GET("/listeners", api.GetListeners)
		protected.POST("/listeners", api.CreateListener)
		protected.DELETE("/listeners/:name", api.DeleteListener)

		// File operations
		protected.POST("/upload", api.UploadFile)
		protected.GET("/loot/:filename", api.DownloadLootFile)
	}

	return router
}