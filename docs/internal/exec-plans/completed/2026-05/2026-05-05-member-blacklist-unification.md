---
type: internal
audience: maintainers, backend-dev, frontend-dev
status: archived
authoritative-source: this file
last-verified: 2026-05-07
---

# Member Blacklist Unification Plan Index

**Goal:** Replace split Koishi/admission blacklist sources with one backend-owned member blacklist supporting guild and global scopes.

**Architecture:** The work is split into backend and Koishi plans so each plan stays under the repository file-size limit. Implement backend first because OpenAPI, schema, and source validation define the contract that Koishi consumes. Implement Koishi second after generated/shared types are available.

**Tech Stack:** Go 1.26, Gin, pgx/PostgreSQL, OpenAPI 3.1, oapi-codegen, TypeScript, Koishi, node:test.

**Status:** Complete. Backend, Koishi, Admin, and console integration are implemented on `feat/member-blacklist-unification`; follow-up review fixes are folded into the branch through `1dbd03f`.

**Verification Snapshot:** The branch has fresh backend, shared client, Koishi workspace, OpenAPI lint/generate, and whitespace verification from the implementation pass. Current behavior is authoritative in the source, OpenAPI contract, migration baseline, and tests.

---

## Execution Order

- [x] **Step 1: Implement backend plan**

Use [2026-05-05-member-blacklist-unification-backend.md](2026-05-05-member-blacklist-unification-backend.md).

Expected result: backend schema, OpenAPI, generated types, service, routes, admission integration, expiry behavior, and admission tests pass.

- [x] **Step 2: Implement Koishi plan**

Use [2026-05-05-member-blacklist-unification-koishi.md](2026-05-05-member-blacklist-unification-koishi.md).

Expected result: Koishi shared client, request flow, commands, console API, and blacklist UI consume backend member blacklist APIs.

- [x] **Step 3: Run integrated verification**

Run:

```bash
cd server
go test -count=1 -timeout=60s ./internal/modules/admission
make check-drift
cd ../bots/koishi
corepack yarn tsx --test packages/shared/src/platform/index.test.ts plugins/stuhelper-core/src/core/modules/event-handlers-admission.test.ts plugins/stuhelper-core/src/core/modules/config-commands.test.ts plugins/stuhelper-core/src/core/modules/member-manage-commands.test.ts
corepack yarn tsc --noEmit
cd ../..
git diff --check
```

Expected: all commands pass.
