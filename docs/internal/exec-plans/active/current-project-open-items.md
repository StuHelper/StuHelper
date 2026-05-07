---
type: internal
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-05-07
---

# 当前项目待办

本文件是执行计划的唯一活跃入口。历史计划、已完成阶段和已废弃方案不再用未勾选
checkbox 表示当前待办。

## 已确认活跃任务

当前没有从计划文档中确认出的活跃未完成开发任务。

## 待立项候选

下列内容来自历史计划的未完成描述，但没有被采纳为当前执行计划：

| 候选项 | 来源 | 当前状态 |
|--------|------|----------|
| Koishi 群管中心高阶运营工作流：更细粒度报表、历史版本、更复杂处置编排 | `docs/internal/exec-plans/archived/2026-04-19-koishi-moderation-center-implementation.md` | 待产品确认，不作为活跃任务 |
| Open Platform v1 前置条件 | `docs/design/open-platform-v1.md` | 依赖 IAM v2 决策，不作为当前执行计划 |

## 归档原则

- `docs/internal/exec-plans/active/` 只保留当前要推进的计划。
- 已完成计划进入 `docs/internal/exec-plans/completed/`。
- 被后续 ADR、设计或实现取代的计划进入 `docs/internal/exec-plans/archived/`
  或保留在 `docs/internal/design-snapshots/` 中并标记为历史快照。
- Runbook、QA checklist、发布检查表不是项目开发计划，不在本文件中跟踪。
