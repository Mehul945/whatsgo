package main

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"whatsgo/internal/api"
	"whatsgo/internal/auth"
	"whatsgo/internal/config"
	"whatsgo/internal/db"
	"whatsgo/internal/webhook"
	"whatsgo/internal/whatsapp"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	cfg := config.Load()
	log.Printf("DatabaseURL: %s", cfg.DatabaseURL)
	log.Printf("DataDir: %s", cfg.DataDir)

	// Setup data directory
	if err := os.MkdirAll(cfg.DataDir, 0755); err != nil {
		log.Fatal("Failed to create data directory:", err)
	}

	// Initialize database
	database, err := db.Connect(cfg.DatabaseURL)
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}
	defer database.Close()

	// Initialize JWT
	auth.InitJWT(cfg.JWTSecret)

	// Initialize WhatsApp manager
	manager := whatsapp.NewManager(cfg.DataDir, cfg.EchoBot)

	// Initialize webhook dispatcher
	dispatcher := webhook.NewDispatcher(database, cfg.WebhookWorkers, cfg.WebhookQueueSize)
	dispatcher.Start()
	manager.SetDatabase(database)
	manager.SetWebhookDispatcher(dispatcher)

	// Reconnect devices that were previously connected
	manager.AutoReconnect()

	// Setup Gin
	if os.Getenv("GIN_MODE") == "" {
		gin.SetMode(gin.ReleaseMode)
	}

	r := gin.New()
	r.Use(gin.Recovery())
	r.Use(gin.Logger())

	// CORS
	r.Use(cors.New(cors.Config{
		AllowOrigins:     cfg.AllowedOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS", "PATCH"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization", "X-Requested-With"},
		ExposeHeaders:    []string{"Content-Length", "Content-Type"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	// Register routes
	api.RegisterAuthRoutes(r, database)
	api.RegisterDeviceRoutes(r, database, manager)
	api.RegisterMessageRoutes(r, database, manager)
	api.RegisterDashboardRoutes(r, database, manager)
	api.RegisterWebhookRoutes(r, database)
	api.RegisterAPIKeyRoutes(r, database)
	api.RegisterUserRoutes(r, database)
	api.RegisterAdminRoutes(r, database)
	api.RegisterMetaRoutes(r, database, manager)
	api.RegisterMetaWebhookRoutes(r, database, manager)

	// Static files - serve frontend build
	webDir := filepath.Join(cfg.DataDir, "..", "web", "dist")
	if _, err := os.Stat(webDir); err == nil {
		r.NoRoute(func(c *gin.Context) {
			path := c.Request.URL.Path
			fullPath := filepath.Join(webDir, path)

			// Check if file exists and is not a directory
			if stat, err := os.Stat(fullPath); err == nil && !stat.IsDir() {
				c.File(fullPath)
				return
			}

			// Fallback to index.html for SPA routing
			c.File(filepath.Join(webDir, "index.html"))
		})
	}

	// Health check
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok", "time": time.Now().Format(time.RFC3339)})
	})

	log.Printf("WhatsGo server starting on port %s", cfg.Port)
	if err := r.Run(":" + cfg.Port); err != nil {
		log.Fatal("Server failed:", err)
	}
}
