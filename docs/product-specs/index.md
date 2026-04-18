# 产品规格

按业务域拆分的功能规格，描述当前代码库已实现的功能与业务规则。
本页只保留域索引；角色说明以 `/Users/zxy/Code/StuHelper/docs/PRODUCT.md` 为准，能力/授权细节以对应规格页与 `/Users/zxy/Code/StuHelper/docs/product-specs/rbac-authorization.md` 为准。

## 域索引

| 域 | 权威规格 | 备注 |
|----|----------|------|
| 认证 | [auth-sso.md](auth-sso.md) | 登录、回调、会话、手机号 OTP |
| 课程与评课 | [course-review.md](course-review.md) | 课程实体、评课、回复、举报、收藏 |
| 用户系统 | [user-system.md](user-system.md) | 实名、学生认证、手机号绑定、学校配置 |
| 授权 | [rbac-authorization.md](rbac-authorization.md) | capability、组织 scope、OpenFGA |
| 通知 | [notification.md](notification.md) | 通知列表、未读数、SSE |
| 审计 | [audit-logging.md](audit-logging.md) | 审计日志与留痕 |
