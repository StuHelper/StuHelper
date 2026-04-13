# 产品概览

StuHelper（航小伴）是面向校园的信息与社区平台。

## 产品组成

| 层 | 代码入口 | 功能 |
|----|----------|------|
| 统一登录 | `modules/auth` + Zitadel | OIDC 登录、手机号验证码登录、会话管理 |
| 主站 | `clients/web` | 首页、课程、教师、评课、搜索、用户中心、通知 |
| 管理后台 | `clients/admin` | 评课运营、用户审核、学校配置、系统配置 |
| 后端 API | `server/cmd/stuhelper` | `/api/v1` 业务接口、健康检查、SSE |
| 授权增强 | `pkg/fga` + OpenFGA | 资源级关系校验 |
| 观测 | `infra/observability` | Prometheus / Loki / Tempo / Grafana / Alertmanager |

## 业务域

| 域 | 内容 |
|----|------|
| 课程实体 | 院系、课程分类、学期、课程详情、教师统计 |
| 评课社区 | 评测、回复、投票、举报、草稿、收藏、搜索 |
| 用户系统 | 实名认证、学生认证、手机号绑定、学校配置 |
| 通知中心 | 通知列表、未读数、已读、SSE 推送 |
| 管理运营 | 举报处理、评课审核、内容标记、教师维护、敏感词 |

## 用户角色

| 角色 | 值 | 能力 |
|------|----|------|
| 游客 | — | 浏览课程、教师、公开评课预览 |
| 登录用户 | `user` | 查看更多内容，管理个人资料 |
| 已认证学生 | `verified_student` | 查看完整评课，发布评课 |
| 学校管理员 | `school_admin` | 审核内容，处理举报 |
| 志愿者 | `moderator` | 内容审核（无法查看实名信息） |
| 平台管理员 | `super_admin` | 全平台运维 |

## 授权模型

- **角色**：Zitadel JWT claims
- **能力**：后端静态展开，不依赖本地 RBAC 表
- **访问事实**：实名状态、学生认证状态、学校归属
- **OpenFGA**：具体资源的操作权限判断

## 通知现状

通知入口已经统一到 `/api/v1/course/review/user/notifications/*`。
其中 SSE 推送由独立通知模块实现，但仍挂在同一路径前缀下。

## 技术栈

| 组件 | 技术 |
|------|------|
| 前端 | Vue 3.5+ / TypeScript 5+ / Vite 6+ / Pinia / Element Plus |
| 管理后台 | Vben Admin 5 |
| 后端 | Go 1.26+ / Gin / pgx |
| 数据库 | PostgreSQL 17 / Redis 7 |
| 认证 | Zitadel OIDC |
| 资源授权 | OpenFGA |
| 契约 | OpenAPI 3.1 |
| 部署 | Docker Compose / GitLab CI/CD |
| 观测 | OpenTelemetry + Grafana LGTM + Alertmanager |
