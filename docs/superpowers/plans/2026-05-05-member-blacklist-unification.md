# Member Blacklist Unification Plan Index

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Replace split Koishi/admission blacklist sources with one backend-owned member blacklist supporting guild and global scopes.

**Architecture:** The work is split into backend and Koishi plans so each plan stays under the repository file-size limit. Implement backend first because OpenAPI, schema, and source validation define the contract that Koishi consumes. Implement Koishi second after generated/shared types are available.

**Tech Stack:** Go 1.26, Gin, pgx/PostgreSQL, OpenAPI 3.1, oapi-codegen, TypeScript, Koishi, node:test.

---

## Execution Order

- [ ] **Step 1: Implement backend plan**

Use [2026-05-05-member-blacklist-unification-backend.md](2026-05-05-member-blacklist-unification-backend.md).

Expected result: backend schema, OpenAPI, generated types, service, routes, admission integration, expiry behavior, and admission tests pass.

- [ ] **Step 2: Implement Koishi plan**

Use [2026-05-05-member-blacklist-unification-koishi.md](2026-05-05-member-blacklist-unification-koishi.md).

Expected result: Koishi shared client, request flow, commands, console API, and blacklist UI consume backend member blacklist APIs.

- [ ] **Step 3: Run integrated verification**

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
