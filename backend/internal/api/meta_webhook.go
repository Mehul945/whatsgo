package api

import (
	"encoding/json"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"whatsgo/internal/db"
	"whatsgo/internal/whatsapp"
)

// RegisterMetaWebhookRoutes registers Meta-compatible webhook verification
// and inbound webhook receiver endpoints.
//
// GET  /webhook  — Hub verification (Frappe / Meta handshake)
// POST /webhook  — Receive inbound events from external services
func RegisterMetaWebhookRoutes(r *gin.Engine, database *db.DB, manager *whatsapp.Manager) {
	// ── GET /webhook — Meta hub verification ────────────────────────────────
	// Frappe WhatsApp (and Meta Cloud API) send a GET request to verify the
	// webhook endpoint before subscribing. The request contains:
	//   hub.mode         = "subscribe"
	//   hub.verify_token = the token configured in Frappe settings
	//   hub.challenge    = a random string to echo back
	//
	// WhatsGo checks the verify token against each device's webhook_secret
	// in the database. If any device's secret matches, verification succeeds.
	// This keeps the verify token per-tenant/per-device rather than a global
	// env var.
	r.GET("/webhook", func(c *gin.Context) {
		mode := c.Query("hub.mode")
		token := c.Query("hub.verify_token")
		challenge := c.Query("hub.challenge")

		if mode != "subscribe" || token == "" {
			log.Printf("[meta-webhook] Verification failed: invalid mode=%q or empty token", mode)
			c.JSON(http.StatusForbidden, gin.H{"error": "Verification failed"})
			return
		}

		// Look up the verify token against all devices' webhook_secret
		var deviceID int
		err := database.QueryRow(
			"SELECT id FROM devices WHERE webhook_secret = ? LIMIT 1",
			token,
		).Scan(&deviceID)

		if err != nil {
			log.Printf("[meta-webhook] Verification failed: no device matches token")
			c.JSON(http.StatusForbidden, gin.H{"error": "Verification failed"})
			return
		}

		log.Printf("[meta-webhook] Verification successful for device %d", deviceID)
		c.String(http.StatusOK, challenge)
	})

	// ── POST /webhook — Receive inbound events ─────────────────────────────
	// Frappe or other Meta-compatible services may POST events here.
	// This endpoint accepts the payload, logs it, and returns 200 OK so
	// the sender does not retry.
	r.POST("/webhook", func(c *gin.Context) {
		var payload map[string]interface{}
		if err := c.ShouldBindJSON(&payload); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid JSON"})
			return
		}

		raw, _ := json.Marshal(payload)
		log.Printf("[meta-webhook] Inbound event received: %s", string(raw))

		// Acknowledge receipt immediately (Meta expects 200 within 5 s)
		c.JSON(http.StatusOK, gin.H{"status": "received"})
	})
}
