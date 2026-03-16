# 身份认证与开放平台接入

这一组文档只回答“身份从哪里来、怎样安全地接入、怎样最小化地开放给其他应用”。

它不负责定义航小伴内部的业务授权模型。
航小伴的业务授权文档在 `docs/modules/policy/`。

## 当前定位

- `sso.stuhelper.com` 使用 Casdoor 作为统一身份平台
- 航小伴和未来开发者平台都是接入该 SSO 的应用
- 第三方应用也会通过 OAuth/OIDC 接入

## 这个模块负责什么

- 登录、注册、单点登录
- OAuth/OIDC 接入
- 会话与 Token
- 账号绑定与基础安全
- 对外应用可申请的最小身份事实与 scope

## 这个模块不负责什么

- 航小伴课程级、分类级、内容级管理员
- 学校、学生/老师、认证状态组合出的业务权限
- 内容所有者、任课教师等资源关系

这些都属于应用业务授权。

## 文档索引

| 文档 | 说明 |
| --- | --- |
| [01-casdoor-sso.md](01-casdoor-sso.md) | StuHelper 生态中的 Casdoor SSO 边界与接入方式 |
| [02-ldap.md](02-ldap.md) | 学校 LDAP 认证相关设计与现状 |
| [03-account.md](03-account.md) | 账号体系、账号绑定、身份事实边界 |
| [04-security.md](04-security.md) | 会话、Token、安全基线 |
| [05-open-platform-claims-and-scopes.md](05-open-platform-claims-and-scopes.md) | 第三方应用能申请哪些身份事实、如何最小化开放 |
