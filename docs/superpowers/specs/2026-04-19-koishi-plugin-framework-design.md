# Koishi Plugin Framework Design

> 日期：2026-04-19
> 状态：approved-for-implementation
> 范围：StuHelper 仓库内新增 Koishi 子工作区与官方插件框架，不实现具体业务规则。

## 背景

StuHelper 现有仓库包含 `server/`、`clients/`、`infra/` 与 `docs/`，尚未包含 QQ 机器人运行时或 Koishi/NapCat 相关代码。项目未来需要接入 QQ 机器人能力，第一阶段目标不是完整业务实现，而是先建立一套可持续扩展的 Koishi 插件框架，用于承载以下业务方向：

- 用户在 StuHelper SSO 注册后绑定 QQ 号。
- 用户完成学生认证后与 QQ 身份联动。
- 用户加入指定群时，根据绑定与认证状态执行自动放行、禁言提醒、解除禁言、超时踢出。

NapCat 与 Koishi 已由外部独立部署，因此当前仓库的首要任务是提供一套可在本仓库内协同开发、可由外部 Koishi 实例挂载的插件工作区。

## 目标

在仓库根目录新增 `bots/koishi/`，使用 Koishi 官方模板初始化为独立工作区，并提供最小可运行的插件框架，满足以下条件：

- 不污染 `server/` 与 `clients/` 的现有边界。
- 保留 Koishi 官方 boilerplate 的 workspace 能力。
- 形成清晰的插件拆分与共享代码层。
- 为后续接入 QQ 绑定、群管、管理员命令提供稳定骨架。

## 非目标

本次设计不包含以下内容：

- 不实现 StuHelper 平台后端新增 API。
- 不实现真实的 QQ 绑定码校验或学生认证流程。
- 不接入生产环境 NapCat 与 Koishi 实例。
- 不实现具体群管理策略细节。

## 总体方案

采用仓库内独立子系统方案：

- 在仓库根目录新增 `bots/koishi/`。
- 通过 `npm create koishi@latest bots/koishi -- -t @koishijs/boilerplate -y` 使用官方模板初始化。
- 保留官方 boilerplate 自带的 workspace 能力，不使用 `--prod`。
- 将 StuHelper 机器人能力拆分为多个插件，由入口插件统一装配。

该方案的优点：

- 与主平台边界清晰。
- 不将机器人运行时强耦合到 Go 后端。
- 保持与 Koishi 官方脚手架一致，后续升级成本较低。
- 便于为本地联调、CI、单元测试单独演进。

## 目录结构

目标目录结构如下：

```text
StuHelper/
└── bots/
    └── koishi/
        ├── koishi.yml
        ├── package.json
        ├── packages/
        │   └── shared/
        │       └── src/
        │           ├── config/
        │           ├── logger/
        │           ├── platform/
        │           └── types/
        └── plugins/
            ├── stuhelper-core/
            ├── stuhelper-binding/
            ├── stuhelper-group-guard/
            └── stuhelper-admin/
```

### 目录职责

- `bots/koishi/`
  - Koishi 独立工作区根目录。
- `packages/shared/`
  - 共享配置、平台客户端、日志与类型定义。
- `plugins/stuhelper-core/`
  - 入口插件，负责装配其余插件与共享依赖。
- `plugins/stuhelper-binding/`
  - 绑定流程插件，预留未来的 QQ 绑定能力。
- `plugins/stuhelper-group-guard/`
  - 群准入与群管理流程插件，预留未来的禁言、提醒、踢出能力。
- `plugins/stuhelper-admin/`
  - 管理员命令与运维辅助能力。

## 系统边界

### StuHelper 平台负责

- SSO 身份与登录态。
- 学生认证权威状态。
- QQ 绑定关系权威状态。
- 群准入规则与审计记录。

### Koishi 插件负责

- 处理 QQ 平台事件。
- 调用 StuHelper 平台 API 查询或提交动作。
- 执行群聊动作，如禁言、解除禁言、踢出、发送提醒。
- 保存少量运行时状态，例如待处理记录、扫描任务与幂等键。

### NapCat 负责

- 作为 QQ 协议与 OneBot 适配层。
- 提供事件与动作 API。
- 不承载 StuHelper 业务判断。

## 插件拆分

### `stuhelper-core`

- 注册与装配 StuHelper 插件集。
- 提供统一的配置注入、日志实例与平台客户端实例。
- 控制各插件的启用顺序。

### `stuhelper-binding`

- 负责未来的 QQ 私聊绑定入口。
- 定义绑定流程相关命令与事件处理的扩展点。
- 当前阶段只保留插件入口与配置结构。

### `stuhelper-group-guard`

- 负责未来的入群事件处理与状态查询。
- 定义禁言、提醒、解除禁言、踢出等流程的扩展点。
- 当前阶段只保留插件入口与配置结构。

### `stuhelper-admin`

- 负责未来的管理员命令。
- 当前阶段只保留插件入口与配置结构。

## 配置模型

StuHelper 自定义插件配置使用 Koishi `Schema` 暴露，便于在控制台中直接配置。第一版框架至少定义以下配置对象：

- `platform.baseUrl`
- `platform.serviceToken`
- `binding.command`
- `binding.codeTtlMinutes`
- `guard.targetGroups`
- `guard.muteDurationSeconds`
- `guard.kickAfterMinutes`
- `guard.reminderTemplate`
- `guard.exemptUsers`
- `scheduler.scanIntervalSeconds`

NapCat 与 OneBot 连接配置不放入 StuHelper 自定义插件中，而是继续由 `koishi.yml` 的适配器配置负责。

## 运行与测试策略

第一阶段的目标是“框架可启动、插件可加载、目录可扩展”，因此验证重点如下：

- Koishi 工作区可成功初始化。
- 工作区依赖可安装。
- `shared` 与 4 个插件包能被 workspace 正确识别。
- TypeScript 构建或类型检查可以通过。
- `koishi.yml` 能挂载 `stuhelper-core`，并由其装配其余 StuHelper 插件。

## 文档与仓库集成

实现过程中需要同步补齐以下内容：

- 增加 `bots/koishi/README.md`，说明该子系统职责与启动方式。
- 在根目录 `README.md` 与 `AGENTS.md` 中补充 Koishi 子系统入口。
- 不修改现有 `server/`、`clients/` 主体架构。

## 设计结论

本次实现应严格限定在“建立官方模板 + 形成企业级插件骨架”这一范围内。先把边界、目录、配置模型与插件拆分做对，再进入后续真实业务能力的开发。
