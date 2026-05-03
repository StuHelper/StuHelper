# Admission Koishi Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make Koishi the QQ executor for admission sessions: auto-approve join, mute, send canonical auth links, remind, unmute, kick, blacklist, forward materials, and accept QQ admin review commands.

**Architecture:** Keep backend as the source of truth. `stuhelper-group-guard` creates and executes backend admission sessions, `packages/shared` owns the platform client, and `stuhelper-admin` forwards review commands to backend after local command permission checks.

**Tech Stack:** Koishi, TypeScript, `@koishijs/plugin-mock`, SQLite test runtime, generated platform HTTP client, Node test runner.

---

## File Structure

- Modify `bots/koishi/packages/shared/src/{platform/index.ts,types/index.ts,config/index.ts}`.
- Modify `bots/koishi/plugins/stuhelper-group-guard/src/{member-guard,events,index}.ts` and tests in the same folder.
- Modify `bots/koishi/plugins/stuhelper-admin/src/{commands,index}.ts` and tests.
- Add focused helpers `bots/koishi/plugins/stuhelper-group-guard/src/admission-format.ts` and `bots/koishi/plugins/stuhelper-admin/src/admission-review-commands.ts` if existing files approach 300 lines.

## Task 1: Platform Admission Client

**Files:** `bots/koishi/packages/shared/src/platform/index.ts`, `bots/koishi/packages/shared/src/types/index.ts`

- [ ] **Step 1: Write failing client tests**

Add tests that assert paths and payloads for `createAdmissionSession`, `listPendingAdmissionActions`, `recordAdmissionEvent`, `listPendingFreshmanForwards`, `markFreshmanForwarded`, and `reviewFreshmanApplication`.

- [ ] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test packages/shared/src/platform/*.test.ts`
Expected: FAIL because methods do not exist.

- [ ] **Step 3: Implement types**

Add `AdmissionSessionCreateRequest`, `AdmissionSessionCreateResult`, `AdmissionBotAction`, `AdmissionBotEventRequest`, `FreshmanForwardItem`, and `FreshmanReviewRequest`. Use readonly fields and string union status values.

- [ ] **Step 4: Implement client methods**

Use service token auth already in `createRequest`. Add endpoints `/api/v1/bot/admission/sessions`, `/api/v1/bot/admission/sessions/pending`, `/api/v1/bot/admission/sessions/{id}/events`, `/api/v1/bot/admission/freshman/applications/pending-forward`, `/api/v1/bot/admission/freshman/applications/{id}/forwarded`, and `/api/v1/bot/admission/freshman/applications/{id}/review`.

- [ ] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test packages/shared/src/platform/*.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/packages/shared/src && git commit -m "feat: add admission platform client"`

## Task 2: Join Admission Session

**Files:** `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/admission-format.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/events.test.ts`

- [ ] **Step 1: Write failing join tests**

Test a new unverified QQ member calls backend `createAdmissionSession`, mutes for returned `initialMuteDurationSeconds`, and sends `https://auth.stuhelper.com/admission/a/<code>?qq=<qq>`. Assert no `buaa.team` and no `sso.stuhelper.com` appear in group text.

- [ ] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/events.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement formatter**

Create `formatAdmissionReminder(input)` returning `@user 请在 X 分钟内完成 StuHelper 学生身份认证：\n<url>\n通过后自动解除禁言，超时将移出群聊。` and use `h.at(memberId)` for mention.

- [ ] **Step 4: Replace local join authority**

In `handleGuildMemberAdded`, after policy exemption checks, call backend create session instead of `getQQVerificationStatus`. Save local guard record only as execution cache with backend `sessionID`, `deadlineAt`, and `nextReminderAt`.

- [ ] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/events.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-group-guard/src bots/koishi/packages/shared/src && git commit -m "feat: create admission sessions on join"`

## Task 3: Reminder, Release, Kick, and Blacklist Actions

**Files:** `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.test.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/store.ts`

- [ ] **Step 1: Write failing action tests**

Mock backend pending actions: `remind`, `release`, `kick`, `blacklist`. Assert bot sends reminder, unmutes with duration `0`, sends pre-kick warning then kicks, kicks with blacklist flag when requested, and reports each result through `recordAdmissionEvent`.

- [ ] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement action dispatcher**

Replace direct local scan decisions with `listPendingAdmissionActions({ platform, botSelfId })`. Route actions to one function per action: `executeReminder`, `executeRelease`, `executeKick`, and `executeBlacklist`. Each function returns a concrete event payload with `success`, `action`, `messageID` when available, and explicit error text on failure.

- [ ] **Step 4: Preserve visible failures**

On platform API or bot API error, call backend `recordAdmissionEvent` with `success=false` and leave `lastBotError` visible. Do not swallow errors after reporting.

- [ ] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts plugins/stuhelper-group-guard/src/events.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-group-guard/src && git commit -m "feat: execute admission bot actions"`

## Task 4: Material Forwarding To Management Group

**Files:** `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/admission-format.ts`, `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.test.ts`

- [ ] **Step 1: Write failing forward tests**

Mock one pending freshman application with management group `9001`, image URL, applicant masked name, school, major, QQ, application ID, and expiry. Assert bot sends one image message plus summary text to group `9001`, then calls `markFreshmanForwarded`.

- [ ] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts`
Expected: FAIL.

- [ ] **Step 3: Implement forward scan**

During scheduled scan, call `listPendingFreshmanForwards`. For each item, send `h.image(materialURL)` only when backend explicitly returns a material URL and policy enables raw forwarding. Text must include application ID and approved/rejected command examples.

- [ ] **Step 4: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-group-guard/src/member-guard.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-group-guard/src && git commit -m "feat: forward freshman review materials"`

## Task 5: QQ Admin Review Commands

**Files:** `bots/koishi/plugins/stuhelper-admin/src/admission-review-commands.ts`, `bots/koishi/plugins/stuhelper-admin/src/commands.ts`, `bots/koishi/plugins/stuhelper-admin/src/index.test.ts`, `bots/koishi/plugins/stuhelper-admin/src/index.ts`

- [ ] **Step 1: Write failing command tests**

Assert `新生审核查看 A123`, `新生审核通过 A123`, `新生审核通过 A123 +30d`, `新生审核驳回 A123 材料不清晰`, and `新生黑名单解除 123456` call backend with operator QQ, guild ID, channel ID, raw command text, and optional expiry override.

- [ ] **Step 2: Run failing tests**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-admin/src/index.test.ts`
Expected: FAIL.

- [ ] **Step 3: Inject platform client**

Update `stuhelper-admin` to create `platform = createPlatformClient(config.platform)` and pass it to command registration. Keep existing local moderation commands unchanged.

- [ ] **Step 4: Implement commands**

Parse `+Nd` as positive integer days, reject zero/negative/non-integer values, and return backend error messages explicitly. Use local command authority checks before API calls; backend remains final authorization.

- [ ] **Step 5: Run and commit**

Run: `cd bots/koishi && corepack yarn tsx --test plugins/stuhelper-admin/src/index.test.ts && corepack yarn tsc --noEmit`
Expected: PASS.
Commit: `git add bots/koishi/plugins/stuhelper-admin/src && git commit -m "feat: add freshman review commands"`

## Task 6: Workspace Verification

**Files:** `bots/koishi/README.md`, `bots/koishi/scripts/startup-smoke.mjs`, `bots/koishi/plugins/stuhelper-core/src/p5-config-contract.test.ts`

- [ ] **Step 1: Update runtime docs and smoke expectations**

Document that `STUHELPER_PLATFORM_BASE_URL` and `STUHELPER_PLATFORM_SERVICE_TOKEN` now require admission bot scopes. Keep `koishi.yml` plugin loading unchanged.

- [ ] **Step 2: Run complete Koishi checks**

Run: `cd bots/koishi && corepack yarn test`
Expected: PASS, including startup smoke on port `5140`.

- [ ] **Step 3: Commit**

Run: `git add bots/koishi && git commit -m "test: verify admission koishi workspace"`

## Self-Review

- Spec coverage: Koishi covers auto join handling, default mute, canonical auth link, reminders, release, timeout kick, permanent blacklist action, material forwarding, QQ review commands, and explicit error reporting.
- No placeholders: every task names files, commands, expected results, and commit boundary.
- Type consistency: all bot API calls are routed through `PlatformClient`; backend remains the only authority for session state.
