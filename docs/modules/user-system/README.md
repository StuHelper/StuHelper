# 用户系统模块

用户系统负责身份审核和校内身份事实，不负责 SSO 登录。

## 代码范围

| 代码位置 | 作用 |
| --- | --- |
| `server/internal/modules/user` | 实名认证、学生认证、学校配置、系统配置 |
| `server/internal/modules/ldap` | LDAP 校验与用户信息查询 |
| `server/internal/pkg/crypto/pii` | 证件号加密 |

## 当前能力

| 子域 | 说明 |
| --- | --- |
| 实名认证 | 提交证件信息、自动学籍匹配、后台审核 |
| 学生认证 | LDAP 校验或手动审核 |
| 学校配置 | 学校认证方式、手动表单字段、同意文案 |
| 系统配置 | 后台可维护的全局配置项 |

## 当前规则

- 学生认证前必须先完成实名认证
- 证件号不落明文，只保存密文和 `person_uid`
- 学校配置更新走合并更新，不会把未提交字段清空
- 学生认证后台和实名认证后台都走应用内 capability

## 接口入口

- 公开与用户态接口见 [../../reference/api-overview.md](../../reference/api-overview.md)
- 后台接口在 `/api/v1/admin/identities`、`/student-verifications`、`/school-configs`、`/system-configs`
