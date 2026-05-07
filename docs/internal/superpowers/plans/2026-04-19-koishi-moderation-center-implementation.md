---
type: internal
audience: backend-dev, maintainers
status: archived
authoritative-source: this file
last-verified: 2026-04-19
---

# Koishi Moderation Center Implementation Plan

> 现状说明：本文是群管中心历史实现快照，不再作为当前项目待办来源。
>
> 当前已实现：QQ 绑定命令、入群禁言/解禁/超时踢出、关键词处理、复读检测、撤回留痕、举报命令、骰子/抽禁言、完整的 `群审*` 文本命令、`stuhelper-core` 自定义控制台、举报面板、群模板/群绑定面板、SQLite 留痕与自动化测试。
>
> 历史未实现候选：更重型的运营工作流，例如更细粒度报表、历史版本和更复杂的处置编排。该候选已转入 `docs/internal/exec-plans/active/current-project-open-items.md` 的"待立项候选"，不作为活跃执行任务。
>
> 归档说明：下方 checklist 表示历史计划闭环或被后续 Koishi/core 重构替代，不再表示当前待办。

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在现有 `bots/koishi/` 工作区内落地可运行的 StuHelper 群管中心基础版本，覆盖群管域模型、规则处理、管理员命令和自定义 Koishi Console 管理面板。

**Architecture:** 保留 `stuhelper-core` 作为唯一装配入口；将“平台 API 与共享配置”继续放在 `packages/shared`，新增独立的 `packages/moderation-core` 承载群管域模型、存储查询和规则引擎；`stuhelper-group-guard` 负责高频事件处理，`stuhelper-admin` 负责文本兜底命令，`stuhelper-console` 负责 Web 管理台扩展。

**Tech Stack:** Koishi 4、TypeScript、Yarn workspace、SQLite、@koishijs/console、Vue 3、Node.js test runner、@koishijs/plugin-mock

---

## File Structure

- Create: `bots/koishi/packages/moderation-core/`
- Create: `bots/koishi/plugins/stuhelper-console/`
- Modify: `bots/koishi/packages/shared/src/config/index.ts`
- Modify: `bots/koishi/packages/shared/src/types/index.ts`
- Modify: `bots/koishi/plugins/stuhelper-group-guard/src/*`
- Modify: `bots/koishi/plugins/stuhelper-admin/src/index.ts`
- Modify: `bots/koishi/plugins/stuhelper-core/src/index.ts`
- Modify: `bots/koishi/package.json`
- Modify: `bots/koishi/koishi.yml`
- Modify: `bots/koishi/README.md`

### Task 1: 建立群管中心领域核心包

**Files:**
- Create: `bots/koishi/packages/moderation-core/package.json`
- Create: `bots/koishi/packages/moderation-core/tsconfig.json`
- Create: `bots/koishi/packages/moderation-core/src/index.ts`
- Create: `bots/koishi/packages/moderation-core/src/constants.ts`
- Create: `bots/koishi/packages/moderation-core/src/types.ts`
- Create: `bots/koishi/packages/moderation-core/src/models.ts`
- Create: `bots/koishi/packages/moderation-core/src/store.ts`
- Create: `bots/koishi/packages/moderation-core/src/rule-engine.ts`
- Create: `bots/koishi/packages/moderation-core/src/action-service.ts`
- Test: `bots/koishi/packages/moderation-core/src/rule-engine.test.ts`

- [x] 写失败测试，覆盖警告阈值表达式、复读命中和关键词动作决策。
- [x] 运行 `cd bots/koishi && corepack yarn test:unit --test-name-pattern \"moderation-core\"`，确认新测试先失败。
- [x] 实现群管域常量、模型注册、存储访问和规则引擎最小闭环。
- [x] 复跑对应单测，确认转绿。

### Task 2: 扩展共享配置与类型

**Files:**
- Modify: `bots/koishi/packages/shared/src/config/index.ts`
- Modify: `bots/koishi/packages/shared/src/types/index.ts`
- Modify: `bots/koishi/packages/shared/src/index.ts`

- [x] 扩展核心配置，增加 `moderation`、`fun`、`console`、`ai` 等分组。
- [x] 定义事件日志、复核队列、成员处罚、角色权限、关键词和娱乐状态等共享类型。
- [x] 确保 `@stuhelper/koishi-shared` 只保留平台 API、配置工厂和跨插件复用的轻量类型。

### Task 3: 升级群管插件为事件驱动执法引擎

**Files:**
- Modify: `bots/koishi/plugins/stuhelper-group-guard/src/index.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/commands.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/events.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/message-guard.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/report-service.ts`
- Modify: `bots/koishi/plugins/stuhelper-group-guard/src/index.test.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/message-guard.test.ts`

- [x] 先写失败测试，覆盖入群准入、关键词警告、复读处罚、撤回留痕和举报入队。
- [x] 在插件中接入 `database`、`commands` 和 `moderation-core`，拆分事件监听与命令注册。
- [x] 让消息账本、告警累计、自动禁言、人工复核队列和娱乐命令都写入 SQLite。
- [x] 复跑插件测试，确认核心链路通过。

### Task 4: 增强文本管理员命令

**Files:**
- Modify: `bots/koishi/plugins/stuhelper-admin/src/index.ts`
- Create: `bots/koishi/plugins/stuhelper-admin/src/index.test.ts`

- [x] 先写失败测试，覆盖成员列表、批量禁言、警告查询、踢人申请和复核队列查看。
- [x] 使用 Koishi 通用命令体系注册分层管理员命令，并为高风险动作保留“提交复核”而非直接执行。
- [x] 为每条命令设置明确 `authority`，避免依赖自定义裸判断。
- [x] 复跑命令测试。

### Task 5: 新增自定义 Console 管理台插件

**Files:**
- Create: `bots/koishi/plugins/stuhelper-console/package.json`
- Create: `bots/koishi/plugins/stuhelper-console/tsconfig.json`
- Create: `bots/koishi/plugins/stuhelper-console/src/index.ts`
- Create: `bots/koishi/plugins/stuhelper-console/src/services/overview.ts`
- Create: `bots/koishi/plugins/stuhelper-console/src/services/queues.ts`
- Create: `bots/koishi/plugins/stuhelper-console/src/services/logs.ts`
- Create: `bots/koishi/plugins/stuhelper-console/client/index.ts`
- Create: `bots/koishi/plugins/stuhelper-console/client/page.vue`
- Create: `bots/koishi/plugins/stuhelper-console/client/composables/useModerationActions.ts`

- [x] 先写服务层失败测试，覆盖概览数据聚合和高风险动作监听。
- [x] 使用 `ctx.inject(['console'], ...)`、`ctx.console.addEntry()` 和 `DataService` 暴露控制台数据。
- [x] 使用自定义页面实现三栏工作台最小版本，包含总览、队列、日志和详情面板。
- [x] 为控制台操作监听设置显式 `authority`，并让所有高风险动作进入人工复核队列。

### Task 6: 更新装配、运行配置与文档

**Files:**
- Modify: `bots/koishi/plugins/stuhelper-core/src/index.ts`
- Modify: `bots/koishi/package.json`
- Modify: `bots/koishi/koishi.yml`
- Modify: `bots/koishi/README.md`
- Modify: `docs/internal/superpowers/specs/2026-04-19-koishi-moderation-center-design.md`

- [x] 将 `stuhelper-console` 纳入 `stuhelper-core` 装配。
- [x] 启用 Koishi Console 所需认证与页面入口，固定端口仍为 `5140`。
- [x] 更新 README，说明插件职责、数据边界和验证命令。
- [x] 如实现与原设计不一致，以代码为准同步修正文档。

### Task 7: 全量验证

**Files:**
- Verify only

- [x] 运行 `cd bots/koishi && corepack yarn build`
- [x] 运行 `cd bots/koishi && corepack yarn test`
- [x] 运行 `cd bots/koishi && NODE_ENV=development timeout 30s corepack yarn dev`
- [x] 确认控制台插件加载、端口固定在 `5140`、现有绑定/群审能力未回归。
