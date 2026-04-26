---
type: internal
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-04-25
---

# 2026-04-25 Koishi 插件重构执行计划

> 目的：把 `bots/koishi/` 当前的"god plugin + 三套并存 WebUI"格局，收敛为"保留唯一真实运行 UI + 内部模块化 server"。
> 范围：仅限 `bots/koishi/`；不动 `server/`、`clients/`、`infra/`。
> 关联决策：[adr/0006-koishi-core-ui-as-single-webui-entry.md](../../../adr/0006-koishi-core-ui-as-single-webui-entry.md)

## 1. 背景与基线

### 1.1 P1 前基线：三套插件代码并存

下表记录 P1 清理前的仓库状态，作为删除 `stuhelper-console` / `stuhelper-platform` 的决策依据；P1 完成后这两个实验包不再保留在 `bots/koishi/plugins/`。

| 插件 | 路径 | LOC（client，非测试） | koishi.yml 是否加载 | 角色 |
|------|------|---------------------|---------------------|------|
| `stuhelper-core` | `plugins/stuhelper-core/` | 18 226 | 是 | 当前唯一运行入口；承载 11 个 view 的 WebUI 与 22 个内部模块 |
| `stuhelper-console` | `plugins/stuhelper-console/` | 6 163 | 否 | 4 月新写的重设计实验，覆盖 5 个分区子集，**功能子集而非替代品** |
| `stuhelper-platform` | `plugins/stuhelper-platform/` | 804 | 否 | 实验性平台化抽象（module registry / runtime） |

证据：
- `bots/koishi/koishi.yml` 的 `group:stuhelper:` 段只列 `stuhelper-core`
- `bots/koishi/plugins/stuhelper-core/src/legacy/legacy-wrapper.ts` 的 `applyLegacyFeatures` 函数通过三处 `ctx.plugin()` 私下拉起 `binding`、`group-guard`、`admin`
- `bots/koishi/plugins/stuhelper-core/client/composables/use-console-pages.ts` 的 `VIEW_COMPONENTS` 映射列出当前真实 11 个 view：dashboard / config / warns / blacklist / identity / review / roles / logs / chat / subscriptions / settings
- `stuhelper-console` 仅覆盖 dashboard / enforcement / identity / policy / audit 5 个分区，缺失 chat / warns / blacklist / subscriptions / settings 等真实功能

### 1.2 P1 前风险：三个插件都注册 `/stuhelper`

- `stuhelper-core/client/index.ts` 的 `ctx.page({ path: '/stuhelper', authority: 4, ... })`
- `stuhelper-console/client/index.ts` 的 `ctx.page({ path: '/stuhelper', ... })`（无 authority）
- `stuhelper-platform/client/index.ts` 的 `ctx.page({ path: '/stuhelper', ... })`（无 authority）

当前因为 `koishi.yml` 只加载 core，不构成运行时冲突；但只要任意两个并存就会互相覆盖。

### 1.3 server 端 god plugin

`stuhelper-core/src/index.ts` 的 `apply()` 函数同时承担：

1. `StuhelperGroupCenterService` 注册
2. Console UI entry
3. Console API（websocket / page-api / review-action / governance-action）
4. 22 个 `MODULE_CLASSES` 命令式实例化
5. `applyLegacyFeatures(ctx, config)` 私下 `ctx.plugin(binding, ...)`、`ctx.plugin(groupGuard, ...)`、`ctx.plugin(admin, ...)`
6. `validateConsoleAdminPassword(...)` 启动期 ENV 校验
7. `registerReviewClaimRecovery(...)` 启动期数据 recovery（含 interval + dispose）

## 2. 决策摘要

### 2.1 保留 `stuhelper-core` 作为唯一 WebUI 入口

理由见 [ADR-0006](../../../adr/0006-koishi-core-ui-as-single-webui-entry.md)。要点：

- core 的 11 个 view 是产品事实源
- console / platform 是功能子集，重写 UI 等于产品倒退
- core 的 client 已有 `composables/` + `models/` + `components/` 文件级解耦
- 真正未模块化的是 server 端

### 2.2 拆 server，不拆 UI

- UI 端不引入新的"shell + module 装配"模式
- 不做"多插件贡献 UI 子页"
- server 端把 god plugin 拆成 7 档显式装配 + 内部模块 registry
- `binding` / `group-guard` / `admin` 三个独立插件最终回归 `koishi.yml` 显式加载，不再被 core 私下拉起

## 3. 不做的事情（明确边界）

下列内容曾在评审过程中讨论但**本计划明确不做**：

| 不做 | 原因 |
|------|------|
| 建立 `stuhelper-shell` 新入口插件 | core UI 已是真实 11 页控制台，新建 shell 是重复造轮子 |
| 把 22 个内部模块拆成 22 个 koishi workspace 包 | 模块间互相依赖（多数依赖 `ModerationStore`），强行拆包会引入循环依赖 |
| 让 UI 子页来自不同插件贡献 | Koishi 多 page 模型代价远超收益，单 UI 入口对一个 11 页控制台是正解 |
| 把 console / platform 的代码捞成 `packages/` 库 | 死代码换目录是 anti-pattern；需要时从 git 历史参考即可 |
| 引入运行时兼容开关（如 `STUHELPER_LEGACY_PLUGINS=1`） | 与 [ADR-0006](../../../adr/0006-koishi-core-ui-as-single-webui-entry.md) §Decision 中"不引入兼容路径"原则冲突；双装载会导致重复命令注册 |
| 提交 HAR 录制作为 API 基线 | 含 cookie / 时间戳 / 噪声，泄漏与脆弱；改用 server 端 listener 单测扩充 DTO 契约 |
| 提交 11 张视觉回归截图作为基线 | 视觉回归在当前阶段成本高、易脆；改用最小 UI smoke |
| 把 knip / ts-prune 写成现成门禁 | 仓库当前未配置；引入需独立工具链 PR |
| 强制每阶段打 git tag | 干净 PR 边界已能定位"上一个稳定状态" |

## 4. 阶段计划

### P0a Playwright 基础设施

#### 目标
建立 Playwright 测试基础设施，验证"启动 koishi → 浏览器访问 → spec 跑完 → 关闭"全链路工作；不涉及业务页面登录与 SPA 路由。

#### 工作内容

1. 新增 `bots/koishi/playwright.config.ts`，与 `clients/` 配置完全隔离（独立 testDir、独立浏览器存储）
2. 新增 `bots/koishi/e2e/smoke.spec.ts`：最小 spec，访问 Koishi Console 首页根路径，断言页面 title 包含 "Koishi"，无 `pageerror`
3. 新增 `bots/koishi/scripts/ui-smoke.mjs`：复用 `startup-smoke.mjs` 的临时配置 + 端口释放 + spawn 模式；先执行 `yarn build` 刷新 production `dist/`，再启动 koishi 并等待 console 监听就绪，spawn `playwright test`，结束后 SIGTERM 关闭 koishi 进程组
4. 在 `bots/koishi/package.json` 增加 `"@playwright/test": "latest"` 到 `devDependencies`（与 `clients/` 完全独立，不通过 monorepo hoist）
5. 在 `bots/koishi/package.json` 增加 `"test:ui": "node scripts/ui-smoke.mjs"` script
6. CI 入口 `yarn test` 改为 `yarn build && yarn test:unit && yarn test:startup && yarn test:ui`

#### 退出标准

- `corepack yarn install` 干净，新依赖落入 `bots/koishi/yarn.lock`
- `corepack yarn test:ui` 在本地稳定通过 3 次
- 关闭 koishi 进程组干净（无端口残留）
- 所有现有 `test:unit`、`test:startup` 仍通过

#### 风险与回滚

- Playwright 浏览器下载阻塞 CI → 在 P0a PR 描述中说明首次运行需 `npx playwright install chromium`
- Koishi Console 启动时 WebSocket 异步连接 → `ui-smoke.mjs` 等待 stdout 中"server listening"日志后才启动 spec
- `test:ui` 使用 production browser entry，`dist/` 是 ignored 产物 → `ui-smoke.mjs` 启动前强制 `yarn build`，避免源码与浏览器执行 bundle 不同步
- 回滚：单 PR revert

---

### P0b 登录 fixture + 11 view smoke

#### 目标
在 P0a 基础设施之上，建立 11 个 view 的端到端导航回归基线，作为 P1-P6 的护栏。

#### 工作内容

1. 新增 `bots/koishi/e2e/fixtures/auth.ts`：worker-scoped 登录 fixture，访问 Koishi Console 登录页 → 输入 admin 用户名 + ENV 注入的 `STUHELPER_CONSOLE_ADMIN_PASSWORD` → 提交 → 等待跳转完成 → 等待 Koishi 侧边栏 `/stuhelper` entry 出现并通过 router link warm-up 页面
2. 新增 `bots/koishi/e2e/stuhelper-views.spec.ts`：
   - 使用 `auth.ts` 登录 fixture 共享已登录的 Koishi Console page
   - 通过 TopNavigation 依次点击 11 个 view（dashboard / config / warns / blacklist / identity / review / roles / logs / chat / subscriptions / settings），不通过 `page.goto` 重载 SPA
   - 每个 view 断言：URL hash 包含目标 view id、view-specific anchor 出现、`pageerror` 与 console error/warning 均无未放行输出
3. 删除 P0a 阶段的 `e2e/smoke.spec.ts` 临时 spec（被业务 spec 替代）

#### 退出标准

- `corepack yarn test:ui` 在本地**严格 12/12 通过 10 次连续**（加固原因见下）
- 11 个 view 全部可达且无前端错误（pageerror + console.error/warning，allowlist 必须窄范围显式声明）
- 至少 2 个**负面用例**验证断言敏感度：
  - 故意改 anchor 文本 → spec 必须 fail
  - 故意改 view ID（让 nav click 找不到对应 button）→ spec 必须 fail
- CI 跑通

> **加固说明**（2026-04-25）：P0b 第一版用"3 次连续通过"作为稳定标准，被 codex 复核
> 在 fresh checkout 重跑时打出 40% flake（每次 fail 的 view 不同——SPA 重载竞态）。
> 原因是每个 test 各自 `page.goto` 触发 page reload，丢失 fixture 的 SPA warm-up，
> Koishi `ctx.console.addEntry` 异步注入路由的竞态再次出现。修法：所有 test 共享
> worker-scope page，通过 nav button click 在 SPA 内 `pushState` 切 view，不再
> 重载。同时 per-view 断言加 view-specific anchor（防 dashboard fallback 静默通过）、
> URL 断言（防 nav click 失败）、console.error/warning 监听（抓 Vue Router 警告
> 等非 pageerror 信号）。

#### 风险与回滚

- Koishi auth 登录是 WebSocket socket event 而非传统 HTTP cookie → 优先走 UI 输入登录路径；如 SPA 登录页 selector 不稳定，退路是 `page.evaluate()` 直接发 `login/password` socket event
- SPA 路由切换异步加载组件导致断言提前 → 用 `expect(locator).toBeVisible()` 等待，禁用 hard `waitForTimeout`
- 多 test 共享 page 的副作用累积（如 console listener、状态污染）→ tracker 用 `await using` 在 test 退出时显式 dispose 监听
- 回滚：单 PR revert（保留 P0a 基础设施）

---

### P1 删除未启用 UI 包

#### 目标
仓库内只保留真实运行的 WebUI 代码。

#### 工作内容

1. 删除 `bots/koishi/plugins/stuhelper-console/` 整个目录
2. 删除 `bots/koishi/plugins/stuhelper-platform/` 整个目录
3. 清理引用：
   - `bots/koishi/plugins/stuhelper-core/package.json` 的 `koishi-plugin-stuhelper-console` 依赖项（如存在）
   - `bots/koishi/plugins/stuhelper-core/src/runtime-contract.test.ts` 中关于 console package runtime entry 的断言
   - `bots/koishi/tsconfig.json` 中 `koishi-plugin-stuhelper-platform` 与 `koishi-plugin-stuhelper-platform/*` 两条 path 映射
   - `bots/koishi/yakumo.yml` 中相关引用
   - `yarn.lock` 重新生成
4. 更新 `bots/koishi/README.md`：删除 `stuhelper-console`、`stuhelper-platform` 两条说明
5. 更新 `docs/guides/koishi-development.md` 表格：删除 `stuhelper-console`、`stuhelper-platform` 两行

#### 退出标准

- `yarn install` 干净
- `yarn build` 通过
- `yarn test:unit` 通过
- `yarn test:startup` 通过（仍只加载 core）
- `yarn test:ui` 11 个 view 全部通过
- `git grep -r "stuhelper-console\|stuhelper-platform"` 在 `bots/koishi/` 下零结果（除归档文档）

#### 风险与回滚

- 隐藏依赖：某测试或脚本静默引用了被删包 → 退出标准的 build/test 全绿覆盖
- 回滚：revert PR

---

### P2 core UI 内部清理

#### 目标
不改语义、不改 API、不改样式语境，只做证据充分的删除与机械整理。

#### 工作内容

每条作为**独立 PR**，禁止混合：

1. **PR-2a 按需注册 Octicons**：
   - 用 `rg "stuhelperGroupCenter:octicons\\.[a-zA-Z-]+"` 列出实际引用的图标
   - `stuhelper-core/client/index.ts` 中 `Octicons.getAll()` 调用块替换为显式列表
   - `bundle size` 在 PR 描述中前后对比
2. **PR-2b 删除未引用组件/文件**：
   - 用 `rg <component-name>` 在 `client/` 全量搜索引用
   - 仅删除零引用项；不删"看似没用但被运行时字符串解析"的项
   - 同步删除对应单测
3. **PR-2c 重复 CSS 合并**：
   - 把 `client/styles/` 内重复定义合并到单一来源
   - 不动 `tokens.css` 的语义
4. **PR-2d 工具函数提取**（可选）：
   - 把 `*View.vue` 内重复的纯函数提到 `client/utils/`
   - 不改 component 语义

#### 通用红线（每个 P2 PR 都必须遵守）

- `git diff` 净删除（删除行数 > 增加行数），或总变更行数 < 200
- 不改 `<template>` 语义结构
- 不改 props / emits 接口
- 不改 `send(...)` 事件名
- 不改 Console API 路径
- `yarn test:unit && yarn test:startup && yarn test:ui` 全绿

#### 退出标准

- `Octicons.getAll()` 不再出现在 `stuhelper-core/client/`
- 所有 `client/components/*.vue` 在 `client/` 内有引用证据
- CSS 重复定义清零

#### 风险与回滚

- 字符串拼图标名称导致静态扫描漏报 → 仅按 allowlist 删除，allowlist 在 PR 描述里展示
- 删 dead code 误删了运行时反射使用的代码 → P0 ui smoke 兜底
- 回滚：单 PR revert

---

### P3 core 入口拆分

#### 目标
`stuhelper-core/src/index.ts` 的 `apply()` 只表达装配顺序，不承载细节。

#### 工作内容

把 `apply()` 当前 7 类职责拆成 7 个独立文件：

| 新文件 | 责任 |
|--------|------|
| `src/setup/register-core-service.ts` | `ctx.plugin(StuhelperGroupCenterService)` |
| `src/setup/register-console-entry.ts` | `consoleCtx.console.addEntry(resolveBrowserEntry())` |
| `src/setup/register-console-api.ts` | websocket / page-api / review-action / governance-action 注册 |
| `src/setup/register-runtime-modules.ts` | 22 个 `MODULE_CLASSES` 实例化（保留当前命令式风格） |
| `src/setup/register-background-jobs.ts` | `registerReviewClaimRecovery(...)` 等带 interval/dispose 的后台任务 |
| `src/setup/register-legacy-plugins.ts` | `applyLegacyFeatures(ctx, config)` 仍调用 `legacy-wrapper.ts` |
| `src/setup/register-console-preflight.ts` | `validateConsoleAdminPassword(...)`，仍在 `console/auth` 注入路径执行 |

`src/index.ts:apply()` 收敛为：

```ts
export function apply(ctx: Context, config: Config) {
  registerCoreService(ctx)
  registerConsoleEntry(ctx)
  registerConsolePreflight(ctx, config)
  registerConsoleApi(ctx, config)
  registerBackgroundJobs(ctx)
  registerRuntimeModules(ctx)
  registerLegacyPlugins(ctx, config)
}
```

#### 退出标准

- `src/index.ts` 不超过 50 行
- 所有装配函数有签名为 `(ctx: Context, config?: Config) => void`
- `yarn test:unit && yarn test:startup && yarn test:ui` 全绿
- **行为零变更**，按以下维度逐项验证：
  - 装配顺序与原 `apply()` 一致
  - `ctx.inject(['console', 'database', 'stuhelperGroupCenter', 'auth'], ...)` 的依赖集合不变（admin password 校验 + Console API 注册路径）
  - `ctx.inject(['database', 'stuhelperGroupCenter'], ...)` 的依赖集合不变（recovery + 模块初始化路径）
  - `ready` event 监听点不变
  - 22 个 `MODULE_CLASSES` 的 register / init 顺序不变

#### 风险与回滚

- 装配顺序改变导致初始化死锁 → 严格保留原顺序
- inject 依赖漂移导致服务不可用 → 退出标准已显式锁定 inject 依赖集合
- 回滚：单 PR revert

---

### P4a Runtime registry + BaseModule adapter

#### 目标
把 22 个 `MODULE_CLASSES` 装配从命令式 `new` 改为 registry 装配，**零业务改动**。

#### 工作内容

1. 新增 `src/runtime/registry.ts`：
   - 最小契约：
     ```ts
     export interface RuntimeModule {
       readonly id: string
       readonly order?: number
       create(ctx: Context, deps: ModuleDeps): RuntimeModuleInstance
     }
     export interface RuntimeModuleInstance {
       init(): Promise<void> | void
       dispose(): Promise<void> | void
     }
     ```
   - 不做 register/start/ready 全套；只做 create / init / dispose
   - 不做依赖图运行时调度
2. 新增 `src/runtime/base-module-adapter.ts`：把现有 `BaseModule` 子类包装成 `RuntimeModule`，零业务改动
3. `src/setup/register-runtime-modules.ts`（P3 产物）改为：
   - 从 registry 读取所有模块定义
   - 通过 adapter 装配
   - 顺序由 `RuntimeModule.order` 决定，默认与原 `MODULE_CLASSES` 一致
4. 新增 `docs/internal/exec-plans/active/koishi-runtime-modules-deps.md`：
   - 列出 22 个模块及其依赖（人工审计）
   - 用于 P4b 排序参考
   - 不作为运行时数据消费

#### 退出标准

- `src/setup/register-runtime-modules.ts` 不再出现 `new ModuleType(...)` 字样
- `src/index.ts` 中的 `MODULE_CLASSES` 常量数组移到 registry 注册点
- 22 个模块全部走 adapter 装配
- 所有现有单测通过 + `test:startup` + `test:ui` 全绿
- **行为零变更**，按以下维度逐项验证：
  - 装配顺序与 P3 末态一致
  - 模块 register / init 调用次数与原实现一致
  - `ready` event 触发时机不变
  - `dispose` 在插件卸载时被调用

#### 风险与回滚

- adapter 漏包装某个生命周期 hook → 单元测试覆盖每个模块的 `init` / `dispose`
- 回滚：单 PR revert

---

### P4b 逐个原生 RuntimeModule

#### 目标
把 22 个模块从 `BaseModule` 子类改写为原生 `RuntimeModule`。

#### 工作内容

按 `koishi-runtime-modules-deps.md` 的依赖图，从叶子节点（依赖最少）开始，**每个模块一个独立 PR**：

1. 重写为原生 `RuntimeModule` 实现
2. 删除该模块对 `BaseModule` 的依赖
3. 单测覆盖
4. 通过 P0 ui smoke 验证 UI 不受影响

预期顺序示例（实际以依赖图为准）：

1. 命令型简单模块：`HelpModule`、`DiceModule`、`BanmeModule`
2. 配置/日志：`ConfigModule`、`LogModule`、`EventModule`、`StatusModule`
3. 内容处理：`KeywordModule`、`AntiRecallModule`、`AntirepeatModule`、`RepeatModule`
4. 群成员管理：`MemberManageModule`、`MessageManageModule`、`OrderManageModule`
5. 业务逻辑：`WelcomeModule`、`SubscriptionModule`、`ReportModule`
6. 高依赖：`AIModule`、`AuthModule`、`GetAuthModule`、`WarnModule`、`crossGroupModule`

#### 退出标准

- `BaseModule` 与 `BaseModuleAdapter` 在 `stuhelper-core/src/` 下零引用
- 删除 `BaseModule` 与 `BaseModuleAdapter` 文件
- `core/modules/` 目录全部为原生 `RuntimeModule`

#### 风险与回滚

- 依赖图遗漏导致原生模块 setup 时依赖未就绪 → 每个 PR 独立 test:startup
- 模块行为变化 → P0 ui smoke + 现有单测兜底
- 回滚：单 PR revert（每模块独立 PR）

---

### P5a 配置传递验证

#### 目标
在做 P5 之前，先验证 YAML anchor 是否在当前 koishi 链路里可用。

#### 工作内容

在分支上做以下三个验证，每个独立提交：

1. **加载验证**：
   - 在 `koishi.yml` 加 anchor 与引用：
     ```yaml
     _shared: &platform_config
       baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
       serviceToken: ${{ env.STUHELPER_PLATFORM_SERVICE_TOKEN }}
     stuhelper-binding:
       platform: *platform_config
     stuhelper-group-guard:
       platform: *platform_config
     stuhelper-admin:
       platform: *platform_config
     ```
   - 启动 koishi，断言三个插件读到的 `platform.baseUrl` 一致
2. **Console 编辑器持久化验证**：
   - 启动 koishi → 通过 Console 配置面板修改某插件的不相关字段 → 保存
   - 检查 `koishi.yml` 是否仍保留 anchor 形式（关键：编辑器可能展开 anchor 写回）
3. **HMR 验证**：
   - 在 dev 模式下修改 anchor 内的 ENV 变量
   - 触发热重载
   - 断言三个插件都拿到新值

#### 退出标准

- 验证 1、2、3 全部通过 → P5 走 YAML anchor 路径
- 验证 1 通过、2 失败 → P5 走"三处显式重复 + 单一 ENV 占位符派生"退路

退路示例：

```yaml
stuhelper-binding:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
stuhelper-group-guard:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
stuhelper-admin:
  platform:
    baseUrl: ${{ env.STUHELPER_PLATFORM_BASE_URL }}
```

表面三处重复，实际单一 ENV 来源；改 ENV 一处全改。

#### 风险与回滚

- 验证 2 在不同 koishi-plugin-config 版本下行为不同 → 在 P5a 中固定当前 lockfile 版本
- 回滚：分支不合并

---

### P5 binding / group-guard / admin 显式装载

#### 目标
三个独立插件回归 `koishi.yml` 显式加载，core 不再 `ctx.plugin()` 它们。

#### 工作内容

1. 修改 `bots/koishi/koishi.yml` 在 `group:stuhelper:` 下加：
   ```yaml
   group:stuhelper:
     stuhelper-core:q1wm1r: {}
     stuhelper-binding:<id>: { platform: ... }
     stuhelper-group-guard:<id>: { platform: ..., guard: ..., scheduler: ..., moderation: ..., fun: ..., ai: ... }
     stuhelper-admin:<id>: { platform: ..., admin: ..., moderation: ..., fun: ... }
   ```
   配置传递格式按 P5a 验证结果选择 anchor 或显式 ENV 派生
2. **直接删除** `legacy-wrapper.ts` 内 `applyLegacyFeatures()` 函数体里的 `ctx.plugin(binding/groupGuard/admin)` 三行调用——不加注释、不加迁移标记。停止装载这件事必须在一处明确表达，避免读者在死代码里搜寻"装载点"。
3. `register-legacy-plugins.ts`（P3 产物）保留文件，body 改为空函数体（仅保留导出与签名）；该文件作为"装配点列表"的一部分仍出现在 `src/index.ts:apply()` 的调用链里，只是不再做任何装配。文件本身留待 P6 删除。
4. `stuhelper-core` 的 `Config` schema 中 `binding` / `guard` / `scheduler` / `moderation` / `fun` / `ai` / `admin` 子配置块标记 deprecated，运行时校验仍接受但日志 warn

#### 退出标准

- `koishi.yml` 显式列出 4 个插件
- 三个独立插件的命令、事件监听、数据库表注册不重复
- `yarn test:startup` 通过
- `yarn test:ui` 11 个 view 全绿
- 删除 `stuhelper-binding` 等命令后 core UI 受影响的功能仍工作（兼容性子目标）

#### 风险与回滚

- 双装载导致命令/事件重复注册 → 工作内容 #2 已直接删除 `legacy-wrapper.ts` 的三行 `ctx.plugin()`，`koishi.yml` 显式装载与 legacy 装载不允许在 P5 PR 中并存
- 配置传递在生产环境与开发环境差异 → P5a 已覆盖；本步只修配置文件
- 回滚：单 PR revert（恢复 `legacy-wrapper.ts` 的三行调用 + 删除 `koishi.yml` 显式条目）

---

### P6 删除 legacy-wrapper

#### 目标
彻底退出 god plugin 状态。

#### 工作内容

1. 删除 `bots/koishi/plugins/stuhelper-core/src/legacy/` 整个目录
2. 删除 `bots/koishi/plugins/stuhelper-core/src/setup/register-legacy-plugins.ts`
3. `stuhelper-core` 的 `Config` schema 删除已 deprecated 的子配置块，schema 收敛为仅 core 自身所需字段
4. 更新 `bots/koishi/README.md`、`docs/guides/koishi-development.md`：
   - "core 装配 binding/group-guard/admin" 描述全部删除
   - 改为"四插件由 koishi.yml 显式装载"

#### 退出标准

- `bots/koishi/plugins/stuhelper-core/src/legacy/` 不存在
- `git grep -r "applyLegacyFeatures\|legacy-wrapper"` 在 `bots/koishi/` 下零结果
- `yarn test:startup` 通过
- `yarn test:ui` 11 个 view 全绿

#### 风险与回滚

- core 的 Config schema 收敛后旧 koishi.yml 报错 → 项目处于"非生产"状态，不需要兼容旧配置；P5 时已要求迁移
- 回滚：单 PR revert

## 5. 验证基线

| 工具 | 范围 | 何时引入 |
|------|------|---------|
| `test:unit` | 各 plugin / package 单元测试 | 已有 |
| `test:startup` | 真实 koishi 启动烟雾 | 已有 |
| `test:ui` | Playwright 11 view smoke | P0 引入 |
| `yarn build` | 全 workspace 构建 | 已有 |

每个 P1-P6 PR 必须通过上述四项才能合并。任意一项失败 → PR 不合并；不引入"运行时兼容开关"绕过失败。

## 6. 待决策事项

| 决策 | 时机 | 选项 | 倾向 |
|------|------|------|------|
| YAML anchor 是否可用 | P5a | (a) 可用 → P5 走 anchor (b) 不可用 → P5 走 ENV 派生退路 | 由 P5a 验证决定 |
| P2 是否引入 knip / ts-prune | P2 之前 | (a) 引入（独立工具链 PR） (b) 不引入，仅用 rg + build 证据 | (b)，工具引入推迟到有需要时 |
| 是否需要 ADR-0007（server 模块化路径） | P4 启动前 | (a) 写 (b) 在本计划内涵盖即可 | (b)，本计划已记录决策与替代方案 |

## 7. 引用

- 关键决策：[adr/0006-koishi-core-ui-as-single-webui-entry.md](../../../adr/0006-koishi-core-ui-as-single-webui-entry.md)
- Koishi 开发指南：[docs/guides/koishi-development.md](../../../guides/koishi-development.md)
- 项目原则：本计划遵循的"不引入兼容路径"与"保留产品事实源"两条原则在 [ADR-0006](../../../adr/0006-koishi-core-ui-as-single-webui-entry.md) §Decision 中明文写出

## 8. 状态追踪

| 阶段 | 状态 | 完成日期 | PR |
|------|------|---------|----|
| P0a Playwright 基础设施 | 已完成 | 2026-04-25 | (本分支单 commit) |
| P0b 登录 fixture + 11 view smoke | 已完成（含 codex 复核 H1/M1/M2/M3/M4 加固） | 2026-04-25 | (本分支两个 commit：基线 + 加固) |
| P1 删除未启用 UI 包 | 已完成 | 2026-04-25 | (本分支单 commit) |
| P2 core UI 内部清理 | 已完成 | 2026-04-26 | (本分支多个 commit) |
| P3 core 入口拆分 | 已完成 | 2026-04-26 | `20e40472` |
| P4a Runtime registry + adapter | 已完成 | 2026-04-26 | (本分支单 commit) |
| P4b 逐个原生 RuntimeModule | 已完成（P4b-1 至 P4b-22 全部 native，并删除 BaseModule / adapter 兼容层） | 2026-04-26 | (本分支 P4b commits) |
| P5a 配置传递验证 | 未开始 | - | - |
| P5 三插件显式装载 | 未开始 | - | - |
| P6 删除 legacy-wrapper | 未开始 | - | - |

每完成一个阶段，回到本表更新状态、日期与 PR 链接。
