---
type: adr
audience: maintainers, backend-dev, ops
status: current
authoritative-source: server/internal/platform/casdoor + server/internal/pkg/oidc
last-verified: 2026-07-30
---

# ADR-0007: Casdoor 作为唯一身份提供方

**Date**: 2026-05-01
**Status**: accepted
**Deciders**: 项目 owner

## Context

StuHelper 需要一个身份提供方承担用户生命周期、登录方式、MFA、会话和 OIDC token 签发。早期设计曾评估 Zitadel 与 Keycloak，仓库中一度存在面向 Zitadel 的实现与迁移计划。

需要一条决策记录固定最终形态，避免后续开发把已被放弃的 Zitadel 路线当作现状或目标。

## Decision

Casdoor 是唯一身份提供方，部署在 `sso.stuhelper.com`，作为公开 OIDC issuer。

该选择由项目 owner 直接决定，不做比较型选型分析。Zitadel 与 Keycloak 不作为候选保留。

切换性质为绿地架构：不做兼容数据迁移，历史 external subject、session 与 token 全部失效。

Casdoor 的职责严格限定在身份侧。业务授权不经过 Casdoor：

- Casdoor 提供主体认证、会话、token 签发和登录层 MFA 证据。
- 业务事实（实名认证、学生认证、学校归属、QQ 绑定、资源归属）以 StuHelper 数据库为准。
- 管理员角色及 scope 以 PostgreSQL 授权账本为唯一管理真源，资源关系投影到 OpenFGA。
- Casdoor 中的业务 role claim 和 role membership 不参与 StuHelper allow/deny。
- 这些事实由 Authorization Service 组合，作为业务模块的唯一授权入口。

禁止向业务模块暴露 Casdoor 的 Casbin、Enforce 或 GetPermissions 能力。
授权控制面的完整决策见 [ADR-0008](0008-postgresql-authorization-control-plane.md)。

## Alternatives Considered

### Zitadel
- **Pros**: org 原生多租户模型，org-scoped role claim 可直接承载学校作用域。
- **Cons**: 把学校作用域嵌入 IDP 的 role claim，使业务授权事实分散到身份层；access token 与 ID token 的 claim 配置模型更复杂。
- **Why not**: owner 直接决策不采用。作用域必须来自数据库与 OpenFGA 投影，不从 role 名解析。

### Keycloak
- **Pros**: 成熟度与生态最广。
- **Cons**: 运维面与配置复杂度高于本项目规模所需。
- **Why not**: owner 直接决策不采用。

## Consequences

### Positive
- 身份层与授权层边界清晰：IDP 只管身份，授权决策集中在 Authorization Service。
- IdP token 不再承载 StuHelper 授权事实，角色与作用域来自可审计的数据库账本。
- 公开 OIDC issuer 与业务开放平台职责分离。

### Negative
- Casdoor 把 access token 与 ID token 签成共享同一 claim payload 的 JWT，因此两者都必须执行字段最小化，不能只约束 ID token。

### Risks
- Casdoor 作为身份真源，本地用户投影存在漂移可能。缓解方式是周期性 reconciliation 与漂移指标。
- `sso.stuhelper.com` 属于独立运行的 SSO 系统，不在本仓库的修改范围内。
