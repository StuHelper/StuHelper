---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-07-31
---

# 用户系统

> 状态：现行

## 两级认证

### 实名认证

确认"这个人是谁"。

**大陆身份证**：提交姓名 + 证件号 → 加密后存入数据库。只有当前账号已经完成学生认证，并且在该账号所属学校的学籍表中同时满足“证件记录一致、姓名一致、命中学号属于当前账号”时才自动通过；缺少任一账号绑定证据时进入人工审核。当前实现不调用第三方二要素核验服务。

**非大陆证件**：上传证件正面照和手持/自拍照 → 人工审核；背面照可选。

**人工审核材料门槛**：任何进入人工审核的申请都必须同时具备正面照和手持/自拍照。大陆身份证可以先不上传照片尝试自动匹配，但只有事务内锁定并复核学籍绑定仍满足自动通过条件时才会成功；不能自动通过时直接返回“需要照片”，不会创建一个缺少材料的 pending 申请。

提交流程分两步：
1. `POST /api/v1/user/identity/uploads` — 上传照片
2. `POST /api/v1/user/identity` — 提交证件信息 + 已上传照片 key

支持证件类型：`MAINLAND_ID` / `HK_MACAU` / `TW` / `PASSPORT`

照片只接受 JPEG、PNG 或 WebP，单文件不超过 5 MiB；每个已登录用户最多上传 12 次/24 h。后端只接受由当前用户上传、槽位匹配且对象存储可验证的私有 key，不接受外部 URL 或跨用户 key。

### 学生认证

确认"这个人是否在校学生"。

**LDAP**：学号 + 密码 → 后端 LDAP bind → 成功自动通过 → 同步 `verified_student` 角色。

**学籍邮箱 OTP**：学号 + 姓名 → 按学校代码路由到外部只读学籍源 → 匹配后由后端固定派生学校邮箱 → 邮箱 OTP 通过后自动认证。客户端不能替换后端派生的邮箱；学籍源不可用时返回服务不可用，不得作为“不匹配”处理。同一用户/学校的 Redis 冷却键已经存在时返回 429；Redis transport failure 返回 503，不得把依赖故障伪装成“请稍后重发”的冷却命中。

Admission 的 `academic-match` 与 `request-otp` 都会在 OTP 冷却检查前访问外部学籍源，因此在
认证之后共用同一个 Redis 用户预算：每用户 5 次/分钟，两条路由合计计数；第 6 次返回 429，
Redis 限流依赖不可用时返回 503，且不得继续调用外部学籍源。`verify-otp` 不访问学籍源，保留
自身的 OTP 尝试次数状态机，不占用这项查询预算。该预算与 OTP 的同用户/学校 60 秒发送冷却
是两层不同约束。

**手工表单**：学校配置动态表单 → 提交 → `pending` → 管理员审核。

学号只接受 1–50 位 ASCII 字母、数字、点、下划线和连字符，且首位必须是字母或数字；姓名规范化后最多 80 个 Unicode 字符，并拒绝控制字符、零宽格式字符和私用区字符。相同学号返回冲突姓名、空姓名或不一致学号时，学籍匹配失败关闭。

## 状态机

实名：数据库使用 `Verified bool` 字段 + `RejectionReason`，API 层映射为：`none`（未提交）→ `pending`（已提交未审核）→ `approved`（Verified=true）/ `rejected`（RejectionReason 非空）。大陆身份证仅在“已认证学生账号 + 同校 + 账号绑定学号 + 姓名 + 证件记录”全部一致时自动通过；自动匹配结果会在写入事务内再次锁定并校验学生档案和学校配置。其他情况转人工审核。

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
- `manual_form_fields.admission.emailIdentityPolicy`：学籍邮箱策略、邮箱域和姓名匹配要求

## 系统配置

`system_configs` 全局 key-value：文案、预览长度、业务开关。

## PII 保护

- `doc_number_enc`：AES-256-GCM 密文
- `person_uid`：HMAC 稳定标识
- `doc_photo_*`：对象存储 key，审核时签发短时 URL

## 后台审核

实名认证审核、学生认证审核、学校配置管理、系统配置管理。

实名认证列表不返回材料 key 或签名 URL。审核员必须先通过全局 `user:identity:read` capability 和 step-up MFA，再调用详情端点按需获取短时签名 URL；每次成功查看都会写入材料访问审计。批准操作要求 `user:identity:review` capability、step-up MFA，以及完整且可从对象存储验证的正面照和手持/自拍照。

审核通过后更新应用数据库；学生/新生能力由 DB 业务事实派生。Casdoor 不得写入或同步学生、
新生或 scoped admin 业务角色；目标 StuHelper organization `IsAdmin` 到 `super_admin` 的窄映射
由 ADR-0009 单独定义，与学生认证流程无关。

## 账号资料状态呈现

`/account/profile` 的基础账号资料和邮箱来自当前已认证用户信息；手机号、QQ、实名与学生认证
状态则必须以验证状态接口成功返回为准。首次加载完成前不得把未知状态显示成“未验证”或
“未绑定”；请求失败时保留可靠的账号/邮箱信息，隐藏未经确认的负面结论，并显示可重试的
错误状态。只有一次完整状态读取成功后，页面才可展示“未验证”“未绑定”等确定性结果。

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
| `/api/v1/admin/identities` | GET | 实名审核列表（不含材料） |
| `/api/v1/admin/identities/{userID}` | GET / PUT | 按需读取签名材料 / 提交实名审核决定 |
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
