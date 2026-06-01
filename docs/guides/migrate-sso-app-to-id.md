---
type: guide
audience: backend-dev, ops
status: deprecated
authoritative-source: docs/design/open-platform-v1.md
last-verified: 2026-05-30
superseded-by: docs/design/open-platform-v1.md
---

# 历史文档：不要迁移到 id.stuhelper.com

> 本指南已废弃。当前 B/2B 架构不再把第三方应用从 `sso.stuhelper.com` 迁移到 `id.stuhelper.com`。`id.stuhelper.com` 是 legacy disabled host，生产公网入口必须返回 404。

当前目标状态见 [StuHelper Open Platform v1](../design/open-platform-v1.md)：

- `https://sso.stuhelper.com` 是唯一公开登录认证系统和 OIDC issuer。
- 第三方应用直接接入 Casdoor discovery / authorization code + PKCE。
- StuHelper 业务数据通过 `https://stuhelper.com/api/open/v1/*` 获取。
- 学生认证、学校/校区、QQ 绑定、业务资料、授权审计和撤销都由 StuHelper Open Platform 管理。
- Casdoor `User.Properties` 不是第三方业务数据 API，最多作为低敏摘要缓存。
- `id.stuhelper.com` 不提供 OAuth endpoint、OIDC discovery、账号中心或旧路径兼容。

如果发现生产 runbook、应用配置或 Nginx 配置仍要求：

```text
issuer=https://id.stuhelper.com
authorization_endpoint=https://id.stuhelper.com/oauth2/authorize
WEB_VITE_IDENTITY_URL=https://id.stuhelper.com
IDENTITY_ISSUER=https://id.stuhelper.com
```

这是旧架构残留，必须改回 B/2B 配置：

```text
issuer=https://sso.stuhelper.com
WEB_VITE_IDENTITY_URL=
IDENTITY_ISSUER=
CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
```
