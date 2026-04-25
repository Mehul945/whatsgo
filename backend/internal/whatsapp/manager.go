package whatsapp

import (
	"context"
	"crypto/sha256"
	"errors"
	"fmt"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"go.mau.fi/whatsmeow"
	"go.mau.fi/whatsmeow/proto/waE2E"
	"go.mau.fi/whatsmeow/store/sqlstore"
	"go.mau.fi/whatsmeow/types"
	"go.mau.fi/whatsmeow/types/events"
	waLog "go.mau.fi/whatsmeow/util/log"
	"google.golang.org/protobuf/proto"
	"whatsgo/internal/db"
	"whatsgo/internal/webhook"
)

// Manager holds multiple WhatsApp clients
type Manager struct {
	clients         map[int]*ClientWrapper
	mu              sync.RWMutex
	dataDir         string
	db              *db.DB
	webhook         *webhook.Dispatcher
	echoBot         bool
	receiptStatuses map[string]string
	receiptMu       sync.Mutex
}

type ClientWrapper struct {
	DeviceID    int
	UserID      int
	Client      *whatsmeow.Client
	Store       *sqlstore.Container
	QRChannel   chan string // base64 QR codes
	Status      string
	PhoneNumber string
	JID         types.JID
	EventID     uint32
}

func NewManager(dataDir string, echoBot bool) *Manager {
	return &Manager{
		clients:         make(map[int]*ClientWrapper),
		dataDir:         dataDir,
		echoBot:         echoBot,
		receiptStatuses: make(map[string]string),
	}
}

func (m *Manager) SetDatabase(database *db.DB) {
	m.db = database
}

func (m *Manager) SetWebhookDispatcher(dispatcher *webhook.Dispatcher) {
	m.webhook = dispatcher
}

func (m *Manager) CreateDevice(deviceID int, userID int, name string) (*ClientWrapper, error) {
	return m.loadDevice(deviceID, userID, "connecting", false)
}

func (m *Manager) LoadDevice(deviceID int, userID int) (*ClientWrapper, error) {
	return m.loadDevice(deviceID, userID, "disconnected", true)
}

func (m *Manager) loadDevice(deviceID int, userID int, status string, loadExisting bool) (*ClientWrapper, error) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[deviceID]; exists {
		if loadExisting {
			return m.clients[deviceID], nil
		}
		return nil, fmt.Errorf("device %d already exists", deviceID)
	}

	deviceDir := filepath.Join(m.dataDir, "devices", fmt.Sprintf("%d", deviceID))
	if err := os.MkdirAll(deviceDir, 0755); err != nil {
		return nil, fmt.Errorf("create device dir: %w", err)
	}

	dbPath := filepath.Join(deviceDir, "store.db")
	container, err := sqlstore.New(context.Background(), "sqlite3", "file:"+dbPath+"?_foreign_keys=on", waLog.Noop)
	if err != nil {
		return nil, fmt.Errorf("create store: %w", err)
	}

	store, err := container.GetFirstDevice(context.Background())
	if err != nil {
		return nil, fmt.Errorf("load device store: %w", err)
	}

	clientLog := waLog.Noop
	client := whatsmeow.NewClient(store, clientLog)

	wrapper := &ClientWrapper{
		DeviceID:  deviceID,
		UserID:    userID,
		Client:    client,
		Store:     container,
		QRChannel: make(chan string, 10),
		Status:    status,
	}
	if store.ID != nil {
		wrapper.JID = *store.ID
		wrapper.PhoneNumber = wrapper.JID.User
	}

	m.clients[deviceID] = wrapper
	return wrapper, nil
}

func (m *Manager) ConnectDevice(deviceID int) error {
	m.mu.RLock()
	wrapper, exists := m.clients[deviceID]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("device %d not found", deviceID)
	}

	// Always register event handler first, even if already connected
	if wrapper.EventID == 0 {
		wrapper.EventID = wrapper.Client.AddEventHandler(func(evt interface{}) {
			m.handleEvent(deviceID, wrapper.UserID, evt)
		})
		log.Printf("[Device %d] Event handler registered", deviceID)
	}

	if wrapper.Client.IsConnected() {
		if wrapper.Client.IsLoggedIn() {
			wrapper.Status = "connected"
		} else {
			wrapper.Status = "connecting"
		}
		return nil
	}

	if wrapper.Client.Store.ID == nil {
		// Need QR pairing
		qrChan, _ := wrapper.Client.GetQRChannel(context.Background())
		go func() {
			for evt := range qrChan {
				if evt.Event == "code" {
					select {
					case wrapper.QRChannel <- evt.Code:
					default:
					}
				}
			}
		}()
	}

	err := wrapper.Client.Connect()
	if err != nil {
		if errors.Is(err, whatsmeow.ErrAlreadyConnected) {
			if wrapper.Client.IsLoggedIn() {
				wrapper.Status = "connected"
			} else {
				wrapper.Status = "connecting"
			}
			return nil
		}
		wrapper.Status = "disconnected"
		return fmt.Errorf("connect: %w", err)
	}

	if wrapper.Client.Store.ID != nil {
		wrapper.JID = *wrapper.Client.Store.ID
		wrapper.PhoneNumber = wrapper.JID.User
		wrapper.Status = "connected"
	}

	log.Printf("[Device %d] Connected successfully (phone: %s)", deviceID, wrapper.PhoneNumber)
	return nil
}

func (m *Manager) DisconnectDevice(deviceID int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	wrapper, exists := m.clients[deviceID]
	if !exists {
		return fmt.Errorf("device %d not found", deviceID)
	}

	wrapper.Client.Disconnect()
	wrapper.Status = "disconnected"
	close(wrapper.QRChannel)
	delete(m.clients, deviceID)
	return nil
}

func (m *Manager) DeleteDevice(deviceID int) error {
	m.DisconnectDevice(deviceID)

	deviceDir := filepath.Join(m.dataDir, "devices", fmt.Sprintf("%d", deviceID))
	os.RemoveAll(deviceDir)

	return nil
}

func (m *Manager) GetDevice(deviceID int) *ClientWrapper {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.clients[deviceID]
}

func (m *Manager) GetDeviceStatus(deviceID int) string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if wrapper, exists := m.clients[deviceID]; exists {
		return wrapper.Status
	}
	return "disconnected"
}

func (m *Manager) SendTextMessage(deviceID int, recipient string, content string) (string, error) {
	wrapper := m.GetDevice(deviceID)
	if wrapper == nil {
		return "", fmt.Errorf("device not found")
	}
	if wrapper.Status != "connected" {
		if wrapper.Client.Store.ID == nil {
			return "", fmt.Errorf("device not connected")
		}
		if err := m.ConnectDevice(deviceID); err != nil {
			return "", fmt.Errorf("device not connected: %w", err)
		}
		wrapper = m.GetDevice(deviceID)
		if wrapper == nil || wrapper.Status != "connected" {
			return "", fmt.Errorf("device not connected")
		}
	}

	jid, err := parseJID(recipient)
	if err != nil {
		return "", err
	}

	msg := &waE2E.Message{
		Conversation: proto.String(content),
	}

	resp, err := wrapper.Client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", fmt.Errorf("send message: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) SendImageMessage(deviceID int, recipient string, imageData []byte, caption string) (string, error) {
	wrapper := m.GetDevice(deviceID)
	if wrapper == nil {
		return "", fmt.Errorf("device not found")
	}
	if wrapper.Status != "connected" {
		if wrapper.Client.Store.ID == nil {
			return "", fmt.Errorf("device not connected")
		}
		if err := m.ConnectDevice(deviceID); err != nil {
			return "", fmt.Errorf("device not connected: %w", err)
		}
		wrapper = m.GetDevice(deviceID)
		if wrapper == nil || wrapper.Status != "connected" {
			return "", fmt.Errorf("device not connected")
		}
	}

	jid, err := parseJID(recipient)
	if err != nil {
		return "", err
	}

	uploaded, err := wrapper.Client.Upload(context.Background(), imageData, whatsmeow.MediaImage)
	if err != nil {
		return "", fmt.Errorf("upload image: %w", err)
	}

	msg := &waE2E.Message{
		ImageMessage: &waE2E.ImageMessage{
			Caption:       proto.String(caption),
			URL:           proto.String(uploaded.URL),
			DirectPath:    proto.String(uploaded.DirectPath),
			MediaKey:      uploaded.MediaKey,
			FileEncSHA256: uploaded.FileEncSHA256,
			FileSHA256:    uploaded.FileSHA256,
			FileLength:    proto.Uint64(uint64(len(imageData))),
			Mimetype:      proto.String("image/jpeg"),
		},
	}

	resp, err := wrapper.Client.SendMessage(context.Background(), jid, msg)
	if err != nil {
		return "", fmt.Errorf("send image: %w", err)
	}

	return resp.ID, nil
}

func (m *Manager) handleEvent(deviceID int, userID int, evt interface{}) {
	switch v := evt.(type) {
	case *events.Connected:
		m.updateDeviceStatus(deviceID, "connected")
	case *events.Disconnected:
		m.updateDeviceStatus(deviceID, "disconnected")
	case *events.LoggedOut:
		m.updateDeviceStatus(deviceID, "disconnected")
	case *events.Message:
		m.handleMessage(deviceID, userID, v)
	case *events.Receipt:
		m.handleReceipt(deviceID, v)
	}
}

func (m *Manager) updateDeviceStatus(deviceID int, status string) {
	m.mu.Lock()
	if wrapper, exists := m.clients[deviceID]; exists {
		wrapper.Status = status
	}
	m.mu.Unlock()

	// Persist status to database for auto-reconnect on restart
	if m.db != nil {
		m.db.Exec("UPDATE devices SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE id = ?", status, deviceID)
	}
	log.Printf("[Device %d] Status changed to: %s", deviceID, status)
}

func (m *Manager) handleMessage(deviceID int, userID int, msg *events.Message) {
	if msg.Info.IsFromMe {
		return
	}

	// Guard against nil message
	if msg.Message == nil {
		log.Printf("[Device %d] Received event with nil message body (ID: %s), skipping", deviceID, msg.Info.ID)
		return
	}

	// Extract message info
	from := msg.Info.Sender.User
	if from == "" {
		from = msg.Info.Chat.User
	}
	msgID := msg.Info.ID
	ts := msg.Info.Timestamp

	log.Printf("[Device %d] Received message from %s (ID: %s)", deviceID, from, msgID)

	// Determine message type and content
	var msgType string
	var content string
	var mediaData []byte
	var mimeType string

	if msg.Message.GetConversation() != "" {
		msgType = "text"
		content = msg.Message.GetConversation()
	} else if msg.Message.GetExtendedTextMessage() != nil {
		msgType = "text"
		content = msg.Message.GetExtendedTextMessage().GetText()
	} else if msg.Message.GetImageMessage() != nil {
		msgType = "image"
		img := msg.Message.GetImageMessage()
		content = img.GetCaption()
		mimeType = img.GetMimetype()
		mediaData, _ = m.GetDevice(deviceID).Client.Download(context.Background(), img)
	} else if msg.Message.GetVideoMessage() != nil {
		msgType = "video"
		vid := msg.Message.GetVideoMessage()
		content = vid.GetCaption()
		mimeType = vid.GetMimetype()
		mediaData, _ = m.GetDevice(deviceID).Client.Download(context.Background(), vid)
	} else if msg.Message.GetAudioMessage() != nil {
		msgType = "audio"
		aud := msg.Message.GetAudioMessage()
		mimeType = aud.GetMimetype()
		mediaData, _ = m.GetDevice(deviceID).Client.Download(context.Background(), aud)
	} else if msg.Message.GetDocumentMessage() != nil {
		msgType = "document"
		doc := msg.Message.GetDocumentMessage()
		content = doc.GetCaption()
		mimeType = doc.GetMimetype()
		mediaData, _ = m.GetDevice(deviceID).Client.Download(context.Background(), doc)
	} else {
		msgType = "text"
		content = "[Unsupported message type]"
	}

	// Save media if present
	var mediaURL string
	var mediaSHA256 string
	if len(mediaData) > 0 {
		h := sha256.New()
		h.Write(mediaData)
		mediaSHA256 = fmt.Sprintf("%x", h.Sum(nil))[:16]
		mediaDir := filepath.Join(m.dataDir, "media", fmt.Sprintf("%d", deviceID))
		os.MkdirAll(mediaDir, 0755)
		ext := extFromMime(mimeType)
		mediaPath := filepath.Join(mediaDir, mediaSHA256+ext)
		os.WriteFile(mediaPath, mediaData, 0644)
		mediaURL = mediaPath
	}

	m.saveInboundMessage(deviceID, userID, from, msgID, msgType, content, mediaURL, mimeType, mediaSHA256, ts)

	// Echo bot mode
	if m.echoBot && msgType == "text" && content != "" {
		wrapper := m.GetDevice(deviceID)
		if wrapper != nil {
			reply := fmt.Sprintf("Echo: %s", content)
			jid, _ := parseJID(from)
			wrapper.Client.SendMessage(context.Background(), jid, &waE2E.Message{
				Conversation: proto.String(reply),
			})
		}
	}
}

func (m *Manager) saveInboundMessage(deviceID int, userID int, from string, msgID string, msgType string, content string, mediaURL string, mimeType string, mediaSHA256 string, ts time.Time) {
	if m.db == nil {
		log.Printf("[Device %d] Cannot save inbound message: database is nil", deviceID)
		return
	}
	if msgID == "" {
		log.Printf("[Device %d] Cannot save inbound message: empty message ID", deviceID)
		return
	}

	result, err := m.db.Exec(
		`INSERT INTO messages (device_id, user_id, direction, recipient, message_type, content, media_url, media_mime_type, media_sha256, status, whatsapp_message_id, created_at, updated_at)
		SELECT ?, ?, 'inbound', ?, ?, ?, ?, ?, ?, 'delivered', ?, ?, CURRENT_TIMESTAMP
		WHERE NOT EXISTS (
			SELECT 1 FROM messages WHERE device_id = ? AND direction = 'inbound' AND whatsapp_message_id = ?
		)`,
		deviceID, userID, from, msgType, content, mediaURL, mimeType, mediaSHA256, msgID, ts, deviceID, msgID,
	)
	if err != nil {
		log.Printf("[Device %d] Failed to save inbound message %s: %v", deviceID, msgID, err)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	if rowsAffected > 0 {
		log.Printf("[Device %d] Saved inbound message from %s (ID: %s, type: %s)", deviceID, from, msgID, msgType)
		m.queueInboundWebhook(deviceID, from, msgID, msgType, content, ts)
	}
}

func (m *Manager) handleReceipt(deviceID int, receipt *events.Receipt) {
	status := receiptStatus(receipt.Type)
	if status == "" || m.db == nil {
		return
	}

	for _, messageID := range receipt.MessageIDs {
		messageID := string(messageID)
		if messageID == "" {
			continue
		}

		result, err := m.db.Exec(
			"UPDATE messages SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE device_id = ? AND direction = 'outbound' AND whatsapp_message_id = ? AND status IN ("+allowedPreviousStatuses(status)+")",
			status, deviceID, messageID,
		)
		if err != nil {
			continue
		}

		rowsAffected, _ := result.RowsAffected()
		if rowsAffected > 0 {
			m.queueMessageStatusWebhook(deviceID, messageID, status, receipt)
		} else {
			m.cacheReceiptStatus(deviceID, messageID, status)
		}
	}
}

func (m *Manager) ApplyCachedReceiptStatus(deviceID int, messageID string) {
	if m.db == nil || messageID == "" {
		return
	}

	status := m.takeCachedReceiptStatus(deviceID, messageID)
	if status == "" {
		return
	}

	_, _ = m.db.Exec(
		"UPDATE messages SET status = ?, updated_at = CURRENT_TIMESTAMP WHERE device_id = ? AND direction = 'outbound' AND whatsapp_message_id = ? AND status IN ("+allowedPreviousStatuses(status)+")",
		status, deviceID, messageID,
	)
}

func receiptStatus(receiptType types.ReceiptType) string {
	switch receiptType {
	case types.ReceiptTypeDelivered, types.ReceiptTypeSender:
		return "delivered"
	case types.ReceiptTypeRead, types.ReceiptTypeReadSelf, types.ReceiptTypePlayed, types.ReceiptTypePlayedSelf:
		return "read"
	default:
		return ""
	}
}

func allowedPreviousStatuses(status string) string {
	if status == "read" {
		return "'pending','sent','delivered'"
	}
	return "'pending','sent'"
}

func receiptStatusRank(status string) int {
	switch status {
	case "delivered":
		return 1
	case "read":
		return 2
	default:
		return 0
	}
}

func receiptCacheKey(deviceID int, messageID string) string {
	return fmt.Sprintf("%d:%s", deviceID, messageID)
}

func (m *Manager) cacheReceiptStatus(deviceID int, messageID string, status string) {
	m.receiptMu.Lock()
	defer m.receiptMu.Unlock()

	key := receiptCacheKey(deviceID, messageID)
	if receiptStatusRank(status) > receiptStatusRank(m.receiptStatuses[key]) {
		m.receiptStatuses[key] = status
	}
}

func (m *Manager) takeCachedReceiptStatus(deviceID int, messageID string) string {
	m.receiptMu.Lock()
	defer m.receiptMu.Unlock()

	key := receiptCacheKey(deviceID, messageID)
	status := m.receiptStatuses[key]
	delete(m.receiptStatuses, key)
	return status
}

func (m *Manager) queueMessageStatusWebhook(deviceID int, messageID string, status string, receipt *events.Receipt) {
	if m.webhook == nil {
		return
	}

	eventType := "message_" + status
	var webhookURL, webhookEvents, webhookSecret string
	err := m.db.QueryRow(
		"SELECT COALESCE(webhook_url, ''), COALESCE(webhook_events, ''), COALESCE(webhook_secret, '') FROM devices WHERE id = ?",
		deviceID,
	).Scan(&webhookURL, &webhookEvents, &webhookSecret)
	if err != nil || webhookURL == "" || !webhookEventEnabled(webhookEvents, eventType) {
		return
	}

	m.webhook.Queue(webhook.Event{
		DeviceID:  deviceID,
		EventType: eventType,
		Payload: map[string]interface{}{
			"event":      eventType,
			"device_id":  deviceID,
			"message_id": messageID,
			"status":     status,
			"timestamp":  receipt.Timestamp.Unix(),
			"recipient":  receipt.Chat.User,
		},
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		Attempt:       1,
	})
}

func (m *Manager) queueInboundWebhook(deviceID int, from string, messageID string, msgType string, content string, ts time.Time) {
	if m.webhook == nil || m.db == nil {
		return
	}

	eventType := "message_received"
	var webhookURL, webhookEvents, webhookSecret string
	err := m.db.QueryRow(
		"SELECT COALESCE(webhook_url, ''), COALESCE(webhook_events, ''), COALESCE(webhook_secret, '') FROM devices WHERE id = ?",
		deviceID,
	).Scan(&webhookURL, &webhookEvents, &webhookSecret)
	if err != nil || webhookURL == "" || !webhookEventEnabled(webhookEvents, eventType) {
		return
	}

	m.webhook.Queue(webhook.Event{
		DeviceID:  deviceID,
		EventType: eventType,
		Payload: map[string]interface{}{
			"event":      eventType,
			"device_id":  deviceID,
			"from":       from,
			"message_id": messageID,
			"type":       msgType,
			"content":    content,
			"timestamp":  ts.Unix(),
		},
		WebhookURL:    webhookURL,
		WebhookSecret: webhookSecret,
		Attempt:       1,
	})
}

func webhookEventEnabled(events string, eventType string) bool {
	for _, event := range strings.Split(events, ",") {
		if strings.TrimSpace(event) == eventType {
			return true
		}
	}
	return false
}

func parseJID(phone string) (types.JID, error) {
	if phone == "" {
		return types.JID{}, fmt.Errorf("empty phone number")
	}
	// Remove any non-digit characters except +
	return types.NewJID(phone, types.DefaultUserServer), nil
}

func extFromMime(mime string) string {
	switch mime {
	case "image/jpeg":
		return ".jpg"
	case "image/png":
		return ".png"
	case "image/gif":
		return ".gif"
	case "video/mp4":
		return ".mp4"
	case "audio/ogg", "audio/opus":
		return ".ogg"
	case "audio/mpeg":
		return ".mp3"
	case "application/pdf":
		return ".pdf"
	default:
		return ".bin"
	}
}

// AutoReconnect loads and connects all devices that have active sessions from the database.
// This should be called on server startup to resume receiving inbound messages.
func (m *Manager) AutoReconnect() {
	if m.db == nil {
		log.Println("[AutoReconnect] Database not set, skipping")
		return
	}

	rows, err := m.db.Query(
		"SELECT id, user_id FROM devices WHERE status IN ('connected', 'connecting')",
	)
	if err != nil {
		log.Printf("[AutoReconnect] Failed to query devices: %v", err)
		return
	}
	defer rows.Close()

	var devices []struct {
		ID     int
		UserID int
	}
	for rows.Next() {
		var d struct {
			ID     int
			UserID int
		}
		if err := rows.Scan(&d.ID, &d.UserID); err != nil {
			continue
		}
		devices = append(devices, d)
	}

	for _, d := range devices {
		go func(deviceID, userID int) {
			_, err := m.LoadDevice(deviceID, userID)
			if err != nil {
				log.Printf("[AutoReconnect] Failed to load device %d: %v", deviceID, err)
				return
			}

			if err := m.ConnectDevice(deviceID); err != nil {
				log.Printf("[AutoReconnect] Failed to connect device %d: %v", deviceID, err)
				return
			}

			log.Printf("[AutoReconnect] Device %d reconnected successfully", deviceID)
		}(d.ID, d.UserID)
	}

	if len(devices) > 0 {
		log.Printf("[AutoReconnect] Reconnecting %d device(s)...", len(devices))
	}
}
