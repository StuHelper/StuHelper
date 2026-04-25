---
type: adr
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-04-25
---

# ADR-0006: 保留 stuhelper-core 作为唯一 Koishi WebUI 入口

**Date**: 2026-04-25
**Status**: accepted
**Implementation status**: accepted, not yet implemented. P1 of the execution plan will delete the unused packages.
**Deciders**: project owner（Xauryan）+ Claude/Codex 联合评审
**Supersedes**:
- [`docs/internal/superpowers/specs/2026-04-24-koishi-stuhelper-platform-redesign-design.md`](../internal/superpowers/specs/2026-04-24-koishi-stuhelper-platform-redesign-design.md)（提议把 `stuhelper-platform` 作为唯一入口）
- [`docs/internal/superpowers/plans/2026-04-24-koishi-stuhelper-platform-redesign.md`](../internal/superpowers/plans/2026-04-24-koishi-stuhelper-platform-redesign.md)（对应实施计划）

## Context

`bots/koishi/plugins/` 下当前并存三套 WebUI 实现：`stuhelper-core`、`stuhelper-console`、`stuhelper-platform`。三者都把页面注册到 `/stuhelper`，但 `koishi.yml` 当前只加载 `stuhelper-core`，所以**当前不构成运行时冲突；只要任意两者并行启用就会互相覆盖**。

完整事实证据（loaded 状态、各 client LOC、view 数量、私下装配链路）见执行计划 [§1 背景与现状](../internal/exec-plans/active/2026-04-25-koishi-plugin-restructure.md#1-背景与现状)。

四轮评审过程中曾提出过几个迁移方向：

1. 建立 `stuhelper-shell` 新入口，把 UI 子页拆分到多个插件贡献
2. 以 `stuhelper-console` 为下一代主 UI 替换 core
3. 保留 core 作为唯一 UI 入口，删除未加载的两套实验代码

最终选择方向 3。本 ADR 记录该决策。

## Decision

**保留 `stuhelper-core` 作为 Koishi 控制台唯一 WebUI 入口；删除 `stuhelper-console` 与 `stuhelper-platform`；后续模块化只针对 server 端。**

具体落实（详细阶段见执行计划 §4）：

1. UI 端不引入新 shell、不做"多插件贡献 UI 子页"的装配模式
2. `stuhelper-console/` 与 `stuhelper-platform/` 整目录删除，不抽取代码到 `packages/`
3. server 端的 god plugin 状态由 `stuhelper-core/src/index.ts` 内部拆分（7 档显式装配 + 模块 registry）解决
4. `binding` / `group-guard` / `admin` 三个独立插件最终回归 `koishi.yml` 显式加载，不再被 core 的 `legacy-wrapper.ts` 私下 `ctx.plugin()` 拉起

### 决策原则

本决策遵循的两条工程原则（用于在执行过程中判断边界情况）：

- **不引入兼容路径（no compatibility shims）**：迁移期不并存"新路径 + 老路径 + 运行时开关"。每一步迁移要么彻底切换、要么不做；回滚通过 `git revert` 完成，不通过环境变量在运行时选择路径。这避免"双装载导致重复命令注册"等真实风险。
- **保留产品事实源**：当前真实运行的 UI（11 个 view）是产品事实源，不能被功能子集替换。架构演进的代价不应由产品功能承担。

相关：项目工程原则总集见 [docs/design/core-beliefs.md](../design/core-beliefs.md)。

执行细节见 [exec-plans/active/2026-04-25-koishi-plugin-restructure.md](../internal/exec-plans/active/2026-04-25-koishi-plugin-restructure.md)。

## Alternatives Considered

### Alternative 1: stuhelper-shell + 多插件贡献 UI

新建 `stuhelper-shell` 插件作为 `/stuhelper` 唯一入口，11 个领域模块各自贡献 sidebar 项与子页，shell 装配。

- **Pros**：模块边界最清晰；理论上每个模块可独立启停 UI
- **Cons**：
  - 引入路由 / store / 样式 / 权限 / 加载顺序 / 版本契约的跨插件协调问题，复杂度本身堪比 IAM 迁移
  - 11 页控制台不需要"多插件贡献"的灵活性——它是单一产品，不是 marketplace
  - 工作量大、回归面广、收益与代价不对等
- **Why not**：架构复杂度匹配的是产品形态，而不是某种通用理想。当前是单一管控台，不是模块市场

### Alternative 2: 以 stuhelper-console 为下一代 UI

把 4 月新写的 `stuhelper-console` 作为重设计主线，逐步替换 core。这是 `superpowers/specs/2026-04-24-koishi-stuhelper-platform-redesign-design.md` 与对应 plan 的方向。

- **Pros**：console 代码更新；composable / model 拆分更现代；token 体系更系统
- **Cons**：
  - **`stuhelper-console` 是 core 的功能子集而非替代品**：仅覆盖 5 个分区（dashboard / enforcement / identity / policy / audit），缺失 chat / warns / blacklist / subscriptions / settings 等 6 个真实运行的 view
  - 切换会立即造成产品功能断档
  - 把 console 补齐到 11 个 view 的工作量本质上等于"重写"，并非"重设计"
- **Why not**：用功能子集替代功能全集是产品倒退，非工程优化

### Alternative 3: 保留 core，删除 console / platform，内部拆 server（采纳）

- **Pros**：
  - 零产品功能损失
  - 与本 ADR §Decision 的"不引入兼容路径"原则一致
  - 删除未加载实验代码后，新增的代码改动都直接作用于真实运行路径，不会再有"实验代码"与"运行代码"的认知分裂
  - server 端拆分可独立推进，与 UI 解耦
- **Cons**：
  - core/client 的 18K 行历史代码继续承载，短期内不会"焕然一新"
  - 部分 console 代码（设计 token、小型测试、实验性 module-registry 思路）会从 active 代码退到 git 历史
- **Why chosen**：cons 都是"美感与实验性"代价；pros 是"产品稳定性与工程聚焦"。前者远不抵后者

## Consequences

### Positive

- `bots/koishi/` 内只剩一套真实运行的 WebUI，新成员理解成本下降
- 后续重构所有改动都直接验证产品功能，没有"改了 console 但 console 不跑"的伪改进
- server 端拆分边界清晰：从"god plugin 拆出 7 档" + "BaseModule → Runtime registry"
- `binding` / `group-guard` / `admin` 显式装载后，三者真正回到独立 koishi 插件身份

### Negative

- `stuhelper-core/client/` 的 18K 行代码在可见未来不会被重写，只做"按需注册图标 / 删 dead code / 合并重复 CSS"等机械整理
- `stuhelper-console` 的 design token 与 model test 思路不会自动迁移到 core；需要时由人工对照 git 历史搬运
- core 的 `Config` schema 短期内仍带 `binding` / `guard` / `moderation` / `fun` / `ai` / `admin` 等历史子块，直到 P5/P6 完成才能收敛

### Risks

- **风险**：删除 console / platform 后发现某测试或脚本静默引用被删包
  - **缓解**：执行计划 P1 的退出标准要求 `yarn build && yarn test:unit && yarn test:startup && yarn test:ui` 全绿
- **风险**：保留 core/client 长期 = 容忍其内部复杂度
  - **缓解**：P2 阶段每个 PR 净删除 + 红线（不改语义、不改 API），逐步降低 LOC
- **风险**：未来某天确实需要"多插件贡献 UI"
  - **缓解**：本 ADR 不是永久决策；当真有此需要时再写新 ADR superseded 本条。当前阶段不为假想未来设计

## References

- 执行计划：[exec-plans/active/2026-04-25-koishi-plugin-restructure.md](../internal/exec-plans/active/2026-04-25-koishi-plugin-restructure.md)
- 工程原则：[docs/design/core-beliefs.md](../design/core-beliefs.md)
- 当前运行入口：`bots/koishi/koishi.yml`（`group:stuhelper:` 段）
- legacy 装配点：`bots/koishi/plugins/stuhelper-core/src/legacy/legacy-wrapper.ts`（`applyLegacyFeatures` 内的 `ctx.plugin(binding/groupGuard/admin)` 三处调用）
