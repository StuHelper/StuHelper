# Logging and Audit

The logging system consists of three parts: structured application logs, request logs, and operation audit.

## Code Locations

| Location | Purpose |
| --- | --- |
| `server/internal/pkg/logger` | Zap logger, field propagation, sensitive value masking |
| `server/internal/pkg/middleware/logging.go` | Request logging middleware |
| `server/internal/pkg/audit` | Authentication and business audit events |
| `server/internal/modules/course/review/*log*` | Review admin operation log write, query, cleanup |

## Capabilities

### Application Logs

Structured logging using Zap with:

- Console or JSON output format (configurable)
- Request context field propagation (request ID, user ID)
- Sensitive value masking (PII, tokens)

### Request Logs

Middleware-level logging for every HTTP request:

- Request path and method
- Response status code
- Request duration
- Request ID for tracing

### Audit Logs

Authentication and critical business events:

- `user.login` / `user.login_failed`
- `user.logout` / `user.logout_all`
- `token.refresh`
- Admin operations (review moderation, report handling, batch operations)

### Operation Log Query

Admin users can query operation logs through the API:

| Endpoint | Purpose |
| --- | --- |
| `/api/v1/course/review/admin/logs` | Query review admin operation logs |

## Configuration

Logging configuration is loaded from `internal/pkg/config`:

| Environment Variable | Description | Default |
| --- | --- | --- |
| `LOG_LEVEL` | Minimum log level (`debug`, `info`, `warn`, `error`) | `info` |
| `LOG_FORMAT` | Output format (`console`, `json`) | `console` |
| `APP_ENV` | Application environment (`development`, `production`) | `development` |

## Usage Examples

### Structured Logging

```go
import "server/internal/pkg/logger"

// With context fields
logger.Info(ctx, "review created",
    "reviewID", review.ID,
    "courseID", review.CourseID,
    "userID", userID,
)

// Error with context
logger.Error(ctx, "failed to create review",
    "error", err,
    "courseID", courseID,
)
```

### Audit Event

```go
import "server/internal/pkg/audit"

audit.Log(ctx, audit.Event{
    Action:  "user.login",
    UserID:  userID,
    Details: map[string]any{
        "method": "casdoor_sso",
        "ip":     clientIP,
    },
})
```

## Storage

| Log Type | Storage | Retention |
| --- | --- | --- |
| Application logs | stdout/stderr | Managed by container runtime |
| Request logs | stdout/stderr | Managed by container runtime |
| Operation audit | `admin_operation_logs` table | Queryable via admin API |

## Related Documentation

- [Backend Quality Guidelines](../../.trellis/spec/backend/quality-guidelines.md)
- [Backend Logging Guidelines](../../.trellis/spec/backend/logging-guidelines.md)
