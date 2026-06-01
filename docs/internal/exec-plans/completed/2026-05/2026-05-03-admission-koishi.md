---
type: internal
audience: maintainers, backend-dev
status: archived
authoritative-source: this file
last-verified: 2026-05-07
---

# Admission Koishi Implementation Plan

**Goal:** Make Koishi the QQ executor for admission sessions: auto-approve join, mute, send canonical auth links, remind, unmute, kick, blacklist, forward materials, and accept QQ admin review commands.

**Architecture:** Keep backend as the source of truth. `stuhelper-core` remains the only `guild-member-request` owner and appends admission auto-approve after existing request rules. `stuhelper-group-guard` creates and executes backend admission sessions after `guild-member-added`, `packages/shared` owns the platform client, and `stuhelper-admin` forwards review commands to backend after local command permission checks.

**Tech Stack:** Koishi, TypeScript, `@koishijs/plugin-mock`, SQLite test runtime, generated platform HTTP client, Node test runner.

**Status:** Complete. Koishi platform admission client methods, join handling, guarded member cache, action execution, freshman material forwarding, QQ admin review commands, and workspace tests are implemented.

**Implementation Notes:** Subsequent hardening made admission action scans resilient to single-action failures and added backend-unavailable fallback handling for newly joined members.

---

## File Structure

- Modify `bots/koishi/packages/shared/src/{platform/index.ts,types/index.ts,config/index.ts,guard/index.ts}`.
- Modify `bots/koishi/plugins/stuhelper-core/src/core/modules/{event-handlers,event-handlers-admission.test}.ts`.
- Modify `bots/koishi/plugins/stuhelper-group-guard/src/{member-guard,events,index}.ts` and tests in the same folder.
- Modify `bots/koishi/plugins/stuhelper-admin/src/{commands,index}.ts` and tests.
- Add focused helpers `bots/koishi/plugins/stuhelper-group-guard/src/admission-format.ts` and `bots/koishi/plugins/stuhelper-admin/src/admission-review-commands.ts` if existing files approach 300 lines.

## Task 1: Platform Admission Client

**Files:** `bots/koishi/packages/shared/src/platform/index.ts`, `bots/koishi/packages/shared/src/types/index.ts`

- [x] **Step 1: Write failing client tests**

Add tests that assert paths and payloads for `createAdmissionSession`, `recordJoinRequestEvent`, `listPendingAdmissionActions`, `recordAdmissionEvent`, `listPendingFreshmanForwards`, `markFreshmanForwarded`, and `reviewFreshmanApplication`.

- [x] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test packages/shared/src/platform/*.test.ts`
Expected: FAIL because methods do not exist.

- [x] **Step 3: Implement types**

Add `AdmissionSessionCreateRequest`, `AdmissionSessionCreateResult`, `AdmissionJoinRequestEvent`, `AdmissionBotAction`, `AdmissionBotEventRequest`, `FreshmanForwardItem`, and `FreshmanReviewRequest`. Use readonly fields and string union status values.

- [x] **Step 4: Implement client methods**

Use service token auth already in `createRequest`. Add endpoints `/api/v1/bot/admission/sessions`, `/api/v1/bot/admission/join-requests/events`, `/api/v1/bot/admission/sessions/pending`, `/api/v1/bot/admission/sessions/{id}/events`, `/api/v1/bot/admission/freshman/applications/pending-forward`, `/api/v1/bot/admission/freshman/applications/{id}/forwarded`, and `/api/v1/bot/admission/freshman/applications/{id}/review`.

- [x] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test packages/shared/src/platform/*.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/packages/shared/src && git commit -m "feat: add admission platform client"`

## Task 2: Join Request And Admission Session

**Files:** `bots/koishi/plugins/stuhelper-core/src/core/modules/{event-handlers,event-handlers-admission.test}.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/{events,member-guard,admission-format}.ts`, `bots/koishi/packages/shared/src/guard/index.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/{events,index}.test.ts`

- [x] **Step 1: Write failing request tests**

Test `stuhelper-core` keeps one `guild-member-request` listener, preserves existing blacklist/cooldown/level/keyword ordering, and only calls admission auto-approve after those rules do not reject. If approval fails, assert Koishi calls `recordJoinRequestEvent` with `success=false` and leaves the error visible. Assert `stuhelper-group-guard` does not register `guild-member-request`.

- [x] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-core/src/core/modules/event-handlers-admission.test.ts plugins/stuhelper-group-guard/src/events.test.ts`
Expected: FAIL.

- [x] **Step 3: Implement request listener**

Extend `stuhelper-core/src/core/modules/event-handlers.ts:handleGuildMemberRequest` after `acceptIfKeywordMatches`. Resolve request id from the same session, load backend admission policy, honor `autoApproveJoin`, call `session.bot.handleGuildMemberRequest`, and report failures through the platform client. Do not add a second `ctx.on('guild-member-request')` in `stuhelper-group-guard`.

- [x] **Step 4: Write failing added-member tests**

Test a new unverified QQ member calls backend `createAdmissionSession`, mutes for returned `initialMuteDurationSeconds`, and sends `https://join.stuhelper.com/verify/<token>?qq=<qq>`. Assert no `buaa.team` and no `sso.stuhelper.com` appear in group text.

- [x] **Step 5: Implement formatter and cache schema**

Create `formatAdmissionReminder(input)` returning `@user 请在 X 分钟内完成 StuHelper 学生身份认证：\n<url>\n通过后自动解除禁言，超时将移出群聊。` and use `h.at(memberId)` for mention.
Extend `GuardMemberRecord` with backend `admissionSessionID`, `nextReminderAt`, and optional `manualReviewDeadlineAt`; update `store.ts` save/update paths.

- [x] **Step 6: Replace local join authority**

In `handleGuildMemberAdded`, after policy exemption checks, call backend create session instead of `getQQVerificationStatus`. Save local guard record only as execution cache with backend `sessionID`, `deadlineAt`, and `nextReminderAt`.

- [x] **Step 7: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-core/src/core/modules/event-handlers-admission.test.ts plugins/stuhelper-group-guard/src/events.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-core/src bots/koishi/plugins/stuhelper-group-guard/src bots/koishi/packages/shared/src && git commit -m "feat: create admission sessions on join"`

## Task 3: Reminder, Release, Kick, and Blacklist Actions

**Files:** `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.test.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/store.ts`

- [x] **Step 1: Write failing action tests**

Mock backend pending actions: `remind`, `release`, `kick`, `blacklist`. Assert bot sends reminder, unmutes with duration `0`, sends pre-kick warning then kicks, kicks with blacklist flag when requested, and reports each result through `recordAdmissionEvent`.

- [x] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts`
Expected: FAIL.

- [x] **Step 3: Implement action dispatcher**

Replace direct local scan decisions with `listPendingAdmissionActions({ platform, botSelfId })`. Route actions to one function per action: `executeReminder`, `executeRelease`, `executeKick`, and `executeBlacklist`. Each function returns a concrete event payload with `success`, `action`, `messageID` when available, and explicit error text on failure.

- [x] **Step 4: Preserve visible failures**

On platform API or bot API error, call backend `recordAdmissionEvent` with `success=false` and leave `lastBotError` visible. Do not swallow errors after reporting.

- [x] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts plugins/stuhelper-group-guard/src/events.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-group-guard/src && git commit -m "feat: execute admission bot actions"`

## Task 4: Material Forwarding To Management Group

**Files:** `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/admission-format.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.test.ts`

- [x] **Step 1: Write failing forward tests**

Mock one pending freshman application with `managementGuildIDs: ['9001', '9002']`, image URL, applicant masked name, school, major, QQ, application ID, and expiry. Assert bot sends one image message plus summary text to both groups, then calls `markFreshmanForwarded`.

- [x] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts`
Expected: FAIL.

- [x] **Step 3: Implement forward scan**

During scheduled scan, call `listPendingFreshmanForwards`. For each item, send `h.image(materialURL)` only when backend explicitly returns a material URL and policy enables raw forwarding. Send it directly to every configured QQ management group with the application summary, then mark forwarded only after all sends succeed; v1 does not add signed URL handling, IP binding, watermark composition, or private download proxy. Text must include application ID and approved/rejected command examples.

- [x] **Step 4: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-group-guard/src && git commit -m "feat: forward freshman review materials"`

## Task 5: QQ Admin Review Commands

**Files:** `bots/koishi/plugins/stuhelper-admin/src/admission-review-commands.ts`, `bots/koishi/plugins/stuhelper-admin/src/commands.ts`, `bots/koishi/plugins/stuhelper-admin/src/index.test.ts`, `bots/koishi/plugins/stuhelper-admin/src/index.ts`

- [x] **Step 1: Write failing command tests**

Assert `新生审核查看 A123`, `新生审核通过 A123`, `新生审核通过 A123 +30d`, `新生审核驳回 A123 材料不清晰`, and `新生黑名单解除 123456` call backend with operator QQ, guild ID, channel ID, raw command text, and optional expiry override. Mock backend 403 codes for unbound/no-capability/wrong-management-guild and assert fixed Chinese replies.

- [x] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-admin/src/index.test.ts`
Expected: FAIL.

- [x] **Step 3: Inject platform client**

Update `stuhelper-admin` to create `platform = createPlatformClient(config.platform)` and pass it to command registration. Keep existing local moderation commands unchanged.

- [x] **Step 4: Implement commands**

Parse `+Nd` as positive integer days and reject zero/negative/non-integer values. Backend enforces `maxExtensionDays`; bot returns backend error messages explicitly. Use local command authority checks before API calls; backend remains final authorization.

- [x] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-admin/src/index.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-admin/src && git commit -m "feat: add freshman review commands"`

## Task 6: Workspace Verification

**Files:** `bots/koishi/README.md`, `bots/koishi/scripts/startup-smoke.mjs`, `bots/koishi/plugins/stuhelper-core/src/p5-config-contract.test.ts`

- [x] **Step 1: Update runtime docs and smoke expectations**

Document that `STUHELPER_PLATFORM_BASE_URL` and `STUHELPER_PLATFORM_SERVICE_TOKEN` now require admission bot scopes. Clarify which local guard policy fields remain local command defaults and which admission policy fields come from backend. Keep `koishi.yml` plugin loading unchanged.

- [x] **Step 2: Run complete Koishi checks**

Run: `cd bots/koishi && corepack yarn test`
Expected: PASS, including startup smoke on port `5140`.

- [x] **Step 3: Commit**

Run: `git add bots/koishi && git commit -m "test: verify admission koishi workspace"`

## Self-Review

- Spec coverage: Koishi covers auto join handling, default mute, canonical auth link, reminders, release, timeout kick, permanent blacklist action, material forwarding, QQ review commands, and explicit error reporting.
- No placeholders: every task names files, commands, expected results, and commit boundary.
- Type consistency: all bot API calls are routed through `PlatformClient`; backend remains the only authority for session state.
