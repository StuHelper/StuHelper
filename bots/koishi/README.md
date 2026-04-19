# StuHelper Koishi Workspace

该目录是 StuHelper 的 QQ 机器人插件工作区，基于 Koishi 官方 boilerplate 初始化。

## 边界

- `server/` 仍然是 StuHelper 的业务权威系统。
- `bots/koishi/` 负责 QQ 机器人运行时与插件开发。
- NapCat 作为外部部署的 OneBot 适配层，不在本目录内实现。

## 当前插件骨架

- `packages/shared`：共享配置、日志、平台客户端与类型。
- `plugins/stuhelper-core`：入口插件，统一装配其余插件。
- `plugins/stuhelper-binding`：预留 QQ 绑定流程。
- `plugins/stuhelper-group-guard`：预留群准入与群管流程。
- `plugins/stuhelper-admin`：预留管理员命令与运维能力。

## 本地命令

```bash
cd bots/koishi
corepack yarn install
corepack yarn dev
corepack yarn build
corepack yarn workspaces list
```

## 说明

当前阶段只完成框架与插件边界，不包含真实业务逻辑，也不直接连接生产 NapCat/Koishi 实例。
