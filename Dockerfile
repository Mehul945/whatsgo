# =============================================================================
# Stage 1: Build the frontend
# =============================================================================
FROM node:20-alpine AS frontend-builder

WORKDIR /build

COPY package.json package-lock.json ./
RUN npm ci --ignore-scripts

COPY src/ src/
COPY index.html vite.config.ts tsconfig*.json tailwind.config.js postcss.config.js components.json ./
RUN npm run build

# =============================================================================
# Stage 2: Build the Go backend
# =============================================================================
FROM golang:1.25-bookworm AS backend-builder

WORKDIR /build

# Install build dependencies for CGO (required by go-sqlite3)
RUN apt-get update && apt-get install -y --no-install-recommends \
    gcc libc6-dev sqlite3 libsqlite3-dev \
    && rm -rf /var/lib/apt/lists/*

# Cache Go modules
COPY backend/go.mod backend/go.sum ./
RUN go mod download

# Copy source and build
COPY backend/ .

# CGO_ENABLED=1 is required for mattn/go-sqlite3
RUN CGO_ENABLED=1 GOOS=linux go build -ldflags="-s -w" -o /build/whatsgo ./cmd/server

# =============================================================================
# Stage 3: Minimal production image
# =============================================================================
FROM debian:bookworm-slim

RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates curl sqlite3 libsqlite3-0 \
    && rm -rf /var/lib/apt/lists/*

# Create non-root user
RUN groupadd -r whatsgo && useradd -r -g whatsgo -d /app -s /sbin/nologin whatsgo

WORKDIR /app

# Copy Go binary
COPY --from=backend-builder /build/whatsgo /app/whatsgo

# Copy frontend build into /app/web/dist (where the Go server looks for it)
COPY --from=frontend-builder /build/dist /app/web/dist

# Create data directory
RUN mkdir -p /app/data && chown -R whatsgo:whatsgo /app

USER whatsgo

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=5s --start-period=10s --retries=3 \
    CMD ["sh", "-c", "curl -sf http://localhost:8080/health || exit 1"]

ENTRYPOINT ["/app/whatsgo"]
