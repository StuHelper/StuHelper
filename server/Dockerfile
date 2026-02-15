# Build stage
FROM golang:1.24-alpine AS builder

WORKDIR /app

# Install build dependencies
RUN apk add --no-cache git ca-certificates tzdata

# Copy go mod files first for better caching
COPY go.mod go.sum ./
RUN go mod download

# Copy source code
COPY . .

# Build binary (TARGETARCH is auto-set by Docker BuildKit: amd64, arm64, etc.)
ARG TARGETARCH
RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/bin/stuhelper \
    ./cmd/stuhelper

# Final stage
FROM alpine:3.19

WORKDIR /app

# Install runtime dependencies
RUN apk add --no-cache ca-certificates tzdata

# Copy binary from builder
COPY --from=builder /app/bin/stuhelper /app/stuhelper

# Create non-root user with explicit UID
RUN adduser -D -u 1000 appuser

# M-84: 创建专用临时目录并限制权限，避免使用全局 /tmp
RUN mkdir -p /app/tmp && chown appuser:appuser /app/tmp && chmod 700 /app/tmp

# Make app directory read-only (except tmp)
RUN chmod -R 555 /app && chmod 700 /app/tmp

# Set dedicated temp directory
ENV TMPDIR=/app/tmp

USER appuser

# Expose port
EXPOSE 8080

# Health check
HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health || exit 1

# Run
ENTRYPOINT ["/app/stuhelper"]
