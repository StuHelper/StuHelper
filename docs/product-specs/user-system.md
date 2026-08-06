---
type: product-spec
audience: product, backend-dev, frontend-dev
status: current
authoritative-source: server/api/openapi.yaml + docs/product-specs/student-verification-and-group-admission.md
last-verified: 2026-08-05
---

# 用户系统

> 状态：现行。学生认证、手机号验证和群聊准入的完整业务规则以
> [学生认证与群聊准入](student-verification-and-group-admission.md) 为准。

## 1. 领域边界

用户系统只负责当前登录账号的聚合展示、QQ 绑定和账号级资料入口，不再保存或暴露旧的
`verified=true` 学生档案，也没有独立的旧“实名认证”申请域。

- Casdoor 是登录身份、会话和规范化手机号的唯一可写权威源。
- PostgreSQL 保存 StuHelper 业务用户、Casdoor 手机号的受保护只读投影、手机号验证凭据和 QQ 绑定。
- 学生身份由独立学生认证域的活动凭据实时派生；姓名、证件号、手机号或 QQ 绑定都不能单独授予学生身份。
- 群聊 admission 只消费最小学生资格与平台绑定结果，不写用户学生认证状态。
- OpenFGA 是可重建资源关系投影，不是账号、学生凭据或手机号真源。

## 2. 当前账号聚合面

`GET /api/v1/user/me` 返回账号中心需要的最小聚合信息：

- 展示名和可选头像；
- 当前手机号的脱敏展示值；
- `studentVerificationStatus: none | approved`，由活动学生凭据实时派生；
- `phoneBound`，只表示当前账号存在规范手机号，不替代手机号验证凭据；
- 当前 capability 列表。

学生资格、手机号验证和 QQ 绑定是三个独立响应。前端不得通过某一项推断另外两项，也不得在请求失败或
首次加载完成前把未知状态显示成“未认证”或“未绑定”。

## 3. 学生认证

学生认证使用独立的 `/api/v1/student-verification/*` API、学校适配器、版本化本地学籍快照、人工材料
审核和有期限凭据。主要方法包括：

- 用户可见的“实名信息校验”；
- 每校独立适配的学校 SSO；
- 学号邮箱接收验证码或主动向指定邮箱发送挑战；
- 可复用的人工材料审核。

旧 `/api/v1/user/identity*`、`/api/v1/user/profile*`、`/api/v1/user/schools` 以及旧 Admin 身份/学生/
新生审核接口已经停用，不是兼容入口。上线迁移直接清空旧学生认证事实，不迁移、不备份、不双读、不
生成兼容凭据；用户需要通过新系统重新认证。

## 4. 手机号验证

手机号是账号级发布风控门槛，不是学生身份证据。学生认证页面不强制绑定手机号。

绑定或换号时用户必须当次手工输入中国大陆手机号。服务端创建有状态 operation，并自行选择路径：

1. 北航用户的规范学号、姓名和手工输入号码与当前活动学籍快照一致时，以
   `school_roster_phone_match` 完成，可免短信；
2. 其他情况不向用户泄露学校是否保存该号码，统一切换到 `sms_possession` 短信验证码；
3. 验证成功后先更新 Casdoor，再回读规范值并刷新 PostgreSQL 受保护投影与手机号凭据；
4. 任一阶段部分失败时保持发布门槛关闭，由幂等 operation 和对账任务继续处理；本地投影不得反向覆盖
   Casdoor；
5. 换号和解绑仍通过 Casdoor 完成，同时递增 revision、撤销旧凭据并失效消费方缓存。

内容发布服务只调用独立手机号门槛契约，不直接读取 Casdoor 表、用户手机号投影或学籍手机号。

## 5. QQ 绑定与机器人联动

用户在主站生成一次性绑定码，再到 StuHelper QQ 机器人私聊发送 `绑定 <code>` 建立绑定。

- 一个 StuHelper 账号最多绑定一个 QQ 号，一个 QQ 号最多绑定一个 StuHelper 账号，由数据库唯一约束
  保证。
- 绑定码一次性消费并带明确有效期；过期、已消费或账号冲突均 fail closed。
- QQ 绑定不授予学生身份；机器人查询时由服务端实时组合绑定与当前学生资格。
- 机器人内部接口只接受独立服务凭据，不接受用户 Cookie、用户 JWT 或浏览器直接调用。
- QQ 申请入群会话属于 admission 域；学生认证系统不保存群号、群策略或 admission session。

机器人查询结果为：

- `unbound`：QQ 尚未绑定 StuHelper 账号；
- `bound_unverified`：已经绑定，但当前没有合格学生凭据；
- `verified`：已经绑定，且当前学生资格满足查询策略。

## 6. 隐私与错误边界

- 普通账号聚合响应不返回姓名、证件号、完整手机号、学校密码、人工材料或内部学籍字段。
- 完整手机号的可写操作只发生在 Casdoor 适配层；日志、Trace、指标 label、审计摘要和 outbox 不保存
  完整号码或号码 HMAC。
- 用户可见的学生认证文案使用中性、真实的“实名信息校验”，不披露 Oracle、表名、字段名或内部匹配
  顺序，也不虚构未实际调用的第三方 provider。
- 账号、学籍主体或手机号冲突统一提供不可枚举错误和人工核查入口，不披露另一账号信息。

## 7. 当前端点

| 路径 | 方法 | 说明 |
|---|---|---|
| `/api/v1/user/me` | GET | 当前账号最小聚合面 |
| `/api/v1/user/qq-binding` | GET | 当前 QQ 绑定；未绑定时 `data: null` |
| `/api/v1/user/qq-binding/code` | POST | 生成一次性 QQ 绑定码 |
| `/api/v1/bot/qq-binding/consume` | POST | Koishi 消费绑定码并建立绑定 |
| `/api/v1/bot/qq-users/{qqID}/verification` | GET | 按 QQ 查询绑定和实时学生资格 |
| `/api/v1/student-verification/schools` | GET | 当前可用学校与方法能力 |
| `/api/v1/student-verification/applications` | POST | 创建独立学生认证申请 |
| `/api/v1/student-verification/credentials` | GET | 查询当前用户学生凭据 |
| `/api/v1/student-verification/eligibility` | GET | 查询最小实时学生资格 |
| `/api/v1/account/phone` | GET / DELETE | 查询手机号状态或发起解绑 |
| `/api/v1/account/phone/operations` | POST | 创建手机号绑定 operation |
| `/api/v1/account/phone/change-operations` | POST | 创建换号 operation |
| `/api/v1/admin/student-verification/*` | 多种 | 学校方法、快照、审核、凭据、冲突与连接器控制面 |
| `/api/v1/admin/system-configs` | GET / PUT | 非认证专属的系统配置 |

字段、状态码和全部子路径以 `server/api/openapi.yaml` 为权威来源。

## 8. 代码入口

| 组件 | 位置 |
|---|---|
| 账号与 QQ 绑定 | `server/internal/modules/user/` |
| 学生认证与手机号凭据 | `server/internal/modules/studentverification/` |
| 校园连接器网关 | `server/internal/modules/campusconnector/` |
| 校园侧连接器 | `server/internal/campusconnector/`、`server/cmd/campus-connector/` |
| 群聊准入 | `server/internal/modules/admission/` |
| 主站账号入口 | `clients/web/src/modules/user/` |
| 管理控制面 | `clients/admin/apps/web-ele/src/views/users/` |
