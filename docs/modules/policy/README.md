# 授权与策略模块

这一组文档描述航小伴内部的业务授权模型，而不是统一身份登录。

## 状态

🟡 目标架构已明确，当前代码仍处于从“本地简单 RBAC + `isAdmin`”向“应用级授权 + 资源关系 + 属性规则”收敛的过程中。

## 负责什么

- 航小伴应用级管理员
- 模块级管理员
- 课程、分类、内容级授权关系
- owner / teacher-of-course 默认策略
- 基于 `schoolID`、学生/老师、实名认证、学生认证的访问规则
- 部分可见、发布限制、风险标签等内容策略

## 不负责什么

- 登录、注册、单点登录
- OAuth/OIDC 应用接入
- 第三方应用 client 和 callback 管理
- 平台级管理员

这些都属于 `docs/modules/auth/`。

## 文档索引

| 文档 | 说明 |
| --- | --- |
| [01-hangxiaoban-authorization-model.md](01-hangxiaoban-authorization-model.md) | 航小伴的最终授权模型：角色、关系、属性、内容策略 |
| [02-policy-evaluation-order.md](02-policy-evaluation-order.md) | 统一授权判断顺序：认证、粗粒度能力、关系、属性、内容整形 |
