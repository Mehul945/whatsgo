package config

import (
	"os"
	"strconv"
	"strings"
)

type Config struct {
	Port             string
	JWTSecret        string
	DatabaseURL      string
	DataDir          string
	WebhookWorkers   int
	WebhookQueueSize int
	RateLimitRPM     int
	MaxMediaSizeMB   int
	AllowedOrigins   []string
	EchoBot          bool
}

func Load() *Config {
	webhookWorkers, _ := strconv.Atoi(os.Getenv("WEBHOOK_WORKERS"))
	if webhookWorkers == 0 {
		webhookWorkers = 10
	}
	webhookQueueSize, _ := strconv.Atoi(os.Getenv("WEBHOOK_QUEUE_SIZE"))
	if webhookQueueSize == 0 {
		webhookQueueSize = 1000
	}
	rateLimitRPM, _ := strconv.Atoi(os.Getenv("RATE_LIMIT_RPM"))
	if rateLimitRPM == 0 {
		rateLimitRPM = 100
	}
	maxMediaSize, _ := strconv.Atoi(os.Getenv("MAX_MEDIA_SIZE"))
	if maxMediaSize == 0 {
		maxMediaSize = 16
	}

	origins := os.Getenv("ALLOWED_ORIGINS")
	if origins == "" {
		origins = "http://localhost:3000,http://localhost:5173,http://localhost:4173,http://127.0.0.1:3000,http://127.0.0.1:5173,http://127.0.0.1:4173"
	}

	return &Config{
		Port:             getEnv("PORT", "8080"),
		JWTSecret:        getEnv("JWT_SECRET", "whatsgo-secret-key-change-in-production"),
		DatabaseURL:      getEnv("DATABASE_URL", "sqlite://./data/whatsgo.db"),
		DataDir:          getEnv("DATA_DIR", "./data"),
		WebhookWorkers:   webhookWorkers,
		WebhookQueueSize: webhookQueueSize,
		RateLimitRPM:     rateLimitRPM,
		MaxMediaSizeMB:   maxMediaSize,
		AllowedOrigins:   strings.Split(origins, ","),
		EchoBot:          os.Getenv("ECHO_BOT") == "1",
	}
}

func getEnv(key, fallback string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return fallback
}
