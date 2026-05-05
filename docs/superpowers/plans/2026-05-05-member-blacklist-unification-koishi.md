# Member Blacklist Koishi Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Move Koishi blacklist reads and writes from local JSON to backend member blacklist APIs.

**Architecture:** `@stuhelper/koishi-shared` exposes backend member blacklist methods. `stuhelper-core` request handling, commands, console API, and UI call those methods. `blacklist.json` stops being a business write path.

**Tech Stack:** TypeScript, Koishi, node:test, Vue.

---

## File Map

- Modify: `bots/koishi/packages/shared/src/types/index.ts`
- Modify: `bots/koishi/packages/shared/src/platform/index.ts`
- Modify: `bots/koishi/packages/shared/src/platform/index.test.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/modules/event-support.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/modules/event-handlers.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/modules/config-commands.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/modules/member-manage-commands.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/modules/report-violation.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/core/api/index.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/client/api.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/client/types.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/client/components/BlacklistView.vue`

## Task 1: Shared Client And Request Access Flow

- [ ] **Step 1: Write failing shared client tests**

In `platform/index.test.ts`, assert calls:

```ts
['GET', '/api/v1/bot/member-blacklist/access?platform=mock&subjectType=qq_user&guildID=guild-1&subjectID=10001']
['GET', '/api/v1/bot/member-blacklist?pageSize=50']
['POST', '/api/v1/bot/member-blacklist']
['POST', '/api/v1/bot/member-blacklist/blk-1/release']
['POST', '/api/v1/bot/member-blacklist/release-by-subject']
```

Run: `cd bots/koishi && corepack yarn tsx --test packages/shared/src/platform/index.test.ts`
Expected: FAIL.

- [ ] **Step 2: Add shared types and client methods**

Replace old `AdmissionQQAccess` and `releaseAdmissionBlacklist` methods with:

```ts
getMemberBlacklistAccess(input: MemberBlacklistAccessQuery): Promise<MemberBlacklistAccessDecision>
listMemberBlacklist(input: MemberBlacklistListQuery): Promise<MemberBlacklistListResult>
createMemberBlacklist(input: MemberBlacklistCreateRequest): Promise<MemberBlacklistEntry>
releaseMemberBlacklist(id: string, input: MemberBlacklistReleaseRequest): Promise<void>
releaseMemberBlacklistBySubject(input: MemberBlacklistReleaseBySubjectRequest): Promise<void>
```

- [ ] **Step 3: Write failing request flow tests**

In `event-handlers-admission.test.ts`, add:

```ts
test('guild-member-request rejects backend member blacklist decision', async () => {})
test('guild-member-request does not approve or reject when access check times out', async () => {})
```

Expected first test rejects with reason `您在黑名单中`; timeout test has no approval call and reaches admission guarded flow.

- [ ] **Step 4: Update request flow**

In `event-support.ts`, replace `getAdmissionQQAccess` dependency with `getMemberBlacklistAccess` and `recordJoinRequestEvent`. In `event-handlers.ts`, call backend access after legacy cooldown/level/keyword checks and before admission auto-approve.

- [ ] **Step 5: Verify and commit**

Run:

```bash
cd bots/koishi
corepack yarn tsx --test packages/shared/src/platform/index.test.ts plugins/stuhelper-core/src/core/modules/event-handlers-admission.test.ts
corepack yarn tsc --noEmit
```

Expected: PASS.

Commit:

```bash
git add bots/koishi/packages/shared bots/koishi/plugins/stuhelper-core/src/core/modules
git commit -m "feat(koishi): use member blacklist access"
```

## Task 2: Commands And Moderation Writes

- [ ] **Step 1: Write failing command tests**

Add or update tests for:

```ts
config -b -a 10001 --global
config -b -r 10001
kick 10001 -b
moderation action kick_blacklist
```

Expected API calls:

```ts
source: 'manual_admin'
source: 'kick_blacklist'
source: 'moderation_action'
scopeType: 'guild' | 'global'
```

- [ ] **Step 2: Replace local JSON writes**

Remove command writes to `host.data.blacklist.setAll`, `set`, and `delete`. Route `config -b`, `kick -b`, and `kick_blacklist` through `PlatformClient` member blacklist methods. If backend write fails after QQ kick, return a visible failure message and log it.

- [ ] **Step 3: Verify and commit**

Run:

```bash
cd bots/koishi
corepack yarn tsx --test plugins/stuhelper-core/src/core/modules/config-commands.test.ts plugins/stuhelper-core/src/core/modules/member-manage-commands.test.ts
corepack yarn tsc --noEmit
```

Expected: PASS.

Commit:

```bash
git add bots/koishi/plugins/stuhelper-core/src/core/modules
git commit -m "feat(koishi): write member blacklist through backend"
```

## Task 3: Console API And UI

- [ ] **Step 1: Write failing console tests**

Update component/API tests so blacklist list/add/remove no longer call `data.blacklist` directly. Expected listener behavior:

```ts
list -> platform.listMemberBlacklist({ pageSize: 50 })
add -> platform.createMemberBlacklist({ source: 'manual_admin', subjectID: '10001', scopeType: 'guild', guildID: 'guild-1' })
remove -> platform.releaseMemberBlacklist(id, { releaseReasonCode: 'manual_pardon' })
```

- [ ] **Step 2: Update API listeners**

In `core/api/index.ts`, replace `data.blacklist` list/add/remove with backend client calls. Enforce console guild scope before sending guild scoped create/release requests.

- [ ] **Step 3: Update Vue client**

In `BlacklistView.vue`, render backend fields: subjectID, scopeType, guildID, source, reasonCode, reasonText, createdAt, expiresAt, releasedAt. Add scope selector. Global create requires an explicit confirmation dialog.

- [ ] **Step 4: Verify and commit**

Run:

```bash
cd bots/koishi
corepack yarn tsx --test plugins/stuhelper-core/src/core/api/*.test.ts plugins/stuhelper-core/client/*.test.ts
corepack yarn tsc --noEmit
```

Expected: PASS.

Commit:

```bash
git add bots/koishi/plugins/stuhelper-core/src/core/api bots/koishi/plugins/stuhelper-core/client
git commit -m "feat(koishi): update blacklist console"
```
