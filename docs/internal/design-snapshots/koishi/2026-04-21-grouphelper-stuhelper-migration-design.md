---
type: internal
audience: maintainers, backend-dev, frontend-dev
status: snapshot
authoritative-source: this file
last-verified: 2026-04-21
---

# Grouphelper → StuHelper Koishi Migration Design

**Status:** approved-by-user

## 1. Goal

将 `grouphelper` 作为新的 Koishi 群管基座，逐步替换当前 `stuhelper-core + stuhelper-group-guard + stuhelper-admin + stuhelper-console` 的分散实现，同时保留 StuHelper 不可替代的业务闭环：

- QQ 绑定与学生认证闭环
- 人工复核工作流
- 群模板/群绑定/命令策略
- Platform API / 审计 / 主系统联动

这次迁移的目标不是“并列安装两个插件”，而是把 `stuhelper-core` 演进为 StuHelper 定制化版本的 `grouphelper`。

## 2. User Decisions

以下内容已经由用户明确拍板，本设计直接采用：

- 持久化统一改为 `sqlite`
- 不引入新插件命名，继续沿用 `stuhelper-core`
- `stuhelper-binding` 保持独立插件
- 迁移方向采用“以 grouphelper 为基座做增量定制”，而不是只搬 client 或并列安装

## 3. Source Comparison and Q2 Decision

### 3.1 Keyword / verification

`grouphelper` 的 `keyword` 模块更完整，覆盖：

- 入群审核关键词
- 违禁词
- 自动撤回 / 禁言 / 踢出
- 群级配置与 WebUI

StuHelper 当前 `message-guard.ts` 的优势不在“关键词管理”，而在：

- 与 `ModerationStore` 的事件、警告、复核联动
- 与学生认证禁言流程同域

**结论：**

- 泛化的关键词/违禁词管理以 `grouphelper keyword` 为新基座
- StuHelper 的“学生身份认证”不能退化成纯关键词验证，仍然保留独立认证闭环
- 后续迁移时，`keyword` 负责一般群管词法规则，`identity guard` 继续负责平台认证状态核验

### 3.2 Report / AI moderation

`grouphelper report` 明显比当前 `report-service.ts` 更强，包含：

- 更完整的 AI prompt 与上下文
- 举报者滥用限制
- 更多处罚组合
- 对应成熟 WebUI

StuHelper 当前实现的优势在于：

- 处罚动作进入人工复核队列
- 事件流写入现有审计体系
- 与 `ModerationActionService` / `ModerationStore` 对齐

**结论：**

- 以 `grouphelper report` 作为新的 AI 举报处理基座
- 处罚执行层不直接照搬 upstream，必须接入 StuHelper 的 review queue / audit event / command policy

### 3.3 Warn

`grouphelper warn` 功能和管理面都比当前 StuHelper 的零散警告逻辑完整。

**结论：**

- 以 `grouphelper warn` 替换现有 warn 底座

### 3.4 Auth / roles / command permissions

`grouphelper auth` 提供的是“角色 + 权限节点”系统，适合做 WebUI 和操作员体验基座。  
StuHelper 现有 `CommandPolicy` 语义更强，尤其是：

- authority 与角色白名单的组合判定
- 与人工复核动作绑定
- 与主项目的命令策略语义一致

**结论：**

- 复用 `grouphelper auth` 的管理体验和角色抽象
- `CommandPolicy` 继续作为敏感命令授权真相来源
- 不允许用 `grouphelper auth` 直接覆盖 `CommandPolicy`

## 4. Storage Architecture

上游 `grouphelper` 使用 JSON 文件做持久化。  
StuHelper Koishi 工作区已经启用 `@koishijs/plugin-database-sqlite`，且用户明确要求统一用 sqlite。

因此迁移采用：

- 使用 Koishi `database` 作为唯一持久化边界
- 不保留 JSON store 作为运行时 fallback
- 新增 `stuhelper_grouphelper_state` 作为 sqlite 命名空间状态表，承载 upstream JSON store 的过渡态数据
- 后续需要高查询能力的领域数据，再逐步拆到专用表

这不是兼容补丁，而是迁移期的正式存储设计：  
先把“文件存储”变成“数据库命名空间状态存储”，再逐步细化模型。

## 5. Runtime Shape

目标形态：

- `stuhelper-core`
  - 承载 grouphelper 风格的核心服务、模块注册、WebUI、API
  - 内聚 warn / keyword / report / auth / config / subscription / status 等通用群管能力
- `stuhelper-binding`
  - 保持独立，负责绑定码消费与账号绑定
- StuHelper delta 保留为 `stuhelper-core` 内的定制模块
  - 身份认证闭环
  - review workflow
  - command policy
  - platform / sms / audit integration

迁移过程中允许 `stuhelper-core` 暂时继续加载旧插件，但这只是过渡态，不是最终架构。

## 6. UI Strategy

最终目标不是保留当前 `stuhelper-console` 的定制控制台，而是把 `grouphelper` 的 9 个视图 UI 迁入 `stuhelper-core`：

- Dashboard
- Blacklist
- Logs
- Roles
- Settings
- Config
- Warns
- Subscription
- Chat

随后再把 StuHelper 专属的认证、复核、模板、平台联动信息并入该 UI。

## 7. Migration Sequence

### Phase 1

- 把 upstream 源码引入仓库并保留来源信息
- 在 `stuhelper-core` 中建立 sqlite 化的 grouphelper 数据层
- 注册新的 `ctx.groupHelper` 服务

### Phase 2

- 先迁移基础设施模块：auth / config / subscription / log
- 接入 WebSocket API 与上游 UI

### Phase 3

- 迁移 warn / keyword / report
- 将执行动作接入 StuHelper review / policy / audit

### Phase 4

- 将身份认证闭环并入新基座
- 删除 `stuhelper-group-guard`、`stuhelper-admin`、`stuhelper-console` 的被替代部分

## 8. Non-Goals

本轮不做以下错误路线：

- 不并列保留两套长期 UI
- 不继续扩写旧 `stuhelper-console` 架构
- 不在运行时保留 JSON 文件 fallback
- 不新建一个平行命名插件来绕开 `stuhelper-core`

## 9. Verification

每个迁移批次都必须满足：

- `tsx --test` 有新增回归测试
- `yarn build` 通过
- `yarn test:startup` 不因新基座注册而失败

首批实施范围聚焦：

- upstream 引入
- sqlite 数据层
- `ctx.groupHelper` 服务骨架
