package api

import (
	"database/sql"
	"net/http"

	"github.com/gin-gonic/gin"
	"whatsgo/internal/auth"
	"whatsgo/internal/db"
	"whatsgo/internal/models"
)

func RegisterAuthRoutes(r *gin.Engine, database *db.DB) {
	authGroup := r.Group("/api/auth")
	{
		authGroup.POST("/register", func(c *gin.Context) {
			var req struct {
				Email    string `json:"email" binding:"required,email"`
				Password string `json:"password" binding:"required,min=6"`
				Name     string `json:"name" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			var exists int
			err := database.QueryRow("SELECT 1 FROM users WHERE email = ?", req.Email).Scan(&exists)
			if err != sql.ErrNoRows {
				c.JSON(http.StatusConflict, gin.H{"success": false, "error": gin.H{"code": "EMAIL_EXISTS", "message": "Email already registered", "status": 409}})
				return
			}

			hash, err := auth.HashPassword(req.Password)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Password hashing failed", "status": 500}})
				return
			}

			res, err := database.Exec(
				"INSERT INTO users (email, password_hash, name) VALUES (?, ?, ?)",
				req.Email, hash, req.Name,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Registration failed", "status": 500}})
				return
			}

			userID, _ := res.LastInsertId()
			accessToken, refreshToken, err := auth.GenerateTokens(int(userID), req.Email, "user")
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Token generation failed", "status": 500}})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"data": gin.H{
					"user": gin.H{
						"id":    userID,
						"email": req.Email,
						"name":  req.Name,
						"role":  "user",
					},
					"access_token":  accessToken,
					"refresh_token": refreshToken,
				},
			})
		})

		authGroup.POST("/login", func(c *gin.Context) {
			var req struct {
				Email    string `json:"email" binding:"required,email"`
				Password string `json:"password" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			var user models.User
			var passwordHash string
			err := database.QueryRow(
				"SELECT id, email, name, role, password_hash FROM users WHERE email = ? AND active = 1",
				req.Email,
			).Scan(&user.ID, &user.Email, &user.Name, &user.Role, &passwordHash)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password", "status": 401}})
				return
			}

			if !auth.CheckPassword(req.Password, passwordHash) {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "INVALID_CREDENTIALS", "message": "Invalid email or password", "status": 401}})
				return
			}

			accessToken, refreshToken, err := auth.GenerateTokens(user.ID, user.Email, user.Role)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Token generation failed", "status": 500}})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"user": gin.H{
						"id":    user.ID,
						"email": user.Email,
						"name":  user.Name,
						"role":  user.Role,
					},
					"access_token":  accessToken,
					"refresh_token": refreshToken,
				},
			})
		})

		authGroup.POST("/refresh", func(c *gin.Context) {
			var req struct {
				RefreshToken string `json:"refresh_token" binding:"required"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			claims, err := auth.ParseToken(req.RefreshToken)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "INVALID_TOKEN", "message": "Invalid refresh token", "status": 401}})
				return
			}

			var user models.User
			err = database.QueryRow(
				"SELECT id, email, name, role FROM users WHERE id = ? AND active = 1",
				claims.UserID,
			).Scan(&user.ID, &user.Email, &user.Name, &user.Role)
			if err != nil {
				c.JSON(http.StatusUnauthorized, gin.H{"success": false, "error": gin.H{"code": "USER_NOT_FOUND", "message": "User not found", "status": 401}})
				return
			}

			accessToken, refreshToken, err := auth.GenerateTokens(user.ID, user.Email, user.Role)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Token generation failed", "status": 500}})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"access_token":  accessToken,
					"refresh_token": refreshToken,
				},
			})
		})
	}
}
