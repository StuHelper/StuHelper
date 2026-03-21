# Type Safety

> How this backend keeps contract and runtime types aligned.

---

## Treat OpenAPI-generated DTOs as the transport contract

The project is spec-first. Request and response shapes start in `server/api/`, then generate:

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

That means transport DTO truth lives in OpenAPI, not in handwritten Gin structs copied around the codebase.

Handwritten request structs inside handlers are still allowed for binding ergonomics, but they must mirror the spec instead of inventing a second contract.

Representative files:

- `server/api/openapi.yaml`
- `server/internal/api/gen/server.gen.go`
- `clients/shared/src/types/api.gen.ts`

---

## Keep type boundaries explicit across layers

The current backend uses three distinct shapes:

- handler request/response DTOs
- service params and result types
- repository row and query types

Do not collapse them into one giant shared struct when the layers have different needs.

Examples already present in the codebase:

- review handlers bind request structs, then pass typed params like `PostReviewParams`
- services expose result structs like `GetCourseReviewsResult`
- repositories define query-specific params such as `ListByCourseWithSortParams`

This separation is what lets the code evolve without leaking SQL or HTTP details across layers.

---

## Prefer concrete types over `any` or map-shaped payloads

`any` should be rare and justified.

Current acceptable cases are narrow:

- audit payloads where the output is intentionally schema-light
- JSON blobs that are stored or relayed as opaque data
- helper hooks that must accept polymorphic old/new values for logging

Everywhere else, use concrete structs, aliases, or typed params.

Bad pattern:

```go
func (s *Service) Update(ctx context.Context, payload map[string]any) error
```

Better:

```go
type UpdateSchoolConfigParams struct {
	SchoolID string
	LdapConfig *LDAPConfigInput
	ManualFormFields []ManualFieldInput
}
```

---

## Preserve omission vs zero-value semantics deliberately

This project already has real partial-update flows. The code must distinguish:

- field omitted
- field provided as empty string / false / empty array
- field provided as a real value

That usually means using pointers or dedicated input types at the handler/service boundary.

Representative examples:

- RBAC update flows that use explicit update inputs instead of forcing every field on every request
- school-config updates that merge partial fields instead of clearing unspecified values

Do not flatten partial updates into non-pointer fields when omission matters.

---

## Match nullability to runtime truth

OpenAPI 3.1 source schemas use explicit null unions when a field may really be absent or null at runtime.

The backend should mirror that honestly:

- use pointers for optional scalar fields
- use nil slices only when the contract allows absence
- initialize empty slices for list responses when the API promises `[]` instead of `null`

Example already followed in review stats code:

- when a query returns no teacher courses, the service normalizes to `[]TeacherCourse{}`

Do not casually return `null` where frontend code expects an array.

---

## Keep enum-like values whitelisted in code

Several backend flows depend on typed string domains even when Go does not use custom enum types.

Examples:

- review status
- report status
- admin actions
- vote type
- rating dimension keys

Current project style is to defend these with:

- binding tags in handlers
- whitelist maps or switch statements in services and repositories

Do not pass raw user-controlled sort, status, or action strings into SQL or business branches without a whitelist.

---

## Avoid stringly typed IDs once parsed

HTTP params may arrive as strings, but once parsed they should move through the stack in their real type:

- numeric resource IDs become `int64`
- UUID-like IDs stay strings only where the system really stores them as string IDs

Current examples:

- `courseID`, `teacherID`, `departmentID` are parsed into `int64`
- review and reply IDs remain string IDs because the database stores them that way

Do not keep everything as raw path strings just because Gin gives you strings.

---

## Generated code is not the place to fix typing mistakes

If a field shape is wrong, fix the source spec or the handwritten runtime mapping, then regenerate.

Never patch:

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

This is especially important for:

- renamed fields
- required vs optional drift
- nullable mismatches
- enum/domain drift

---

## Common failure modes to catch in review

- A handler request struct no longer matches the OpenAPI field names or requiredness
- A partial update lost omission semantics because pointers were removed
- A list endpoint now returns `null` after a refactor
- A repository started accepting raw sort strings without a whitelist
- A service now returns `map[string]any` where a concrete result type should exist
- Someone hand-edited generated DTOs instead of fixing the source contract
