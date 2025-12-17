package api

import (
	"simplec2/pkg/config"
	"simplec2/teamserver/service"
	"simplec2/teamserver/websocket"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

// API holds the configuration and dependencies for the API handlers.
type API struct {
	Config          *config.TeamServerConfig
	BeaconService   service.BeaconService
	TaskService     service.TaskService
	ListenerService service.ListenerService
	SessionService  *service.SessionService
	PortFwdService  service.PortFwdService
	Hub             *websocket.Hub
}

// NewRouter sets up the API routes and returns the Gin engine.
func NewRouter(cfg *config.TeamServerConfig, beaconService service.BeaconService, taskService service.TaskService, listenerService service.ListenerService, sessionService *service.SessionService, portFwdService service.PortFwdService, hub *websocket.Hub) *gin.Engine {
	router := gin.Default()

	// Add CORS middleware
	corsConfig := cors.DefaultConfig()
	corsConfig.AllowAllOrigins = true // For development; in production, lock this down.
	corsConfig.AllowHeaders = append(corsConfig.AllowHeaders, "Authorization", "X-Upload-ID", "X-Chunk-Number")
	router.Use(cors.New(corsConfig))

	api := &API{
		Config:          cfg,
		BeaconService:   beaconService,
		TaskService:     taskService,
		ListenerService: listenerService,
		SessionService:  sessionService,
		PortFwdService:  portFwdService,
		Hub:             hub,
	}

	// 获取 JWT 签名密钥
	jwtSecret := config.GetJWTSecret(cfg.Auth.JWTSecret)

	// Public group for authentication
	auth := router.Group("/api/auth")
	{
		auth.POST("/login", api.Login())
		auth.POST("/logout", api.Logout())
	}

	// Protected group for C2 operations
	protected := router.Group("/api")
	protected.Use(api.AuthMiddlewareWithSession(jwtSecret))
	{
		// WebSocket endpoint
		protected.GET("/ws", api.serveWs)

		// Beacon management
		protected.GET("/beacons", api.GetBeacons)
		protected.GET("/beacons/:beacon_id", api.GetBeacon)
		protected.DELETE("/beacons/:beacon_id", api.DeleteBeacon)

		// Task management
		protected.POST("/beacons/:beacon_id/tasks", api.CreateTaskForBeacon)
		protected.GET("/beacons/:beacon_id/tasks", api.GetTasksForBeacon)
		protected.GET("/tasks/:task_id", api.GetTask)
		protected.DELETE("/tasks/:task_id", api.CancelTask)

		// Listener management
		protected.GET("/listeners", api.GetListeners)
		protected.POST("/listeners", api.CreateListener)
		protected.DELETE("/listeners/:name", api.DeleteListener)
		protected.POST("/listeners/:name/start", api.StartListener)
		protected.POST("/listeners/:name/stop", api.StopListener)
		protected.POST("/listeners/:name/restart", api.RestartListener)

		// File operations
		protected.POST("/upload/init", api.UploadInit)
		protected.POST("/upload/chunk", api.UploadChunk)
		protected.POST("/upload/complete", api.UploadComplete)
		protected.GET("/loot/*filepath", api.DownloadLootFile)

		// Tunnel management
		protected.POST("/tunnels/start", api.StartTunnel)
		protected.GET("/tunnels", api.ListTunnels)
		protected.GET("/tunnels/:tunnel_id", api.GetTunnel)
		protected.POST("/tunnels/:tunnel_id/close", api.CloseTunnel)
	}

	return router
}
