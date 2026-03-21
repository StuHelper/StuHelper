# Error Handling

> How errors are handled in this project.

---

## Use the unified response envelope everywhere

Backend HTTP responses use `server/internal/pkg/response/response.go`.

The real API envelope is:

```json
{
	"success": false,
	"error": {
		"code": "A0110002",
		"message": "you have already reviewed this course"
	}
}
```

Success responses use the same wrapper:

```json
{
  "success": true,
  "data": { ... }
}
```

Representative files:

- `server/internal/pkg/response/response.go`
- `server/internal/modules/course/review/review.go`
- `server/internal/pkg/middleware/logging.go`

Do not return ad hoc JSON error shapes from handlers.

---

## Use structured error codes, not free-form strings

The error code catalog lives in `server/internal/pkg/errs/codes.go`.

Current rules that are already real:

- error codes are 8 characters
- categories are grouped by client, system, and upstream failures
- domain-specific codes exist for auth, course, review, and more
- handlers may use a general helper like `response.BadRequest(...)`, but should attach a domain code when the case is specific enough

Example:

```go
case errors.Is(err, ErrSensitiveContent):
	response.BadRequest(c, "content contains sensitive words", errs.ErrSensitiveContent)
```

Use the shared constants in `errs`, not one-off code strings.

---

## Let services return business errors, then map them in handlers

The current project pattern is explicit mapping in the handler layer.

What this means in practice:

- service methods return sentinel or wrapped business errors
- handlers use `errors.Is(...)` to classify those errors
- handlers choose the HTTP status and response helper
- unexpected failures are logged server-side and returned as generic 500 responses

Representative example from `server/internal/modules/course/review/review.go`:

```go
if err != nil {
	switch {
	case errors.Is(err, ErrAlreadyReviewed):
		response.Conflict(c, "you have already reviewed this course", errs.ErrReviewExists)
	case errors.Is(err, ErrCourseNotFound):
		response.NotFound(c, "course not found", errs.ErrCourseNotFound)
	default:
		logger.FromGin(c).Error("failed to create review", zap.Error(err))
		response.InternalError(c, "failed to create review")
	}
	return
}
```

This is more accurate than saying "the project always has one global error translator." It does not.

---

## Keep client messages safe and logs detailed

The project intentionally separates:

- what the client sees
- what the logs retain

Current behavior:

- internal failures are logged with `zap.Error(err)`
- client responses avoid exposing stack traces, SQL text, or infrastructure details
- middleware recovery logs the panic and stack, then returns a generic internal error
- `response.ErrorWithDetails(...)` exists, but the details must already be sanitized

Example from `server/internal/pkg/response/response.go`:

```go
// 调用方必须确保 Details 不包含敏感信息
```

If the error detail is only useful to developers, log it instead of returning it.

---

## Handle middleware errors as protocol-level failures

Some failures are resolved before business handlers run.

Examples already implemented in middleware:

- invalid or oversized request IDs
- CSRF failures
- panic recovery
- rate limiting
- request body size limits

Representative files:

- `server/internal/pkg/middleware/logging.go`
- `server/cmd/stuhelper/main.go`

Do not duplicate these checks in every handler when middleware already owns the boundary.

---

## API Error Responses

### Standard response helpers

Use the shared helpers in `server/internal/pkg/response/response.go`:

- `Success`
- `Created`
- `BadRequest`
- `Unauthorized`
- `Forbidden`
- `NotFound`
- `Conflict`
- `InternalError`
- `RateLimitExceeded`
- `ServiceUnavailable`

### When to use them

- `Created` for resource creation endpoints
- `BadRequest` for invalid params, bad payload shape, or invalid domain input
- `NotFound` when the referenced resource does not exist
- `Conflict` for duplicate or mutually exclusive state
- `InternalError` for unexpected server-side failure

### Example contract

#### Wrong

```go
c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
```

#### Correct

```go
response.BadRequest(c, "invalid request parameters")
```

---

## Common Mistakes

### Common Mistake: leaking internal detail to clients

**Symptom**: API responses expose raw `err.Error()` messages.

**Cause**: returning the original error string directly from the handler.

**Fix**: log the detailed error, then return a stable user-facing message with the shared response helper.

**Prevention**: treat `response.ErrorWithDetails(...)` as opt-in and only use sanitized details.

---

### Common Mistake: skipping domain-specific error codes

**Symptom**: frontend receives a generic bad request code for a known business case.

**Cause**: handler uses `response.BadRequest(...)` without passing the relevant `errs.*` code.

**Fix**: attach the domain error code when the failure is a known business rule.

**Example**:

```go
response.BadRequest(c, "content contains sensitive words", errs.ErrSensitiveContent)
```

---

### Common Mistake: claiming there is a global auto-mapper when there is not

This repo does not currently centralize all business error mapping in one place.

The current reality is explicit per-handler mapping. Document and follow that pattern instead of inventing a more abstract system.

---

### Common Mistake: returning success after partial failure in a multi-step flow

If a business operation depends on multiple writes, keep it inside the service transaction path and return one result for the whole operation.

---

## What is still evolving

A few things exist but are not fully uniform yet:

- `ValidationError(...)` exists, but many handlers still use plain `BadRequest(...)`
- not every module maps domain errors with the same level of detail
- some older handlers are less explicit than the newer review module

When in doubt, follow the newer review handlers and the shared response helpers rather than inventing a new error style.
