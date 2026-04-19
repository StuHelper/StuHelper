# Koishi Plugin Framework Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** 在 StuHelper 仓库内建立基于 Koishi 官方模板的独立工作区，并产出可扩展的 StuHelper 插件框架。

**Architecture:** 使用 `bots/koishi/` 作为独立 Node.js 子系统，通过 Koishi 官方 boilerplate 初始化 workspace。以 `packages/shared` 提供共享能力，以 `plugins/stuhelper-*` 提供插件边界，入口插件 `stuhelper-core` 负责装配其余插件。

**Tech Stack:** Node.js 24、npm create、Koishi、TypeScript、workspace、Yarn Berry（官方模板默认）

---

## File Structure

- Create: `bots/koishi/`
- Create: `bots/koishi/README.md`
- Create: `bots/koishi/packages/shared/`
- Create: `bots/koishi/plugins/stuhelper-core/`
- Create: `bots/koishi/plugins/stuhelper-binding/`
- Create: `bots/koishi/plugins/stuhelper-group-guard/`
- Create: `bots/koishi/plugins/stuhelper-admin/`
- Modify: `README.md`
- Modify: `AGENTS.md`

### Task 1: 初始化官方 Koishi 工作区

**Files:**
- Create: `bots/koishi/`

- [ ] **Step 1: 确认目标目录不存在**

Run: `test ! -e bots/koishi`
Expected: 命令返回 0。

- [ ] **Step 2: 执行官方脚手架**

Run:

```bash
npm create koishi@latest bots/koishi -- -t @koishijs/boilerplate -y
```

Expected: 成功生成 `bots/koishi/package.json`、`bots/koishi/koishi.yml` 与 boilerplate 文件。

- [ ] **Step 3: 安装依赖**

Run:

```bash
cd bots/koishi && corepack yarn install
```

Expected: 依赖安装成功，生成 `.yarn/install-state.gz` 或缓存状态文件，无 install error。

- [ ] **Step 4: 记录脚手架结果**

Run:

```bash
find bots/koishi -maxdepth 2 | sort | sed -n '1,120p'
```

Expected: 输出中包含 `package.json`、`koishi.yml`、`packages` 或 `plugins` 根目录。

### Task 2: 建立 StuHelper 插件与共享包骨架

**Files:**
- Create: `bots/koishi/packages/shared/package.json`
- Create: `bots/koishi/packages/shared/tsconfig.json`
- Create: `bots/koishi/packages/shared/src/index.ts`
- Create: `bots/koishi/packages/shared/src/config/index.ts`
- Create: `bots/koishi/packages/shared/src/logger/index.ts`
- Create: `bots/koishi/packages/shared/src/platform/index.ts`
- Create: `bots/koishi/packages/shared/src/types/index.ts`
- Create: `bots/koishi/plugins/stuhelper-core/package.json`
- Create: `bots/koishi/plugins/stuhelper-core/tsconfig.json`
- Create: `bots/koishi/plugins/stuhelper-core/src/index.ts`
- Create: `bots/koishi/plugins/stuhelper-binding/package.json`
- Create: `bots/koishi/plugins/stuhelper-binding/tsconfig.json`
- Create: `bots/koishi/plugins/stuhelper-binding/src/index.ts`
- Create: `bots/koishi/plugins/stuhelper-group-guard/package.json`
- Create: `bots/koishi/plugins/stuhelper-group-guard/tsconfig.json`
- Create: `bots/koishi/plugins/stuhelper-group-guard/src/index.ts`
- Create: `bots/koishi/plugins/stuhelper-admin/package.json`
- Create: `bots/koishi/plugins/stuhelper-admin/tsconfig.json`
- Create: `bots/koishi/plugins/stuhelper-admin/src/index.ts`

- [ ] **Step 1: 建立目录**

Run:

```bash
mkdir -p \
  bots/koishi/packages/shared/src/config \
  bots/koishi/packages/shared/src/logger \
  bots/koishi/packages/shared/src/platform \
  bots/koishi/packages/shared/src/types \
  bots/koishi/plugins/stuhelper-core/src \
  bots/koishi/plugins/stuhelper-binding/src \
  bots/koishi/plugins/stuhelper-group-guard/src \
  bots/koishi/plugins/stuhelper-admin/src
```

Expected: 所有目录创建成功。

- [ ] **Step 2: 写入最小共享包代码**

Code:

```ts
export * from './config'
export * from './logger'
export * from './platform'
export * from './types'
```

Expected: `packages/shared` 暴露统一出口。

- [ ] **Step 3: 写入各插件最小入口**

Code:

```ts
import { Context, Schema } from 'koishi'

export interface Config {}

export const Config: Schema<Config> = Schema.object({})

export function apply(ctx: Context) {
  ctx.logger('<plugin-name>').info('plugin loaded')
}
```

Expected: 每个插件都具备 Koishi 标准入口。

- [ ] **Step 4: 让 core 插件装配其余插件**

Code:

```ts
import binding from 'koishi-plugin-stuhelper-binding'
import groupGuard from 'koishi-plugin-stuhelper-group-guard'
import admin from 'koishi-plugin-stuhelper-admin'

export function apply(ctx: Context) {
  ctx.plugin(binding)
  ctx.plugin(groupGuard)
  ctx.plugin(admin)
}
```

Expected: `stuhelper-core` 成为唯一挂载入口。

### Task 3: 定义最小共享配置与平台客户端边界

**Files:**
- Modify: `bots/koishi/packages/shared/src/config/index.ts`
- Modify: `bots/koishi/packages/shared/src/platform/index.ts`
- Modify: `bots/koishi/packages/shared/src/types/index.ts`

- [ ] **Step 1: 定义配置类型与 Schema 工厂**

Code:

```ts
export interface StuhelperPlatformConfig {
  baseUrl: string
  serviceToken: string
}

export interface StuhelperBindingConfig {
  command: string
  codeTtlMinutes: number
}
```

Expected: 平台、绑定、群管配置有明确类型。

- [ ] **Step 2: 定义平台客户端接口**

Code:

```ts
export interface PlatformClient {
  getHealth(): Promise<void>
}

export function createPlatformClient(config: StuhelperPlatformConfig): PlatformClient {
  return {
    async getHealth() {
      const url = new URL('/healthz', config.baseUrl)
      const response = await fetch(url, {
        headers: { Authorization: `Bearer ${config.serviceToken}` },
      })
      if (!response.ok) throw new Error(`platform request failed: ${response.status}`)
    },
  }
}
```

Expected: 后续业务插件可以依赖接口而不是散落 `fetch`。

- [ ] **Step 3: 定义未来业务状态的预留类型**

Code:

```ts
export type VerificationState =
  | 'unbound'
  | 'bound_unverified'
  | 'verified'
  | 'muted_pending_verification'
  | 'expired_pending_kick'
```

Expected: 未来业务状态在共享包中集中定义。

### Task 4: 调整 Koishi 配置与运行入口

**Files:**
- Modify: `bots/koishi/koishi.yml`
- Modify: `bots/koishi/package.json`

- [ ] **Step 1: 在 `koishi.yml` 中挂载 `stuhelper-core`**

Config snippet:

```yml
plugins:
  group:server:
    server: {}
  group:basic:
    help: {}
  group:stuhelper:
    koishi-plugin-stuhelper-core: {}
```

Expected: Koishi 启动时能加载 StuHelper 入口插件。

- [ ] **Step 2: 保留官方模板默认插件，去除与本次框架无关的最重依赖入口**

Expected: 配置文件保持最小可运行，不强行引入无关平台适配器。

- [ ] **Step 3: 验证工作区识别**

Run:

```bash
cd bots/koishi && corepack yarn workspaces list
```

Expected: 输出包含 `shared` 与 4 个 `stuhelper-*` 包。

### Task 5: 本地验证与仓库文档集成

**Files:**
- Create: `bots/koishi/README.md`
- Modify: `README.md`
- Modify: `AGENTS.md`

- [ ] **Step 1: 写子系统 README**

Content requirements:

```md
# Koishi Bot Workspace

- 说明该目录是 StuHelper 的 QQ 机器人插件工作区
- 说明 Koishi/NapCat 运行边界
- 说明本地安装与启动命令
```

Expected: 新开发者知道该子系统做什么、怎么启动。

- [ ] **Step 2: 更新根 README 与 AGENTS**

Expected: 文档中出现 `bots/koishi/` 的入口说明。

- [ ] **Step 3: 运行最小验证**

Run:

```bash
cd bots/koishi && corepack yarn build
cd bots/koishi && corepack yarn workspaces list
```

Expected: 构建成功，workspace 识别成功。

- [ ] **Step 4: 记录 git 变更**

Run:

```bash
git status --short
```

Expected: 输出仅包含本次新增的 Koishi 相关文件与文档更新。
