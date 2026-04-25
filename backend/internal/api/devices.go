package api

import (
	"database/sql"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/skip2/go-qrcode"
	"whatsgo/internal/auth"
	"whatsgo/internal/db"
	"whatsgo/internal/models"
	"whatsgo/internal/whatsapp"
)

func ensureManagedDevice(c *gin.Context, database *db.DB, manager *whatsapp.Manager, deviceID int) (*whatsapp.ClientWrapper, bool) {
	userID := c.GetInt("user_id")

	var ownerID int
	err := database.QueryRow("SELECT user_id FROM devices WHERE id = ?", deviceID).Scan(&ownerID)
	if err == sql.ErrNoRows {
		c.JSON(http.StatusNotFound, gin.H{"success": false, "error": gin.H{"code": "DEVICE_NOT_FOUND", "message": "Device not found", "status": 404}})
		return nil, false
	}
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": "Database error", "status": 500}})
		return nil, false
	}

	if ownerID != userID {
		role, _ := c.Get("role")
		if role != "admin" {
			c.JSON(http.StatusForbidden, gin.H{"success": false, "error": gin.H{"code": "FORBIDDEN", "message": "Access denied", "status": 403}})
			return nil, false
		}
	}

	if wrapper := manager.GetDevice(deviceID); wrapper != nil {
		return wrapper, true
	}

	wrapper, err := manager.LoadDevice(deviceID, ownerID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "DEVICE_LOAD_ERROR", "message": err.Error(), "status": 500}})
		return nil, false
	}

	return wrapper, true
}

func RegisterDeviceRoutes(r *gin.Engine, database *db.DB, manager *whatsapp.Manager) {
	devices := r.Group("/api/devices")
	devices.Use(auth.JWTMiddleware())
	devices.Use(auth.TenantMiddleware(database))
	{
		devices.GET("", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			rows, err := database.Query(
				`SELECT id, user_id, name,
					COALESCE(phone_number, ''),
					COALESCE(jid, ''),
					COALESCE(push_name, ''),
					status,
					COALESCE(webhook_url, ''),
					COALESCE(webhook_events, ''),
					COALESCE(webhook_secret, ''),
					created_at, updated_at
				FROM devices WHERE user_id = ? ORDER BY created_at DESC`,
				userID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}
			defer rows.Close()

			var deviceList []models.Device
			for rows.Next() {
				var d models.Device
				err := rows.Scan(&d.ID, &d.UserID, &d.Name, &d.PhoneNumber, &d.JID, &d.PushName, &d.Status, &d.WebhookURL, &d.WebhookEvents, &d.WebhookSecret, &d.CreatedAt, &d.UpdatedAt)
				if err != nil {
					continue
				}
				// Update status from manager if client exists
				if status := manager.GetDeviceStatus(d.ID); status != "" {
					d.Status = status
				}
				deviceList = append(deviceList, d)
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": deviceList})
		})

		devices.POST("", func(c *gin.Context) {
			userID := c.GetInt("user_id")
			var req struct {
				Name       string `json:"name" binding:"required"`
				WebhookURL string `json:"webhook_url"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			res, err := database.Exec(
				"INSERT INTO devices (user_id, name, webhook_url, status) VALUES (?, ?, ?, 'connecting')",
				userID, req.Name, req.WebhookURL,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			deviceID, _ := res.LastInsertId()
			_, err = manager.CreateDevice(int(deviceID), userID, req.Name)
			if err != nil {
				// Rollback
				database.Exec("DELETE FROM devices WHERE id = ?", deviceID)
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.JSON(http.StatusCreated, gin.H{
				"success": true,
				"data": gin.H{
					"id":     deviceID,
					"name":   req.Name,
					"status": "connecting",
				},
			})
		})

		devices.GET("/:id/qr", func(c *gin.Context) {
			deviceID, _ := strconv.Atoi(c.Param("id"))
			wrapper, ok := ensureManagedDevice(c, database, manager, deviceID)
			if !ok {
				return
			}

			if wrapper.Client.Store.ID != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "ALREADY_PAIRED", "message": "Device is already paired", "status": 400}})
				return
			}

			var qrData string
			select {
			case qrData = <-wrapper.QRChannel:
			case <-time.After(20 * time.Second):
				c.JSON(http.StatusRequestTimeout, gin.H{"success": false, "error": gin.H{"code": "QR_TIMEOUT", "message": "QR code is not ready yet. Try connecting again.", "status": 408}})
				return
			}

			png, err := qrcode.Encode(qrData, qrcode.Medium, 256)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.Data(http.StatusOK, "image/png", png)
		})

		devices.POST("/:id/connect", func(c *gin.Context) {
			deviceID, _ := strconv.Atoi(c.Param("id"))
			if _, ok := ensureManagedDevice(c, database, manager, deviceID); !ok {
				return
			}

			err := manager.ConnectDevice(deviceID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "CONNECT_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			// Update DB status
			database.Exec("UPDATE devices SET status = 'connecting' WHERE id = ?", deviceID)

			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "connecting"}})
		})

		devices.POST("/:id/disconnect", func(c *gin.Context) {
			deviceID, _ := strconv.Atoi(c.Param("id"))
			err := manager.DisconnectDevice(deviceID)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "DISCONNECT_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			database.Exec("UPDATE devices SET status = 'disconnected', phone_number = NULL, jid = NULL, push_name = NULL WHERE id = ?", deviceID)

			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"status": "disconnected"}})
		})

		devices.DELETE("/:id", func(c *gin.Context) {
			deviceID, _ := strconv.Atoi(c.Param("id"))
			manager.DeleteDevice(deviceID)
			database.Exec("DELETE FROM devices WHERE id = ?", deviceID)
			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Device deleted"}})
		})

		devices.PUT("/:id/webhook", func(c *gin.Context) {
			deviceID, _ := strconv.Atoi(c.Param("id"))
			var req struct {
				WebhookURL    string   `json:"webhook_url"`
				WebhookEvents []string `json:"webhook_events"`
				WebhookSecret string   `json:"webhook_secret"`
			}
			if err := c.ShouldBindJSON(&req); err != nil {
				c.JSON(http.StatusBadRequest, gin.H{"success": false, "error": gin.H{"code": "VALIDATION_ERROR", "message": err.Error(), "status": 400}})
				return
			}

			events := strings.Join(req.WebhookEvents, ",")
			_, err := database.Exec(
				"UPDATE devices SET webhook_url = ?, webhook_events = ?, webhook_secret = ? WHERE id = ?",
				req.WebhookURL, events, req.WebhookSecret, deviceID,
			)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"success": false, "error": gin.H{"code": "INTERNAL_ERROR", "message": err.Error(), "status": 500}})
				return
			}

			c.JSON(http.StatusOK, gin.H{"success": true, "data": gin.H{"message": "Webhook settings updated"}})
		})
	}
}
