# Member Blacklist Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the backend source of truth for scoped member blacklists.

**Architecture:** PostgreSQL stores `member_blacklist_entries`. Admission service owns source validation, active access decisions, expiration release, admission failure count coupling, and audit events. Bot/Admin routes expose only allowed entry points.

**Tech Stack:** Go 1.26, Gin, pgx/PostgreSQL, OpenAPI 3.1, oapi-codegen.

---

## File Map

- Modify: `server/api/openapi.yaml`
- Modify: `server/api/components/schemas/admission.yaml`
- Modify: `server/api/paths/bot-admission.yaml`
- Modify: `server/api/paths/admin-admission.yaml`
- Modify: `server/migrations/000001_initial_schema.up.sql`
- Create: `server/internal/modules/admission/models_blacklist.go`
- Create: `server/internal/modules/admission/repository_blacklist.go`
- Create: `server/internal/modules/admission/service_blacklist.go`
- Create: `server/internal/modules/admission/handler_blacklist.go`
- Modify: `server/internal/modules/admission/handler.go`
- Modify: `server/internal/modules/admission/service_session.go`
- Modify: `server/internal/modules/admission/service_expiry.go`
- Modify: `server/internal/pkg/capability/catalog.go`

## Task 1: Contract And Schema

- [ ] **Step 1: Write failing route tests**

In `server/internal/modules/admission/route_contract_test.go`, replace old blacklist route assertions with:

```go
routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/member-blacklist/access")
routeassert.Exists(t, routes, http.MethodGet, "/api/v1/bot/member-blacklist")
routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/member-blacklist")
routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/member-blacklist/:id/release")
routeassert.Exists(t, routes, http.MethodPost, "/api/v1/bot/member-blacklist/release-by-subject")
routeassert.Exists(t, routes, http.MethodGet, "/api/v1/admin/member-blacklist")
routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/member-blacklist")
routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/member-blacklist/:id/release")
routeassert.Exists(t, routes, http.MethodPost, "/api/v1/admin/member-blacklist/release-by-subject")
```

Run: `cd server && go test -count=1 -timeout=60s ./internal/modules/admission -run TestAdmissionRoutes`
Expected: FAIL because routes are absent.

- [ ] **Step 2: Add OpenAPI contract**

Add schemas in `server/api/components/schemas/admission.yaml`: `MemberBlacklistEntry`, `MemberBlacklistCreateRequest`, `MemberBlacklistReleaseRequest`, `MemberBlacklistReleaseBySubjectRequest`, `MemberBlacklistAccessDecision`, `MemberBlacklistSource`, `MemberBlacklistScopeType`, `MemberBlacklistReleaseReasonCode`.

Add paths in `bot-admission.yaml` and `admin-admission.yaml` for list, create, access, release by ID, and release by subject. Remove these refs from `openapi.yaml`: `/api/v1/bot/admission/qq-users/{qqID}/access`, `/api/v1/bot/admission/blacklist/{qqID}/release`, `/api/v1/admin/admission/blacklist/{qqID}/release`.

- [ ] **Step 3: Add schema baseline**

Add `member_blacklist_entries` to `server/migrations/000001_initial_schema.up.sql` with CHECK constraints:

```sql
CONSTRAINT chk_member_blacklist_scope CHECK (
  (scope_type = 'guild' AND guild_id IS NOT NULL) OR
  (scope_type = 'global' AND guild_id IS NULL)
)
```

Add indexes:

```sql
CREATE UNIQUE INDEX member_blacklist_global_active_key
  ON public.member_blacklist_entries (platform, subject_type, subject_id)
  WHERE released_at IS NULL AND scope_type = 'global';
CREATE UNIQUE INDEX member_blacklist_guild_active_key
  ON public.member_blacklist_entries (platform, subject_type, subject_id, guild_id)
  WHERE released_at IS NULL AND scope_type = 'guild';
CREATE INDEX member_blacklist_access_idx
  ON public.member_blacklist_entries (platform, subject_type, subject_id, scope_type, guild_id)
  WHERE released_at IS NULL;
```

- [ ] **Step 4: Generate and verify**

Run: `cd server && make generate && make check-drift`
Expected: PASS.

Commit:

```bash
git add server/api server/internal/api/gen clients/shared/src/types/api.gen.ts server/migrations/000001_initial_schema.up.sql
git commit -m "feat: add member blacklist contract"
```

## Task 2: Repository And Service

- [ ] **Step 1: Write failing service tests**

Create `server/internal/modules/admission/service_blacklist_test.go` with these tests:

```go
func TestMemberBlacklistAccessPrefersGlobal(t *testing.T)
func TestMemberBlacklistGuildScopeDoesNotBlockOtherGuild(t *testing.T)
func TestMemberBlacklistRejectsSourceForWrongEntryPoint(t *testing.T)
func TestMemberBlacklistReleaseBySubjectRequiresScope(t *testing.T)
func TestMemberBlacklistExpiredRowsDoNotBlockAccess(t *testing.T)
func TestMemberBlacklistCreateReleasesExpiredRowBeforeInsert(t *testing.T)
func TestMemberBlacklistGlobalUniqueDoesNotDuplicateOnNullGuild(t *testing.T)
```

Run: `cd server && go test -count=1 -timeout=60s ./internal/modules/admission -run TestMemberBlacklist`
Expected: FAIL because service does not exist.

- [ ] **Step 2: Add focused models**

Create `models_blacklist.go` with scope/source/release constants, `MemberBlacklistEntry`, `MemberBlacklistCreateInput`, `MemberBlacklistAccessQuery`, `MemberBlacklistListFilter`, `MemberBlacklistReleaseInput`, `MemberBlacklistReleaseBySubjectInput`, and an internal `blacklistEntryPoint` enum for admin, bot, internal.

- [ ] **Step 3: Add repository**

Create `repository_blacklist.go` with:

```go
CreateMemberBlacklistTx(ctx context.Context, tx pgx.Tx, input MemberBlacklistCreateInput, now time.Time) (*MemberBlacklistEntry, error)
ReleaseExpiredMemberBlacklistTx(ctx context.Context, tx pgx.Tx, key memberBlacklistKey, now time.Time) error
GetMemberBlacklistAccess(ctx context.Context, query MemberBlacklistAccessQuery, now time.Time) (*MemberBlacklistEntry, error)
ListMemberBlacklist(ctx context.Context, filter MemberBlacklistListFilter) ([]MemberBlacklistEntry, int, error)
ReleaseMemberBlacklistByIDTx(ctx context.Context, tx pgx.Tx, input MemberBlacklistReleaseInput, now time.Time) error
ReleaseMemberBlacklistBySubjectTx(ctx context.Context, tx pgx.Tx, input MemberBlacklistReleaseBySubjectInput, now time.Time) error
```

Access SQL must be read-only and include `released_at IS NULL AND (expires_at IS NULL OR expires_at > $now)`.

- [ ] **Step 4: Add service validation and audit**

Create `service_blacklist.go`. Enforce:

```text
admin API: manual_admin
bot API: manual_admin with created_from=koishi_console, kick_blacklist, moderation_action
internal service: admission_failure
migration_*: scripts only
```

Create and release paths write `member_blacklist.created` and `member_blacklist.released` audit events in the same transaction.

- [ ] **Step 5: Verify and commit**

Run: `cd server && go test -count=1 -timeout=60s ./internal/modules/admission -run TestMemberBlacklist`
Expected: PASS.

Commit:

```bash
git add server/internal/modules/admission
git commit -m "feat: add member blacklist service"
```

## Task 3: Routes And Admission Integration

- [ ] **Step 1: Write failing integration tests**

Add tests for:

```go
func TestBotMemberBlacklistRejectsAdmissionFailureSource(t *testing.T)
func TestAdmissionFailureCreatesGuildMemberBlacklist(t *testing.T)
func TestAdmissionBlacklistManualPardonResetsFailureCount(t *testing.T)
func TestExpiredAdmissionBlacklistSweepResetsFailureCount(t *testing.T)
func TestDuplicateKickEventDoesNotDoubleCountFailure(t *testing.T)
func TestMemberBlacklistListPaginatesAndFilters(t *testing.T)
```

Run: `cd server && go test -count=1 -timeout=60s ./internal/modules/admission -run 'Blacklist|Routes'`
Expected: FAIL.

- [ ] **Step 2: Wire handlers and permissions**

Create `handler_blacklist.go`. In `handler.go`, remove old admission QQ access/release routes. Add bot routes using new service-account scopes and admin routes using member blacklist read/manage capabilities.

- [ ] **Step 3: Integrate admission event path**

In `service_session.go`, make successful kick/blacklist idempotent: if session is already `expired_kicked`, return without incrementing. When failure count reaches policy limit, create or reuse a guild scoped `admission_failure` member blacklist entry in the same transaction.

- [ ] **Step 4: Add expiry worker behavior**

In `service_expiry.go`, add a worker method that releases expired member blacklist rows. For `source='admission_failure'`, reset matching `group_admission_failures.failure_count` to 0 in the same transaction.

- [ ] **Step 5: Verify and commit**

Run:

```bash
cd server
go test -count=1 -timeout=60s ./internal/modules/admission
make check-drift
```

Expected: PASS.

Commit:

```bash
git add server/internal/modules/admission server/internal/pkg/capability clients/shared/src/constants server/api
git commit -m "feat: wire member blacklist backend"
```
