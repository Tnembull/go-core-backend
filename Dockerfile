# ==============================================================================
# Multi-Stage Hardened Production Dockerfile for Go Core Backend
# Minimal attack surface, Non-Root Execution, Static Binary
# ==============================================================================

# Stage 1: Build & Compile
FROM golang:alpine AS builder

WORKDIR /app

# Install security certificates & build dependencies
RUN apk add --no-cache ca-certificates git

COPY go.mod go.sum ./
RUN go mod download

COPY . .

# Compile static binary with optimizations
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build \
    -ldflags="-w -s -extldflags '-static'" \
    -o /app/server cmd/server/main.go

# Stage 2: Hardened Minimal Alpine Runtime
FROM alpine:3.20 AS runner

WORKDIR /app

# Install CA certificates and curl for container health check
RUN apk --no-cache add ca-certificates curl tzdata && \
    addgroup -g 10001 appgroup && \
    adduser -u 10001 -G appgroup -s /sbin/nologin -D appuser

# Copy static binary from builder
COPY --from=builder --chown=appuser:appgroup /app/server /app/server

# Security: Run as non-root user
USER appuser

ENV PORT=8080
ENV APP_ENV=production

EXPOSE 8080

HEALTHCHECK --interval=15s --timeout=3s --start-period=5s --retries=3 \
  CMD curl -f http://127.0.0.1:8080/healthz || exit 1

ENTRYPOINT ["/app/server"]
