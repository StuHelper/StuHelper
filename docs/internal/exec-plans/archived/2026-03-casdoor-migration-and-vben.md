---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# Casdoor → Zitadel + OpenFGA 迁移 & Vben Admin 引入

> 状态：已归档（主线完成）
> 归档时间：2026-03-29
> 启动日期：2026-03-23
> 前提：开发阶段，零迁移负担，旧数据可全部丢弃
> 备注：本计划的主干事项已在代码中完成；当时的后续跟踪文档现已统一归档到 `/Users/zxy/Code/StuHelper/docs/exec-plans/archived/`。

## 阶段概览

| 阶段 | 内容 | 预估 | 状态 |
|------|------|------|------|
| A-P1 | 基础设施（Zitadel + OpenFGA Docker + 初始化） | 2-3 天 | 完成 |
| A-P2 | SSO 重写（OIDC 客户端 + Auth handler + 中间件） | 3-5 天 | 完成 |
| A-P3 | 权限重写（RoleCapabilities + 删 RBAC + OpenFGA） | 3-4 天 | 完成（与 A-P2 合并） |
| B | Vben Admin（脚手架 + 全部页面重建，与 A-P3 并行） | 5-7 天 | 完成 |
| A-P4 | 认证流程 + 清理 + 文档 | 2-3 天 | 完成 |

## A-P1: 基础设施准备

### A1.1 docker-compose.yml 新增 Zitadel + OpenFGA

- Zitadel 服务（PostgreSQL 后端，独立 database `zitadel`）
- OpenFGA 服务（PostgreSQL 后端，独立 database `openfga`）
- OpenFGA migrate 初始化服务

### A1.2 Zitadel 初始化配置

- 创建 Organization: stuhelper-platform + buaa
- 创建 Project: hangxiaoban
- 创建 Application: web (OIDC 授权码+PKCE) / admin (OIDC 授权码+PKCE)
- 定义 Project Roles: super_admin / school_admin / moderator / verified_student / user
- 配置 SMTP + 短信

### A1.3 OpenFGA 模型导入

- 参照 docs/design-docs/openfga-model.fga
- 创建 Store + 导入模型 + 写入初始关系

### A1.4 Go 配置层

- CasdoorConfig → ZitadelConfig（Issuer/ClientID/ClientSecret/ProjectID）
- 新增 OpenFGAConfig（APIUrl/StoreID/AuthorizationModelID）

### A1.5 腾讯云短信转发（~50 行 Go）

## A-P2: SSO 重写

### A2.1 标准 OIDC 客户端

- 新建 `server/internal/pkg/oidc/client.go`（go-oidc/v3）
- Provider 自动发现 + JWKS 验证

### A2.2 Zitadel Claims 结构

- 新建 `server/internal/pkg/oidc/claims.go`
- 解析 `urn:zitadel:iam:org:project:roles`

### A2.3 重写 Auth Handler

- login/callback/refresh/logout/me 全部重写
- 删除所有 Casdoor SDK 调用
- shadow user 同步保留（external_id 改为 Zitadel sub）

### A2.4 重写 Auth 中间件

- Cookie: 本地 JWKS 验证（5min TTL）
- Bearer: Zitadel introspection（即时吊销）
- 角色 → 能力展开注入 context

### A2.5 前端 Auth 适配

- shared/api/auth.ts 适配新响应结构
- web + admin auth store 适配

## A-P3: 权限重写

### A3.1 RoleCapabilities 静态映射

- 5 角色 → 能力集 Go 常量 map
- ExpandRoles() 函数

### A3.2 删除 RBAC

- DROP 6 张表，删除对应 repository/service/handler
- 中间件简化为 RequireCapability

### A3.3 OpenFGA 集成

- Go 客户端封装（Check/Write/Delete）
- 资源创建时写关系 tuple
- 操作前 Check 权限

## B: Vben Admin（完成）

### B1 脚手架

- Vben Admin 5.7.0 官方完整仓库克隆到 `clients/admin/`（保留所有 app 变体、文档和演示）
- 保留 `apps/web-ele` + `packages/` + `internal/`，移除其他 app 变体
- Auth adapter：HttpOnly Cookie + CSRF Token，OIDC 重定向登录
- 权限路由：`meta.authority` = capability 字符串，由 accessCodes 匹配
- API 客户端：`withCredentials: true`，响应格式适配 `{success, data, error}`
- Vite proxy 转发 `/api` 到 Go 后端 `:8080`

### B2 页面重建

- Dashboard / 评课管理 / 举报管理 / 教师管理 / 敏感词管理
- 操作日志 / 实名审核 / 学生审核 / 学校配置 / 系统配置
- 角色管理 → 已删除（Zitadel Console 接管）

### B3 新增

- 用户列表页（占位，API 待对接）

### B4 清理

- `clients/admin/` 旧代码待删除（新项目在 `admin/` 目录）

## A-P4: 认证流程 + 清理

- 学生认证通过后异步同步 Zitadel 角色
- 删除 pkg/sso/ 全部、pkg/jwt/validator.go
- 移除 casdoor-go-sdk 依赖
- 更新 OpenAPI spec + 重新生成
- 更新文档

## 关键新建文件

| 文件 | 内容 |
|------|------|
| `server/internal/pkg/oidc/client.go` | OIDC 客户端 |
| `server/internal/pkg/oidc/claims.go` | Zitadel Claims |
| `server/internal/pkg/fga/client.go` | OpenFGA 客户端 |
| `server/internal/pkg/sms/tencent.go` | 短信转发 |
| `infra/zitadel/` | 部署和初始化 |
| `infra/openfga/` | 模型和初始化 |

## 关键删除文件

| 文件 | 原因 |
|------|------|
| `server/internal/pkg/sso/*` | Casdoor SDK |
| `server/internal/pkg/jwt/validator.go` | Casdoor JWT |
| `server/internal/modules/rbac/repository_*.go` | RBAC DB 查询 |
| `server/internal/modules/rbac/service_permissions.go` | RBAC 权限计算 |
| `server/internal/modules/rbac/service_users.go` | RBAC 用户角色 |
| `server/internal/modules/rbac/service_groups.go` | RBAC 用户组 |
| `clients/admin/` 全部 | 被 Vben Admin 替代 |

## 关键新建文件（B 阶段）

| 文件 | 内容 |
|------|------|
| `clients/admin/` | Vben Admin 5.7.0 完整官方仓库（二开基础） |
| `clients/admin/apps/web-ele/src/` | Element Plus 变体，StuHelper 管理后台主入口 |

## 决策日志

| 日期 | 决策 |
|------|------|
| 2026-03-23 | 计划确认。零迁移负担，旧数据全部丢弃，直接重写。 |
| 2026-03-23 | A-P1 完成：docker-compose 新增 Zitadel + OpenFGA，Go 配置层 Casdoor → Zitadel，infra/ 目录创建 |
| 2026-03-23 | A-P2+A-P3 合并完成：新建 pkg/oidc（OIDC 客户端 + Claims），重写 auth handler 全部（login/callback/refresh/logout/me），重写 auth 中间件（OIDC JWKS + 能力注入），重写 token service（移除 JWT 验证器），删除 pkg/sso/ + pkg/jwt/ 全部，删除 RBAC 23 文件（仅保留 middleware.go），新增 RoleCapabilities 静态映射 + ExpandRoles，course/user 模块全部从 PermissionService 迁移到 RequireCapability |
| 2026-03-23 | B 阶段完成：Vben Admin 5.7.0 部署到 admin/ 独立目录，Element Plus 变体。重写 auth 为 OIDC SSO + HttpOnly Cookie（非 Bearer），API 层适配 CSRF + {success,data,error} 格式。11 个管理页面迁移 + 1 个新增用户列表（占位）。typecheck + build 通过。 |
| 2026-03-24 | DB 清理：删除 8 张旧 RBAC 表（roles/permissions/role_permissions/user_roles/user_groups/user_group_members/user_group_permissions/user_permissions）及种子数据，角色→能力映射完全由 Go 常量 capability.RoleCapabilities 取代 |
| 2026-03-24 | OpenFGA 集成：FGA 客户端接入 wire 链路（可选，StoreID 为空时 nil），评课创建时异步写入 FGA tuple（author/course/school），举报创建时同步写入 FGA tuple，管理员 hide/delete 操作增加 FGA Check（FGA 未配置时回退到能力中间件） |
| 2026-03-24 | Zitadel 角色同步：创建 ManagementClient（REST API v2 + PAT 鉴权），学生认证通过/驳回时异步调用 GrantProjectRole/RevokeProjectRole。PAT 未配置时仅记日志。 |
| 2026-03-24 | 腾讯云短信：创建 pkg/sms（TC3-HMAC-SHA256 签名 + 内部 HTTP 端点 /internal/sms/send），Zitadel Action 可通过此端点发送验证码。配置层新增 SMSConfig。 |
| 2026-03-24 | Docker 简化：Zitadel 开发环境改用 Login V1（内置），移除 zitadel-login 容器和 Traefik 依赖。Redis healthcheck 修复（添加 REDIS_PASSWORD 到 environment）。镜像固定版本。 |
| 2026-03-25 | Docker 架构对齐官方 deploy/compose：恢复 Traefik + Login V2 三容器模式（proxy → zitadel-api + zitadel-login）。新增 /api 前缀路由 + strip-prefix 中间件（优先级 200）。路由命名规范 `-web` 后缀。ZITADEL_PORT 拆分为内部 8080 + ZITADEL_EXTERNAL_PORT。X-Forwarded-Proto 改用 ZITADEL_PUBLIC_SCHEME 变量。新增 access log stdout 配置。.env.example 新增 Docker 层变量区块。 |
