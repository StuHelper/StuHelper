---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# StuHelper IAM 架构迁移：Zitadel + OpenFGA

> 状态：大部分完成。OIDC / capability 核心链路、基础设施、清理均已完成。剩余：自定义登录 UI、SMTP 配置。

## Goal

将 StuHelper 生态的身份认证和授权体系从 Casdoor + 应用自建 RBAC 迁移到 Zitadel + OpenFGA 架构，实现：

- **标准化身份层**：OIDC 认证的 Zitadel 替代 Casdoor，消除 SDK 质量问题
- **关系型授权**：OpenFGA 处理资源级权限（谁能操作哪个具体资源），支持权限继承链
- **权限模型简化**：删除 6 张 RBAC 表，角色→能力映射降级为 Go 静态常量
- **生态级身份复用**：认证结果存 Zitadel 元数据，任何未来应用可直接复用

## 架构决策记录

### D1: SSO 从 Casdoor 迁移到 Zitadel

**Context**: Casdoor SDK 质量差（goroutine+select 包装 context、全局 HTTP client），无 OIDC 认证，多租户能力弱，无 Token 定制能力。

**Decision**: 迁移到 Zitadel（Go 原生、OIDC 认证、原生多租户、Actions V2）。

**Consequences**: 需要额外开发腾讯云短信转发服务（~50 行 Go）和自定义登录 UI（手机号登录）。但消除了持续的 SDK workaround 成本，获得了多租户和开放平台能力。

### D2: 权限模型采用方案 B2（复合角色 + 应用侧静态展开）

**Context**: 当前 6 张 RBAC 表 + 复杂 SQL UNION 查询只是为了做"角色→能力"映射。26 个权限的 scope_roles 字段本质上就是角色到能力的另一种写法。

**Decision**:
- Zitadel 管理 5 个粗粒度角色：super_admin / school_admin / moderator / verified_student / user
- 角色→能力映射为 Go 常量 map，中间件内存展开，零 DB 查询
- 不使用 Actions V2（现阶段）——内存展开已足够，避免 webhook 复杂度

**Consequences**: 丢失个人权限覆盖（override grant/deny）和组批量管理。但这两个功能前端页面尚未开发、无实际使用场景。如果将来需要，加一张 `capability_overrides` 表即可。

### D3: 引入 OpenFGA 处理资源级授权

**Context**: 当前权限模型只能回答"张三是管理员"，不能回答"张三能不能删除评课#100"。随着志愿者、学校管理员、课程负责人等角色出现，需要基于关系链推导权限。

**Decision**: 引入 OpenFGA（Google Zanzibar 模型的开源实现），建模 ecosystem → school → course → review 关系链。选 OpenFGA 而非 SpiceDB，因为 PostgreSQL 支持更成熟、多租户（Store 隔离）、社区更大。

**Consequences**: 新增一个服务（OpenFGA，~100MB RAM）。资源创建时需写入关系 tuple，有额外的写入开销。但获得了完整的关系型授权能力。

### D4: 认证分层——过程和真相源在应用，Zitadel 只存粗粒度角色

**Context**: 实名认证和学生认证涉及 PII 收集、第三方 API 调用、学校 LDAP 查询、管理员审核等复杂流程。认证可能被撤销、过期、复审驳回。

**Decision**:
- 认证过程、材料、**状态真相源**均在应用 DB（user_identities / user_profiles）
- Zitadel 只存粗粒度角色（verified_student），作为 Token claim 的粗筛依据
- 认证撤销时：应用 DB 立即更新 + 异步同步 Zitadel 角色移除
- 需要精确判断的业务场景查应用 DB 实时状态，不依赖 Token claim

**Consequences**: Token claim 中的认证角色可能有短暂延迟（Token TTL 内），但业务授权的最终判断查 DB 实时状态，不受 Token 生命周期影响。

### D5: 多租户模型

**Decision**:
- StuHelper 生态 = Zitadel Instance（sso.stuhelper.com）
- 平台管理组织 = Zitadel Organization（stuhelper-platform）
- 每所学校 = Zitadel Organization（buaa / tsinghua / ...）
- 航小伴 = Zitadel Project，包含 Web / Admin / Mobile 三个 Application
- 第三方接入 = Project Grant 给第三方 Organization

### D6: 不拆分认证为独立服务

**Context**: 学生认证是否应该是独立应用？

**Decision**: 不拆。航小伴是唯一触发认证流程的应用，拆分只增加复杂度。保持模块边界清晰即可，将来出现第二个需要触发认证的应用时再考虑拆分。

### D7: 中国本土化适配

**Decision**:
- 腾讯云短信：Zitadel 通用 HTTP SMS 提供商 + Go 薄转发服务
- 手机号登录：自定义登录 UI（Vue）+ Zitadel Session API
- Google Fonts：自托管字体文件替换
- 容器镜像：推到腾讯云/阿里云私有镜像仓库

## Requirements

### 身份层（Zitadel）

- [x] Zitadel 自托管部署（Docker Compose + PostgreSQL）
- [x] 创建 Organization 结构（stuhelper-platform + buaa）
- [x] 创建 Project hangxiaoban + Application（web / admin / mobile）
- [x] 定义 Project Roles（super_admin / school_admin / moderator / verified_student / user）
- [ ] 配置 SMTP（邮件验证）
- [x] 配置通用 HTTP SMS（腾讯云短信转发）
- [ ] 自定义登录 UI（Vue，支持手机号验证码登录）
- [x] Google Fonts 替换为自托管

### SSO 迁移

- [x] 替换 SSO 客户端：删除 `pkg/sso/` 整个包，用标准 Go OIDC 库
- [x] 重写 JWT 验证：标准 OIDC 验证，不再需要 Casdoor 特定 claims struct
- [x] 重写 login/callback/refresh/logout handler
- [x] 重写 Token cookie 管理
- [x] 重写 auth 中间件（Cookie 本地 JWKS + Bearer introspection，从 Token 读角色和元数据）
- ~~用户数据迁移脚本~~ — 不需要，无旧用户

### 权限简化

- [x] 定义角色→能力静态映射（Go 常量 map）
- [x] 重写中间件：Token 读角色 → 内存展开能力 → 注入 context
- [x] 删除 6 张 RBAC 表（roles / permissions / role_permissions / user_roles / user_groups / user_group_members / user_group_permissions / user_permissions）
- [x] 删除 RBAC module 的权限计算相关代码
- [x] RBAC 管理 API 迁移：角色管理 → Zitadel Management API

### 授权层（OpenFGA）

- [x] OpenFGA 自托管部署
- [x] 编写授权模型（ecosystem / school / course / review / report / user_profile）
- [x] 编写 Go 客户端封装（Check / Write / Delete）
- [x] 资源创建时自动写入关系 tuple
- [x] 关键操作前调 OpenFGA Check
- [x] 区分志愿者/管理员权限边界（删帖 vs 看实名信息）

### 认证流程

- [x] 实名认证：应用触发 → 腾讯云核验 → 结果存应用 DB（user_identities，真相源）
- [x] 学生认证：LDAP/手动 → 审核 → 结果存应用 DB（user_profiles，真相源）→ 异步同步 Zitadel 粗粒度角色（verified_student）
- [x] 认证撤销：应用 DB 立即更新 → 异步移除 Zitadel 角色
- [x] 前端适配：粗筛从 Token 角色读，精确判断调应用接口查实时状态

### 文档更新

- [x] 重写 `docs/architecture/ecosystem-identity-and-authorization.md`
- [x] 重写 `docs/modules/auth/` 全部文档
- [x] 重写 `docs/modules/rbac/README.md`
- [x] 重写 `docs/modules/policy/` 全部文档
- [x] 标记 `docs/casdoor/` 为废弃或删除
- [x] 更新 `docs/architecture/README.md` 索引
- [x] 更新 `.trellis/spec/` 下的开发指南

### 清理

- [x] 删除 `server/internal/pkg/sso/` 整个包
- [x] 删除 `server/internal/pkg/jwt/validator.go`
- [x] **保留并简化** `server/internal/pkg/token/blacklist.go`（仅用于紧急吊销）
- [x] **保留并简化** `server/internal/modules/auth/user_sync.go`（shadow user 同步）
- [x] 删除 L1+L2 用户缓存（`pkg/sso/cache.go`）
- [x] **保留 `users` 表**（shadow user，业务外键锚点），删除 RBAC 6 张表
- [x] Bearer token 走 Zitadel introspection，Cookie 走短 TTL + refresh
- [x] 更新 OpenAPI spec

## Acceptance Criteria

- [x] 用户可通过 Zitadel 登录（邮箱+密码、手机号+验证码）
- [x] Token claim 携带角色和组织（不携带认证状态元数据）
- [x] 管理后台根据 Token 角色展开的能力控制页面/按钮可见性
- [x] 志愿者可删帖但不能看发帖人实名信息（OpenFGA 区分）
- [x] 学校管理员可管理本校内容（OpenFGA 关系链验证）
- [x] 超级管理员可管理全生态（OpenFGA 继承）
- [x] 实名认证和学生认证状态真相源在应用 DB，Zitadel 只存粗粒度角色
- [x] 已认证学生的 verified_student 角色在 Zitadel 中分配，Token 自动携带
- [x] shadow user 表保留，业务外键链完整
- [x] Bearer token 走 Zitadel introspection 实现即时吊销
- [x] 所有旧 Casdoor 代码和 RBAC 6 张表已删除
- [x] `go build` / `go test` / 前端 lint 全部通过
- [x] 架构文档已更新

## Definition of Done

- 所有 Acceptance Criteria 通过
- go build / go vet / go test 通过
- 前端 TypeScript 类型检查通过
- 架构文档和模块文档已更新
- OpenAPI spec 和 generated types 已同步
- 无残留 Casdoor 引用

## Out of Scope

- Actions V2 Token 预计算（现阶段用应用侧内存展开替代）
- RBAC 管理 admin 前端页面重建（Zitadel Console 替代）
- 完整的开放平台 API（仅预留 Project Grant 架构能力）
- 学校管理端独立应用（保持为航小伴管理后台）
- git 历史中的私钥清理（filter-repo，单独处理）
- uni-app x 移动端适配（架构上预留 OIDC Native Application）
- SpiceDB（选择 OpenFGA）
- Keycloak（选择 Zitadel）

## Technical Notes

### 关键文件行动清单

| 文件 | 行动 |
|------|------|
| `server/internal/pkg/sso/` (client.go, cache.go, state.go) | 删除，替换为标准 OIDC 库 |
| `server/internal/pkg/jwt/validator.go` | 删除，Zitadel 标准验证 |
| `server/internal/pkg/token/blacklist.go` | **保留并简化**，仅用于紧急吊销 |
| `server/internal/pkg/token/service.go` | **保留并简化**，OIDC 验证 + introspection |
| `server/internal/modules/auth/handler_login.go` | 重写 |
| `server/internal/modules/auth/handler_userinfo.go` | 重写 |
| `server/internal/modules/auth/handler_session.go` | 重写 |
| `server/internal/modules/auth/user_sync.go` | **保留并简化**，shadow user 同步 |
| `server/internal/modules/rbac/repository_users.go` | 删除 |
| `server/internal/modules/rbac/service_permissions.go` | 删除 |
| `server/internal/modules/rbac/middleware.go` | 重写 |
| `server/internal/pkg/capability/capability.go` | 保留 + 扩展（加 RoleCapabilities map） |
| `server/internal/pkg/middleware/auth.go` | 重写（Cookie 本地验证 / Bearer introspection） |
| `server/scripts/init.sql` 中 RBAC 6 张表 | 删除（`users` 表保留） |

### 新增依赖

| 依赖 | 用途 |
|------|------|
| `github.com/coreos/go-oidc/v3` | 标准 OIDC 验证 |
| `github.com/openfga/go-sdk` | OpenFGA Go 客户端 |
| `github.com/zitadel/zitadel-go/v3` | Zitadel Management API（可选，也可直接用 HTTP） |

### 新增基础设施

| 组件 | 部署方式 | 资源 |
|------|----------|------|
| Zitadel | Docker Compose | ~200MB RAM |
| OpenFGA | Docker Compose | ~100MB RAM |
| SMS 转发服务 | 内嵌到航小伴或独立 | ~10MB |

### 文档更新计划

| 文档 | 行动 |
|------|------|
| `docs/architecture/iam-architecture-design.md` | 已创建，本次迁移的主文档 |
| `docs/architecture/openfga-model.fga` | 已创建，OpenFGA 模型定义 |
| `docs/architecture/ecosystem-identity-and-authorization.md` | 重写：四层结构更新为 Zitadel + OpenFGA |
| `docs/architecture/README.md` | 更新索引 |
| `docs/modules/auth/01-casdoor-sso.md` | 重写为 `01-zitadel-sso.md` |
| `docs/modules/auth/README.md` | 更新 |
| `docs/modules/rbac/README.md` | 重写：简化模型说明 |
| `docs/modules/policy/01-authorization-model.md` | 重写：OpenFGA 模型 |
| `docs/modules/policy/02-policy-evaluation.md` | 重写：新决策链 |
| `docs/casdoor/README.md` | 标记废弃或删除 |
| `.trellis/spec/backend/authorization-architecture.md` | 重写 |
| `.trellis/spec/guides/ecosystem-identity-and-authorization.md` | 重写 |

## Implementation Plan

### Phase 1: 基础设施准备
- Zitadel Docker Compose 部署脚本
- OpenFGA Docker Compose 部署脚本
- Zitadel 初始化配置（组织/项目/应用/角色）
- OpenFGA 模型导入

### Phase 2: SSO 迁移（核心依赖，最大工作量）
- 标准 OIDC 客户端替换 Casdoor SDK
- Auth handler 重写（login/callback/refresh/logout）
- Auth 中间件重写
- 腾讯云短信转发服务
- 自定义登录 UI
- 用户迁移脚本

### Phase 3: 权限简化
- RoleCapabilities 静态映射
- 新中间件（Token → 角色 → 能力）
- 删除 RBAC 表和代码
- Admin 前端适配

### Phase 4: OpenFGA 集成
- Go 客户端封装
- 资源创建时写关系
- 操作前 Check 权限
- 志愿者/管理员权限区分

### Phase 5: 认证流程迁移
- 实名认证：结果存应用 DB（真相源），不写 Zitadel 元数据
- 学生认证：结果存应用 DB（真相源），异步同步 Zitadel 粗粒度角色（verified_student）
- 认证撤销：应用 DB 立即更新 + 异步移除 Zitadel 角色
- 前端适配：粗筛从 Token 角色读，精确判断查应用接口

### Phase 6: 清理和文档
- 删除旧代码
- 更新文档
- 更新 OpenAPI spec
- 最终测试
