# Admission Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the backend source of truth for QQ admission sessions, old-student credentials, freshman manual review, policies, expiry, audit, and bot/admin APIs.

**Architecture:** Add a focused `server/internal/modules/admission` module and keep existing `user` as the owner of accounts, QQ bindings, and role sync. Admission owns session state, material applications, policies, blacklists, and calls explicit user/storage gateways.

**Tech Stack:** Go, Gin, pgx, PostgreSQL migrations, Redis OTP, OpenAPI 3.1, Casdoor OIDC role sync, S3-compatible object storage.

---

## File Structure

- Create `server/internal/modules/admission/{models,errors,repository,repository_session,repository_policy,repository_application,service,service_session,service_freshman,service_policy,handler,handler_user,handler_bot,handler_admin,material_store,route_contract_test,service_session_test,service_freshman_test,repository_integration_test}.go`.
- Create `server/api/paths/{admission,admin-admission,bot-admission}.yaml` and `server/api/components/schemas/admission.yaml`.
- Create `server/migrations/000033_admission_verification.{up,down}.sql`.
- Modify `server/api/openapi.yaml`, `server/internal/app/modules.go`, `server/internal/app/modules_auth.go`, `server/internal/modules/user/service_qq_binding.go`, `server/internal/modules/user/service.go`, `server/internal/pkg/capability/capability.go`, `server/internal/pkg/capability/capability_test.go`, `server/internal/platform/serviceaccount/constants.go`, and `server/internal/platform/serviceaccount/verifier_integration_test.go`.

## Task 1: OpenAPI Contract

**Files:** `server/api/openapi.yaml`, `server/api/paths/admission.yaml`, `server/api/paths/admin-admission.yaml`, `server/api/paths/bot-admission.yaml`, `server/api/components/schemas/admission.yaml`, `server/internal/modules/user/route_contract_test.go`

- [ ] **Step 1: Add failing route contract tests**

Add route assertions for `GET /api/v1/admission/sessions/:token`, `POST /api/v1/admission/sessions/:token/link`, `POST /api/v1/admission/freshman/applications`, `POST /api/v1/bot/admission/sessions`, `GET /api/v1/bot/admission/sessions/pending`, `POST /api/v1/bot/admission/sessions/:id/events`, `POST /api/v1/bot/admission/freshman/applications/:id/review`, `GET /api/v1/admin/admission/policies`, `PUT /api/v1/admin/admission/policies/:id`, `GET /api/v1/admin/freshman-verifications`, and `PUT /api/v1/admin/freshman-verifications/:id`.

- [ ] **Step 2: Run failing contract test**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/user -run TestUserRoutes`
Expected: FAIL because the admission routes do not exist.

- [ ] **Step 3: Add OpenAPI paths and schemas**

Define schemas `AdmissionSession`, `AdmissionPolicy`, `FreshmanApplication`, `FreshmanReviewRequest`, `BotAdmissionSessionCreateRequest`, `BotAdmissionEventRequest`, `CameraCaptureRequest`, and `SchoolEmailOTPRequest`. Use `serviceTokenAuth` for bot routes, cookie/bearer auth for user/admin routes, and keep `qq` as display-only string.

- [ ] **Step 4: Generate and verify**

Run: `cd server && make generate && make lint-spec && make check-drift`
Expected: PASS and generated Go/TS types include admission operations.

- [ ] **Step 5: Commit**

Run: `git add server/api server/internal/modules/user/route_contract_test.go clients/shared/src/types/api.gen.ts server/internal/api/gen && git commit -m "feat: add admission api contract"`

## Task 2: Database Model

**Files:** `server/migrations/000033_admission_verification.{up,down}.sql`, `server/internal/modules/admission/models.go`, `server/internal/modules/admission/repository_integration_test.go`

- [ ] **Step 1: Write migration integration test**

Create a test that inserts one policy, one session, one freshman application, one material, and one failure row; assert token hashes are unique and `freshman_material_manual` rows require `expires_at`.

- [ ] **Step 2: Run failing repository test**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run TestAdmissionMigration`
Expected: FAIL because the module and tables are missing.

- [ ] **Step 3: Add migration**

Create tables `user_verification_credentials`, `group_admission_policies`, `group_admission_sessions`, `freshman_verification_applications`, `freshman_verification_materials`, `group_admission_failures`, and `admission_audit_events`. Add unique indexes for token hash, active QQ session per group, pending freshman application per user/school, and credential subject hash.

- [ ] **Step 4: Add immutable models**

Define Go structs with typed status constants: `AdmissionStatusJoinedMuted`, `AdmissionStatusLinked`, `AdmissionStatusVerified`, `AdmissionStatusExpiredKicked`, `ApplicationStatusPending`, `ApplicationStatusApproved`, and `ApplicationStatusRejected`.

- [ ] **Step 5: Run repository test**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run TestAdmissionMigration`
Expected: PASS.

- [ ] **Step 6: Commit**

Run: `git add server/migrations server/internal/modules/admission && git commit -m "feat: add admission persistence model"`

## Task 3: User Gateway and Capabilities

**Files:** `server/internal/modules/user/service_qq_binding.go`, `server/internal/modules/user/service.go`, `server/internal/modules/user/service_qq_binding_test.go`, `server/internal/pkg/capability/capability.go`, `server/internal/pkg/capability/capability_test.go`, `server/internal/platform/serviceaccount/constants.go`

- [ ] **Step 1: Write failing user service tests**

Add tests for `EnsureQQBindingForUser(ctx, userID, qqID, nickname)` covering create, same binding idempotent, user-bound-to-other-QQ conflict, and QQ-bound-to-other-user conflict.

- [ ] **Step 2: Add capability tests**

Assert `freshman_provisional` expands to the same review capabilities as `verified_student`, and `super_admin` includes `admission:policy:update`, `admission:freshman:review`, and `admission:blacklist:manage`.

- [ ] **Step 3: Run failing tests**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/user -run TestEnsureQQBindingForUser && go test -count=1 -timeout=60s ./server/internal/pkg/capability`
Expected: FAIL because the new service method, role, and capabilities are absent.

- [ ] **Step 4: Implement user gateway**

Add exported method `EnsureQQBindingForUser(ctx context.Context, userID int64, qqID string, qqNickname *string) (*QQBinding, error)` that reuses `resolveQQBindingConflict` inside `repo.WithTx` and never overwrites an existing mismatched binding.

- [ ] **Step 5: Add capabilities and bot scopes**

Add constants `AdmissionPolicyRead`, `AdmissionPolicyUpdate`, `AdmissionFreshmanRead`, `AdmissionFreshmanReview`, `AdmissionBlacklistManage`, plus service scopes `bot.admission.session`, `bot.admission.event`, and `bot.admission.review`.

- [ ] **Step 6: Run tests and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/user ./server/internal/pkg/capability ./server/internal/platform/serviceaccount`
Expected: PASS.
Commit: `git add server/internal/modules/user server/internal/pkg/capability server/internal/platform/serviceaccount && git commit -m "feat: add admission identity gateways"`

## Task 4: Admission State Machine

**Files:** `server/internal/modules/admission/{errors,repository,repository_session,repository_policy,service,service_session,handler,handler_user,handler_bot}.go`, `server/internal/app/modules.go`

- [ ] **Step 1: Write failing service tests**

Cover create session, token preview without consume, token mismatch rejection, login return without token consume, linked session consume, verified transition, expired kicked transition, and three failures causing permanent blacklist.

- [ ] **Step 2: Run failing tests**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run 'TestAdmissionSession|TestAdmissionToken|TestAdmissionFailureBlacklist'`
Expected: FAIL because service logic is missing.

- [ ] **Step 3: Implement repository methods**

Implement `CreateSession`, `GetSessionByTokenHash`, `LinkSessionToUser`, `MarkVerified`, `ListBotPendingActions`, `RecordBotEvent`, `IncrementFailure`, and `ApplyBlacklist`. Use parameterized SQL only.

- [ ] **Step 4: Implement service methods**

Implement `CreateBotSession`, `PreviewToken`, `LinkTokenToUser`, `RecordBotEvent`, and `ListPendingBotActions`. Hash tokens with HMAC, store only hash, bind `platform + guild_id + qq_id`, and consume only after authenticated link.

- [ ] **Step 5: Register routes**

Wire admission handlers in `registerAPIRoutes`, protected user routes with `authMW`, admin routes under existing admin group, and bot routes with service credential scopes.

- [ ] **Step 6: Run tests and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission ./server/internal/app`
Expected: PASS.
Commit: `git add server/internal/modules/admission server/internal/app && git commit -m "feat: add admission session state machine"`

## Task 5: Freshman, Email, and Expiry

**Files:** `server/internal/modules/admission/{service_freshman,repository_application,material_store,handler_admin}.go`, `server/internal/modules/notification/templates.go`

- [ ] **Step 1: Write failing tests**

Cover camera-only material accept, PDF rejection, MIME sniff rejection, oversized image rejection, freshman channel closed after October 1 12:00, approve with default expiry, approve with `+N days`, reject with reason, and expiry job revoking `freshman_provisional`.

- [ ] **Step 2: Run failing tests**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run 'TestFreshman|TestCamera|TestExpiry|TestSchoolEmail'`
Expected: FAIL.

- [ ] **Step 3: Implement material storage**

Use `storage.Service.Put(ctx, storage.DefaultMountKey, objectKey, bytes, contentType)`. Accept only decoded JPEG/PNG/WebP bytes after `image.DecodeConfig`; reject PDF signature `%PDF`, non-image MIME, and configured size larger than `10 MiB`.

- [ ] **Step 4: Implement review and role projection**

On approval, insert `user_verification_credentials(kind='freshman_material_manual')`, set `freshman_provisional` Casdoor role, mark linked sessions verified, and write audit. On expiry, remove only `freshman_provisional`.

- [ ] **Step 5: Implement school email OTP**

Store OTP in Redis under `admission:email_otp:<userID>:<schoolID>`, require school domain allowlist, five-minute TTL, one-minute resend cooldown, and five attempts. Store credential `school_email_otp` with email hash and masked display.

- [ ] **Step 6: Run backend verification and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission ./server/internal/modules/notification && cd server && make check-drift`
Expected: PASS.
Commit: `git add server/internal/modules/admission server/internal/modules/notification server/api server/internal/api/gen clients/shared/src/types/api.gen.ts && git commit -m "feat: add admission freshman verification"`

## Self-Review

- Spec coverage: backend covers admission tokens, SSO handoff endpoints, QQ binding, credentials, freshman material review, policy, failure blacklist, bot/admin APIs, role expiry, and audit.
- No placeholders: all tasks name files, test commands, expected outcomes, and commit boundaries.
- Type consistency: use `freshman_provisional` as a distinct role with review capabilities, not as a persisted formal `verified_student` credential.
