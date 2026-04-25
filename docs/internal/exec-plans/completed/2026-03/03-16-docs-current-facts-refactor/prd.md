---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# Refactor docs system around current facts

## Goal

重构 `docs/` 文档系统，让目录、索引、模块说明和技术说明都围绕当前代码库的真实结构组织。文档只描述当前事实，直接服务开发和维护。

## Requirements

- 按当前项目约定整理 `tutorials / guides / reference / architecture / modules` 这套文档骨架
- 让根索引和各子目录索引都能快速把读者带到正确文档
- 技术文档只描述当前代码、接口、数据结构和运行方式
- 删除或改写文档里的变更历史、迁移叙事、规划叙事和否定式表达
- 让模块文档和代码入口保持对齐，避免出现和当前实现脱节的说明
- 优化文档之间的链接关系，减少重复表述

## Acceptance Criteria

- `docs/README.md` 和各子目录 `README.md` 都能清楚说明自己的职责和入口
- 文档内容以当前代码库事实为准，不保留变更历史段落
- 文档内容不出现 `不是`、`不在`、`不再`、`尚未实现` 这类表述
- 规划性质或占位性质的技术文档被删除或改写成当前事实说明
- 模块、架构、参考文档与当前代码目录和契约保持一致

## Technical Notes

- 事实源优先级以 `server/api/openapi.yaml`、`server/scripts/init.sql`、当前代码和测试为准
- 当前 `docs/` 已有 34 篇 Markdown 文档，需先清理索引和风格，再处理模块与专题文档
- 需要同步更新 Trellis 记录，因为这是项目级文档系统调整
