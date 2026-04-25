package api

import (
	"fmt"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"whatsgo/internal/auth"
	"whatsgo/internal/db"
	"whatsgo/internal/models"
	"whatsgo/internal/whatsapp"
)

func RegisterMessageRoutes(r *gin.Engine, database *db.DB, manager *whatsapp.Manager) {
	messages := r.Group("/api/messages")
	messages.Use(auth.JWTMiddleware())
	messages.Use(auth.TenantMiddleware(database))
	{
		messages.GET("", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))
			direction := c.Query("direction")
			deviceID := c.Query("device_id")
			status := c.Query("status")

			query := `SELECT id, device_id, user_id, direction,
				COALESCE(recipient, ''),
				message_type,
				COALESCE(content, ''),
				COALESCE(media_url, ''),
				COALESCE(media_mime_type, ''),
				status,
				COALESCE(whatsapp_message_id, ''),
				COALESCE(error_message, ''),
				created_at,
				updated_at
				FROM messages WHERE user_id = ?`
			args := []interface{}{userID}

			if direction != "" {
				query += " AND direction = ?"
				args = append(args, direction)
			}
			if deviceID != "" {
				query += " AND device_id = ?"
				args = append(args, deviceID)
			}
			if status != "" {
				query += " AND status = ?"
				args = append(args, status)
			}

			countQuery := "SELECT COUNT(*) FROM messages WHERE user_id = ?"
			countArgs := []interface{}{userID}
			if direction != "" {
				countQuery += " AND direction = ?"
				countArgs = append(countArgs, direction)
			}
			if deviceID != "" {
				countQuery += " AND device_id = ?"
				countArgs = append(countArgs, deviceID)
			}
			if status != "" {
				countQuery += " AND status = ?"
				countArgs = append(countArgs, status)
			}

			var total int
			database.QueryRow(countQuery, countArgs...).Scan(&total)

			query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
			args = append(args, limit, offset)

			rows, err := database.Query(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}
			defer rows.Close()

			var messageList []models.Message
			for rows.Next() {
				var m models.Message
				if err := rows.Scan(&m.ID, &m.DeviceID, &m.UserID, &m.Direction, &m.Recipient, &m.MessageType, &m.Content, &m.MediaURL, &m.MediaMimeType, &m.Status, &m.WhatsAppMessageID, &m.ErrorMessage, &m.CreatedAt, &m.UpdatedAt); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
					return
				}
				messageList = append(messageList, m)
			}

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data":    messageList,
				"meta": gin.H{
					"page":   offset/limit + 1,
					"limit":  limit,
					"total":  total,
					"offset": offset,
				},
			})
		})

		messages.POST("/send", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			var req struct {
				DeviceID  int    `json:"device_id" binding:"required"`
				Recipient string `json:"recipient" binding:"required"`
				Type      string `json:"type" binding:"required"`
				Content   string `json:"content"`
				MediaURL  string `json:"media_url"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			if _, ok := ensureManagedDevice(c, database, manager, req.DeviceID); !ok {
				return
			}

			var msgID string
			var msgStatus = "pending"
			var errorMsg string
			var err error

			switch req.Type {
			case "text":
				msgID, err = manager.SendTextMessage(req.DeviceID, req.Recipient, req.Content)
				if err != nil {
					msgStatus = "failed"
					errorMsg = err.Error()
				} else {
					msgStatus = "sent"
				}
			case "image":
				// For demo purposes, send as text with image notice
				msgID, err = manager.SendTextMessage(req.DeviceID, req.Recipient, fmt.Sprintf("[Image] %s", req.Content))
				if err != nil {
					msgStatus = "failed"
					errorMsg = err.Error()
				} else {
					msgStatus = "sent"
				}
			default:
				msgID, err = manager.SendTextMessage(req.DeviceID, req.Recipient, req.Content)
				if err != nil {
					msgStatus = "failed"
					errorMsg = err.Error()
				} else {
					msgStatus = "sent"
				}
			}

			res, err := database.Exec(
				"INSERT INTO messages (device_id, user_id, direction, recipient, message_type, content, status, whatsapp_message_id, error_message) VALUES (?, ?, 'outbound', ?, ?, ?, ?, ?, ?)",
				req.DeviceID, userID, req.Recipient, req.Type, req.Content, msgStatus, msgID, errorMsg,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			messageID, _ := res.LastInsertId()
			manager.ApplyCachedReceiptStatus(req.DeviceID, msgID)

			c.JSON(http.StatusOK, gin.H{
				"success": true,
				"data": gin.H{
					"id":         messageID,
					"status":     msgStatus,
					"message_id": msgID,
				},
			})
		})
	}
}

func RegisterDashboardRoutes(r *gin.Engine, database *db.DB, manager *whatsapp.Manager) {
	dashboard := r.Group("/api/dashboard")
	dashboard.Use(auth.JWTMiddleware())
	{
		dashboard.GET("/stats", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			now := time.Now()
			today := now.Format("2006-01-02")
			monthStart := now.Format("2006-01") + "-01"

			var stats models.DashboardStats

			// Active devices
			database.QueryRow(
				"SELECT COUNT(*) FROM devices WHERE user_id = ? AND status = 'connected'",
				userID,
			).Scan(&stats.ActiveDevices)

			// Messages today
			database.QueryRow(
				"SELECT COUNT(*) FROM messages WHERE user_id = ? AND date(created_at) = ?",
				userID, today,
			).Scan(&stats.MessagesToday)

			// Messages this month
			database.QueryRow(
				"SELECT COUNT(*) FROM messages WHERE user_id = ? AND created_at >= ?",
				userID, monthStart,
			).Scan(&stats.MessagesThisMonth)

			// Webhook deliveries today
			database.QueryRow(
				"SELECT COUNT(*) FROM webhook_logs WHERE device_id IN (SELECT id FROM devices WHERE user_id = ?) AND date(created_at) = ?",
				userID, today,
			).Scan(&stats.WebhookDeliveries)

			c.JSON(http.StatusOK, gin.H{"success": true, "data": stats})
		})
	}
}

func RegisterWebhookRoutes(r *gin.Engine, database *db.DB) {
	webhooks := r.Group("/api/webhooks")
	webhooks.Use(auth.JWTMiddleware())
	{
		webhooks.GET("/logs", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			rows, err := database.Query(
				"SELECT w.id, w.device_id, w.event_type, w.payload, w.url, w.status_code, w.response_time_ms, w.success, w.error, w.attempt_number, w.created_at FROM webhook_logs w INNER JOIN devices d ON w.device_id = d.id WHERE d.user_id = ? ORDER BY w.created_at DESC LIMIT ? OFFSET ?",
				userID, limit, offset,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}
			defer rows.Close()

			var logs []models.WebhookLog
			for rows.Next() {
				var w models.WebhookLog
				rows.Scan(&w.ID, &w.DeviceID, &w.EventType, &w.Payload, &w.URL, &w.StatusCode, &w.ResponseTimeMs, &w.Success, &w.Error, &w.AttemptNumber, &w.CreatedAt)
				logs = append(logs, w)
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": logs})
		})
	}
}
