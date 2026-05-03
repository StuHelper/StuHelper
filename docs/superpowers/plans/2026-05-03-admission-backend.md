# Admission Backend Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Build the backend source of truth for QQ admission sessions, credentials, freshman review, policies, expiry, blacklist, audit, and bot/admin APIs.

**Architecture:** Add `server/internal/modules/admission` for session, policy, application, material, blacklist, and review workflows. Keep `user` as the owner of accounts, QQ bindings, and IAM role outbox; admission calls explicit user/storage/audit gateways and never treats Koishi SQLite as identity truth.

**Tech Stack:** Go, Gin, pgx, PostgreSQL migrations, Redis OTP, OpenAPI 3.1, existing `audit_events`, IAM outbox, Casdoor role sync, object storage.

---

## File Structure

- Create `server/internal/modules/admission/{models,errors,repository,repository_session,repository_policy,repository_application,repository_failure,service,service_session,service_student,service_freshman,service_operator,service_expiry,service_policy,handler,handler_user,handler_bot,handler_admin,material_store,route_contract_test,service_session_test,service_student_test,service_freshman_test,service_operator_test,service_expiry_test,repository_integration_test}.go`.
- Create `server/api/paths/{admission,admin-admission,bot-admission}.yaml` and `server/api/components/schemas/admission.yaml`.
- Create `server/migrations/000033_admission_verification.{up,down}.sql`.
- Modify `server/api/openapi.yaml`, `server/internal/app/modules.go`, `server/internal/app/modules_auth.go`, `server/internal/modules/user/service_qq_binding.go`, `server/internal/modules/user/service.go`, `server/internal/modules/user/external_sync.go`, `server/internal/modules/notification/templates.go`, `server/internal/pkg/capability/capability.go`, and `server/internal/platform/serviceaccount/constants.go`.

## Task 1: OpenAPI Contract

**Files:** `server/api/openapi.yaml`, `server/api/paths/admission.yaml`, `server/api/paths/admin-admission.yaml`, `server/api/paths/bot-admission.yaml`, `server/api/components/schemas/admission.yaml`, `server/internal/modules/admission/route_contract_test.go`

- [ ] **Step 1: Write failing route tests**

Assert routes for user session preview/link, freshman application/camera capture, email OTP, school SSO, bot sessions/actions/events/join-request-events/forward/review, admin policies/sessions/freshman reviews/blacklist release. Add error code expectations for `admission.qq_mismatch`, `admission.token_consumed`, and `admission.token_expired`.

- [ ] **Step 2: Run failing test**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run TestAdmissionRoutes`
Expected: FAIL because admission routes do not exist.

- [ ] **Step 3: Add schemas**

Define `AdmissionSession`, `AdmissionMe`, `AdmissionPolicy`, `FreshmanApplication`, `FreshmanReviewRequest`, `BotAdmissionSessionCreateRequest`, `BotAdmissionEventRequest`, `BotAdmissionJoinRequestEvent`, `BotFreshmanReviewRequest`, `CameraCaptureRequest`, `SchoolEmailOTPRequest`, and `SchoolEmailOTPVerifyRequest`. `AdmissionMe` includes `projectionPending`. Bot review body must include `operatorQQID`, `guildID`, `channelID`, `rawCommand`, and optional `expiresInDays`.

- [ ] **Step 4: Generate and commit**

Run: `cd server && make generate && make lint-spec && make check-drift`
Expected: PASS.
Commit: `git add server/api server/internal/modules/admission/route_contract_test.go clients/shared/src/types/api.gen.ts server/internal/api/gen && git commit -m "feat: add admission api contract"`

## Task 2: Persistence Model

**Files:** `server/migrations/000033_admission_verification.{up,down}.sql`, `server/internal/modules/admission/models.go`, `server/internal/modules/admission/repository_integration_test.go`

- [ ] **Step 1: Write failing migration test**

Insert policy, session, freshman application, material, credential, and failure rows. Assert token hash uniqueness, pending freshman uniqueness, credential expiry requirement for `freshman_material_manual`, and that a new session is allowed after old status becomes `expired_kicked`.

- [ ] **Step 2: Run failing test**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run TestAdmissionMigration`
Expected: FAIL.

- [ ] **Step 3: Add tables and indexes**

Create `user_verification_credentials`, `group_admission_policies`, `group_admission_sessions`, `freshman_verification_applications`, `freshman_verification_materials`, and `group_admission_failures`. Do not create a separate admission audit table; use existing `audit_events`. Add `management_guild_ids TEXT[] NOT NULL DEFAULT '{}'`, `UNIQUE(platform,guild_id,qq_id) WHERE status IN ('joined_muted','linked')`, indexes for pending applications, failures by `(platform,guild_id,failure_count DESC)`, and material lookup by application.

- [ ] **Step 4: Add models**

Define status constants `joined_muted`, `linked`, `material_submitted`, `verified`, `expired_kicked`, `cancelled`, `pending`, `approved`, and `rejected`. Add policy fields `linkWaitSeconds`, `submissionWaitSeconds`, `manualReviewTimeoutSeconds`, `maxMaterialBytes`, `maxExtensionDays`, and `managementGuildIDs []string`.

- [ ] **Step 5: Run and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run TestAdmissionMigration`
Expected: PASS.
Commit: `git add server/migrations server/internal/modules/admission && git commit -m "feat: add admission persistence model"`

## Task 3: Identity, Capability, And Bot Scopes

**Files:** `server/internal/modules/user/{service_qq_binding.go,service.go,service_qq_binding_test.go,external_sync.go}`, `server/internal/pkg/capability/{capability.go,capability_test.go}`, `server/internal/platform/serviceaccount/{constants.go,verifier_integration_test.go}`

- [ ] **Step 1: Write failing identity tests**

Cover `EnsureQQBindingForUserTx(ctx, tx, userID, qqID, nickname)` create, idempotent same binding, user-bound-to-other-QQ conflict, QQ-bound-to-other-user conflict, and no nested transaction.

- [ ] **Step 2: Write failing capability/scope tests**

Assert role `freshman_provisional` has the same capability set as `verified_student` but remains a distinct role. Assert `super_admin` has admission capabilities. Assert `KoishiRuntimeScopes()` contains existing QQ scopes plus `bot.admission.session`, `bot.admission.event`, `bot.admission.review`, and `bot.admission.forward`.

- [ ] **Step 3: Implement gateway and constants**

Add Tx gateway methods without changing existing `ConsumeQQBindingCode`. Add `AdmissionPolicyRead`, `AdmissionPolicyUpdate`, `AdmissionFreshmanRead`, `AdmissionFreshmanReview`, `AdmissionBlacklistManage`, `AdmissionSessionRead`, and service account scopes.

- [ ] **Step 4: Run and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/user ./server/internal/pkg/capability ./server/internal/platform/serviceaccount`
Expected: PASS.
Commit: `git add server/internal/modules/user server/internal/pkg/capability server/internal/platform/serviceaccount && git commit -m "feat: add admission identity gateways"`

## Task 4: Session State Machine

**Files:** `server/internal/modules/admission/{repository_session,repository_policy,repository_failure,service_session,handler_user,handler_bot}.go`, `server/internal/app/modules.go`

- [ ] **Step 1: Write failing state tests**

Cover create session, token preview without consume, `qq` mismatch, expired token, consumed token, linked status, link success resetting `submission_wait_deadline_at` from link time, material-submitted status, verified status, and kick event incrementing failure count.

- [ ] **Step 2: Implement session repository**

Implement `CreateSession`, `GetSessionByTokenHashForUpdate`, `MarkTokenConsumedAndLinked`, `MarkMaterialSubmitted`, `MarkVerified`, and `CancelSession`. Use `FOR UPDATE` when consuming tokens.

- [ ] **Step 3: Implement policy and failure repositories**

Implement policy lookup with global/default fallback, action listing by deadlines, `RecordBotEvent`, `IncrementFailureFromKickEvent`, and `ApplyBlacklistTx`. Backend increments failure count when a bot kick/expired event succeeds.

- [ ] **Step 4: Implement atomic link service**

`LinkTokenToUser` must run one `repo.WithTx` containing token lock, token consumed check, QQ query match, `EnsureQQBindingForUserTx`, and session `user_id` update. Add concurrency test: two users link the same token; exactly one succeeds and the other receives `token_consumed`.

- [ ] **Step 5: Register handlers and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run 'TestAdmissionSession|TestAdmissionToken|TestAdmissionFailureBlacklist'`
Expected: PASS.
Commit: `git add server/internal/modules/admission server/internal/app && git commit -m "feat: add admission session state machine"`

## Task 5: Student Verification, Freshman Material, And Email OTP

**Files:** `server/internal/modules/admission/{service_student,service_freshman,repository_application,material_store,handler_user}.go`, `server/internal/modules/notification/templates.go`

- [ ] **Step 1: Write failing verification tests**

Cover school SSO login only for configured schools, return URL whitelist/state binding, callback writing `school_sso` credential, camera-capture endpoint rejecting PDF/non-image/oversized bytes, channel close time, and duplicate pending application rejection.

- [ ] **Step 2: Implement school SSO flow**

Use the configured school SSO provider behind Casdoor. Bind OIDC state to admission session and same-origin admission return target; callback writes `user_verification_credentials(kind=school_sso)` and marks the linked session verified.

- [ ] **Step 3: Implement material flow**

Accept only decoded JPEG/PNG/WebP bytes from the dedicated camera-capture endpoint and store through `storage.Service.Put`. Store `sha256`, size, content type, and object key. Do not add any generic upload endpoint; backend enforces image format and size, while frontend tests prove the only submission UI is live camera capture.

- [ ] **Step 4: Implement email OTP**

Allow OTP request only for authenticated users with a linked admission session. Store Redis key `admission:email_otp:<userID>:<schoolID>`, five-minute TTL, one-minute resend cooldown, five attempts, domain whitelist, `subject_hash = HMAC(school_id || normalized_email)`, and masked `subject_display`.

- [ ] **Step 5: Run and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run 'TestStudent|TestFreshman|TestCamera|TestSchoolEmail'`
Expected: PASS.
Commit: `git add server/internal/modules/admission server/internal/modules/notification && git commit -m "feat: add student admission verification"`

## Task 6: Review Authorization And Policy Admin

**Files:** `server/internal/modules/admission/{service_operator,handler_bot,handler_admin,service_policy}.go`

- [ ] **Step 1: Write failing operator tests**

Cover unbound operator QQ rejected, operator bound to user without `admission:freshman:review` rejected, command from non-whitelisted management guild rejected, and authorized operator approve/reject accepted.

- [ ] **Step 2: Implement operator authorization**

Bot service token authenticates the bot only. Review service must resolve `operatorQQID -> user_qq_bindings -> user_id -> capability grant`, verify `guildID` is in policy `managementGuildIDs`, enforce `maxExtensionDays`, and write audit details including raw command text.

- [ ] **Step 3: Implement admin handlers**

Admin review uses logged-in user capability/MFA. Bot and Admin review call the same service method after building different operator contexts.

- [ ] **Step 4: Run and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission -run 'TestOperator|TestAdmissionPolicy|TestFreshmanReview'`
Expected: PASS.
Commit: `git add server/internal/modules/admission && git commit -m "feat: add admission review authorization"`

## Task 7: Projection, Expiry, Notifications, And Audit

**Files:** `server/internal/modules/admission/{service_freshman,service_expiry}.go`, `server/internal/modules/user/external_sync.go`, `server/internal/modules/notification/templates.go`

- [ ] **Step 1: Write failing projection/expiry tests**

Cover approve enqueues `freshman_provisional` role sync through IAM outbox, session verified does not wait for Casdoor projection, `/api/v1/admission/me` exposes projection pending/ready status, expired credentials enqueue role removal, and only `freshman_provisional` is removed.

- [ ] **Step 2: Implement outbox projection**

Add `enqueueFreshmanProvisionalRoleSyncTx` following existing verified-student outbox pattern. Approval transaction writes credential, application status, session verified, unified audit event, notification, and outbox job.

- [ ] **Step 3: Implement expiry worker**

Add admission background job with bounded batch size from `outbox.IAMWorkerBatchSize`, polling interval from existing worker config, and explicit logs. Expired provisional credentials enqueue role removal and mark expiry processed.

- [ ] **Step 4: Add notifications and audit**

Add notification templates for freshman approved, rejected, and near-expiry. Reuse `audit_events`; admin/QQ review and blacklist release use `category='admin_operation'`; expiry revocation, bot action result, and auto-approve failure use `category='domain_event'`; join-request approve failure uses `resource_type='admission.join_request'`; do not create a separate admission audit table.

- [ ] **Step 5: Run and commit**

Run: `go test -count=1 -timeout=60s ./server/internal/modules/admission ./server/internal/modules/user ./server/internal/modules/notification && cd server && make check-drift`
Expected: PASS.
Commit: `git add server/internal/modules/admission server/internal/modules/user server/internal/modules/notification server/api server/internal/api/gen clients/shared/src/types/api.gen.ts && git commit -m "feat: add admission projection and expiry"`

## Self-Review

- Spec coverage: backend covers admission tokens, atomic QQ binding, SSO handoff, credentials, freshman material review, policy, failure blacklist, operator QQ authorization, bot/admin APIs, role expiry, notification, and audit.
- No placeholders: all tasks name files, commands, expected outcomes, and commit boundaries.
- Type consistency: `freshman_provisional` is a distinct role with the same capabilities as `verified_student`, not a formal verified-student credential.
