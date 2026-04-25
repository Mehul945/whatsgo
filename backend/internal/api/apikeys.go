package api

import (
	"database/sql"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"whatsgo/internal/auth"
	"whatsgo/internal/db"
	"whatsgo/internal/models"
)

func RegisterAPIKeyRoutes(r *gin.Engine, database *db.DB) {
	keys := r.Group("/api/apikeys")
	keys.Use(auth.JWTMiddleware())
	{
		keys.GET("", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			rows, err := database.Query(
				"SELECT id, user_id, device_id, name, key_prefix, last_used_at, created_at, revoked_at FROM api_keys WHERE user_id = ? AND revoked_at IS NULL ORDER BY created_at DESC",
				userID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}
			defer rows.Close()

			var keyList []models.APIKey
			for rows.Next() {
				var k models.APIKey
				var deviceID sql.NullInt64
				rows.Scan(&k.ID, &k.UserID, &deviceID, &k.Name, &k.KeyPrefix, &k.LastUsedAt, &k.CreatedAt, &k.RevokedAt)
				if deviceID.Valid {
					devID := int(deviceID.Int64)
					k.DeviceID = &devID
				}
				keyList = append(keyList, k)
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": keyList})
		})

		keys.POST("", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			var req struct {
				Name     string `json:"name" binding:"required"`
				DeviceID *int   `json:"device_id"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			fullKey, prefix, hash, err := auth.GenerateAPIKey()
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Key generation failed", "status": 500}})
				return
			}

			var deviceID interface{}
			if req.DeviceID != nil {
				deviceID = *req.DeviceID
			} else {
				deviceID = nil
			}

			_, err = database.Exec(
				"INSERT INTO api_keys (user_id, device_id, name, key_prefix, key_hash) VALUES (?, ?, ?, ?, ?)",
				userID, deviceID, req.Name, prefix, hash,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"data": gin.H{
					"name":      req.Name,
					"api_key":   fullKey,
					"key_prefix": prefix,
					"created_at": time.Now(),
				},
			})
		})

		keys.DELETE("/:id", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			id := c.Param("id")

			_, err := database.Exec(
				"UPDATE api_keys SET revoked_at = ? WHERE id = ? AND user_id = ?",
				time.Now(), id, userID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "API key revoked"}})
		})
	}
}

func RegisterUserRoutes(r *gin.Engine, database *db.DB) {
	users := r.Group("/api/user")
	users.Use(auth.JWTMiddleware())
	{
		users.GET("/profile", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			var user models.User
			err := database.QueryRow(
				"SELECT id, email, name, role, active, rate_limit, created_at, updated_at FROM users WHERE id = ?",
				userID,
			).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &user.Active, &user.RateLimit, &user.CreatedAt, &user.UpdatedAt)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": user})
		})

		users.PUT("/profile", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			var req struct {
				Name     string `json:"name"`
				Email    string `json:"email"`
				Password string `json:"password"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			updates := []string{}
			args := []interface{}{}

			if req.Name != "" {
				updates = append(updates, "name = ?")
				args = append(args, req.Name)
			}
			if req.Email != "" {
				updates = append(updates, "email = ?")
				args = append(args, req.Email)
			}
			if req.Password != "" {
				hash, err := auth.HashPassword(req.Password)
				if err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Password hashing failed", "status": 500}})
					return
				}
				updates = append(updates, "password_hash = ?")
				args = append(args, hash)
			}

			if len(updates) == 0 {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": "No fields to update", "status": 400}})
				return
			}

			args = append(args, userID)
			query := "UPDATE users SET " + joinUpdates(updates) + ", updated_at = CURRENT_TIMESTAMP WHERE id = ?"
			_, err := database.Exec(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Profile updated"}})
		})
	}
}

func RegisterAdminRoutes(r *gin.Engine, database *db.DB) {
	admin := r.Group("/api/admin")
	admin.Use(auth.JWTMiddleware())
	admin.Use(auth.AdminMiddleware())
	{
		admin.GET("/users", func(c *gin.Context) {
			rows, err := database.Query(
				"SELECT id, email, name, role, active, rate_limit, created_at, updated_at FROM users ORDER BY created_at DESC",
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}
			defer rows.Close()

			var users []models.User
			for rows.Next() {
				var u models.User
				rows.Scan(&u.ID, &u.Email, &u.Name, &u.Role, &u.Active, &u.RateLimit, &u.CreatedAt, &u.UpdatedAt)
				users = append(users, u)
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": users})
		})
	}
}

func joinUpdates(updates []string) string {
	result := ""
	for i, u := range updates {
		if i > 0 {
			result += ", "
		}
		result += u
	}
	return result
}
