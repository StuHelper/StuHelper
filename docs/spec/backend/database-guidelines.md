# Database Guidelines

> Database patterns and conventions for this project.

---

## Use PostgreSQL with handwritten SQL

This backend does not use an ORM.

The real stack is:

- PostgreSQL
- `pgx` / `pgxpool`
- a project wrapper in `server/internal/pkg/db/db.go`
- handwritten SQL in repository files

Representative examples:

- `server/internal/pkg/db/db.go`
- `server/internal/modules/course/review/repository_review_query.go`
- `server/scripts/init.sql`

That means database behavior should be documented in terms of SQL, transactions, indexes, and repository conventions, not ORM models.

---

## Write queries in repositories, not handlers

Keep SQL in the repository layer.

Real conventions in this repo:

- handlers parse HTTP input and return API responses
- services own orchestration, validation, and transaction boundaries
- repositories own SQL text, joins, filters, and scan logic
- all query inputs must be parameterized with `$1`, `$2`, and so on
- dynamic `ORDER BY` is only allowed when the final SQL fragment comes from a hardcoded whitelist

Example from `server/internal/modules/course/review/repository_review_query.go`:

```go
var allowedSortOrders = map[string]string{
	SortTime:   "r.created_at DESC",
	SortLikes:  "r.like_count DESC, r.created_at DESC",
	SortRating: "r.avg_rating DESC, r.created_at DESC",
}
```

```go
orderBy, ok := allowedSortOrders[p.Sort]
if !ok {
	orderBy = allowedSortOrders["time"]
}
qb.WriteString(` ORDER BY ` + orderBy)
```

Use this pattern when sort behavior must be configurable.

---

## Use the DB wrapper for timeouts, retries, and transactions

`server/internal/pkg/db/db.go` is part of the contract.

What it already does:

- applies a query timeout
- retries one time on transient connection errors
- exposes `Query`, `QueryRow`, `Exec`, and `WithTx`
- records Prometheus metrics for query timing and status
- uses a longer timeout for transactions than for single queries

Example:

```go
func (d *DB) WithTx(ctx context.Context, fn func(ctx context.Context, tx pgx.Tx) error) error {
	txTimeout := d.timeout * txTimeoutMultiplier
	ctx, cancel := context.WithTimeout(ctx, txTimeout)
	defer cancel()
	...
}
```

Use `WithTx(...)` when a write flow needs multiple dependent statements or lock-sensitive checks.

---

## Match query shape to actual product needs

Common query patterns already in use:

- pagination with `COUNT(*) OVER()` so the API can return list data and total count in one query
- `LATERAL` joins for top-N-per-parent queries
- selective dynamic filters with `strings.Builder`
- secondary repository-level validation for sort values, even if the handler already validated input
- explicit index design for common `WHERE + ORDER BY` combinations

Example from `server/internal/modules/course/review/repository_review_query.go`:

```sql
SELECT r.id, ..., COUNT(*) OVER() AS total
FROM reviews r
LEFT JOIN courses c ON c.id = r.course_id
LEFT JOIN teachers t ON t.id = r.teacher_id
WHERE r.course_id = $1 AND r.status IN ('published', 'hidden')
ORDER BY r.created_at DESC
LIMIT $2 OFFSET $3
```

Example from `server/scripts/init.sql`:

```sql
CREATE INDEX IF NOT EXISTS idx_reviews_course_status_created
ON reviews(course_id, status, created_at DESC);
```

Design indexes around the real filter and sort path, not around abstract entity fields.

---

## Change schema through SQL scripts for now

The current repo does not have a mature standalone migration framework.

Today, schema bootstrap lives in:

- `server/scripts/init.sql`
- `server/scripts/seed.sql`

Treat this as the current reality.

When you change schema:

1. Update `server/scripts/init.sql`.
2. Update seed data if the feature depends on bootstrap records.
3. Add or update constraints and indexes in the same change.
4. If the schema change affects API payloads, also update OpenAPI and regenerate code.

Cross-layer reminder:

- backend contract source: `server/api/openapi.yaml`
- generated Go code: `server/internal/api/gen/`
- generated frontend types: `clients/shared/src/types/api.gen.ts`

Do not document a fake migration workflow that the repo does not actually use yet.

---

## Use the project naming scheme consistently

Observed naming conventions:

### Tables and columns

- snake_case table names: `review_votes`, `course_rating_stats`
- snake_case columns: `created_at`, `term_id`, `user_hash`
- timestamp columns usually use `TIMESTAMPTZ`

### Constraints and indexes

- foreign keys use `fk_*`
- unique constraints use `uq_*`
- check constraints use `chk_*`
- indexes use `idx_*`

Examples from `server/scripts/init.sql`:

- `fk_reviews_course`
- `uq_review_votes_user`
- `chk_review_reports_status`
- `idx_review_reports_status_created`

### ID strategy

This project uses a mixed ID strategy:

- user-visible entities use `BIGSERIAL`: `courses`, `teachers`, `departments`
- internal business records use `VARCHAR(36)` IDs: `reviews`, `review_votes`, `review_reports`, `notifications`
- new internal IDs are generated in Go, not by a database UUID function

Keep following that split unless there is a deliberate architectural change.

---

## Examples to follow

Use these files as references:

- `server/internal/pkg/db/db.go` — timeout, retry, transaction wrapper
- `server/internal/modules/course/review/repository_review_query.go` — repository-owned query construction
- `server/scripts/init.sql` — real schema, indexes, constraints, and naming

---

## Common Mistakes

### Do not introduce ORM-style access patterns

This codebase is already built around handwritten SQL and repository methods.

Wrong direction:

- hiding SQL behind a new ORM layer for one module
- mixing ORM objects with existing pgx repositories

Correct direction:

- extend the existing repository with a focused SQL method

### Do not trust raw sort or filter strings

Wrong:

```go
qb.WriteString(" ORDER BY " + userInput)
```

Correct:

```go
orderBy, ok := allowedSortOrders[p.Sort]
if !ok {
	orderBy = allowedSortOrders[SortTime]
}
qb.WriteString(` ORDER BY ` + orderBy)
```

### Do not assume migration tooling exists when it does not

If you change schema, update the SQL bootstrap files that the repo actually uses today.

### Do not move transactional checks outside the transaction

If correctness depends on read-then-write ordering, keep the whole sequence inside `WithTx(...)`.

### Do not generate new internal UUIDs in ad hoc SQL

New internal record IDs should follow the project convention of Go-side UUID generation.

---

## Call out what is still evolving

These areas are real, but not fully standardized yet:

- schema evolution is still script-based, not migration-tool-based
- some historical SQL comments and older patterns remain in bootstrap scripts
- not every module has the same depth of repository test coverage

Document the current state honestly instead of inventing a cleaner system than the repo actually has.
