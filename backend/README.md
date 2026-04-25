# WhatsGo - Meta WhatsApp API Replacement

A self-hosted, multi-tenant WhatsApp Business API replacement built in Go using the `whatsmeow` library. Provides the same core capabilities as the official Meta WhatsApp API — sending/receiving messages, media handling, webhooks, and template messages — but runs on your own infrastructure with full data sovereignty.

## Features

- **Multi-tenant**: Multiple users, each with isolated data and devices
- **Multi-device**: Manage multiple WhatsApp numbers per account
- **Meta API Compatible**: Drop-in replacement — just change the base URL
- **Real-time Webhooks**: Async delivery with HMAC signature verification and retry
- **Media Support**: Send/receive images, videos, audio, documents
- **QR Code Pairing**: Easy device linking via WhatsApp mobile app
- **Dashboard**: React-based admin panel for device and message management
- **Rate Limiting**: Per-user token bucket (configurable)
- **API Keys**: Scoped per-device or per-account

## Architecture

```
┌─────────────┐     ┌──────────────┐     ┌─────────────────┐
│   React     │────▶│   Go/Gin     │────▶│   whatsmeow     │
│  Dashboard  │     │   Backend    │     │ WhatsApp Client │
└─────────────┘     └──────────────┘     └─────────────────┘
                           │
                           ▼
                    ┌──────────────┐
                    │   SQLite     │
                    │   (default)  │
                    └──────────────┘
```

## Quick Start

### Prerequisites
- Go 1.21+
- Node.js 18+ (for frontend development)
- WhatsApp mobile app for QR pairing

### 1. Clone and Build Backend

```bash
cd backend
go mod tidy
go build -o whatsgo ./cmd/server
```

### 2. Configure Environment

```bash
cp .env.example .env
# Edit .env with your settings:
# - JWT_SECRET: strong random string
# - DATABASE_URL: sqlite://data.db (default) or postgres://... 
# - DATA_DIR: where to store media and device credentials
```

### 3. Run Backend

```bash
./whatsgo
# Server starts on http://localhost:8080
```

### 4. Build Frontend (optional — for development)

```bash
cd ..
npm install
npm run build
# Dist files are served automatically by Go server at /
```

## API Usage

### Authentication

Dashboard uses JWT tokens. Meta-compatible API uses API keys:

```bash
# Get API key from dashboard or generate via API
curl -X POST http://localhost:8080/api/apikeys \
  -H "Authorization: Bearer <jwt_token>" \
  -d '{"name": "Production"}'
```

### Send a Message (Meta API Compatible)

```bash
curl -X POST http://localhost:8080/v1/messages \
  -H "Authorization: Bearer <api_key>" \
  -H "Content-Type: application/json" \
  -d '{
    "messaging_product": "whatsapp",
    "recipient_type": "individual",
    "to": "+1234567890",
    "type": "text",
    "text": {"body": "Hello from WhatsGo!"}
  }'
```

### Receive Webhooks

Configure webhook URL on your device. WhatsGo sends events:

```json
{
  "object": "whatsapp_business_account",
  "entry": [{
    "id": "<device_id>",
    "changes": [{
      "value": {
        "messaging_product": "whatsapp",
        "metadata": {
          "display_phone_number": "<phone>",
          "phone_number_id": "<device_id>"
        },
        "messages": [{
          "from": "<phone>",
          "id": "<msg_id>",
          "timestamp": "<ts>",
          "type": "text",
          "text": {"body": "Hello!"}
        }]
      }
    }]
  }]
}
```

## Environment Variables

| Variable | Default | Description |
|----------|---------|-------------|
| `PORT` | `8080` | Server port |
| `JWT_SECRET` | — | JWT signing key |
| `DATABASE_URL` | `sqlite://data.db` | Database connection |
| `DATA_DIR` | `./data` | Media and credential storage |
| `WEBHOOK_WORKERS` | `10` | Webhook goroutine pool |
| `WEBHOOK_QUEUE_SIZE` | `1000` | Event queue buffer |
| `RATE_LIMIT_RPM` | `100` | Requests per minute per user |
| `MAX_MEDIA_SIZE` | `16` | Max upload in MB |
| `ALLOWED_ORIGINS` | — | CORS origins |
| `ECHO_BOT` | `0` | Auto-reply with received text |

## Database Schema

SQLite default. Tables:
- `users` — accounts with role-based access
- `devices` — WhatsApp connections per user
- `messages` — inbound/outbound message log
- `api_keys` — scoped API credentials
- `webhook_logs` — delivery attempt history

## Webhook Events

- `message_received` — inbound message
- `message_sent` — outbound message acknowledged
- `message_delivered` — recipient received
- `message_read` — recipient read
- `device_connected` — WhatsApp client online
- `device_disconnected` — WhatsApp client offline

Each webhook includes:
- `X-WhatsGo-Signature`: HMAC-SHA256 of payload
- `X-WhatsGo-Event`: Event type
- `User-Agent: WhatsGo/1.0`

## Docker

```dockerfile
FROM golang:1.22-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o whatsgo ./cmd/server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/
COPY --from=builder /app/whatsgo .
COPY --from=builder /app/web/dist ./web/dist
EXPOSE 8080
CMD ["./whatsgo"]
```

## Project Structure

```
backend/
  cmd/server/
    main.go              # Entry point
  internal/
    auth/                # JWT, bcrypt, middleware
    db/                  # SQLite connection & migrations
    models/              # Data structures
    whatsapp/            # whatsmeow client manager
    api/                 # HTTP handlers (REST + Meta compat)
    webhook/             # Async delivery with retry
    config/              # Environment config
frontend/
  src/
    pages/               # Dashboard, Devices, Messages, etc.
    stores/              # Zustand auth & UI state
    lib/api.ts           # API client
```

## Security

- Row-level isolation: every query filters by `user_id`
- Device isolation: users can only access their own devices
- API key scoping: per-device or account-wide
- Rate limiting: token bucket per user
- Credential encryption: whatsmeow stores encrypted at rest
- Webhook HMAC: per-device signature verification

## License

MIT
