---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-04-19
---

# 用户系统

> 状态：现行

## 两级认证

### 实名认证

确认"这个人是谁"。

**大陆身份证**：提交姓名 + 证件号 → 后端调用腾讯云二要素核验 → 加密后存入数据库。命中学籍表则自动通过，否则转人工。

**非大陆证件**：上传证件照片 → 人工审核。

提交流程分两步：
1. `POST /api/v1/user/identity/uploads` — 上传照片
2. `POST /api/v1/user/identity` — 提交证件信息 + 已上传照片 key

支持证件类型：`MAINLAND_ID` / `HK_MACAU` / `TW` / `PASSPORT`

### 学生认证

确认"这个人是否在校学生"。

**LDAP**：学号 + 密码 → 后端 LDAP bind → 成功自动通过 → 同步 `verified_student` 角色。

**手工表单**：学校配置动态表单 → 提交 → `pending` → 管理员审核。

## 状态机

实名：数据库使用 `Verified bool` 字段 + `RejectionReason`，API 层映射为：`none`（未提交）→ `pending`（已提交未审核）→ `approved`（Verified=true）/ `rejected`（RejectionReason 非空）。大陆身份证命中学籍表自动通过（auto-verify），其他证件类型转人工审核。

学生：`unverified` → `pending` → `verified` / `rejected`（rejected 可重新提交）

规则：`verified` 不重复提交，`pending` 不允许覆盖。

## 手机号绑定

两步流程：
1. `POST /api/v1/user/profile/bind-phone/otp` — 发送 OTP 验证码到目标手机号
2. `POST /api/v1/user/profile/bind-phone` — 提交手机号 + OTP 验证码完成绑定

仅限中国大陆手机号。绑定手机号不授予学生身份。

## QQ 绑定与机器人联动

用户在主站内生成一次性绑定码，再到 StuHelper QQ 机器人私聊发送 `绑定 <code>` 完成绑定。

核心规则：

- 一个 StuHelper 账号最多绑定一个 QQ 号。
- 一个 QQ 号最多绑定一个 StuHelper 账号。
- 绑定码一次性消费，默认有效期 10 分钟，过期后必须重新生成。
- 已经绑定的账号不再生成新绑定码，接口返回冲突。
- 机器人消费绑定码后，会立即拿到当前学生认证状态，用于受控群的自动放行、继续禁言或超时踢出。
- 机器人内部接口只接受服务令牌，不接受用户 Cookie、用户 JWT 或前端调用。

机器人联动返回的认证状态固定为：

- `unbound`：当前 QQ 号尚未绑定 StuHelper 账号
- `bound_unverified`：已绑定，但学生认证未通过
- `verified`：已绑定，且学生认证已通过

## 学校配置

`school_configs` 每校一条：
- `verification_method`：`ldap` 或 `manual`
- `ldap_config` / `manual_form_fields` / `consent_text`
- `academic_db_table` / `enabled`

## 系统配置

`system_configs` 全局 key-value：文案、预览长度、业务开关。

## PII 保护

- `doc_number_enc`：AES-256-GCM 密文
- `person_uid`：HMAC 稳定标识
- `doc_photo_*`：对象存储 key，审核时签发短时 URL

## 后台审核

实名认证审核、学生认证审核、学校配置管理、系统配置管理。

审核通过后更新应用数据库，必要时同步 Casdoor 角色投影。

## 端点

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/user/me` | GET | 用户综合概览（身份状态、认证状态、手机绑定、能力） |
| `/api/v1/user/identity` | GET / POST | 实名认证 |
| `/api/v1/user/identity/uploads` | POST | 上传证件照片 |
| `/api/v1/user/profile` | GET | 学生认证档案 |
| `/api/v1/user/profile/verify` | POST | 发起学生认证 |
| `/api/v1/user/profile/bind-phone/otp` | POST | 发送绑定手机验证码 |
| `/api/v1/user/profile/bind-phone` | POST | 绑定手机号 |
| `/api/v1/user/qq-binding` | GET | 获取当前用户 QQ 绑定状态 |
| `/api/v1/user/qq-binding/code` | POST | 生成一次性 QQ 绑定码 |
| `/api/v1/user/profile/academic-info` | GET | 学籍信息 |
| `/api/v1/bot/qq-binding/consume` | POST | 机器人消费绑定码并建立绑定 |
| `/api/v1/bot/qq-users/{qqID}/verification` | GET | 机器人按 QQ 号查询绑定与学生认证状态 |
| `/api/v1/user/schools` | GET | 学校列表 |
| `/api/v1/admin/identities` | GET / PUT | 实名审核 |
| `/api/v1/admin/student-verifications` | GET / PUT | 学生审核 |
| `/api/v1/admin/school-configs` | GET / PUT | 学校配置 |
| `/api/v1/admin/system-configs` | GET / PUT | 系统配置 |

## 代码入口

| 组件 | 位置 |
|------|------|
| 用户模块 | `server/internal/modules/user/` |
| LDAP 客户端 | `server/internal/modules/ldap/` |
| PII 加密 | `server/internal/pkg/crypto/pii/` |
| QQ 绑定前端入口 | `clients/web/src/modules/user/views/QQBindingPage.vue` |
