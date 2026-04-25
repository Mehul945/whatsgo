# =============================================================================
# Stage 1: Build the Go binary
# =============================================================================
FROM golang:1.25-bookworm AS builder

WORKDIR /build

# Install build dependencies for CGO (required by go-sqlite3)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev sqlite3 libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Cache Go modules
COPY go.mod go.sum ./
RUN go mod download

# Copy source and build
COPY . .

# CGO_ENABLED=1 is required for mattn/go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /build/whatsgo ./cmd/server

# =============================================================================
# Stage 2: Minimal production image
# =============================================================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl sqlite3 libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -r whatsgo && useradd -r -g whatsgo -d /app -s /sbin/nologin whatsgo

WORKDIR /app

# Copy binary from builder
COPY --from=builder /build/whatsgo /app/whatsgo

# Create data directory
RUN mkdir -p /app/data && chown -R whatsgo:whatsgo /app

USER whatsgo

EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["sh", "-c", "curl -sf http://localhost:8080/health || exit 1"]

ENTRYPOINT ["/app/whatsgo"]
