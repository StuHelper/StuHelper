# 授权与策略模块

这一组文档描述航小伴内部的业务授权，不描述统一身份登录。

## 当前状态

当前主干实现已经完成从 `isAdmin` 业务门禁到应用内 capability 授权的替换：

- 后台路由统一走 `rbac.RequirePermission(...)`
- `/auth/me` 返回应用内 `capabilities`
- 评课可见性和发评资格走访问事实判断

长期上仍然允许继续演进到资源关系授权，但那不是当前运行时的真相源。

## 负责什么

- 应用后台 capability
- 模块级后台门禁
- 基于 `schoolID`、学生认证、实名认证的访问事实
- 内容裁剪和发布限制

## 不负责什么

- SSO 登录和 Casdoor 会话
- 平台管理员
- 第三方应用 OAuth client 与 scope 管理

这些属于 [../auth/](../auth/) 和架构文档里的生态身份边界。

## 文档索引

| 文档 | 说明 |
| --- | --- |
| [01-hangxiaoban-authorization-model.md](01-hangxiaoban-authorization-model.md) | 当前能力模型和长期演进方向 |
| [02-policy-evaluation-order.md](02-policy-evaluation-order.md) | 后端统一授权判断顺序 |
