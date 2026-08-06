ARG GO_IMAGE=golang:1.26.5-alpine3.24@sha256:0178a641fbb4858c5f1b48e34bdaabe0350a330a1b1149aabd498d0699ff5fb2
ARG RUNTIME_IMAGE=alpine:3.24@sha256:28bd5fe8b56d1bd048e5babf5b10710ebe0bae67db86916198a6eec434943f8b
ARG ALPINE_APK_REPOSITORY=
ARG GOPROXY=

# Build stage
FROM ${GO_IMAGE} AS builder

ARG ALPINE_APK_REPOSITORY
ARG GOPROXY

WORKDIR /app

RUN if [ -n "${ALPINE_APK_REPOSITORY}" ]; then \
        sed -i "s|https://dl-cdn.alpinelinux.org/alpine|${ALPINE_APK_REPOSITORY}|g" /etc/apk/repositories; \
    fi

RUN apk add --no-cache git ca-certificates tzdata

COPY go.mod go.sum ./
RUN if [ -n "${GOPROXY}" ]; then go env -w GOPROXY="${GOPROXY}"; fi
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

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/bin/backfill-user-hashes \
    ./cmd/backfill-user-hashes

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/bin/external-student-source-smoke \
    ./cmd/external-student-source-smoke

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/bin/openfga-resource-smoke \
    ./cmd/openfga-resource-smoke

RUN CGO_ENABLED=0 GOOS=linux GOARCH=${TARGETARCH} go build \
    -ldflags="-s -w" \
    -o /app/bin/campus-connector-registry \
    ./cmd/campus-connector-registry

# Final stage
FROM ${RUNTIME_IMAGE}

ARG ALPINE_APK_REPOSITORY

WORKDIR /app

RUN if [ -n "${ALPINE_APK_REPOSITORY}" ]; then \
        sed -i "s|https://dl-cdn.alpinelinux.org/alpine|${ALPINE_APK_REPOSITORY}|g" /etc/apk/repositories; \
    fi

RUN apk add --no-cache ca-certificates tzdata nodejs npm chromium

ENV PLAYWRIGHT_SKIP_BROWSER_DOWNLOAD=1
RUN npm install --omit=dev --prefix /app playwright-core@1.58.2 \
    && npm cache clean --force

COPY --from=builder /app/bin/stuhelper /app/stuhelper
COPY --from=builder /app/bin/migrate-runtime /app/migrate-runtime
COPY --from=builder /app/bin/backfill-user-hashes /app/backfill-user-hashes
COPY --from=builder /app/bin/external-student-source-smoke /app/external-student-source-smoke
COPY --from=builder /app/bin/openfga-resource-smoke /app/openfga-resource-smoke
COPY --from=builder /app/bin/campus-connector-registry /app/campus-connector-registry
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
