package auth

import (
	"database/sql"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"whatsgo/internal/db"
)

func JWTMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Missing authorization header", "status": 401}})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid authorization format", "status": 401}})
			c.Abort()
			return
		}

		claims, err := ParseToken(parts[1])
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid or expired token", "status": 401}})
			c.Abort()
			return
		}

		c.Set("user_id", claims.UserID)
		c.Set("email", claims.Email)
		c.Set("role", claims.Role)
		c.Next()
	}
}

func APIKeyMiddleware(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		authHeader := c.GetHeader("Authorization")
		if authHeader == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Missing API key", "status": 401}})
			c.Abort()
			return
		}

		parts := strings.SplitN(authHeader, " ", 2)
		if len(parts) != 2 || strings.ToLower(parts[0]) != "bearer" {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid API key format", "status": 401}})
			c.Abort()
			return
		}

		apiKey := parts[1]
		var keyID, userID, deviceID int
		var keyHash string
		err := database.QueryRow(
			"SELECT id, user_id, COALESCE(device_id, 0), key_hash FROM api_keys WHERE key_prefix = ? AND revoked_at IS NULL",
			apiKey[:min(8, len(apiKey))],
		).Scan(&keyID, &userID, &deviceID, &keyHash)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid API key", "status": 401}})
			c.Abort()
			return
		}

		if !VerifyAPIKey(apiKey, keyHash) {
			c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "UNAUTHORIZED", "message": "Invalid API key", "status": 401}})
			c.Abort()
			return
		}

		database.Exec("UPDATE api_keys SET last_used_at = ? WHERE id = ?", time.Now(), keyID)
		c.Set("user_id", userID)
		c.Set("api_key_id", keyID)
		if deviceID > 0 {
			c.Set("device_id", deviceID)
		}
		c.Next()
	}
}

func AdminMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		role, exists := c.Get("role")
		if !exists || role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "Admin access required", "status": 403}})
			c.Abort()
			return
		}
		c.Next()
	}
}

func TenantMiddleware(database *db.DB) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		deviceID := c.Param("id")
		if deviceID != "" {
			var ownerID int
			err := database.QueryRow("SELECT user_id FROM devices WHERE id = ?", deviceID).Scan(&ownerID)
			if err != nil && err != sql.ErrNoRows {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Database error", "status": 500}})
				c.Abort()
				return
			}
			if err != sql.ErrNoRows && ownerID != userID.(int) {
				role, _ := c.Get("role")
				if role != "admin" {
					c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "Access denied", "status": 403}})
					c.Abort()
					return
				}
			}
		}
		c.Next()
	}
}

func RateLimitMiddleware(database *db.DB, defaultRPM int) gin.HandlerFunc {
	return func(c *gin.Context) {
		userID, exists := c.Get("user_id")
		if !exists {
			c.Next()
			return
		}

		var rpm int
		err := database.QueryRow("SELECT rate_limit FROM users WHERE id = ?", userID.(int)).Scan(&rpm)
		if err != nil || rpm == 0 {
			rpm = defaultRPM
		}

		now := time.Now().UTC()
		windowStart := now.Truncate(time.Minute)

		var requests int
		err = database.QueryRow(
			"SELECT requests FROM rate_limits WHERE user_id = ? AND window_start = ?",
			userID.(int), windowStart,
		).Scan(&requests)
		if err != nil {
			_, _ = database.Exec(
				"INSERT OR REPLACE INTO rate_limits (user_id, requests, window_start) VALUES (?, 1, ?)",
				userID.(int), windowStart,
			)
			c.Next()
			return
		}

		if requests >= rpm {
			c.JSON(http.StatusTooManyRequests, gin.H{
				"success": false,
				"error": gin.H{"code": "RATE_LIMIT_EXCEEDED", "message": "Too many requests. Try again later.", "status": 429},
			})
			c.Abort()
			return
		}

		_, _ = database.Exec(
			"UPDATE rate_limits SET requests = requests + 1 WHERE user_id = ? AND window_start = ?",
			userID.(int), windowStart,
		)
		c.Next()
	}
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
