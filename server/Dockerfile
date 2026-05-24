# Build stage
FROM golang:1.26-alpine AS builder

WORKDIR /app

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ARG TARGETARCH
ARG VERSION=dev
ARG GIT_COMMIT=unknown
ARG BUILD_TIME=unknown

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w -X main.version=${VERSION} -X main.gitCommit=${GIT_COMMIT} -X main.buildTime=${BUILD_TIME}" \
    -o /app/bin/stuhelper \
    ./cmd/stuhelper

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/bin/migrate-runtime \
    ./cmd/migrate-runtime

# Final stage
FROM alpine:3.21

WORKDIR /app

RUN apk add --no-cache ca-certificates tzdata nodejs npm chromium

ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN npm install --omit=dev --prefix /app playwright-core@1.58.2 \
    && npm cache clean --force

COPY --from=builder /app/bin/stuhelper /app/stuhelper
COPY --from=builder /app/bin/migrate-runtime /app/migrate-runtime
COPY --from=builder /app/migrations /app/migrations
COPY --from=builder /app/scripts/casdoor-runtime-token-probe-runner.mjs /app/casdoor-runtime-token-probe-runner.mjs

RUN adduser -D -u 1000 appuser

RUN mkdir -p /app/tmp && chown appuser:appuser /app/tmp && chmod 700 /app/tmp

RUN chmod -R 555 /app && chmod 700 /app/tmp

ENV TMPDIR=/app/tmp

USER appuser

EXPOSE 8080

HEALTHCHECK --interval=30s --timeout=3s --start-period=5s --retries=3 \
    CMD wget --no-verbose --tries=1 --spider http://localhost:8080/health/live || exit 1

ENTRYPOINT ["/app/stuhelper"]
