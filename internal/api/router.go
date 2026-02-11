package api

import (
	"embed"
	"io/fs"
	"net/http"

	"github.com/gin-gonic/gin"
	"nas-controller/internal/api/handlers"
	"nas-controller/internal/database"
	"nas-controller/internal/docker"
	"nas-controller/internal/services"
)

//go:embed all:static
var staticFiles embed.FS

func NewRouter(
	db *database.DB,
	dockerClient *docker.Client,
	authService *services.AuthService,
	appManager *services.AppManager,
	buildService *services.BuildService,
	portAllocator *services.PortAllocator,
	dataDir string,
) *gin.Engine {
	gin.SetMode(gin.ReleaseMode)
	router := gin.Default()

	// CORS middleware
	router.Use(func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Allow-Credentials", "true")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	})

	// Initialize handlers
	authHandler := handlers.NewAuthHandler(db, authService)
	appHandler := handlers.NewAppHandler(appManager, buildService, dockerClient, dataDir)
	systemHandler := handlers.NewSystemHandler(dockerClient, buildService, db, dataDir)

	// Auth middleware
	authMiddleware := NewAuthMiddleware(db)

	// API routes
	api := router.Group("/api/v1")
	{
		// Auth routes (no auth required)
		auth := api.Group("/auth")
		{
			auth.POST("/login", authHandler.Login)
			auth.POST("/logout", authHandler.Logout)
			auth.GET("/check", authHandler.Check)
		}

		// Protected routes
		protected := api.Group("")
		protected.Use(authMiddleware.Authenticate())
		{
			// Auth
			protected.PUT("/auth/password", authHandler.UpdatePassword)

			// Apps
			protected.GET("/apps", appHandler.ListApps)
			protected.POST("/apps", appHandler.CreateApp)
			protected.POST("/apps/clone", appHandler.CloneRepo)
			protected.GET("/apps/:id", appHandler.GetApp)
			protected.PUT("/apps/:id", appHandler.UpdateApp)
			protected.DELETE("/apps/:id", appHandler.DeleteApp)
			protected.GET("/apps/:id/icon", appHandler.GetAppIcon)

			// App actions
			protected.POST("/apps/:id/build", appHandler.BuildApp)
			protected.POST("/apps/:id/start", appHandler.StartApp)
			protected.POST("/apps/:id/stop", appHandler.StopApp)
			protected.POST("/apps/:id/restart", appHandler.RestartApp)
			protected.POST("/apps/:id/pull", appHandler.PullAndRebuild)
			protected.GET("/apps/:id/check-update", appHandler.CheckUpdate)

			// Logs
			protected.GET("/apps/:id/logs", appHandler.GetLogs)
			protected.DELETE("/apps/:id/logs", appHandler.ClearLogs)
			protected.GET("/apps/:id/build-logs", appHandler.GetBuildLogs)

			// System
			protected.GET("/system/info", systemHandler.GetInfo)
			protected.GET("/system/storage", systemHandler.GetStorage)
			protected.GET("/system/ports", systemHandler.GetPorts)
			protected.POST("/system/prune", systemHandler.PruneImages)
			protected.DELETE("/system/logs", systemHandler.ClearAllLogs)
			protected.POST("/system/check-update", systemHandler.CheckSelfUpdate)
			protected.POST("/system/self-update", systemHandler.SelfUpdate)
		}

		// WebSocket routes (auth via query param)
		api.GET("/apps/:id/logs/stream", authMiddleware.AuthenticateWS(), appHandler.StreamLogs)
		api.GET("/apps/:id/build/stream", authMiddleware.AuthenticateWS(), appHandler.StreamBuild)
	}

	// Health check (no auth)
	router.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	// Serve static files (frontend)
	staticFS, err := fs.Sub(staticFiles, "static")
	if err == nil {
		// Helper to serve index.html
		serveIndex := func(c *gin.Context) {
			content, err := fs.ReadFile(staticFS, "index.html")
			if err != nil {
				c.String(http.StatusInternalServerError, "Failed to load page")
				return
			}
			c.Data(http.StatusOK, "text/html; charset=utf-8", content)
		}

		// Serve index.html for root
		router.GET("/", serveIndex)

		// Handle all other routes - serve static files or fallback to index.html for SPA
		router.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path

			// Remove leading slash for filesystem access
			filePath := path
			if len(filePath) > 0 && filePath[0] == '/' {
				filePath = filePath[1:]
			}

			// Skip empty path
			if filePath == "" {
				serveIndex(c)
				return
			}

			// Try to read the file
			content, err := fs.ReadFile(staticFS, filePath)
			if err != nil {
				// File doesn't exist, serve index.html for SPA routing
				serveIndex(c)
				return
			}

			// Determine content type
			contentType := "application/octet-stream"
			if len(filePath) > 3 {
				switch filePath[len(filePath)-3:] {
				case ".js":
					contentType = "application/javascript"
				case "css":
					contentType = "text/css"
				case "tml":
					contentType = "text/html"
				case "son":
					contentType = "application/json"
				case "svg":
					contentType = "image/svg+xml"
				case "png":
					contentType = "image/png"
				case "jpg":
					contentType = "image/jpeg"
				case "ico":
					contentType = "image/x-icon"
				}
			}

			c.Data(http.StatusOK, contentType, content)
		})
	}

	return router
}
