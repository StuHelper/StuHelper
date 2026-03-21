# Logging Guidelines

> How logging is done in this project.

---

## Use Zap and keep logs structured

The backend uses Zap-based structured logging.

The core setup lives in:

- `server/internal/pkg/logger/logger.go`
- `server/internal/pkg/middleware/logging.go`
- `server/internal/pkg/audit/audit.go`

Current runtime defaults:

- format defaults to JSON
- output defaults to stdout
- file output is optional
- file rotation uses `lumberjack`
- stack traces are attached at error level and above

If you need to add logs, prefer structured fields over formatted strings.

---

## Pick log levels by outcome severity

The codebase already uses these rules:

- `Info` for normal request completion and successful audit events
- `Warn` for recoverable operational problems, fallback behavior, retry signals, or cleanup failures
- `Error` for server errors, panic recovery, and request failures with 5xx semantics

Examples:

- `server/internal/pkg/middleware/logging.go` logs 2xx and 3xx requests as `Info`, 4xx as `Warn`, and 5xx as `Error`
- `server/internal/pkg/db/db.go` logs transient query retry events as `Warn`
- `server/internal/modules/course/handler.go` logs background cleanup failures as `Error`

Do not log expected success paths as warnings just because they are interesting.

---

## Always include useful fields, especially request context

Request logging is already centralized.

Current request log fields include:

- `request_id`
- `method`
- `path`
- `query`
- `status`
- `latency`
- `size`
- `client_ip`
- `user_agent`
- `user_id` when present

Example from `server/internal/pkg/middleware/logging.go`:

```go
fields := []zap.Field{
	zap.String("request_id", requestID),
	zap.String("method", c.Request.Method),
	zap.String("path", path),
	zap.String("query", query),
	zap.Int("status", status),
	zap.Duration("latency", latency),
}
```

When logging inside a request path, prefer `logger.FromGin(c)` or a logger that already carries the request ID.

---

## Treat audit logs as a separate contract

Audit logs are not the same as generic operational logs.

The audit event model is defined in `server/internal/pkg/audit/audit.go` and includes fields such as:

- `audit_type`
- `user_id`
- `username`
- `ip`
- `user_agent`
- `request_id`
- `resource`
- `action`
- `result`
- `duration`
- `details`

Representative example:

```go
logger.L().Info("audit_log", fields...)
```

If an event is security-sensitive or administrator-visible, consider whether it belongs in the audit channel instead of a normal info log.

---

## Never log secrets or raw sensitive values

This project already has masking and redaction behavior. Follow it.

Current protections include:

- sensitive query params are replaced with `[REDACTED]`
- usernames are masked in audit logs
- IP addresses can be masked through helper functions
- user agents are truncated
- recovery returns generic messages to clients while keeping details in logs

Sensitive query param examples from `server/internal/pkg/middleware/logging.go`:

- `code`
- `token`
- `access_token`
- `refresh_token`
- `password`
- `state`
- `nonce`

Do not add logs that print raw tokens, passwords, OAuth codes, or full secret-bearing query strings.

---

## Log retries, fallbacks, and cleanup failures explicitly

This codebase prefers explicit operational logs for non-fatal failure paths.

Examples to follow:

- DB retry warning in `server/internal/pkg/db/db.go`
- rollback warning in `server/internal/pkg/db/db.go`
- cache invalidation warning in `server/internal/modules/course/review/handler.go`
- background cleanup warning/error in `server/internal/modules/course/handler.go`

These logs help explain why behavior degraded without turning a recoverable path into a hard failure.

---

## What to Log

Log these categories when they matter:

- request lifecycle and status codes
- panic recovery details
- retry behavior and rollback failures
- cache invalidation failures that do not block the main request
- background job start/stop/failure outcomes
- audit-worthy security, auth, admin, and moderation actions

Good examples:

- `server/internal/pkg/middleware/logging.go`
- `server/internal/pkg/db/db.go`
- `server/internal/pkg/audit/audit.go`
- `server/internal/modules/course/review/handler.go`

---

## What NOT to Log

Do not log:

- passwords
- access tokens
- refresh tokens
- OAuth authorization codes
- raw CSRF tokens
- raw secret-bearing query strings
- internal stack traces in client responses

Also avoid logging entire request bodies by default. Most handlers do not do that today, and adding it casually would increase privacy and security risk.

---

## Wrong vs Correct

### Wrong

```go
logger.L().Info("oauth callback", zap.String("code", code), zap.String("state", state))
```

### Correct

```go
logger.L().Info("oauth callback received", zap.String("request_id", requestID))
```

If the values are sensitive, keep them out of logs.

---

## What is still evolving

A few areas are standardized less strongly than request logging:

- business-level log field names are not yet as uniform as request and audit logs
- some modules use richer structured fields than others
- the log framework is mature, but the per-feature field schema is still evolving

When adding new logs, prefer the richer structured style already used in middleware, audit, and DB infrastructure.
