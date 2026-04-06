# brainstorm: migrate project_rule into trellis

## Goal

删除 `.project_rule/` 目录，并把其中的项目规则、开发归档，以及你补充提供的旧 `CLAUDE.md` 用户级配置，迁移到 Trellis 体系内的合适位置。迁移完成后，项目的规范入口、历史记录入口、文档引用和自动化检查都应指向 `.trellis/`，避免继续依赖即将删除的旧目录。

## What I already know

* 当前 `.project_rule/` 下有两个核心文件：`project_rules.md` 和 `archiving.md`
* `project_rules.md` 既包含项目级规范，也混入了旧 `CLAUDE.md` 风格的用户偏好和工作方式约束
* `archiving.md` 承担了“历史修改记录 / 关键决策归档”的角色
* 你希望彻底删除 `.project_rule/`，而不是保留兼容壳
* 你还要求把旧 `CLAUDE.md` 的内容一并合并进 Trellis 体系
* `.trellis/spec/` 当前更像“项目规范 / 开发准则”区
* `.trellis/workspace/` 当前更像“开发者工作区 / 日志 / 会话记录”区
* 仓库中已经存在对 `.project_rule/` 的引用，至少包括：
  * `/Users/zxy/Code/StuHelper/README.md`
  * `/Users/zxy/Code/StuHelper/docs/README.md`
  * `/Users/zxy/Code/StuHelper/clients/web/src/utils/__tests__/staleArtifacts.test.ts`
* 当前仓库内的 `AGENTS.md` 主要是 Trellis managed block，本体并没有再显式引用 `.project_rule/`

## Assumptions (temporary)

* 项目级、可复用、面向所有协作者的规范，应该迁入 `.trellis/spec/`
* 历史归档应该迁入 Trellis 里的稳定位置，而不是继续放在开发者私有工作区子目录下
* 旧 `CLAUDE.md` 的“语言偏好 / 工具偏好 / 禁止事项 / 会话启动要求”应被拆分并合并到 Trellis 的长期规范文档，而不是原样堆叠
* 删除 `.project_rule/` 后，仓库内不应再有任何硬编码引用残留

## Open Questions

* 无。当前关键结构性选择已完成。

## Requirements (evolving)

* 删除 `.project_rule/` 目录
* 将 `project_rules.md` 的内容迁入现有 `.trellis/spec/` 文档体系，而不是新增独立规则文件
* 将 `archiving.md` 的内容迁入现有 `.trellis/workspace/` 的 journal 体系，而不是新增独立归档文件
* 将旧 `CLAUDE.md` 中仍然有效的用户级规范合并进新的 Trellis 文档
* 更新仓库内所有对 `.project_rule/` 的引用
* 迁移后的结构应便于后续会话启动时稳定读取
* 迁移方案应尽量降低规范分散和重复维护的风险
* 启动必读规则、语言偏好、工具偏好、归档要求等内容需要映射到现有 Trellis 文档的明确位置
* 项目历史归档需要被吸收到现有 `journal` 链路中
* 旧归档内容需要做去重、合并同类项、按 Trellis 的 journal 风格重写，而不是原样搬运
* `project_rules.md + CLAUDE.md` 的规则分布采用“入口集中 + 细则分流”
* “会话启动必读”和“代码变更后必须更新记录”这类强约束，由 `AGENTS.md` + `.trellis/workflow.md` 共同兜底，`.trellis/spec/guides/index.md` 提供导航说明

## Acceptance Criteria (evolving)

* [ ] 删除 `.project_rule/` 后，仓库内不存在对 `.project_rule/` 的有效引用
* [ ] 原 `project_rules.md` 和旧 `CLAUDE.md` 的有效规范都能在现有 `.trellis/spec/` 文档中找到归宿
* [ ] 原 `archiving.md` 的历史记录都能在精简改写后的 journal 文档中继续维护
* [ ] README、文档索引、测试或脚本中的相关路径都已更新
* [ ] 新的规范入口足够清晰，后续协作者知道“启动先读哪里、改完记到哪里”
* [ ] 项目级总入口、前端规则入口、后端规则入口之间的职责划分清晰，不互相打架

## Definition of Done (team quality bar)

* 文档与引用已同步更新
* 路径相关测试或校验已更新并通过
* 不新增重复规范源
* Trellis 结构中的职责边界清晰
* `.project_rule/` 可被安全删除且不会影响后续协作流程

## Technical Approach

1. 规则迁移

* 将项目级总入口、启动必读说明、跨层通用约定迁入 `.trellis/spec/guides/index.md`
* 将后端相关规则、命名/工具链/质量门槛迁入 `.trellis/spec/backend/index.md` 与 `.trellis/spec/backend/quality-guidelines.md`
* 将前端相关规则、命名/工具链/质量门槛迁入 `.trellis/spec/frontend/index.md` 与 `.trellis/spec/frontend/quality-guidelines.md`
* 将“首要指令”和“改动后必须更新记录”等强约束写入 `AGENTS.md` 与 `.trellis/workflow.md`

2. 归档迁移

* 将 `.project_rule/archiving.md` 精简、去重、合并同类项后，改写进 `.trellis/workspace/wztxy/journal-1.md`
* 保留关键决策、关键文件、原因和注意事项，不保留冗余或重复表述
* 在 `.trellis/workspace/index.md` 与 `.trellis/workspace/wztxy/index.md` 中补充说明，让协作者知道项目历史已并入 journal

3. 引用与校验迁移

* 更新 `README.md`、`docs/README.md`、相关测试中的 `.project_rule/` 路径
* 搜索全仓并清理对 `.project_rule/` 的剩余引用
* 删除 `.project_rule/` 目录

4. 结构原则

* 不新增 `project-rules.md` 或 `project-archiving.md`
* 采用“项目级入口集中、领域细则分流”的结构
* 旧归档进入 journal 时采用 Trellis 风格重写，而不是逐字保留

## Out of Scope (explicit)

* 不在这次迁移里重写全部 frontend/backend 细则内容
* 不在这次迁移里引入新的任务流框架
* 不在这次迁移里大规模调整 `.trellis/workspace/` 的开发者个人日志机制

## Research Notes

### 当前仓库结构与职责

* `.trellis/spec/` 适合放项目级、长期有效的规范
* `.trellis/workspace/` 适合放项目过程记录、工作日志、历史归档
* `.project_rule/project_rules.md` 当前是“项目规范 + 用户偏好 + 工作约束”的混合体
* `.project_rule/archiving.md` 当前是“项目历史归档”，语义上更接近 workspace/history，而不是 spec/rules

### Feasible approaches here

**Approach A: 保留两个专门的 Trellis 文档**

* How it works:
  * 新建 `.trellis/spec/project-rules.md`，吸收 `project_rules.md` 和旧 `CLAUDE.md` 中仍有效的规范
  * 新建 `.trellis/workspace/project-archiving.md`，完整迁移 `archiving.md`
  * 更新 `.trellis/workflow.md`、README、docs、测试中的引用，再删除 `.project_rule/`
* Pros:
  * 迁移边界清晰，实施风险低
  * 保留“规则”和“归档”两个稳定入口，后续容易让 agent 和记忆系统读取
  * 不会把现有 frontend/backend 规范文档搅乱
* Cons:
  * `.trellis/spec/` 和 `.trellis/workspace/` 中会各新增一个顶层文档

**Approach B: 完全融入现有 Trellis 文档** (Chosen)

* How it works:
  * 把 `project_rules.md` 拆散后并入现有 `frontend/`、`backend/`、`guides/`、`workflow.md`
  * 把 `archiving.md` 精简后并入开发者日志体系
* Pros:
  * 表面上更“原生 Trellis”
  * 顶层文件更少
* Cons:
  * 迁移成本和风险最高
  * 容易让“启动必读规范”和“历史归档”失去单独入口
  * 会把原本清晰的历史记录掺进工作日志

**Approach C: 混合方案**

* How it works:
  * 新建 `.trellis/spec/project-rules.md` 承接规则
  * 把 `archiving.md` 合并进 `.trellis/workspace/index.md` 或新增一个 workspace history 区块
* Pros:
  * 规则入口清晰
  * workspace 不增加太多独立文件
* Cons:
  * 归档和 workspace 索引混在一起，长期可读性一般
  * 如果归档继续增长，`workspace/index.md` 会越来越臃肿

## Technical Notes

* 已检查文件：
  * `/Users/zxy/Code/StuHelper/.project_rule/project_rules.md`
  * `/Users/zxy/Code/StuHelper/.project_rule/archiving.md`
  * `/Users/zxy/Code/StuHelper/.trellis/workflow.md`
  * `/Users/zxy/Code/StuHelper/.trellis/workspace/index.md`
  * `/Users/zxy/Code/StuHelper/.trellis/workspace/wztxy/index.md`
  * `/Users/zxy/Code/StuHelper/AGENTS.md`
  * `/Users/zxy/Code/StuHelper/README.md`
  * `/Users/zxy/Code/StuHelper/docs/README.md`
  * `/Users/zxy/Code/StuHelper/clients/web/src/utils/__tests__/staleArtifacts.test.ts`
* 当前已知后续需要同步更新的引用包括 README、docs 索引、测试中的路径断言
* 如果最终选择删除 `.project_rule/`，后续还需要把“每次会话先读哪份文档”的约定迁入 Trellis 启动入口或规范索引中
* 当前 `journal-1.md` 仅包含文件头，尚未写入实际会话内容，因此本次迁移不会和既有长日志结构冲突
* 现有 `backend/index.md`、`frontend/index.md` 更像入口导航；`quality-guidelines.md` 更适合承接“禁止事项 / 工具链 / 质量门槛”；`guides/index.md` 更适合承接项目级思维提醒和通用入口说明
* 已确认规则落点采用：
  * `guides/index.md`：项目级总入口、启动必读、跨层通用约定、规则导航
  * `backend/index.md` + `backend/quality-guidelines.md`：后端规则、工具链、质量门槛
  * `frontend/index.md` + `frontend/quality-guidelines.md`：前端规则、工具链、质量门槛

## Decision (ADR-lite)

**Context**: 需要在删除 `.project_rule/` 的同时，把规则和归档完全迁入 Trellis 体系，且你明确选择“不保留两个专门的新文件”。

**Decision**: 采用“完全融入现有 Trellis 文档结构”的方案。项目规则将拆分后合并进现有 `.trellis/spec/` 与相关说明文档的合适位置；项目归档将经过精简、去重和 Trellis 风格重写后，并入现有 `.trellis/workspace/` 的 journal 体系，而不是新建独立归档文件。

**Consequences**:
* 需要更细致地设计内容落点，避免规则分散和难以检索
* 需要同步修改 README、docs、测试与启动入口，确保引用一致
* 相比专门文件方案，后续更依赖清晰的目录说明和索引更新
* 项目级归档会与开发者会话记录共享载体，需要额外约束格式和入口说明
* 旧归档将不再逐字保留，需要明确“合并同类项”与“保留关键信息”的边界
* 规则文档将采用“项目级入口集中、领域细则分流”的结构，减少新增文件同时保持可读性
* 关键流程约束会双写入 `AGENTS.md` 和 `.trellis/workflow.md`，以提升 agent 遵循率
