---
type: guide
audience: backend-dev, ops
status: current
authoritative-source: bots/koishi/ + server/api/openapi.yaml
last-verified: 2026-05-25
---

# Koishi 机器人开发

## 子系统边界

- `bots/koishi/` 是 StuHelper 的 QQ 机器人工作区，承载 Koishi 插件、群管规则和机器人测试。
- `server/` 仍然是身份、QQ 绑定关系、学生认证状态的权威来源。
- NapCat 作为外部独立部署的 OneBot 适配层，不在本仓库内实现。

## 当前职责划分

| 位置 | 职责 |
|------|------|
| `packages/shared` | 平台 API 客户端、共享配置与基础类型 |
| `packages/moderation-core` | 群管领域模型、SQLite 表、规则引擎、动作服务；内部统一使用 `guildId`、`memberId` 等平台无关命名 |
| `plugins/stuhelper-core` | 当前入口插件，承载完整群管中心页面、控制台 API 与 WebSocket 交互 |
| `plugins/stuhelper-binding` | 私聊 `绑定 <code>` 命令，消费平台绑定码 |
| `plugins/stuhelper-group-guard` | 入群准入、关键词/复读处理、撤回留痕、举报、骰子和抽禁言；待认证成员记录会绑定 `platform + botSelfId`，扫描时按原 bot 路由动作 |
| `plugins/stuhelper-admin` | 提供 `群审状态`、`群审警告`、`群审复核`、`群审禁言`、`群审踢人申请`、`群审拉黑申请` 等文本管理员命令 |

群管中心 WebUI 的唯一运行入口是 `plugins/stuhelper-core`。它通过 Koishi Console 注册 `/stuhelper` 页面；`stuhelper-group-guard` 只提供运行时服务、事件监听和可选公开命令，不注册独立 WebUI。历史 `stuhelper-console` / `stuhelper-platform` 方向已由 ADR-0006 废弃。

## 本地命令

```bash
cd bots/koishi
corepack yarn install
corepack yarn build
corepack yarn test:unit
corepack yarn test:startup
corepack yarn test:ui
corepack yarn test
corepack yarn dev
corepack yarn start
corepack yarn workspaces list
```

说明：

- `yarn dev` 使用源码态启动，便于本地调试。
- `yarn start` 先构建 `lib/` 产物，再以运行时入口启动。
- `yarn test:unit` 会跑 `packages/` 与 `plugins/` 下的 Koishi 单测。
- `yarn test:startup` 会真实拉起一次 Koishi，并在启动前强制释放 `5140` 端口。
- `yarn test:ui` 会先构建工作区，再用临时 SQLite、临时 mock bot 和临时群管中心 JSON 数据目录拉起 Koishi Console，通过 Playwright 覆盖登录 warm-up、群管中心 NavRail、11 个业务视图、ChatDock、聊天接收 / 图片代理 / 发送 / 撤回、全局搜索、处置中心举报驳回、配置治理、订阅、黑名单、警告、日志检索、全局设置、角色权限和系统缓存真实交互，并检查 `pageerror`、未放行的 console error/warning、关键资源加载失败和关键资源 HTTP 4xx/5xx。
- `yarn test` 会串行执行构建、单元测试、启动烟雾验证和 UI smoke；根目录 `make e2e-koishi` 等价于执行 Koishi UI smoke。
- Koishi 工作区通过 Yarn `resolutions` 对 `@koishijs/client@5.30.11` 应用 `.yarn/patches/` 补丁，避免其构建入口以 CJS 方式加载 Vite 和 Vite 插件。升级 `@koishijs/client` 时，先复跑 `corepack yarn build` 和 `make e2e-koishi`；如果上游已改为 ESM Node API，再移除该补丁和 resolution。

## 配置入口

### Koishi 运行配置

- 文件：`bots/koishi/koishi.yml`
- 当前工作区固定监听 `5140`，由 `port: 5140` 与 `maxPort: 5140` 双重约束
- `scripts/startup-smoke.mjs` 会在烟雾验证前先清理占用 `5140` 的进程，避免端口漂移到其他值
- `koishi.yml` 显式加载 `stuhelper-core`、`stuhelper-binding`、`stuhelper-group-guard` 与 `stuhelper-admin`，群管中心 WebUI 挂载到 Koishi Console 的 `/stuhelper`
- `STUHELPER_CONSOLE_ADMIN_PASSWORD` 是 Koishi Console 的管理员密码，必须通过环境变量提供且不能为空
- Docker Compose 部署 Koishi 时，`koishi` service 必须通过 `env_file` 或等价机制注入 `STUHELPER_PLATFORM_BASE_URL`、`STUHELPER_PLATFORM_SERVICE_TOKEN`、`STUHELPER_FRESHMAN_MATERIAL_HOSTS` 和 `STUHELPER_CONSOLE_ADMIN_PASSWORD`
- 本地 SQLite 默认位于 `bots/koishi/data/koishi.db`
- 群管中心 JSON 数据默认位于 Koishi baseDir 下的 `data/stuhelperGroupCenter`；`STUHELPER_GROUP_CENTER_DATA_DIR` 可选覆盖该目录，`test:ui` 会自动指向临时目录以隔离 smoke 数据

### StuHelper 群管中心配置

当前 StuHelper 群管页面由 `stuhelper-core` 提供，保留重构前的群管中心能力。

群管配置来源包括：

- `stuhelper-core` 控制台页面与 WebSocket API
- `packages/shared` 的共享配置 schema
- `moderation-core` 与 `stuhelper-group-guard` 的 SQLite 表和运行时服务

权威定义见：

- `bots/koishi/packages/shared/src/types/index.ts`
- `bots/koishi/packages/shared/src/config/index.ts`

## 平台依赖接口

Koishi 当前依赖的 StuHelper 后端机器人接口包括：

- `POST /api/v1/bot/qq-binding/consume`
- `GET /api/v1/bot/qq-users/{qqID}/verification`
- `POST /api/v1/bot/admission/sessions`
- `GET /api/v1/bot/admission/sessions/pending?platform=qq&botSelfID=<botSelfID>`
- `POST /api/v1/bot/admission/sessions/{id}/events`
- `GET /api/v1/bot/admission/freshman/applications/pending-forward`

这两类接口都不是面向浏览器或普通用户的公开入口，而是面向机器人服务的内部接口：

- 后端通过 `bot_service_credentials` 中的 Koishi 机器凭据控制访问
- Koishi 侧通过 `STUHELPER_PLATFORM_BASE_URL` 读取后端地址，通过 `STUHELPER_PLATFORM_SERVICE_TOKEN` 注入 `platform.serviceToken`，并发送 `Authorization: Bearer <token>`
- `BOT_SERVICE_TOKEN` 只用于后端启动时 bootstrap / rotation；认证路径不直接比较裸环境变量
- 不应复用用户 JWT、Cookie 或前端访问令牌

与机器人联动的用户自助入口仍在主站：

- `GET /api/v1/user/qq-binding`
- `POST /api/v1/user/qq-binding/code`

改动这些接口时：

1. 先修改 `server/api/openapi.yaml`
2. 执行 `cd server && make generate`
3. 再同步调整 `bots/koishi/packages/shared/src/platform/index.ts`

说明：

- 当前后端机器人接口仍然是 QQ 专属契约，因此 `packages/shared/src/platform` 依旧保留 `qq-binding`、`qq-users` 命名。
- 这种 QQ 语义不会继续向群管核心域扩散；`packages/moderation-core` 与 `plugins/stuhelper-group-guard` 内部已经收敛为平台无关成员语义。
- admission 后端 `platform` 字段表示被验证账号的 subject platform，当前只有 `qq`。生产 NapCat / OneBot 的 Koishi runtime platform 可能是 `onebot`，插件会把它映射为 `qq` 后调用 admission API；禁言、踢人和发消息仍使用当前 runtime bot，不切换适配器。
- 当前 admission MVP 可在生产显式设置 `commands.enabled=false`、`moderation.enabled=false`、`freshmanForward.enabled=false`，避免新插件抢旧群管命令、接管消息风控或触发原始材料转发扫描。旧 `student-query` 插件不应因为 admission 上线被整体关闭；如它也监听同一批 admission 群的同一阶段入群验证，应调整旧插件自己的目标群或功能范围。

## 测试策略

- 单元测试使用 Koishi 官方 `@koishijs/plugin-mock`
- 不依赖真实 OneBot / NapCat 即可验证命令和事件流
- 当前自动化已覆盖：
  - QQ 绑定命令
  - 入群禁言、提醒、认证后解禁、超时踢出
  - 关键词命中后的删除、警告和禁言
  - 举报命令与抽禁言命令
  - 撤回事件留痕与提示
  - 控制台批量踢出改走人工复核、控制台复核执行与举报报表聚合
  - 工作区运行时入口与启动烟雾验证
  - 群管中心 WebUI 的 NavRail、11 个业务视图、ChatDock、聊天接收 / 图片代理 / 发送 / 撤回、全局搜索、处置中心举报驳回、配置治理、订阅、黑名单、警告、日志检索、全局设置、角色权限和系统缓存 Playwright UI smoke，并把 `document`、`script`、`stylesheet`、`font`、`image` 关键资源加载失败和 HTTP 4xx/5xx 视为失败
  - 管理员命令与命令权限表初始化
  - 群模板、群绑定和数据库优先级策略解析

## 当前仍由外部负责的部分

- NapCat / OneBot 真实连接配置仍由外部部署管理，不在本仓库提交

## 当前控制台能力

- 总览卡片会展示待复核、待认证成员、开放举报和高风险事件。
- 批量禁言、解除禁言、角色设置会直接执行；踢人和踢人并拉黑会统一进入人工复核队列。
- 复核工作台支持备注后执行或驳回高风险动作。
- 群模板面板支持维护禁言时长、超时踢出和提醒文案；群绑定面板支持按 `platform + guildId` 绑定模板。
- 举报面板会展示最近举报、AI 状态和摘要。

## 开发约束

- `database` 这类 Koishi 服务依赖必须在插件入口显式声明 `inject`
- 命令注册直接使用 `ctx.command()`；不要把不存在的服务名写进 `ctx.inject()`
- 高风险动作要写入 SQLite 审计记录
- 群规则优先在 Koishi 本地闭环处理，不把高频运行态回灌到主后端
- 所有对后端接口的变更都以 OpenAPI 为真源
- 多 bot 场景下，后台扫描任务必须依据记录中的 `platform + botSelfId` 路由到正确 bot，禁止再使用 `ctx.bots[0]` 这类单实例假设

## 相关文档

- [bots/koishi/README.md](../../bots/koishi/README.md)
- [product-specs/user-system.md](../product-specs/user-system.md)
- [docs/reference/api-overview.md](../reference/api-overview.md)
- [design/security-model.md](../design/security-model.md)
