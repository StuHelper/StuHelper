# Quality Guidelines

> Code quality standards for backend development.

---

## Follow the existing toolchain, not an imagined one

The backend quality bar is already encoded in the repo.

Primary entry points:

- `server/Makefile`
- `server/.golangci.yml`
- OpenAPI generation and drift checks under `server/Makefile`

Before calling work complete, backend changes should line up with the current workflow:

```bash
cd server
make fmt
make lint
make test
make build
```

For API contract changes, also run:

```bash
cd server
make lint-spec
make generate
make check-drift
```

If the work touches OpenAPI authoring or generation behavior, also follow:

- `server/api/openapi.yaml` stays on OpenAPI 3.1
- `server/api/oapi-codegen-overlay.yaml` is the only place where 3.0 compatibility shims belong
- `./openapi-tooling.md`

---

## Forbidden Patterns

### Do not hand-edit generated API files

Generated files are outputs, not authoring surfaces.

Examples:

- `server/internal/api/gen/server.gen.go`
- `server/api/openapi.bundled.yaml`
- `clients/shared/src/types/api.gen.ts`

If behavior changes, update the OpenAPI source files and regenerate.

### Do not bypass the response helpers

Avoid ad hoc JSON responses from handlers.

Wrong:

```go
c.JSON(http.StatusOK, gin.H{"ok": true})
```

Correct:

```go
response.Success(c, data)
```

### Do not put SQL in handlers

SQL belongs in repositories. Handlers should stay focused on HTTP concerns.

### Do not trust raw user-controlled sort or filter fragments

Repository code must protect dynamic SQL with a whitelist.

### Do not skip errors

The project enables `errcheck`, and the codebase expects explicit error handling.

### Do not hardcode environment-specific config

Use config/env wiring instead of embedding ports, URLs, secrets, or deployment values in module code.

### Do not grow giant functions when the flow should be split

If a function grows past roughly 200 lines or mixes multiple responsibilities, split it into smaller helpers or layers.

---

## Required Patterns

### Use formatters and linters already configured in the repo

The current backend lint/format stack includes:

- `gofmt`
- `goimports`
- `golangci-lint`
- `govet`
- `staticcheck`
- `gosec`
- `errcheck`

Representative config:

- `server/.golangci.yml`

The old baseline also assumed:

- `gofmt`
- `goimports`
- `go vet`

### Use race-enabled tests

The default test target is:

```bash
go test -race ./...
```

Do not document plain `go test ./...` as the main quality path when the repo already runs the race detector by default.

### Keep API contracts in sync

If an API shape changes:

1. update `server/api/`
2. regenerate code
3. verify drift is clean

`make check-drift` already enforces this.

### Keep comments and tests close to non-obvious regressions

This repo already contains guardrail tests that lock down API behavior.

Example:

- `server/internal/modules/course/review/admin_create_status_test.go`

That test protects the requirement that creation handlers use `response.Created(...)`.

---

## Testing Requirements

### Minimum expectation

For backend behavior changes, the current quality bar is:

- unit or module-level regression coverage for non-trivial logic
- race-safe test execution
- build passes
- contract generation stays in sync

### Good places to add tests

- service rules
- repository query behavior where regressions are subtle
- handler response semantics when status codes or envelopes matter
- contract guardrails for easy-to-regress conventions

### Prefer compile-time and test-time guardrails over tribal knowledge

If an interface boundary, error mapping, or response contract is easy to regress, add an explicit regression test or compile-time conformance check.

Representative examples:

- `server/internal/modules/course/review/admin_create_status_test.go`
- review module tests under `server/internal/modules/course/review/`

### Be honest about coverage depth

The toolchain is fairly mature, but test depth is not uniform across every module. The review module has the strongest regression coverage today. Follow the stronger patterns instead of assuming every area already has the same guardrails.

---

## Code Review Checklist

Review backend changes for these points:

- Is SQL kept in repositories?
- Are handlers using `response.*` helpers?
- Are domain errors mapped intentionally?
- Does the change preserve request, auth, or cache safety boundaries?
- If SQL changed, are indexes, constraints, and script-based schema updates consistent?
- If API contracts changed, was OpenAPI updated and generation re-run?
- Are logs structured and free of secrets?
- Were tests added or updated for the regression surface?
- Does the change still pass `make lint`, `make test`, and `make build`?

---

## Common backend gotchas worth preserving

### Contract drift is a real failure mode

Because the repo is spec-first, changing runtime behavior without updating OpenAPI or generated types creates real breakage.

Use `make check-drift` before declaring API work complete.

### Script-based schema is still the reality

Do not claim a schema change is done if only a repository query changed. The bootstrap SQL in `server/scripts/init.sql` must stay aligned with the runtime expectations.

### New handlers should use the newer conventions

The review module contains many of the strongest modern patterns in this codebase:

- explicit error mapping
- shared response helpers
- rate limiting middleware
- cache invalidation warnings instead of silent failure
- focused regression tests

Prefer those patterns when extending older areas.
