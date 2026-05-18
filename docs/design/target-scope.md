---
type: design
audience: all
status: current
authoritative-source: this file
last-verified: 2026-05-18
---

# 0001. StuHelper Target Scope And Module Boundaries

## 状态

已采纳

## 背景

StuHelper 当前最成熟的能力并不是完整教务，而是：

- 认证与会话
- 用户实名与学生认证
- 课程目录
- 评课社区
- 通知
- 后台治理

代码库已经具备单体 Go API、OpenAPI-first、Casdoor OIDC、OpenFGA、Redis、S3/MinIO、可观测性和 CI/CD 基础。真正缺口不是实验 / 作业 / 评分域，而是新产品边界下的教务读模型、资源共享、可插拔存储、统一审计和外部依赖治理。

## 决策

### 产品范围

StuHelper 固定为校园信息平台，不再按完整教务系统或教学实验平台扩展。

当前主路线包含：

- `auth`
- `user`
- `course` / `review`
- `academics`
- `resource`
- `storage`
- `notification`
- `authorization`
- `open-platform`
- `audit`
- `admin`

### 明确非目标

以下方向不进入当前主路线：

- 实验系统
- 作业系统
- 提交 / 批改 / 评分 / 申诉系统
- 选课 / 退课 / 排课 / 调课写侧
- 面向实验或作业的通用附件平台

### 教务边界

`academics` 只负责：

- 接入外部教务数据
- 标准化并落库
- 提供查询、展示和与本系统用户的关联视图

`academics` 不负责：

- 维护完整教务写侧主数据
- 实现完整选课、排课、退课、调课流程

### 资源与存储边界

`resource` 是业务模块，负责资源元数据、标签、绑定、检索和分享。

`storage` 是基础模块，负责：

- mount
- driver registry
- capability reporting
- health check
- 统一对象访问与错误模型

`resource` 只依赖 `storage` 暴露的稳定接口，不直接依赖 S3 SDK。

### 模块边界策略

短期内保留现有 `course` 目录承载课程目录与评课，以避免在本轮把成熟域和新域的改造耦合到一起。新增域采用清晰一级模块：

- `academics`
- `resource`
- `storage`

### 开放平台边界

`open-platform` 是平台扩展能力，负责第三方应用接入、scope 审批、用户授权、最小化 disclosure API 和审计。

`open-platform` 不负责：

- 替代 Casdoor 注册 / 登录页
- 让第三方直接读取 Casdoor user API
- 把学生认证、实名认证、手机号、学校归属等业务字段塞进第三方 token
- 提供完整 OAuth marketplace、计费、插件运行时或第三方资源代理

第三方 app 只能通过 StuHelper disclosure API 读取已审批且已授权的字段。完整设计见 [open-platform-v1.md](open-platform-v1.md)。

后续若 `course` 持续阻碍 catalog/review/academics 的演进，再拆分为：

- `catalog`
- `review`
- `academics`

## 后果

### 正面结果

- 主路线与非目标边界被显式冻结。
- 新模块不会再被塞进 `course` 大域。
- 教务展示域与资源共享域能够独立演进。
- 第三方教务连接器和网盘驱动有了正确的架构落点。

### 代价

- 文档、OpenAPI、能力模型和目录边界需要持续收口。
- `objectstorage` 已经退到 `storage` abstraction 的底层驱动实现，后续新增文件能力都应经 `storage` 进入。
- 审计与 outbox 已统一到 `audit_events` / `domain_event_outbox`，后续新增副作用不要再回到碎片化表模型。
