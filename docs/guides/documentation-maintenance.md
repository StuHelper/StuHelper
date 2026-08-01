---
type: guide
audience: maintainers, backend-dev, frontend-dev
status: current
authoritative-source: docs/design/documentation-governance.md + scripts/lib/docs-hygiene-lib.mjs
last-verified: 2026-08-01
---

# 文档维护

本文回答的是“当你要新增或修改项目文档时，应该怎么做”。

## 1. 先判断文档应该放哪

| 你要回答的问题 | 去哪里 |
|----------------|--------|
| 如何完成某个操作 | `docs/guides/` |
| 为什么采用某种架构或机制 | `docs/design/` |
| 某个业务域做什么、不做什么 | `docs/product-specs/` |
| 从哪里查稳定事实入口 | `docs/reference/` |
| 某项已采纳架构决策为何如此 | `docs/adr/` |
| 历史计划、评分、设计草案、被替代方案、阶段快照 | `docs/internal/` |
| 当前审计/修复台账 | GitHub Issue / PR；确需随工作树维护时放仓库根目录，收口后由 Git 历史保存 |

如果一份文档同时想回答两类问题，先拆，再写。

## 2. 填写 frontmatter

新建 Markdown 文档时，先复制下面的模板，再写正文：

```yaml
---
type: guide | design | product-spec | reference | adr | internal
audience: backend-dev, frontend-dev
status: current
authoritative-source: this file
last-verified: 2026-04-19
---
```

规则：

- `type` 必须和目录匹配。
- `audience` 只能使用受控值。
- 长期文档不要使用 `snapshot` 或 `archived`。
- `last-verified` 写最近一次和代码核对的日期，不写预计日期。

## 3. 长期文档只写当前状态

- 当前代码、当前流程、当前边界，写进长期文档。
- 历史背景、执行轨迹、阶段评分和被替代设计草案，写进 `docs/internal/`。
- 当前审计台账不进入长期文档导航；使用单一根目录文件时，不再保留第二份同主题报告。
- 不要在长期文档里保留“当时是这样，后来改成那样”的迁移叙事。

## 4. 不要复制真源

下面这些内容不能在人手文档里维护第二份完整副本：

- API 全量路径、参数、字段：去 `server/api/openapi.yaml`
- 数据库全量表结构：去 `server/migrations/`
- capability 常量全集：去 `server/internal/pkg/capability/`

文档应该做的是：

- 解释模块边界
- 给出查找入口
- 说明跨模块约束

## 5. 写完后更新导航

新增长期文档后，至少检查下面三个入口是否需要更新：

- `docs/README.md`
- 同目录索引页，例如 `docs/product-specs/index.md`、`docs/adr/README.md`
- `AGENTS.md` 中的文档导航

## 6. 运行校验

提交前至少执行：

```bash
make check-docs
```

如果你修改了 API 模块摘要，还要执行：

```bash
cd server && make check-doc-sync
```

如果你修改了 OpenAPI 或引用它的文档链路，再执行：

```bash
cd server && make lint-spec
```

## 7. 常见错误

- 把产品规格写成实现说明。
- 在 `reference/` 里手抄一份完整真源。
- 删除旧目录后，没有修正文档链接。
- 给长期文档写“阶段性评分”或“本轮计划”。
- frontmatter 和正文自述不一致。

完整规则见 [../design/documentation-governance.md](../design/documentation-governance.md)。
