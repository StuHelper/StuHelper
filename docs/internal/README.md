---
type: internal
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-05-07
---

# 内部工件索引

`docs/internal/` 只放**临时工件和阶段性快照**，不作为当前代码库的长期规范来源。

这里的文档可以过时，但必须说明自己是：

- 正在执行或已归档的计划
- 某个时间点的评估结论
- 尚未落地的未来设计草案
- 演练记录与过程材料

## 当前目录

| 目录 / 文件 | 用途 | 是否代表当前实现 |
|-------------|------|------------------|
| [exec-plans/README.md](exec-plans/README.md) | 执行计划、历史审计、闭环记录 | 否 |
| [design-snapshots/](design-snapshots/) | 历史设计草案、被替代方案和未来候选快照 | 否 |
| [release-readiness.md](release-readiness.md) | 阶段性发布准备度评分 | 否 |
| [drill-logs/README.md](drill-logs/README.md) | 备份恢复演练记录入口 | 否 |

## 使用规则

- 要看**当前代码事实**：回到 `docs/guides/`、`docs/design/`、`docs/product-specs/`、`docs/reference/`
- 要看**历史过程**：在本目录查计划、审计和评估
- 要看**未来目标但尚未落地的 Koishi 方案**：只看 `design-snapshots/koishi/`，不要把其中内容当成现状
