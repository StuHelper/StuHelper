# 身份认证模块

## 模块概述

统一身份认证模块负责用户身份验证、账号管理和安全存储。

## 认证方式

| 认证方式 | 状态 | 说明 |
|----------|------|------|
| Casdoor SSO | 🟢 已实现 | OAuth2/OIDC 单点登录 |
| LDAP 认证 | 🟡 原型阶段 | 学校 LDAP 直接验证（已有客户端原型） |

## 技术栈

| 组件 | 选型 | 说明 |
|------|------|------|
| SSO 服务 | Casdoor | OAuth2/OIDC 认证服务 |
| Token | JWT | Access Token + Refresh Token |
| 会话存储 | Redis | Token 黑名单、会话状态 |

## 文档索引

| 文档 | 说明 |
|------|------|
| [01-casdoor-sso.md](01-casdoor-sso.md) | Casdoor OAuth2 集成 |
| [02-ldap.md](02-ldap.md) | LDAP 认证设计 |
| [03-account.md](03-account.md) | 账号体系与第三方绑定 |
| [04-security.md](04-security.md) | 安全存储与会话管理 |
