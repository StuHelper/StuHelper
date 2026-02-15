# 技术架构

## 系统架构

采用"调度中心 + 分布式执行节点"的混合架构。

| 节点类型 | 部署环境 | 角色 |
|----------|----------|------|
| 调度中心 | 腾讯云 | 任务调度、数据存储、Web服务 |
| 固定节点 | 校内主机 | SSO模拟、爬虫、高频任务 |
| 移动节点 | 用户手机 | 轻量任务、网络探测 |

## 技术栈

| 层级 | 技术选型 | 说明 |
|------|----------|------|
| 前端 Web | Vue3 + Tailwind CSS v4 | 评课社区 SPA |
| 后端 API | Go + Gin | 高性能 HTTP 服务 |
| API 规范 | OpenAPI 3.0.3 | Spec-First，自动生成前后端代码 |
| 用户认证 | Casdoor | OAuth2/OIDC 单点登录 |
| 数据库 | PostgreSQL 18 | 关系型数据 + JSONB |
| 缓存 | Redis 8 | 会话管理、限流、缓存 |
| 文件存储 | 腾讯云 COS | 前端直传，CDN 加速 |
| 部署 | Docker Compose | 单机容器编排 |

## 分层架构

后端采用 Handler → Service → Repository 三层架构。

详细规范见 `.project_rule/project_rules.md`
