# StuHelper Koishi Workspace

该目录是 StuHelper 的 QQ 机器人工作区，基于 Koishi 官方 boilerplate 初始化，并已接入 StuHelper 平台后端的 QQ 绑定与群管联动能力。

## 边界

- `server/` 仍然是 StuHelper 的业务权威系统，负责 QQ 绑定关系、绑定码与学生认证状态。
- `bots/koishi/` 负责 QQ 机器人运行时、群管逻辑与管理员命令。
- NapCat 作为外部部署的 OneBot 适配层，不在本目录内实现。

## 当前包与入口

`koishi.yml` 当前显式加载 `stuhelper-core`、`stuhelper-binding`、`stuhelper-group-guard` 与 `stuhelper-admin`，保留完整群管中心页面与功能入口。

- `packages/shared`：共享配置、日志、平台客户端与基础类型。
- `packages/moderation-core`：群管领域模型、SQLite 表、规则引擎与动作服务。
- `plugins/stuhelper-core`：当前入口插件，承载完整群管中心页面、控制台 API 与 WebSocket 交互。
- `plugins/stuhelper-binding`：处理私聊 `绑定 <code>` 命令，消费平台绑定码并建立 QQ 绑定。
- `plugins/stuhelper-group-guard`：处理入群准入、关键词命中、撤回留痕、举报流和娱乐命令。
- `plugins/stuhelper-admin`：提供文本管理员命令，用于查看待认证成员、查询警告、查看复核队列、批量禁言以及提交踢人/拉黑复核申请。

## 本地命令

```bash
cd bots/koishi
corepack yarn install
corepack yarn dev      # 源码态调试，适合本地开发
corepack yarn start    # 产物态启动，先构建再启动 Koishi
corepack yarn build    # 生成各本地包的 lib/index.js 与类型声明
corepack yarn test     # packages/plugins 单元测试 + 真实启动烟雾验证
corepack yarn workspaces list
```

启动前要求：

- `STUHELPER_CONSOLE_ADMIN_PASSWORD` 必须为非空值；`koishi.yml` 会把它作为 Koishi Console 管理员密码。
- 本地可直接 `export STUHELPER_CONSOLE_ADMIN_PASSWORD=dev-console-admin-password`，或把同名变量写入仓库根目录 `.env` / 生产环境变量文件。

## 自动化验证

- 单元测试基于 Koishi 官方 `@koishijs/plugin-mock`，不需要连接真实 OneBot/NapCat。
- `test:unit` 会覆盖 `packages/` 与 `plugins/` 下的 Koishi 测试文件。
- 绑定插件测试会验证私聊绑定命令和群聊误用提示。
- 群管插件测试会验证入群禁言、提醒、认证后解禁、超时踢出、关键词处理、模板/群绑定策略解析与撤回留痕。
- 控制台测试会验证高风险批量操作改走人工复核、复核执行、举报报表聚合，以及模板/群绑定保存事件写入 SQLite。
- 启动烟雾验证会真实拉起一次 Koishi，确认四个 StuHelper 插件、群管中心、Console API 与群守卫能力可启动，并固定监听 `5140`。
