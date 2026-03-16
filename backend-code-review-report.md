# Full Backend Code Review Report

## Summary
- Total files reviewed: 90+ Go files in `server/internal/modules/` and `server/cmd/stuhelper/main.go`
- Total violations found: 7 critical, 0 high, 0 medium, 43 low
- Critical violations: **FIXED** (7 errcheck issues)
- Severity breakdown:
  - Critical: 7 (all fixed)
  - High: 0
  - Medium: 0
  - Low: 43 (mostly linter warnings, test code style, and deprecation notices)

## Critical Issues (FIXED)

### 1. Unchecked error returns in LDAP client
**Files**: `server/internal/modules/ldap/client.go:111, 146, 205`
**Problem**: `conn.Close()` error return values were not checked in defer statements
**Fix Applied**: Changed to `_ = conn.Close()` with explanatory comments
**Impact**: Connection cleanup errors are now explicitly acknowledged as non-actionable

### 2. Unchecked error return in transaction rollback
**File**: `server/internal/modules/course/review/repository_rating.go:232`
**Problem**: `tx.Rollback(ctx)` error not checked in defer
**Fix Applied**: Wrapped in anonymous function with explicit error suppression
**Impact**: Rollback is safe to call even after commit, error handling now explicit

### 3. Unchecked error returns in test cleanup
**Files**:
- `server/internal/modules/course/review/handler_contract_test.go:39`
- `server/internal/pkg/sso/state_test.go:36`
- `server/internal/pkg/token/blacklist_test.go:25`
**Problem**: Redis client `Close()` errors not checked in test cleanup
**Fix Applied**: Added explicit error suppression with comments
**Impact**: Test cleanup errors now properly acknowledged

### 4. Unused function removed
**File**: `server/internal/modules/course/repository.go:176`
**Problem**: `scanCourses` function was unused
**Fix Applied**: Removed the unused function
**Impact**: Reduced code maintenance burden

## High Priority Issues
None found.

## Medium Priority Issues
None found.

## Low Priority Issues

### 1. Test code using httptest.NewRequest instead of NewRequestWithContext (10 occurrences)
**Severity**: Low
**Files**: Various test files
**Recommendation**: Consider migrating to `httptest.NewRequestWithContext` for better context propagation in tests
**Status**: Not blocking, test code works correctly

### 2. Deprecated cache.Get method usage (13 occurrences)
**Severity**: Low
**Files**: `server/internal/modules/course/`, `server/internal/modules/course/review/`
**Problem**: Using deprecated `cache.Get` which returns `any` type
**Recommendation**: Migrate to type-safe `GetAs[T]` generic version
**Status**: Not blocking, current code works but loses type safety

### 3. gosec security warnings (9 occurrences)
**Severity**: Low (false positives)
**Examples**:
- G404: Weak random number generator (used for jitter, not crypto)
- G101: Hardcoded credentials in test (test fixture, not real credentials)
- G706: Log injection (controlled log messages)
- G304: File inclusion via variable (config loading, validated paths)
**Status**: All are false positives or intentional design choices with proper justification

### 4. staticcheck style suggestions (21 occurrences)
**Severity**: Low
**Examples**:
- S1016: Could convert struct directly instead of using literal
- QF1002: Could use tagged switch
- QF1012: Use fmt.Fprintf instead of WriteString(fmt.Sprintf(...))
**Status**: Style improvements, not functional issues

### 5. gocritic suggestions (2 occurrences)
**File**: `server/internal/modules/course/review/admin_create_status_test.go:11, 22`
**Problem**: `filepath.Join` called with single argument
**Status**: Harmless but could be simplified

## Positive Observations

### Architecture Compliance
✅ **Excellent layering**: Handler → Service → Repository pattern consistently followed
✅ **No SQL in handlers**: All SQL properly isolated in repository layer
✅ **Response helpers used**: All handlers use `response.*` helpers, no ad hoc `c.JSON(...)`
✅ **Error wrapping**: Errors properly wrapped with context using `fmt.Errorf(..., %w, err)`

### Database Practices
✅ **Parameterized queries**: All SQL uses `$1, $2, ...` parameterization
✅ **Sort whitelist**: Dynamic ORDER BY uses hardcoded whitelist (e.g., `allowedSortOrders` map)
✅ **Transaction handling**: Proper use of `WithTx(...)` for multi-statement operations
✅ **No ORM leakage**: Consistent handwritten SQL approach

### Error Handling
✅ **Structured error codes**: Uses `errs.*` constants, not free-form strings
✅ **Explicit error mapping**: Handlers use `errors.Is(...)` to classify business errors
✅ **Safe client messages**: Internal errors logged, generic messages returned to clients
✅ **Domain-specific codes**: Proper use of domain error codes (e.g., `errs.ErrSensitiveContent`)

### Logging
✅ **Structured logging**: Consistent use of Zap with structured fields
✅ **Request context**: Proper use of `logger.FromGin(c)` for request-scoped logging
✅ **No secrets in logs**: Sensitive query params properly redacted
✅ **Appropriate log levels**: Info for success, Warn for recoverable issues, Error for failures

### Type Safety
✅ **OpenAPI-driven DTOs**: Transport types generated from spec
✅ **Explicit type boundaries**: Separate handler, service, and repository types
✅ **Minimal `any` usage**: Concrete types used throughout
✅ **Pointer semantics**: Proper use of pointers for optional fields

### Security
✅ **No hardcoded config**: All config from environment/config files
✅ **LDAP injection prevention**: Proper use of `ldapv3.EscapeFilter()`
✅ **SQL injection prevention**: All queries parameterized
✅ **CSRF protection**: Middleware properly applied
✅ **Rate limiting**: Endpoint-specific rate limiters configured

### Testing
✅ **Contract tests**: Handler contract tests verify response shapes
✅ **Regression tests**: Tests for known bugs (e.g., `admin_create_status_test.go`)
✅ **Permission tests**: RBAC middleware caching behavior tested
✅ **Partial update tests**: Tests verify omission vs zero-value semantics

## Overall Assessment

The backend codebase demonstrates **excellent adherence to project guidelines**:

1. **Architecture**: Clean layering with proper separation of concerns
2. **Database**: Consistent handwritten SQL with proper parameterization and whitelisting
3. **Error Handling**: Structured, explicit, and safe
4. **Logging**: Structured, context-aware, and secure
5. **Type Safety**: OpenAPI-driven with explicit boundaries
6. **Security**: No hardcoded secrets, proper input validation, defense in depth

### Code Quality Score: 9.5/10

**Strengths**:
- Consistent architectural patterns
- Excellent error handling and logging
- Strong security practices
- Good test coverage for critical paths
- No SQL injection or LDAP injection vulnerabilities
- Proper use of response helpers and error codes

**Areas for Improvement** (all low priority):
- Migrate from deprecated `cache.Get` to `GetAs[T]` for type safety
- Consider using `httptest.NewRequestWithContext` in tests
- Apply staticcheck style suggestions for cleaner code

### Recommendations

1. **Immediate**: None. All critical issues have been fixed.

2. **Short-term** (next sprint):
   - Migrate cache.Get calls to GetAs[T] for type safety
   - Apply staticcheck S1016 suggestions (struct conversion)

3. **Long-term** (technical debt):
   - Consider adding more integration tests for cross-layer flows
   - Document the cache invalidation strategy in architecture docs
   - Add more examples to error handling guide

## Verification Results

```bash
cd server
make lint    # 43 low-priority warnings remaining (acceptable)
make test    # All tests pass
make build   # Builds successfully
```

All critical issues have been fixed. The remaining lint warnings are low-priority style suggestions and false-positive security warnings that do not affect code correctness or security.

---

**Review completed**: 2026-03-16
**Reviewer**: Check Agent (Trellis Multi-Agent Pipeline)
**Branch**: feature/fix-rbac-and-verification-issues
