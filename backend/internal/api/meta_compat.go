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

// RegisterMetaRoutes registers Meta API compatible endpoints
func RegisterMetaRoutes(r *gin.Engine, database *db.DB, manager *whatsapp.Manager) {
	meta := r.Group("/v1")
	meta.Use(auth.APIKeyMiddleware(database))
	meta.Use(auth.TenantMiddleware(database))
	{
		// Send message
		meta.POST("/messages", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			deviceID := c.GetInt("device_id")

			var req MetaMessageRequest
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{
					"messaging_product": "whatsapp",
					"error":             gin.H{"code": "INVALID_REQUEST", "message": err.Error()},
				})
				return
			}

			// If no device_id from API key scope, use from request or first device
			if deviceID == 0 {
				if req.From != "" {
					var id int
					err := database.QueryRow("SELECT id FROM devices WHERE user_id = ? AND (phone_number = ? OR id = ?)", userID, req.From, req.From).Scan(&id)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"messaging_product": "whatsapp",
							"error":             gin.H{"code": "INVALID_DEVICE", "message": "Device not found"},
						})
						return
					}
					deviceID = id
				} else {
					var id int
					err := database.QueryRow("SELECT id FROM devices WHERE user_id = ? AND status = 'connected' ORDER BY id LIMIT 1", userID).Scan(&id)
					if err != nil {
						c.JSON(http.StatusBadRequest, gin.H{
							"messaging_product": "whatsapp",
							"error":             gin.H{"code": "NO_DEVICE", "message": "No connected device found"},
						})
						return
					}
					deviceID = id
				}
			}

			var msgID string
			var msgStatus = "pending"
			var err error

			switch req.Type {
			case "text":
				if req.Text != nil {
					msgID, err = manager.SendTextMessage(deviceID, req.To, req.Text.Body)
					if err != nil {
						msgStatus = "failed"
					} else {
						msgStatus = "sent"
					}
				}
			default:
				msgID, err = manager.SendTextMessage(deviceID, req.To, fmt.Sprintf("[%s] Message", req.Type))
				if err != nil {
					msgStatus = "failed"
				} else {
					msgStatus = "sent"
				}
			}

			// Log message
			_, _ = database.Exec(
				"INSERT INTO messages (device_id, user_id, direction, recipient, message_type, content, status, whatsapp_message_id) VALUES (?, ?, 'outbound', ?, ?, ?, ?, ?)",
				deviceID, userID, req.To, req.Type, req.GetContent(), msgStatus, msgID,
			)
			manager.ApplyCachedReceiptStatus(deviceID, msgID)

			if msgStatus == "failed" {
				c.JSON(http.StatusInternalServerError, gin.H{
					"messaging_product": "whatsapp",
					"error":             gin.H{"code": "SEND_FAILED", "message": err.Error()},
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"messaging_product": "whatsapp",
				"contacts": []gin.H{
					{"input": req.To, "wa_id": req.To},
				},
				"messages": []gin.H{
					{"id": msgID},
				},
			})
		})

		// List messages
		meta.GET("/messages", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			deviceID := c.GetInt("device_id")
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "20"))
			offset, _ := strconv.Atoi(c.DefaultQuery("offset", "0"))

			query := `SELECT id, device_id, direction,
				COALESCE(recipient, ''),
				message_type,
				COALESCE(content, ''),
				status,
				COALESCE(whatsapp_message_id, ''),
				created_at
				FROM messages WHERE user_id = ?`
			args := []interface{}{userID}

			if deviceID > 0 {
				query += " AND device_id = ?"
				args = append(args, deviceID)
			}

			query += " ORDER BY created_at DESC LIMIT ? OFFSET ?"
			args = append(args, limit, offset)

			rows, err := database.Query(query, args...)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{
					"messaging_product": "whatsapp",
					"error":             gin.H{"code": "INTERNAL_ERROR", "message": err.Error()},
				})
				return
			}
			defer rows.Close()

			var messages []gin.H
			for rows.Next() {
				var m models.Message
				if err := rows.Scan(&m.ID, &m.DeviceID, &m.Direction, &m.Recipient, &m.MessageType, &m.Content, &m.Status, &m.WhatsAppMessageID, &m.CreatedAt); err != nil {
					c.JSON(http.StatusInternalServerError, gin.H{
						"messaging_product": "whatsapp",
						"error":             gin.H{"code": "INTERNAL_ERROR", "message": err.Error()},
					})
					return
				}
				messages = append(messages, gin.H{
					"id":         m.ID,
					"direction":  m.Direction,
					"to":         m.Recipient,
					"type":       m.MessageType,
					"content":    m.Content,
					"status":     m.Status,
					"message_id": m.WhatsAppMessageID,
					"timestamp":  m.CreatedAt.Unix(),
				})
			}

			c.JSON(http.StatusOK, gin.H{
				"messaging_product": "whatsapp",
				"data":              messages,
				"paging": gin.H{
					"limit":  limit,
					"offset": offset,
				},
			})
		})

		// Business profile
		meta.GET("/business/profile", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			deviceID := c.GetInt("device_id")

			var d models.Device
			var err error
			if deviceID > 0 {
				err = database.QueryRow(
					"SELECT id, name, phone_number, jid, push_name, status, created_at FROM devices WHERE id = ? AND user_id = ?",
					deviceID, userID,
				).Scan(&d.ID, &d.Name, &d.PhoneNumber, &d.JID, &d.PushName, &d.Status, &d.CreatedAt)
			} else {
				err = database.QueryRow(
					"SELECT id, name, phone_number, jid, push_name, status, created_at FROM devices WHERE user_id = ? AND status = 'connected' ORDER BY id LIMIT 1",
					userID,
				).Scan(&d.ID, &d.Name, &d.PhoneNumber, &d.JID, &d.PushName, &d.Status, &d.CreatedAt)
			}

			if err != nil {
				c.JSON(http.StatusNotFound, gin.H{
					"messaging_product": "whatsapp",
					"error":             gin.H{"code": "NOT_FOUND", "message": "No device found"},
				})
				return
			}

			c.JSON(http.StatusOK, gin.H{
				"messaging_product": "whatsapp",
				"id":                d.ID,
				"name":              d.Name,
				"phone_number":      d.PhoneNumber,
				"status":            d.Status,
				"created_at":        d.CreatedAt.Format(time.RFC3339),
			})
		})

		// Webhooks config
		meta.GET("/webhooks", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			deviceID := c.GetInt("device_id")

			var webhookURL, webhookEvents string
			var query string
			if deviceID > 0 {
				query = "SELECT webhook_url, webhook_events FROM devices WHERE id = ? AND user_id = ?"
				database.QueryRow(query, deviceID, userID).Scan(&webhookURL, &webhookEvents)
			} else {
				query = "SELECT webhook_url, webhook_events FROM devices WHERE user_id = ? LIMIT 1"
				database.QueryRow(query, userID).Scan(&webhookURL, &webhookEvents)
			}

			c.JSON(http.StatusOK, gin.H{
				"messaging_product": "whatsapp",
				"webhook_url":       webhookURL,
				"webhook_events":    webhookEvents,
			})
		})
	}
}

type MetaMessageRequest struct {
	MessagingProduct string `json:"messaging_product"`
	RecipientType    string `json:"recipient_type"`
	To               string `json:"to" binding:"required"`
	Type             string `json:"type" binding:"required"`
	From             string `json:"from"`
	Text             *struct {
		Body string `json:"body"`
	} `json:"text"`
	Image *struct {
		Link    string `json:"link"`
		Caption string `json:"caption"`
	} `json:"image"`
	Template *struct {
		Name     string `json:"name"`
		Language struct {
			Code string `json:"code"`
		} `json:"language"`
	} `json:"template"`
}

func (m MetaMessageRequest) GetContent() string {
	if m.Text != nil {
		return m.Text.Body
	}
	if m.Image != nil {
		return m.Image.Caption
	}
	if m.Template != nil {
		return m.Template.Name
	}
	return ""
}
