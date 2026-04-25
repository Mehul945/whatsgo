package models

import "time"

type User struct {
	ID         int       `json:"id" db:"id"`
	Email      string    `json:"email" db:"email"`
	Name       string    `json:"name" db:"name"`
	Role       string    `json:"role" db:"role"`
	Active     bool      `json:"active" db:"active"`
	RateLimit  int       `json:"rate_limit" db:"rate_limit"`
	CreatedAt  time.Time `json:"created_at" db:"created_at"`
	UpdatedAt  time.Time `json:"updated_at" db:"updated_at"`
}

type Device struct {
	ID            int       `json:"id" db:"id"`
	UserID        int       `json:"user_id" db:"user_id"`
	Name          string    `json:"name" db:"name"`
	PhoneNumber   string    `json:"phone_number" db:"phone_number"`
	JID           string    `json:"jid" db:"jid"`
	PushName      string    `json:"push_name" db:"push_name"`
	Status        string    `json:"status" db:"status"`
	WebhookURL    string    `json:"webhook_url" db:"webhook_url"`
	WebhookEvents string    `json:"webhook_events" db:"webhook_events"`
	WebhookSecret string    `json:"webhook_secret" db:"webhook_secret"`
	StorePath     string    `json:"store_path" db:"store_path"`
	CreatedAt     time.Time `json:"created_at" db:"created_at"`
	UpdatedAt     time.Time `json:"updated_at" db:"updated_at"`
}

type Message struct {
	ID                int       `json:"id" db:"id"`
	DeviceID          int       `json:"device_id" db:"device_id"`
	UserID            int       `json:"user_id" db:"user_id"`
	Direction         string    `json:"direction" db:"direction"`
	Recipient         string    `json:"recipient" db:"recipient"`
	MessageType       string    `json:"message_type" db:"message_type"`
	Content           string    `json:"content" db:"content"`
	MediaURL          string    `json:"media_url" db:"media_url"`
	MediaMimeType     string    `json:"media_mime_type" db:"media_mime_type"`
	MediaSHA256       string    `json:"media_sha256" db:"media_sha256"`
	Status            string    `json:"status" db:"status"`
	WhatsAppMessageID string    `json:"whatsapp_message_id" db:"whatsapp_message_id"`
	ErrorMessage      string    `json:"error_message" db:"error_message"`
	CreatedAt         time.Time `json:"created_at" db:"created_at"`
	UpdatedAt         time.Time `json:"updated_at" db:"updated_at"`
}

type APIKey struct {
	ID         int        `json:"id" db:"id"`
	UserID     int        `json:"user_id" db:"user_id"`
	DeviceID   *int       `json:"device_id" db:"device_id"`
	Name       string     `json:"name" db:"name"`
	KeyPrefix  string     `json:"key_prefix" db:"key_prefix"`
	KeyHash    string     `json:"-" db:"key_hash"`
	LastUsedAt *time.Time `json:"last_used_at" db:"last_used_at"`
	CreatedAt  time.Time  `json:"created_at" db:"created_at"`
	RevokedAt  *time.Time `json:"revoked_at" db:"revoked_at"`
}

type WebhookLog struct {
	ID             int       `json:"id" db:"id"`
	DeviceID       int       `json:"device_id" db:"device_id"`
	EventType      string    `json:"event_type" db:"event_type"`
	Payload        string    `json:"payload" db:"payload"`
	URL            string    `json:"url" db:"url"`
	StatusCode     *int      `json:"status_code" db:"status_code"`
	ResponseBody   string    `json:"response_body" db:"response_body"`
	ResponseTimeMs *int      `json:"response_time_ms" db:"response_time_ms"`
	Success        bool      `json:"success" db:"success"`
	Error          string    `json:"error" db:"error"`
	AttemptNumber  int       `json:"attempt_number" db:"attempt_number"`
	CreatedAt      time.Time `json:"created_at" db:"created_at"`
}

type DashboardStats struct {
	ActiveDevices     int `json:"active_devices"`
	MessagesToday     int `json:"messages_today"`
	MessagesThisMonth int `json:"messages_this_month"`
	WebhookDeliveries int `json:"webhook_deliveries"`
}
