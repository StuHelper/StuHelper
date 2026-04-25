---
type: internal
audience: koishi-dev, maintainers
status: approved-for-implementation
authoritative-source: this file
last-verified: 2026-04-24
---

# Koishi StuHelper Platform Redesign

## 背景

现有 Koishi StuHelper 能力分散在多个插件与配置项中。群管功能继续扩展时，如果仍把模块开关、策略、权限和群配置放在 Koishi 插件配置里，会让新功能必须修改入口配置、重启运行时，并把 WebUI 与运行时能力耦合在一起。

本次重构把 StuHelper 群管插件改造成一个平台化入口：Koishi 配置只负责加载平台插件与基础连接；StuHelper 自身的模块启停、模块配置、权限策略和群策略全部进入新增 WebUI，并持久化到 Koishi database SQLite。

## 目标

- 新增 `koishi-plugin-stuhelper-platform` 作为唯一 StuHelper 平台入口。
- 模块通过统一契约注册：`manifest`、`configSchema`、`permissions`、`commands`、`events`、`webui`。
- 平台负责模块注册、配置持久化、模块状态、审计记录、权限策略、群策略和 WebUI 数据出口。
- 后续新增群管功能时，只新增模块目录或模块包，不修改旧入口配置。
- WebUI 文案保持简洁，不写解释性注释、实现说明或开发备注。

## 非目标

- 不兼容旧 StuHelper 插件配置项。
- 不提供旧配置运行时 fallback。
- 不在本轮实现完整生产级权限认证。
- 不直接迁移所有旧 UI 细节；只保留平台化所需的最小管理体验。

## 架构

采用外部平台插件加内部模块注册的结构：

```text
bots/koishi/plugins/stuhelper-platform/
├── src/
│   ├── index.ts
│   ├── module-contract.ts
│   ├── module-registry.ts
│   ├── platform-models.ts
│   ├── config-store.ts
│   ├── platform-service.ts
│   ├── console-routes.ts
│   └── modules/
└── client/
```

平台插件只暴露最小 Koishi `Config`。业务配置全部通过 `PlatformConfigStore` 读取和保存。

## 数据模型

平台使用 Koishi database 注册 5 张表：

- `stuhelper_module_state`
- `stuhelper_module_config`
- `stuhelper_permission_policy`
- `stuhelper_guild_policy`
- `stuhelper_audit_event`

默认模块状态只读返回，不隐式写库。保存配置、启停模块和未来策略更新必须写审计事件。

## 模块契约

模块导出 `StuhelperModule`：

- `manifest` 描述模块 id、名称、版本、顺序和默认启用状态。
- `configSchema` 描述模块配置结构。
- `permissions` 描述 WebUI 和命令权限。
- `commands` 描述 Koishi 命令注册。
- `events` 描述 Koishi 事件注册。
- `webui` 描述 WebUI 导航和配置页贡献。

平台服务根据 registry 与 store 合成运行时快照，不让 WebUI 直接读写模块内部实现。

## WebUI

WebUI 只使用新增平台页面，不继续使用旧配置项。

首版页面包括：

- 模块总览
- 模块配置
- 群策略
- 权限策略
- 审计记录

页面复用仓库已有 `sh-*` token 和 Koishi Console 页面结构。UI 文案只呈现必要状态和操作，不写说明型备注。

## 迁移策略

清空重来。旧配置不迁移、不读取、不兼容。`koishi.yml` 后续只保留加载平台插件与基础运行依赖。

## 验证策略

- 每个服务层任务先写 focused unit tests。
- 每个 Task 完成后执行规格复核与质量复核。
- Koishi 包级验证优先使用 focused `tsx --test` 和插件 `tsc --noEmit`。
- 不把既有 unrelated test failure 作为当前任务成功或失败依据。
