---
type: design
audience: maintainers, backend-dev, frontend-dev
status: current
authoritative-source: this file + scripts/check-docs-hygiene.mjs
last-verified: 2026-05-07
---

# 文档治理模型

`docs/` 是 StuHelper 的长期文档系统，不是随手堆放 Markdown 的目录。它的目标只有三个：

1. 让读者能快速找到对自己有用的内容。
2. 让文档与代码、契约、迁移保持单一真源关系。
3. 让文档质量尽可能被脚本和 CI 机械化约束，而不是依赖口头约定。

## 目录职责

| 目录 | 文档类型 | 作用 | 不应该放什么 |
|------|----------|------|--------------|
| `guides/` | guide | 操作指南，回答“怎么做” | 设计取舍、历史复盘 |
| `design/` | design | 架构解释，回答“为什么这样设计” | 逐步操作手册、全量事实清单 |
| `product-specs/` | product-spec | 业务边界、规则、范围 | 具体技术机制实现 |
| `reference/` | reference | 导航摘要、稳定查阅入口 | 手写真源副本 |
| `adr/` | adr | 已采纳、代价高、难回退的单项决策 | 实现复盘、变更日志 |
| `internal/` | internal | 临时工件、历史设计快照、阶段性评估、执行计划 | 现行规范与长期说明 |

## 长期文档与临时工件

- `docs/` 根目录下除 `internal/` 以外的内容，都应反映当前代码库的**现行状态**。
- `docs/internal/` 记录的是某一轮执行中的输入、输出、历史设计快照或评估快照，可以过时。
- 历史计划、历史审计、阶段评分不应回流到长期文档目录。

## Frontmatter 规则

所有 `docs/**/*.md` 都必须带 YAML frontmatter，字段固定为：

| 字段 | 含义 | 规则 |
|------|------|------|
| `type` | 文档类型 | 必须与所在目录匹配 |
| `audience` | 目标读者 | 逗号分隔，使用受控值 |
| `status` | 生命周期状态 | 长期文档只允许 `current / draft / deprecated`；内部文档允许 `current / snapshot / archived` |
| `authoritative-source` | 真源 | 可以是当前文件，也可以是源码路径 |
| `last-verified` | 最近与代码核对日期 | `YYYY-MM-DD` |

受控 `audience` 取值：

- `all`
- `backend-dev`
- `frontend-dev`
- `ops`
- `product`
- `qa`
- `maintainers`

受控例外：

- `docs/README.md` 是根索引，`type` 固定为 `reference`
- `docs/adr/README.md` 与 `docs/adr/template.md` 是 ADR 目录配套材料，`type` 固定为 `reference`

## 真源规则

| 事实 | 唯一真源 | 文档职责 |
|------|----------|----------|
| API 契约 | `server/api/openapi.yaml` | 解释模块边界，提供导航入口 |
| 数据库 schema | `server/migrations/` | 解释数据面、模块归属、跨表约束 |
| 能力常量 | `server/internal/pkg/capability/` | 解释授权模型，不复制常量全集 |
| 资源关系模型 | `docs/design/openfga-model.fga` | 作为设计资产保存，并由设计文档解释其含义 |
| 运行时行为 | 源代码与测试 | 文档只解释，不替代测试 |

## Reference 约束

`reference/` 只做“查到入口”，不做“复制真源”。

- 可以写模块分组、目录索引、查找指引。
- 可以写跨模块的稳定规则，例如错误码分类。
- 不维护完整接口表、完整字段表、完整表结构清单。
- API 概要必须服从 `cd server && make check-doc-sync`。

## ADR 约束

只有满足以下条件的决策才应进入 `adr/`：

- 已经采纳，而不是讨论中。
- 影响跨模块协作、维护成本或回滚成本。
- 如果没有文档记录，后续团队会反复争论“为什么当初这样选”。

ADR 不承担以下职责：

- 代替设计文档描述全局架构。
- 记录一次普通重构的变更历史。
- 充当 release note 或审计清单。

## 非 Markdown 例外

长期文档默认只接受 Markdown。确实属于设计真源的结构化资产，可以作为**显式白名单**与文档并存。

当前白名单：

- `docs/design/openfga-model.fga`

新增非 Markdown 长期资产前，应先确认它确实是“设计真源”，而不是生成物、截图或临时草稿。

## 机械化守卫

文档治理不是人工承诺，必须能跑：

- `make check-docs`
  检查 frontmatter、目录归属、绝对路径、retired 路径、相对链接、长期资产白名单。
- `cd server && make check-doc-sync`
  检查 `docs/reference/api-overview.md` 的模块前缀仍覆盖当前 OpenAPI。
- GitLab CI 在 lint 阶段执行同一套规则。

## 变更原则

- 改代码事实，优先改真源，再改文档。
- 改长期文档时，只改当前状态；历史说明放进 `internal/`。
- 一份文档只回答一种问题，避免 “Guide + Reference” 或 “Spec + ADR” 混写。
