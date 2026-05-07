---
type: internal
audience: maintainers, backend-dev
status: current
authoritative-source: this file
last-verified: 2026-05-07
---

# Koishi 规划快照

本目录保存 Koishi 子系统的**规划草案与实现计划快照**。它们用于记录设计思路和未来目标，不代表当前仓库已经具备这些能力。

## 当前真实实现看哪里

- [../../guides/koishi-development.md](../../guides/koishi-development.md)
- [`bots/koishi/`](../../../bots/koishi/)
- `cd bots/koishi && corepack yarn test`

## 当前目录

| 位置 | 含义 | 状态 |
|------|------|------|
| [plans/2026-04-19-koishi-plugin-framework.md](plans/2026-04-19-koishi-plugin-framework.md) | 已完成的工作区初始化计划 | 已归档 |
| [specs/2026-04-19-koishi-plugin-framework-design.md](specs/2026-04-19-koishi-plugin-framework-design.md) | 已归档的框架设计稿 | 已归档 |
| [plans/2026-04-19-koishi-moderation-center-implementation.md](plans/2026-04-19-koishi-moderation-center-implementation.md) | 群管中心实现快照 | 已归档；高阶运营工作流仅作为待立项候选 |
| [specs/2026-04-19-koishi-moderation-center-design.md](specs/2026-04-19-koishi-moderation-center-design.md) | 群管中心目标设计 | 部分能力已实现，仍保留未来目标 |
| [plans/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration.md](plans/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration.md) | `stuhelper-core` 兼容能力并入计划快照 | 已归档 |
| [reports/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration-summary.md](reports/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration-summary.md) | 上述并入工作的阶段总结 | 已归档 |
| [specs/2026-04-20-koishi-console-ui-redesign-design.md](specs/2026-04-20-koishi-console-ui-redesign-design.md) | 控制台 UI 重构设计快照 | 已归档 |
| [specs/2026-04-21-koishi-console-ui-redesign-completion.md](specs/2026-04-21-koishi-console-ui-redesign-completion.md) | 控制台 UI 重构完成总结 | 已归档 |
| [specs/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration-design.md](specs/2026-04-21-stuhelper-core-legacy-capabilities-ui-integration-design.md) | `stuhelper-core` 兼容能力并入设计快照 | 已归档 |
| [specs/2026-04-21-grouphelper-stuhelper-migration-design.md](specs/2026-04-21-grouphelper-stuhelper-migration-design.md) | grouphelper 迁移方向设计快照 | 仍属未来目标 |

## 当前待办入口

当前活跃执行计划与待立项候选统一查看 [../exec-plans/active/current-project-open-items.md](../exec-plans/active/current-project-open-items.md)。

## 阅读提醒

- 当前代码里已经落地的是：QQ 绑定、入群准入、关键词处理、撤回留痕、举报命令、娱乐命令、完整的 `群审*` 文本命令、`stuhelper-core` 自定义 Koishi Console 工作台、群模板/群绑定面板、SQLite 留痕与自动化测试。
- 仍属于未来候选的是：更多运营面板、更细粒度报表和更重型的群治理流程；这些不是当前活跃执行任务。
