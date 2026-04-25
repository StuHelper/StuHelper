---
type: internal
audience: maintainers
status: archived
authoritative-source: this file
last-verified: 2026-04-25
---

> **Superseded by [ADR-0006](../../../adr/0006-koishi-core-ui-as-single-webui-entry.md)** (2026-04-25). 本计划提议把 `stuhelper-platform` 作为唯一入口；该方向已被否决。当前决策是保留 `stuhelper-core` 作为唯一 WebUI 入口、删除 `stuhelper-platform` 与 `stuhelper-console`、内部拆 server。执行计划见 [exec-plans/active/2026-04-25-koishi-plugin-restructure.md](../../exec-plans/active/2026-04-25-koishi-plugin-restructure.md)。本文件保留为历史方案记录。

# Koishi StuHelper Platform Redesign Implementation Plan

> **For agentic workers:** Use `superpowers:subagent-driven-development`. Execute task-by-task. Each task needs implementation, spec review, quality review, and focused verification before moving on.

## Goal

把 StuHelper 群管能力从旧 Koishi 配置项解耦为平台插件和模块系统。`koishi.yml` 只保留最小启动项；模块启停、配置、权限和群策略进入新增 WebUI。

## Completed Baseline

- [x] Task 1: Add platform package skeleton.
- [x] Task 2: Add module contract and registry.
- [x] Task 3: Add platform database models and config store.

## Remaining Tasks

### Task 4: Add platform service snapshot layer

**Files:**
- Create: `bots/koishi/plugins/stuhelper-platform/src/platform-service.ts`
- Create: `bots/koishi/plugins/stuhelper-platform/src/platform-service.test.ts`
- Modify: `bots/koishi/plugins/stuhelper-platform/src/index.ts`

**Requirements:**
- Build `StuhelperPlatformService` from registry and config store.
- Expose `listModules()` with manifest, enabled state, current config, permissions, commands, events and webui metadata.
- Expose `setModuleEnabled()` and `saveModuleConfig()` through the service.
- Keep service independent from concrete Koishi database implementation.
- Export the service from package entry.

**Verification:**
- `cd bots/koishi && yarn tsx --test plugins/stuhelper-platform/src/platform-service.test.ts`
- `cd bots/koishi && ./node_modules/.bin/tsc -p plugins/stuhelper-platform/tsconfig.json --noEmit --skipLibCheck`

### Task 5: Add console API routes

**Files:**
- Create: `bots/koishi/plugins/stuhelper-platform/src/console-routes.ts`
- Create: `bots/koishi/plugins/stuhelper-platform/src/console-routes.test.ts`
- Modify: `bots/koishi/plugins/stuhelper-platform/src/index.ts`

**Requirements:**
- Register platform console routes under one stable API namespace.
- Provide module listing, module enablement, module config save, and audit list endpoints.
- Validate request bodies at route boundary.
- Route handlers call service methods only.

**Verification:**
- `cd bots/koishi && yarn tsx --test plugins/stuhelper-platform/src/console-routes.test.ts`
- `cd bots/koishi && ./node_modules/.bin/tsc -p plugins/stuhelper-platform/tsconfig.json --noEmit --skipLibCheck`

### Task 6: Add first built-in group guard module adapter

**Files:**
- Create: `bots/koishi/plugins/stuhelper-platform/src/modules/group-guard.ts`
- Create: `bots/koishi/plugins/stuhelper-platform/src/modules/group-guard.test.ts`
- Modify: `bots/koishi/plugins/stuhelper-platform/src/index.ts`

**Requirements:**
- Represent group guard as a platform module.
- Declare manifest, config schema, permissions, commands, events and WebUI contribution.
- Do not read old Koishi config.
- Keep existing `stuhelper-group-guard` package untouched unless explicitly required later.

**Verification:**
- `cd bots/koishi && yarn tsx --test plugins/stuhelper-platform/src/modules/group-guard.test.ts`
- `cd bots/koishi && ./node_modules/.bin/tsc -p plugins/stuhelper-platform/tsconfig.json --noEmit --skipLibCheck`

### Task 7: Add platform WebUI shell

**Files:**
- Modify: `bots/koishi/plugins/stuhelper-platform/client/index.ts`
- Create: `bots/koishi/plugins/stuhelper-platform/client/page.vue`
- Create: `bots/koishi/plugins/stuhelper-platform/client/api.ts`
- Create: `bots/koishi/plugins/stuhelper-platform/client/model.ts`
- Create: `bots/koishi/plugins/stuhelper-platform/client/styles.css`

**Requirements:**
- Register only the new platform WebUI page.
- Show module list, module config, group policy, permission policy and audit tabs.
- Keep page copy concise.
- Do not add explanatory comments, implementation notes, or developer remarks in WebUI.

**Verification:**
- `cd bots/koishi && yarn tsx --test plugins/stuhelper-platform/client/model.test.ts`
- `cd bots/koishi && ./node_modules/.bin/tsc -p plugins/stuhelper-platform/tsconfig.json --noEmit --skipLibCheck`

### Task 8: Minimize Koishi config wiring

**Files:**
- Modify: `bots/koishi/koishi.yml`
- Modify: `bots/koishi/plugins/stuhelper-platform/src/index.ts`

**Requirements:**
- Load `stuhelper-platform` as the StuHelper entry.
- Remove old StuHelper module config from Koishi plugin config.
- Keep only runtime dependencies and plugin bootstrap.
- Do not add fallback to old configuration.

**Verification:**
- `cd bots/koishi && yarn test:unit`
- `cd bots/koishi && ./node_modules/.bin/tsc --noEmit --skipLibCheck`

### Task 9: Final integration review

**Requirements:**
- Run focused tests for all new platform files.
- Run plugin typecheck.
- Run `git diff --check`.
- Record known unrelated failures separately.
