---
type: design
audience: backend-dev, frontend-dev, koishi-dev, product
status: approved-for-planning
authoritative-source: server/api/openapi.yaml after implementation
created: 2026-05-03
---

# Koishi 新生群入群认证与学生身份打通设计

## 背景

当前系统已经有 QQ 绑定码、学生认证、Koishi 入群禁言/提醒/解禁/踢出，以及对象存储。新需求是在 QQ 新生群中实现“先入群、禁言、认证、通过后解禁”的完整链路，并把老生认证、新生材料审核、QQ 绑定、Admin 后台和 QQ 管理群审批打通。

后期 QQ 加群问题可写成“访问网站 `buaa. team` 完成认证”，这是为了绕开 QQ 文案拦截和字数限制。`buaa.team` 只做短域名重定向工具，目标是 StuHelper 认证域名，例如 `https://auth.stuhelper.com/admission`。本系统和 `buaa.team` 没有身份、数据或权限关系。

## 目标

- 老生通过学校官方 SSO 或学校邮箱 OTP 任一方式获得正式学生身份。
- 新生只能通过录取通知书或录取证明材料人工审核获得临时学生身份。
- 新生临时身份在有效期内权限等同正式学生，但数据模型上不能伪装成正式 `verified_student`。
- `server/` 是身份、材料、审核、策略、过期、黑名单和审计的权威来源。
- Koishi 是 QQ 执行器：自动通过入群、禁言、提醒、转发材料、接收 QQ 审批指令、解禁、踢出和拉黑。
- 审批同时支持 QQ 管理群和 Admin 后台，两个入口共用同一后端审核服务。

## 非目标

- 不把 `buaa.team` 做成系统组成部分；它只是可替换的短域名重定向工具。
- 不把录取材料人工审核直接写成正式 `verified_student`。
- 不把 Koishi SQLite 作为身份或审核真源。
- 第一版不支持普通图片上传、相册选择、拖拽上传或 PDF 材料。
- URL query 中的 `qq` 参数只展示和一致性校验，不参与信任判断。

## 核心边界

`server/` 保存和裁决：

- 用户账号、QQ 绑定、正式学生认证、临时新生身份。
- 入群认证会话、材料、审核记录、群准入策略、失败累计和黑名单。
- 权限投影、通知和审计事件。

`bots/koishi/` 执行：

- 监听入群事件，自动同意，按后端策略禁言。
- 发送带 token 的认证链接。
- 拉取待提醒、待解禁、待踢出、待拉黑和待转发材料任务。
- 解析 QQ 审批指令并调用后端。
- 上报禁言、提醒、解禁、踢出、拉黑和转发材料结果。

Admin 后台执行：

- 查看待审材料、入群会话、QQ 绑定、失败次数、审核记录。
- 批准、驳回、设置单个申请的临时身份过期时间。
- 配置新生通道、群准入策略和黑名单。

## 身份等级

- `verified_student`：正式学生，来自学校官方 SSO、学校邮箱 OTP、现有 LDAP/学籍认证等强凭据。
- `freshman_provisional`：临时学生，来自新生录取材料人工审核，必须有 `expires_at`。

默认新生通道关闭和临时身份过期时间为当年 10 月 1 日 12:00。管理员可在策略中修改开关和时间；审批时可为单个申请设置 `+N 天` 过期覆盖值。

## 主流程

1. 新人申请加入 QQ 群，bot 自动同意。
2. Koishi 调后端创建入群认证会话。
3. 后端返回认证链接、禁言时长、等待截止时间、提醒间隔等策略。
4. Koishi 默认禁言 30 天，并 @ 新人发送认证链接和截止时间。
5. 用户打开链接；未登录则跳转 SSO 注册或登录，回调后回到 admission 页面。
6. 后端消费 token，将当前登录用户与 token 绑定的 QQ 会话关联，并在安全条件满足时自动建立 QQ 绑定。
7. 用户选择老生或新生认证路径。
8. 认证通过后，Koishi 扫描到状态变化并自动解禁。
9. 等待时间超时仍未通过时，Koishi 发送超时提醒、踢出并上报失败。
10. 同一 QQ 在同一群累计 3 次“进群但未认证超时被踢”后默认永久拉黑。

## 认证链接

认证链接的 canonical URL 使用 StuHelper 域名，例如：

```text
https://auth.stuhelper.com/admission/a/<code>?qq=123456789
```

QQ 加群问题或群内短文案可使用 `buaa. team` 引导用户访问短域名；`https://buaa.team/a/<code>?qq=123456789` 只做 302 到 canonical URL。

`qq` 参数只用于用户观察。后端必须校验 token 绑定的 QQ 与 `qq` 参数一致；不一致时拒绝打开。实际绑定和状态判断只使用 token 记录中的 `qq_id`。

入群 token 规则：

- 默认 1 小时有效，一次性消费。
- 绑定 `session_id + platform + guild_id + qq_id`。
- 数据库只保存 `token_hash`。
- 消费后不能转给另一个用户。
- 登录用户未绑定 QQ 且 token QQ 未绑定其他用户时自动绑定。
- 登录用户已绑定同一 QQ 时继续流程。
- 登录用户绑定其他 QQ 或 token QQ 已绑定其他用户时拒绝覆盖，并引导回退到现有私聊 `绑定 <code>`。

## 数据模型

### `user_verification_credentials`

记录用户通过了哪些认证凭据，不直接等同角色。

字段：`id`, `user_id`, `kind`, `school_id`, `subject_hash`, `subject_display`, `status`, `verified_at`, `expires_at`, `metadata`, `created_at`, `updated_at`。

`kind` 取值：`school_sso`, `school_email_otp`, `freshman_material_manual`。正式学生凭据通常无 `expires_at`；新生材料凭据必须有 `expires_at`。

### `group_admission_sessions`

记录一次 QQ 入群认证会话。

字段：`id`, `policy_id`, `platform`, `guild_id`, `channel_id`, `qq_id`, `qq_nickname`, `user_id`, `status`, `token_hash`, `token_expires_at`, `verification_wait_deadline_at`, `initial_mute_until`, `next_reminder_at`, `last_reminder_at`, `reminder_count`, `muted_at`, `released_at`, `kicked_at`, `kick_warned_at`, `post_kick_notice_sent_at`, `blacklist_applied_at`, `last_bot_error`, `created_at`, `updated_at`。

`status` 取值：`joined_muted`, `linked`, `verified`, `expired_kicked`, `cancelled`。

### `freshman_verification_applications`

记录新生材料申请和审核状态。

字段：`id`, `user_id`, `school_id`, `admission_session_id`, `applicant_name`, `applicant_name_masked`, `department_or_major`, `material_type`, `status`, `provisional_expires_at`, `reviewed_by_user_id`, `reviewed_by_qq_id`, `review_source`, `review_reason`, `created_at`, `updated_at`, `reviewed_at`。

审批通过时，后端在同一事务里写入 `user_verification_credentials(kind=freshman_material_manual)` 并触发权限投影。

### `freshman_verification_materials`

记录摄像头拍摄材料。

字段：`id`, `application_id`, `object_key`, `filename`, `content_type`, `size_bytes`, `sha256`, `uploaded_at`。

第一版只允许摄像头拍摄图片，不提供普通文件上传入口，不接受 PDF。

### `group_admission_policies`

记录群准入策略，按 `platform + guild_id` 生效；没有群级配置时用全局或学校默认值。

默认值：

- `auto_approve_join`: `true`
- `initial_mute_duration_seconds`: 30 天
- `verification_token_ttl_seconds`: 1 小时
- `verification_wait_timeout_seconds`: 1 小时
- `reminder_interval_seconds`: 10 分钟
- `failed_join_limit`: 3 次
- `blacklist_duration_seconds`: `NULL`，表示永久拉黑
- `freshman_channel_closes_at`: 当年 10 月 1 日 12:00
- `freshman_default_expires_at`: 同 `freshman_channel_closes_at`
- `forward_raw_material_to_qq`: 显式开启后才转发原始材料

还需配置提醒文案、踢出提醒文案、管理群 ID、材料转发开关、新生通道开关和审批延长天数上限。

### `group_admission_failures`

聚合同一 QQ 在同一群内的失败次数和黑名单状态。

字段：`platform`, `guild_id`, `qq_id`, `failure_count`, `last_failed_at`, `blacklisted_at`, `blacklist_expires_at`, `reason`, `created_at`, `updated_at`。

默认永久拉黑使用 `blacklist_expires_at = NULL`。管理员可手动解除。

## 认证链路

老生认证二选一通过：

- 学校官方 SSO 登录成功。
- 学校邮箱 OTP 验证成功。

两者都写入 `user_verification_credentials`，但 `kind` 不同。邮箱 OTP 需要学校域名白名单、Redis OTP、冷却、尝试次数和频率限制。邮箱明文不应无必要长期存储，优先保存 hash 和脱敏展示值。

新生通道必须满足：

- 当前时间早于 `freshman_channel_closes_at`。
- 策略启用 `freshman_channel_enabled`。
- 用户没有正式学生身份。
- 用户没有同学校 pending 新生申请。

材料入口只允许摄像头拍摄。移动端使用 camera capture 或 WebRTC；桌面端无摄像头时明确提示换手机打开。后端拒绝 PDF、超大文件、伪装 content type 和非图片内容。

## API 形状

用户侧接口：

- `GET /api/v1/admission/sessions/{token}`
- `POST /api/v1/admission/sessions/{token}/link`
- `GET /api/v1/admission/me`
- `POST /api/v1/admission/freshman/applications`
- `POST /api/v1/admission/freshman/applications/{id}/camera-captures`
- `POST /api/v1/admission/school-email/request-otp`
- `POST /api/v1/admission/school-email/verify-otp`
- `GET /api/v1/admission/school-sso/{schoolID}/login`
- `GET /api/v1/admission/school-sso/{schoolID}/callback`

Bot 内部接口：

- `POST /api/v1/bot/admission/sessions`
- `GET /api/v1/bot/admission/qq-users/{qqID}/access`
- `GET /api/v1/bot/admission/sessions/pending`
- `POST /api/v1/bot/admission/sessions/{id}/events`
- `GET /api/v1/bot/admission/freshman/applications/pending-forward`
- `POST /api/v1/bot/admission/freshman/applications/{id}/forwarded`
- `POST /api/v1/bot/admission/freshman/applications/{id}/review`

Admin 接口：

- `GET /api/v1/admin/admission/policies`
- `PUT /api/v1/admin/admission/policies/{id}`
- `GET /api/v1/admin/admission/sessions`
- `GET /api/v1/admin/freshman-verifications`
- `GET /api/v1/admin/freshman-verifications/{id}`
- `PUT /api/v1/admin/freshman-verifications/{id}`
- `POST /api/v1/admin/admission/blacklist/{qqID}/release`

所有接口先改 OpenAPI，再生成 Go 和 TypeScript 类型。Bot 接口只接受服务令牌；Admin 接口走 capability 和 MFA；用户侧写接口走登录态与 CSRF。

## Koishi 改动

`stuhelper-group-guard`：

- 入群后创建后端 admission session。
- 按策略禁言 30 天。
- 发送带 `qq=` 的认证链接。
- 从后端拉取待提醒、待解禁、待踢出和待拉黑任务。
- 上报执行结果。

`stuhelper-binding`：

- 保留现有 `绑定 <code>`。
- 绑定成功后可触发一次当前 pending admission session 状态刷新。

`stuhelper-admin`：

- 新增 `新生审核通过 <申请ID>`、`新生审核通过 <申请ID> +30d`、`新生审核驳回 <申请ID> <原因>`、`新生审核查看 <申请ID>`、`新生黑名单解除 <qqID>`。
- Koishi 做本地群管权限校验，后端再校验操作者 QQ 是否映射到有权限的 StuHelper 管理员。

## Admin 后台改动

新增“新生认证”模块：

- 列表筛选：状态、学校、群、时间、QQ、失败次数。
- 详情展示：用户、QQ 绑定、入群会话、材料原图、历史审核记录和风险提示。
- 操作：批准、批准并设置 `+N 天`、驳回、解除黑名单。

新增“入群认证策略”模块：

- 新生通道开关。
- 默认关闭时间和临时身份过期时间。
- 入群禁言时长、token TTL、认证等待时间、提醒间隔。
- 失败拉黑阈值和默认永久拉黑。
- 管理群 ID 和是否转发原始材料。

## 权限与审计

新增 capability：

- `admission:policy:read`
- `admission:policy:update`
- `admission:freshman:read`
- `admission:freshman:review`
- `admission:blacklist:manage`

审核审计必须记录申请、目标用户、目标 QQ、审批入口、操作者用户、操作者 QQ、动作、延长天数、过期时间、原因和 QQ 指令原文。

Bot 执行动作必须记录禁言、提醒、解禁、踢出、拉黑和材料转发结果。失败必须暴露到 `last_bot_error`，不能吞掉。

## 验证策略

后端：

- token 消费、QQ 冲突、老生认证、新生申请、审批、过期、失败拉黑的状态机单测。
- 新表约束、唯一索引和事务一致性的 repository 集成测试。
- OpenAPI route contract、Bot service credential 权限测试、邮箱 OTP 限流测试。
- 摄像头材料接口拒绝 PDF、超大文件、伪装 content type 和非图片内容。
- `freshman_provisional` 未过期时能力等同正式学生，到期后撤销。

Koishi：

- 入群创建会话、禁言 30 天、发送带 `qq=` 的链接。
- 每 10 分钟提醒，1 小时等待超时后提醒并踢出，3 次失败后永久拉黑。
- 已通过后解禁，QQ 审核命令成功和失败都有明确反馈。
- 多 bot 场景按 `platform + botSelfId` 路由执行。

前端：

- admission 页面未登录跳 SSO。
- token mismatch 拒绝。
- `qq` 参数只展示，不作为信任依据。
- 摄像头不可用时明确失败。
- 不显示文件上传入口。
- Admin 列表、详情、审批和策略配置可用。

## 实施分解建议

1. 后端数据模型、OpenAPI 和 admission 状态机。
2. 用户 admission 页面、老生认证和摄像头材料提交。
3. Admin 新生审核和策略配置。
4. Koishi admission session、提醒、解禁、踢出、拉黑。
5. QQ 管理群材料转发和指令审批。
6. 过期撤销、能力投影和端到端验证。
