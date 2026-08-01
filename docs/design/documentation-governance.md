---
type: design
audience: all
status: current
authoritative-source: scripts/lib/docs-hygiene-lib.mjs + scripts/check-docs-hygiene.mjs
last-verified: 2026-08-01
---

# 文档治理模型

本文是 StuHelper 文档结构和治理规则的唯一说明。具体操作步骤见
[文档维护指南](../guides/documentation-maintenance.md)，机械校验以
`scripts/lib/docs-hygiene-lib.mjs` 为准。

## 1. 目标与真源

文档系统只承担三项职责：

1. 帮助读者找到正确入口；
2. 解释代码、契约和数据模型之间的边界；
3. 让结构、元数据和链接错误能被 CI 发现。

同一事实只能有一个权威来源：

| 事实 | 唯一真源 | 文档职责 |
|------|----------|----------|
| API operation、字段和响应 | `server/api/openapi.yaml` | 解释模块边界并提供导航 |
| 数据库表、列、约束和索引 | `server/migrations/` | 解释数据归属和跨表不变量 |
| Capability 常量 | `server/internal/pkg/capability/` | 解释授权层次，不复制全集 |
| OpenFGA 关系模型 | `docs/design/openfga-model.fga` | 保存模型资产并解释关系语义 |
| 运行时行为 | 源代码和测试 | 描述稳定机制，不替代回归测试 |

当文档和真源冲突时，以真源为准，并在同一变更中修正文档。不要通过新增另一份说明来
“兼容”冲突。

## 2. 目录职责

长期文档采用 Diátaxis 的职责划分；历史工件单独隔离：

| 位置 | 类型 | 回答的问题 | 不应承载的内容 |
|------|------|------------|----------------|
| `docs/QUICKSTART.md` | guide | 第一次如何跑起来 | 架构历史和全量参考表 |
| `docs/guides/` | guide | 如何完成具体任务 | 设计取舍和阶段评分 |
| `docs/design/` | design | 系统为何这样设计 | 执行计划、待办、审计流水 |
| `docs/product-specs/` | product-spec | 产品行为和验收规则是什么 | 具体实现细节 |
| `docs/reference/` | reference | 去哪里查稳定事实 | 手抄 OpenAPI、schema 或常量全集 |
| `docs/adr/` | adr | 一项高代价决策为何被采纳 | 全局架构说明和实施日志 |
| `docs/internal/` | internal | 某一阶段发生了什么 | 现行规范和长期事实 |

`docs/internal/` 可以保存历史计划、设计快照和阶段性评估，但它不是当前实现的事实来源。
审计台账属于阶段性工作产物，保留在仓库根目录或 Issue/PR 中，不进入长期文档导航。

## 3. 一文档一职责

- 一份文档只回答一种问题；同时承担 guide、reference 和设计说明时应拆分。
- 同一主题只能有一份当前态设计文档。旧文件被新文件替代时，应在同一变更中更新链接并删除
  旧文件；不要长期保留两份都声称 `status: current` 的副本。
- 设计文档解释系统级机制，产品规格固定业务行为，ADR 记录单项决策理由，guide 只写操作步骤。
- 当前态文档使用现在时。执行轨迹、修复数量、阶段结论和未完成清单进入审计台账、Issue、PR
  或 `docs/internal/`。

## 4. Frontmatter 契约

所有 `docs/**/*.md` 必须以 YAML frontmatter 开头，并包含：

```yaml
---
type: guide | design | product-spec | reference | adr | internal
audience: all | backend-dev | frontend-dev | ops | product | qa | maintainers
status: current | draft | deprecated
authoritative-source: <仓库相对路径或 this file>
last-verified: YYYY-MM-DD
---
```

规则如下：

- `type` 必须与目录匹配；`docs/README.md`、`docs/adr/README.md` 和 ADR 模板使用
  `reference`。
- `audience` 可以用逗号组合受控值。
- 长期文档状态只能是 `current`、`draft`、`deprecated`。
- `docs/internal/` 状态可以是 `current`、`snapshot`、`archived`。
- `authoritative-source` 必须指向实际核对的来源；组合来源使用清晰的仓库相对路径。
- `last-verified` 只在实际对照真源后更新，不能把编辑日期冒充核对日期。

## 5. ADR 边界

只有同时满足以下条件的决策进入 `docs/adr/`：

- 已被项目 owner 采纳；
- 跨模块、维护或回滚成本较高；
- 后续维护者需要理解当时为何选择该方案。

ADR 记录上下文、决策、后果和备选方案，不承担全局架构说明、发布记录或任务清单职责。方向改变时
新增 ADR，并在索引中标明取代关系；不要让两条 ADR 同时声称互斥方案均为当前权威。

## 6. Reference 与生成物

`reference/` 只提供模块分组、稳定规则和真源入口：

- 不维护完整 API 路径和字段副本；
- 不维护完整数据库结构副本；
- 不维护 Capability 常量全集；
- OpenAPI 变更后运行 `cd server && make check-doc-sync`。

生成代码和生成契约不能手改。接口变化必须先修改 OpenAPI，再运行生成命令。

## 7. 非 Markdown 资产

长期文档默认只接受 Markdown。当前唯一长期设计资产白名单是：

- `docs/design/openfga-model.fga`

`docs/internal/` 可以保存阶段性非 Markdown 工件，但不得被长期文档当作当前真源。

## 8. CI 当前实际执行的守卫

`make check-docs` 执行测试和文档树校验，当前覆盖：

- 目录布局和 retired path；
- frontmatter 必填字段、类型、受众、状态和日期格式；
- 长期文档相对链接；
- 长期文档中的本机绝对路径；
- 长期非 Markdown 资产白名单；
- `.DS_Store` 仓库卫生。

这些是当前代码真实执行的规则。时态、内容重复和 `last-verified` 对应源码提交时间仍需评审者
人工核对；在脚本真正实现前，不得把它们描述成 CI 已强制的能力。

## 9. 变更流程

1. 先改代码、OpenAPI、migration 或其他事实真源。
2. 找到唯一负责解释该事实的当前态文档并更新；不要新增平行副本。
3. 更新索引和所有旧链接。
4. 实际核对后更新 `last-verified`。
5. 运行 `make check-docs`；涉及 API 摘要时再运行 `cd server && make check-doc-sync`，涉及
   OpenAPI 时运行 `cd server && make lint-spec` 和生成漂移检查。

详细命令和常见错误见[文档维护指南](../guides/documentation-maintenance.md)。
