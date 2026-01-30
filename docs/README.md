# StuHelper 系统设计文档

本目录包含 BUAA StuHelper 系统的完整技术设计文档。

> 产品定位与核心能力详见 [产品介绍](product/overview.md)

## 文档索引

### 概览文档

| 文档 | 说明 |
|------|------|
| [产品介绍](product/overview.md) | 产品定位、核心能力、建设路线 |
| [系统架构](architecture/overview.md) | 技术架构、技术栈、安全设计 |
| [分层架构](architecture/layered-architecture.md) | Handler → Service → Repository 三层架构规范 |

### 模块文档

| 模块 | 文档路径 | 状态 | 说明 |
|------|----------|------|------|
| 身份认证 | [modules/auth/](modules/auth/) | 🟡 开发中 | SSO 登录、账号体系、安全存储 |
| 教学中心 | [modules/course/](modules/course/) | 🟡 开发中 | 评课社区、资料共享 |
| 社群自动化 | [modules/community/](modules/community/) | 🔴 规划中 | QQ群审批、Bot对接 |
| 策略配置 | [modules/policy/](modules/policy/) | 🔴 规划中 | 功能开关、灰度发布 |
| 校园工具箱 | [modules/tools/](modules/tools/) | 🔴 规划中 | 打卡、抢票、空教室 |
| 日志系统 | [modules/logging/](modules/logging/) | 🟡 开发中 | 日志采集、分析告警 |
| 通知中心 | [modules/notification/](modules/notification/) | 🔴 规划中 | 多渠道推送 |
| 举报审核 | [modules/moderation/](modules/moderation/) | 🔴 规划中 | 举报流程、处罚机制 |

### 基础设施文档

| 文档 | 说明 |
|------|------|
| [数据库设计](database/schema.md) | 表结构、索引设计 |
| [API 设计](api/overview.md) | RESTful API 规范 |
| [API 错误码](api/error-codes.md) | 统一错误码规范 |

### 开发文档

| 文档 | 说明 |
|------|------|
| [后端快速入门](development/backend-quickstart.md) | 环境搭建、项目结构、开发流程 |
| [开发规范](development/guide.md) | 代码风格、Git 工作流、三层架构 |
| [代码审查报告](development/backend-code-review.md) | 后端代码审查问题清单与修复进度 |

## 文档约定

### 状态标记

- 🟢 **已实现** - 功能已上线
- 🟡 **开发中** - 正在开发
- 🔴 **规划中** - 尚未开始
