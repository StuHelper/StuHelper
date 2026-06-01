---
type: design
audience: backend-dev, frontend-dev, ops, product
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-05-30
supersedes: id-stuhelper-identity-issuer-design
---

# StuHelper Open Platform v1

> 当前决策：采用 B/2B 架构。`sso.stuhelper.com` 的 Casdoor 是唯一公开登录认证系统和 OIDC issuer；`stuhelper.com` 承载账号中心、开放平台、授权应用、开发者应用、学生认证和 QQ 绑定；`join.stuhelper.com` 只承载加群验证业务闭环；`id.stuhelper.com` 是 legacy disabled host，公网入口必须返回 404。

## 核心边界

| 组件 | 职责 | 不做什么 |
|------|------|----------|
| Casdoor / `sso.stuhelper.com` | 登录、注册、MFA、上游身份源、OIDC/OAuth issuer、token 签发、基础 scope consent | StuHelper 业务事实真源、学生认证审核、QQ 绑定真源、加群验证流程、第三方业务数据 API |
| StuHelper API / `stuhelper.com` | 主站、账号中心、学生认证、QQ 绑定、开放平台 app registry、业务 scope 审批、用户业务授权、Open API、审计、撤销 | 签发独立公开 identity issuer、保存原始登录密码、伪装 Casdoor |
| Join / `join.stuhelper.com` | 加群验证入口和业务闭环，唯一公开链接为 `https://join.stuhelper.com/verify/<token>?qq=<qq>` | 登录系统、第三方开放平台、旧 `/verify` 兼容入口 |
| Legacy ID / `id.stuhelper.com` | 生产入口返回 404，避免用户和应用继续进入旧 identity 方案 | 普通用户入口、OIDC discovery、OAuth endpoint、账号中心、Casdoor 反代 |

## 第三方接入模型

第三方应用直接信任 Casdoor issuer：

```text
issuer = https://sso.stuhelper.com
authorization endpoint = Casdoor discovery 返回值
token endpoint = Casdoor discovery 返回值
jwks = Casdoor discovery 返回值
```

第三方应用不能直接调用 Casdoor Admin API，也不能把 Casdoor `User.Properties` 当作 StuHelper 业务数据接口。登录 token 只证明“用户是谁”和“应用获得了哪些 scope”；学生认证、学校/校区、QQ 绑定、授权记录和业务资料必须通过 StuHelper Open API 读取。

```text
第三方应用
  → sso.stuhelper.com OIDC authorization code + PKCE
  → 得到 Casdoor token，iss=https://sso.stuhelper.com
  → 调用 https://stuhelper.com/api/open/v1/*
  → StuHelper 校验 token、app、scope、用户授权和业务状态
  → 返回按 scope 裁剪后的业务数据
```

StuHelper 后端校验至少包括：

- token issuer 必须是配置的 `CASDOOR_ISSUER=https://sso.stuhelper.com`。
- `client_id` / `aud` / `azp` 必须能映射到已批准的开放平台应用。
- redirect URI、应用状态、密钥状态、scope 审批状态必须仍然有效。
- 用户必须对请求的业务 scope 存在有效 consent，除非该接口是明确的 app-only 资源接口。
- API 响应必须按 granted scopes 做字段级裁剪，不能因为 token 含有 `profile` 或 Casdoor `Properties` 就泄漏业务字段。

## Scope 与数据披露

不要定义笼统的 `properties` scope。授权显示和 API 裁剪都基于业务 scope，而不是 Casdoor `User.Properties` 的原始 key。

| Scope | 授权页展示 | 允许返回的典型字段 |
|-------|------------|--------------------|
| `profile.basic.read` | 读取基础资料 | 昵称、头像、公开 profile 摘要 |
| `email.read` | 读取邮箱地址 | 邮箱、邮箱验证状态 |
| `phone.read` | 读取手机号 | 脱敏或明文手机号，按应用审批结果决定 |
| `stu.student.status.read` | 读取学生认证状态 | `verified_student`、`freshman_provisional`、过期时间摘要 |
| `stu.student.school.read` | 读取认证学校与校区 | 稳定开放学校代码、校区代码、展示名 |
| `stu.qq.status.read` | 读取 QQ 绑定状态 | `qq_bound` |
| `stu.qq.number.read` | 读取 QQ 号码 | QQ 号；敏感 scope，默认不授予普通第三方应用 |
| `resource.read` | 读取已授权资源 | 具体资源仍由 OpenFGA / 业务授权表判断 |
| `resource.write` | 写入已授权资源 | 具体资源仍由 OpenFGA / 业务授权表判断 |

标准 OIDC scope 可以在 StuHelper 内部映射为业务 scope：

| OIDC scope | StuHelper disclosure scope |
|------------|----------------------------|
| `profile` | `profile.basic.read` |
| `email` | `email.read` |
| `phone` | `phone.read` |

`openid` 只表示登录身份，不等价于业务数据授权。`offline_access` 只表示 refresh token 能力，不等价于任何 disclosure scope。

## 授权页面

授权页显示的是 scope 的人类可读名称和说明，不显示原始数据库列名，也不显示 Casdoor `Properties` key。

推荐显示形式：

```text
应用「示例应用」请求获取：

- 登录并识别你的 StuHelper 账号
- 读取你的基础资料
- 读取你的邮箱地址
- 读取你的学生认证状态
- 读取你的认证学校与校区
- 读取你的 QQ 绑定状态
```

如果当前 Casdoor 版本和应用类型能够稳定表达自定义业务 scope、展示 display name / description，并把最终授权结果作为可验证 scope 传给 StuHelper API，可以把上述 scope 同意页放在 `sso.stuhelper.com`。否则，Casdoor 只负责登录与标准 OIDC consent，StuHelper 在 `stuhelper.com` 的 Open Platform 页面承载业务数据授权。无论 UI 放在哪里，业务授权真源都是 StuHelper 数据库里的 app、scope approval、user consent 和审计记录。

## Casdoor Properties 使用原则

Casdoor `User.Properties` 不作为第三方开放数据主链路。若未来确有一方系统需要低敏摘要，可以异步同步少量 namespaced 字段：

```text
stuhelper_verified_student
stuhelper_school_code
stuhelper_campus_code
stuhelper_qq_bound
```

限制：

- 不同步 QQ 号，除非有明确的一方应用场景并单独评审。
- 不同步学生证件、材料图片、审核备注、手机号明文、邮箱 OTP 详情。
- 不同步内部数据库自增 ID；对外学校使用稳定开放代码，校区使用稳定 code。
- Properties 是缓存，不是事实来源；StuHelper 业务库始终是权威。
- token 中若包含 Properties 投影，必须接受 token TTL 内的滞后；敏感判断仍需调用 StuHelper API。

## 开放平台应用生命周期

1. 开发者在 `stuhelper.com/developers/apps` 创建应用，提交展示名、主页、隐私政策、redirect URI 和申请 scope 用途说明。
2. 管理员审核应用、redirect URI 和敏感 scope。
3. StuHelper 使用 Casdoor API 或运维脚本创建/更新对应 Casdoor application，但 Casdoor registry 不是开放平台业务真源。
4. 用户登录第三方应用时，通过 `sso.stuhelper.com` 完成 OIDC 授权码 + PKCE。
5. 用户同意业务 scope 后，第三方应用调用 StuHelper Open API 获取数据。
6. 用户可在 `stuhelper.com/user/authorized-apps` 查看、撤销应用授权。
7. 管理员暂停、吊销应用或 scope 后，StuHelper API 立即 fail-closed；已有 token 不能绕过 app/scope/consent 检查。

## API 与审计

Open Platform API 的权威契约是 `server/api/openapi.yaml`。实现变更必须遵守：

- 先改 OpenAPI，再运行生成。
- API 返回结构按 scope 明确裁剪，禁止“多返回让前端自己藏”。
- 每次 consent grant / deny / revoke、disclosure granted / denied、token replay、app secret rotate、scope approval change 都要进入审计。
- 手机号、QQ 号、学生认证状态、学校/校区等业务字段披露必须能按 app、user、scope、时间追溯。
- refresh token、长期授权、app-only resource scope 的撤销必须在后续 API 检查中即时生效或在明确 TTL 内失效。

## 与入群验证的关系

Open Platform 不承载入群验证入口。加群验证链路固定为：

```text
https://join.stuhelper.com/verify/<token>?qq=<qq>
```

未登录用户在 join 流程中跳转到主站登录入口，由后端生成 Casdoor 登录 URL；Casdoor 登录完成后回到 `stuhelper.com/api/v1/auth/callback`，写入 `.stuhelper.com` 会话 cookie，再回到原始 join admission URL 继续 QQ 绑定、学生认证或新生材料流程。

`stuhelper.com/verify*`、`id.stuhelper.com/verify*`、`join.stuhelper.com/verify` 都不是公开入口，必须返回 404。

## 生产配置基线

```env
IDENTITY_SERVER_ENABLED=false
IDENTITY_ISSUER=
WEB_VITE_IDENTITY_URL=
WEB_VITE_WEB_URL=https://stuhelper.com
WEB_VITE_SSO_URL=https://sso.stuhelper.com
CASDOOR_ISSUER=https://sso.stuhelper.com
CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com
CASDOOR_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_ADMIN_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
CASDOOR_UNIAPP_REDIRECT_URI=https://stuhelper.com/api/v1/auth/callback
ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com
CORS_ORIGINS=https://stuhelper.com,https://join.stuhelper.com,https://sso.stuhelper.com
TOKEN_COOKIE_DOMAIN=.stuhelper.com
```

`id.stuhelper.com` 不得提供 discovery、OAuth endpoint、账号中心、登录页或 Casdoor 反代。
