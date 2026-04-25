package webhook

import (
	"bytes"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"time"

	"whatsgo/internal/db"
)

type Dispatcher struct {
	queue   chan Event
	workers int
	db      *db.DB
}

type Event struct {
	DeviceID      int
	EventType     string
	Payload       map[string]interface{}
	WebhookURL    string
	WebhookSecret string
	Attempt       int
}

func NewDispatcher(db *db.DB, workers int, queueSize int) *Dispatcher {
	return &Dispatcher{
		queue:   make(chan Event, queueSize),
		workers: workers,
		db:      db,
	}
}

func (d *Dispatcher) Start() {
	for i := 0; i < d.workers; i++ {
		go d.worker()
	}
}

func (d *Dispatcher) Queue(event Event) {
	select {
	case d.queue <- event:
	default:
		// Queue full, log and skip
	}
}

func (d *Dispatcher) worker() {
	for event := range d.queue {
		d.dispatch(event)
	}
}

func (d *Dispatcher) dispatch(event Event) {
	if event.WebhookURL == "" {
		return
	}

	payloadBytes, err := json.Marshal(event.Payload)
	if err != nil {
		return
	}

	start := time.Now()
	req, err := http.NewRequest("POST", event.WebhookURL, bytes.NewBuffer(payloadBytes))
	if err != nil {
		d.logDelivery(event, 0, "", 0, false, err.Error())
		return
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "WhatsGo/1.0")
	req.Header.Set("X-WhatsGo-Event", event.EventType)

	if event.WebhookSecret != "" {
		mac := hmac.New(sha256.New, []byte(event.WebhookSecret))
		mac.Write(payloadBytes)
		signature := hex.EncodeToString(mac.Sum(nil))
		req.Header.Set("X-WhatsGo-Signature", "sha256="+signature)
	}

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(req)
	duration := int(time.Since(start).Milliseconds())

	if err != nil {
		d.logDelivery(event, 0, "", duration, false, err.Error())
		if event.Attempt < 3 {
			time.Sleep(time.Duration(event.Attempt*5) * time.Second)
			d.Queue(Event{
				DeviceID:      event.DeviceID,
				EventType:     event.EventType,
				Payload:       event.Payload,
				WebhookURL:    event.WebhookURL,
				WebhookSecret: event.WebhookSecret,
				Attempt:       event.Attempt + 1,
			})
		}
		return
	}
	defer resp.Body.Close()

	var bodyBuf bytes.Buffer
	bodyBuf.ReadFrom(resp.Body)
	body := bodyBuf.String()

	success := resp.StatusCode >= 200 && resp.StatusCode < 300
	d.logDelivery(event, resp.StatusCode, body, duration, success, "")

	if !success && event.Attempt < 3 {
		time.Sleep(time.Duration(event.Attempt*5) * time.Second)
		d.Queue(Event{
			DeviceID:      event.DeviceID,
			EventType:     event.EventType,
			Payload:       event.Payload,
			WebhookURL:    event.WebhookURL,
			WebhookSecret: event.WebhookSecret,
			Attempt:       event.Attempt + 1,
		})
	}
}

func (d *Dispatcher) logDelivery(event Event, statusCode int, responseBody string, durationMs int, success bool, errMsg string) {
	successInt := 0
	if success {
		successInt = 1
	}
	_, _ = d.db.Exec(
		"INSERT INTO webhook_logs (device_id, event_type, payload, url, status_code, response_body, response_time_ms, success, error, attempt_number) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?)",
		event.DeviceID, event.EventType, mustJSON(event.Payload), event.WebhookURL, statusCode, responseBody, durationMs, successInt, errMsg, event.Attempt,
	)
}

func mustJSON(v interface{}) string {
	b, _ := json.Marshal(v)
	return string(b)
}
