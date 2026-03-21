# Directory Structure

> How backend code is organized in this project.

---

## Keep runtime, contracts, and business code separate

This backend is a Go + Gin service with a spec-first API workflow.

Start from these boundaries:

- `server/cmd/stuhelper/main.go` wires the application together.
- `server/api/` stores the OpenAPI source of truth.
- `server/internal/api/gen/` stores generated server-side API code.
- `server/internal/modules/` stores business modules.
- `server/internal/pkg/` stores shared infrastructure and cross-module helpers.
- `server/scripts/` currently stores schema bootstrap SQL and seed data.

Do not describe this project as a generic "controller/service/model" app. The real structure is module-oriented, with manual dependency wiring in `main.go` and SQL-heavy repositories.

---

## Directory Layout

```text
server/
├── cmd/
│   └── stuhelper/
│       └── main.go
├── api/
│   ├── openapi.yaml
│   ├── components/
│   └── paths/
├── internal/
│   ├── api/
│   │   └── gen/
│   ├── modules/
│   │   ├── auth/
│   │   └── course/
│   │       └── review/
│   ├── pkg/
│   │   ├── audit/
│   │   ├── db/
│   │   ├── logger/
│   │   ├── middleware/
│   │   ├── response/
│   │   └── ...
│   └── wire/
├── scripts/
│   ├── init.sql
│   └── seed.sql
└── Makefile
```

`internal/wire/` exists, but the current runtime still assembles dependencies manually in `server/cmd/stuhelper/main.go`. Treat manual wiring as the current source of truth.

---

## Organize business code by module boundary

New backend behavior usually belongs in `internal/modules/<domain>`.

Within a module, the real pattern is:

- `Handler` receives HTTP input, applies middleware expectations, and sends unified responses.
- `Service` owns orchestration, validation, business rules, and transactions.
- `Repository` owns SQL and persistence details.

Representative examples:

- `server/internal/modules/course/handler.go` registers the top-level course routes and mounts the review submodule.
- `server/internal/modules/course/review/handler.go` shows a large domain split into review, reply, draft, favorite, notification, and admin flows.
- `server/internal/modules/course/review/repository_review_query.go` keeps query construction in the repository layer.

For larger modules, do not force everything into a single `handler.go` or `service.go`. The review module is intentionally split across multiple files by use case.

---

## Put shared infrastructure in `internal/pkg`

`internal/pkg` is for cross-module capabilities, not for business features.

Common examples:

- `server/internal/pkg/db/db.go` — pgx pool wrapper, timeouts, retries, and transactions.
- `server/internal/pkg/logger/logger.go` — Zap setup and lifecycle.
- `server/internal/pkg/middleware/logging.go` — request ID, request logging, and panic recovery.
- `server/internal/pkg/response/response.go` — the unified success/error envelope.

If a helper is specific to one business domain, keep it inside that module instead of promoting it into `internal/pkg` too early.

---

## Keep API contracts and generated code out of handwritten handlers

This project uses a spec-first workflow:

- edit OpenAPI sources under `server/api/`
- regenerate code into `server/internal/api/gen/`
- update runtime code in `internal/modules/`

Do not hand-edit generated files.

Representative examples:

- `server/api/openapi.yaml`
- `server/internal/api/gen/server.gen.go`
- `clients/shared/src/types/api.gen.ts`

---

## Use file names that describe the responsibility

Observed naming conventions:

- directories use lowercase names: `auth`, `course`, `review`
- Go files use snake_case or responsibility-based names: `repository_review_query.go`, `handler_teacher_admin.go`
- tests live next to the code they protect: `admin_create_status_test.go`

Prefer names that explain the slice of behavior, especially in larger modules.

Good examples:

- `server/internal/modules/course/review/handler.go`
- `server/internal/modules/course/review/repository_review_query.go`
- `server/internal/modules/course/review/admin_create_status_test.go`

---

## Examples to follow

Use these files as structure references:

- `server/cmd/stuhelper/main.go` — full application wiring and middleware order
- `server/internal/modules/course/handler.go` — top-level module with nested submodule registration
- `server/internal/modules/course/review/handler.go` — large domain broken into focused route groups
- `server/internal/pkg/db/db.go` — infrastructure belongs in `internal/pkg`, not a business module

---

## Avoid these structural mistakes

- Do not put SQL directly in Gin handlers.
- Do not put HTTP response shaping inside repositories.
- Do not move business-specific code into `internal/pkg` just because multiple files use it once.
- Do not assume there is a mature standalone migration framework yet; current schema bootstrap still lives in `server/scripts/*.sql`.
- Do not hand-edit generated API code.
