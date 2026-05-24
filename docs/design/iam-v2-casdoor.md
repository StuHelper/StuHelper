---
type: design
audience: maintainers, backend-dev, ops
status: draft
authoritative-source: this file for the StuHelper IAM v2 target architecture
created: 2026-05-01
last-verified: 2026-05-18
supersedes: 2026-05-01-casdoor-open-platform-iam-design.md
related:
  - open-platform-v1.md (current Open Platform baseline and target)
  - docs/design/authorization-model.md
  - docs/design/auth-and-session.md
scope: Casdoor-centered IAM target architecture; Open Platform scope and consent live in separate current spec
---

# StuHelper IAM v2 — Casdoor 身份平台

## 1. 决策记录

- **IDP 决策**：Casdoor 是最终方案，由项目 owner 直接决策；不再做选型 ADR；不再保留 Zitadel/Keycloak 候选。
- **迁移性质**：绿地架构，不做兼容数据迁移；历史 Zitadel external subject、session、token 全部失效，要求所有用户重新登录。
- **范围拆分**：本文只覆盖 IAM 目标架构（身份、登录、应用 registry、授权决策入口、SMS/Email 通道、Zitadel 退役）。开放平台已拆出独立 spec [`open-platform-v1.md`](open-platform-v1.md)，当前 v1 baseline 已落地；本文只保留 IAM 侧依赖和边界。

> 本 spec 取代 `2026-05-01-casdoor-open-platform-iam-design.md`（commit 8295a1e7）。旧 spec 把 IAM 切换与开放平台 v1 混写，并把 Casdoor Casbin Enforce 引入业务授权路径，已被本文从架构上修正。

## 2. 目标架构（4 层 + 1 网关）

| 层 | 权威 / 职责 | 严格不能做的 |
|----|-------------|--------------|
| **Casdoor** | 身份与一方登录应用 registry：用户生命周期、登录方式、Provider、MFA、会话、token 签发、扁平角色投影 | 业务授权决策；Open Platform 对外 issuer；不向业务模块暴露 Casbin / Enforce / GetPermissions |
| **StuHelper DB** | 业务事实真相源：实名认证、学生认证、学校归属、手机号验证投影、QQ 绑定、课程/评课/资源 owner、Open Platform consent | 完整手机号真相源 |
| **StuHelper Authorization Service** | **业务模块唯一授权入口**：组合 token 主体、DB 事实、OpenFGA 检查；统一 fail-closed | — |
| **OpenFGA** | 资源关系权威：owner/author/school_admin/section_admin/section_moderator/section_reviewer/app→resource 关系 | 直接被业务模块调用；参与登录决策；承担粗粒度 RBAC（已在 Casdoor 解决） |
| **Open Platform / Identity Gateway** | `id.stuhelper.com` 对外 OIDC issuer、第三方应用披露网关、scope/consent/审计/限流/吊销 | **完整逻辑见** [`open-platform-v1.md`](open-platform-v1.md)；Casdoor 只作为上游登录源 |

```text
一方应用 (web / admin / uniapp)
    │
    │ OIDC authorization code + PKCE
    ▼
Casdoor at sso.stuhelper.com
    │
    │ ID token / access token
    ▼
StuHelper API
    │
    │ ┌──────────────────────────────────────┐
    │ │ Authorization Service (单一 PDP)     │
    │ │   ├─ token 主体 (Casdoor JWT claim)  │
    │ │   ├─ DB 业务事实查询                 │
    │ │   └─ OpenFGA 资源关系检查            │
    │ └──────────────────────────────────────┘
    ▼
业务 handler (不直接 import Casdoor SDK / OpenFGA client)
```

## 3. Casdoor 职责（最小化）

### 3.1 用户与登录方式

Casdoor 承载：
- 密码登录；
- 手机号验证码登录（通过 Custom HTTP SMS Provider 转发，见 §9.1）；
- 邮箱验证、找回、通知（通过 SMTP 或 Custom HTTP Email Provider，见 §9.2）；
- MFA（TOTP / WebAuthn 视后续启用顺序）；
- OIDC/OAuth 会话与 token 签发；
- 未来的社交或校园身份源（设计预留，不在 v2 范围）。

> StuHelper 不再自管公开 `/auth/phone/*` 验证码登录链路。手机号验证码登录归 Casdoor；个人中心的补绑 / 更换 UI 可以在 StuHelper，但写入 Casdoor user profile。StuHelper 本地只保留脱敏投影、验证状态和更新时间。

### 3.2 组织与应用

| Casdoor 对象 | 用途 |
|--------------|------|
| Organization `stuhelper` | StuHelper 用户命名空间 |
| Application `stuhelper-web` | 主站 Web/H5 |
| Application `stuhelper-admin` | 管理后台 |
| Application `stuhelper-uniapp` | Native/mobile（uniapp 端） |
| Application `stuhelper-identity` | `id.stuhelper.com` 接入 Casdoor 的一方登录应用 |

**`stuhelper-koishi-console` 不在 v2 范围。** Koishi 控制台已有独立 admin password 自管，是否接 SSO 是单独产品决策，不阻塞 IAM v2。

每个 Casdoor 一方应用必须配置精确 redirect URI、明确的 grant type、token TTL。生产禁止 wildcard redirect URI。第三方应用不再直接进入 Casdoor registry；它们进入 `id.stuhelper.com` 的 open platform registry。

**Redirect 安全 gate**：若当前 Casdoor 版本存在未修复 open redirect advisory，Casdoor 前置网关必须对 `/login/oauth/authorize` 的 `client_id + redirect_uri` 执行 StuHelper DB 精确白名单校验；没有该网关校验不得开放第三方 OAuth。

### 3.3 角色目录（扁平）

Casdoor 仅存以下 7 个扁平角色，**不编码学校 ID 或板块名**：

```
super_admin
school_admin
section_admin
section_moderator
section_reviewer
verified_student
user
```

学校归属、板块归属、资源所属由 OpenFGA tuple 表达（如 `school_admin → school:10006`），或由业务 DB 查询。`verified_student` 仅作为粗粒度入口闸门，**不作为业务事实证据**——敏感判断必须查 DB（见 §6）。

### 3.4 Provider

bootstrap 必须创建并校验：
- **SMS Provider**：Custom HTTP 类型，回调 StuHelper `/internal/sms/send`（见 §9.1）；
- **Email Provider**：仅当 `CASDOOR_EMAIL_PROVIDER_ENABLED=true` 时创建并校验；当前仓库尚无 StuHelper email service，默认延后（见 §9.2）；
- 证书 / 公钥配置；
- 未来的社交或校园 Provider（设计预留）。

生产环境缺必要 Provider（当前为 SMS Provider；Email Provider 以 env 开关显式启用后才算必要）启动失败；开发环境可显式关闭，但必须通过 env + 日志告警暴露。

### 3.5 MFA / Step-up 策略

> 不一刀切"所有管理员强制 WebAuthn/TOTP"，按风险矩阵分级。参考 [NIST SP 800-63B-4](https://pages.nist.gov/800-63-4/sp800-63b.html) 的 federated 认证 + authenticator recovery 要求。

#### 3.5.1 强制 MFA 矩阵

| 角色 | MFA 要求 |
|------|----------|
| `super_admin` | **强制** TOTP 或 WebAuthn；不允许仅密码登录 |
| `school_admin` | **强制** TOTP 或 WebAuthn |
| `section_admin` / `section_moderator` / `section_reviewer` | 默认不强制登录 MFA；但访问敏感端点时触发 step-up（§3.5.2）|
| `verified_student` / `user` | 不强制；用户可选启用 |

#### 3.5.2 Step-up 触发清单

以下操作必须 step-up MFA（即使当前 session 已 MFA 过，距上次 MFA 超过 5 分钟需重新验证）：

- 实名认证审核结果变更（通过 / 拒绝）
- 学籍认证审核结果变更
- 批量删除 / 隐藏评课（≥ 5 条）
- 学校配置修改
- 角色分配 / 撤销
- OpenFGA tuple 直接写入（管理面接口）
- 第三方应用密钥轮换 / 吊销
- 查看实名信息 / 身份证号 / 学号

#### 3.5.3 MFA 注册与恢复

- **注册时机**：用户首次进入需要 MFA 的角色时强制走注册流程
- **支持方式**：TOTP（任何 RFC 6238 客户端）+ WebAuthn（passkey）；至少注册一种
- **Recovery code**：注册时生成 10 个一次性 recovery code，强制下载或抄写后才能完成注册
- **丢失 MFA 自助恢复**：通过 recovery code；recovery code 用完后由 super_admin 人工 reset
- **super_admin MFA reset**：必须**双人复核**——另一名 super_admin 在 **StuHelper IAM console / API**（非 Casdoor console，详见 §3.5.6 ownership 模型）二次确认；reset 操作落审计

#### 3.5.4 MFA 审计

MFA 注册、禁用、reset、recovery code 使用、step-up 失败、step-up 成功 全部进 `audit_events`，保留 1 年（高于现行 90 天管理员操作审计基线，因敏感度更高）。具体保留期见 §14.3。

#### 3.5.5 Enforcement 机制（capability gate）

> Casdoor 内置 MFA 是 organization 级 Optional / Prompt / Required（参考 [Casdoor MFA items](https://casdoor.org/docs/organization/mfa-items/)）——这意味着 **org-level Required 会强制该组织所有用户启用 MFA，不能按角色细化**。同时 Casdoor 官方文档**未明确**说明是否支持 `amr` / `auth_time` claim 签发、是否尊重 OIDC `prompt=login` / `max_age` / `acr_values` 参数。本节按"实施前必须验证 Casdoor capability"原则写，而非假定可用。

**实施前必须实测的 6 个 capability**：

| Capability | 验证方法 |
|-----------|---------|
| C1: Casdoor 是否能"按用户"或"按角色"强制 MFA（不影响普通用户）| 实测 organization MFA Required 配置范围 |
| C2: Casdoor JWT 是否签发 `amr` claim | 解码已 MFA 用户的 JWT 验证 |
| C3: Casdoor JWT 是否签发 `auth_time` claim | 解码新登录 JWT 验证 |
| C4: Casdoor 是否尊重 OIDC `prompt=login` | 用此参数发起 authorize，看是否强制重新登录 |
| C5: Casdoor 是否尊重 OIDC `max_age=0` | 看是否强制重新走完整认证 |
| C6: Casdoor 是否尊重 `acr_values=mfa` | 看 token 是否含 acr 且强制 MFA |

**落地工具**：`infra/ops/casdoor-capability-probe.sh` 是实施前必须跑的证据入口。它会读取真实 Casdoor OIDC metadata，生成含 `prompt=login` / `max_age=0` / `acr_values=mfa` 的 step-up authorize URL，并在提供 `CASDOOR_PROBE_ID_TOKEN` 或 `CASDOOR_PROBE_ID_TOKEN_FILE` 时解码 JWT payload 检查 `amr` / `auth_time` / `acr` 是否存在。缺少必要 env 时脚本以 `78` 明确退出，不产出伪成功。

**根据 capability 实测结果分支选择**：

| 场景 | C1=Yes（精准强制） | C1=No（org 全员强制） |
|------|---------------------|---------------------|
| `super_admin` / `school_admin` 强制 MFA | 直接用 Casdoor 角色级 Required | 接受 org 全员强制 MFA（项目当前用户量小，可承受），或在 Authorization Service 层以本地 MFA proof 兜底拒绝未完成 MFA 的 admin 操作 |

| 场景 | C2/C3/C4/C5/C6 全部 Yes | 任一 No |
|------|------------------------|--------|
| step-up MFA | OIDC reauth：前端跳转 `authorize?prompt=login&max_age=0&acr_values=mfa`，依赖 `auth_time` / `amr` 校验 | StuHelper 本地 step-up：API 层独立 challenge（POST `/api/v1/auth/step-up` 触发 TOTP/WebAuthn 验证 + 5 分钟 grace token），不依赖 Casdoor OIDC 参数 |

**MFA enrollment vs proof（关键区分）**：

- `user_mfa_enrollment` 只证明"该用户已注册可用 MFA 因子"，**不**证明"当前 session 已完成 MFA challenge"；
- `mfa_proof` 才是放行证据：证明当前 session / 当前操作窗口内完成过 MFA challenge；
- admin 入口要求：`user_mfa_enrollment.active = true` + 当前 session 有有效 `mfa_proof`；
- §3.5.2 敏感操作要求：`user_mfa_enrollment.active = true` + `mfa_proof.verified_at >= now - 5min`；
- `mfa_proof` 来源二选一：
  - Casdoor capability C2/C3/C4/C5/C6 全部实测通过时：前端调用 `GET /api/v1/auth/step-up` 获取带 `prompt=login` / `max_age=0` / `acr_values=mfa` 的 OIDC reauth URL；callback 完成后，JWT `amr` 包含 MFA 方法，且 `auth_time` 满足 5 分钟窗口；
  - 任一 capability 不满足时：StuHelper 本地 `/api/v1/auth/step-up` challenge 签发 5 分钟 grace proof（Redis 或签名短 token，绑定 `user_id` / `session_id` / `aud` / `nonce`）。

**强制规则（capability 无关）**：

- super_admin / school_admin 任一 admin 入口或 admin 操作前，Authorization Service 必须同时确认 MFA enrollment + MFA proof；缺 enrollment → `403 mfa_enrollment_required`，缺 proof 或 proof 过期 → `412 step_up_required`；
- MFA enrollment **仅以 DB `user_mfa_enrollment` 为准**（§3.5.6 ownership 模型）；Casdoor 投影漂移**仅触发** drift check / alert，**不阻断**授权放行（除非 exec-plan 明确选择 stricter fail-closed 模式）；
- 无论 Casdoor 是否签发 `amr` claim，MFA enrollment 状态始终由 StuHelper DB 表 `user_mfa_enrollment`（实施时新建）作为真相源；`amr` 只能参与 `mfa_proof` 判断；
- MFA enrollment 的 ownership 模型与同步方向见 §3.5.6（不通过 §6.5 outbox 反向同步——outbox 是 DB → Casdoor 单向）。

#### 3.5.6 MFA Enrollment Ownership

> 选定 **"StuHelper owns privileged MFA"** 模型。原因：privileged MFA（super_admin / school_admin）必须满足双人复核 reset、5 分钟 step-up grace、reset 操作落审计等强约束；如果以 Casdoor console 作为可独立修改的入口，会绕过 StuHelper 审计与流程。

**真相源**：StuHelper DB `user_mfa_enrollment` 表（实施时新建）。

**写路径**（唯一入口）：

1. 所有 privileged MFA 变更（注册 / 禁用 / reset / recovery code 生成）必须通过 StuHelper API（如 `/api/v1/iam/mfa/*`）；
2. StuHelper API 在事务内写 `user_mfa_enrollment` + 审计；privileged MFA factor / recovery code / step-up proof 由 StuHelper 本地 MFA 子系统承载；
3. 默认**不**把 privileged MFA factor 同步到 Casdoor：Casdoor 当前公开文档只明确 organization MFA 配置和用户 profile 自助 MFA，不假设存在可由 StuHelper 后端安全 provision / reset / disable 用户 TOTP/WebAuthn 因子的管理 API；
4. 若 exec-plan 实测证明 Casdoor 支持安全的 MFA factor 管理 API，也必须先写单独 capability ADR，再决定是否新增 Casdoor MFA projection；不得在 IAM v2 默认路径里引入 `iam_casdoor_mfa_sync`；
5. **禁止**直接在 Casdoor console / Casdoor profile UI 修改 super_admin / school_admin 的 privileged MFA 配置；Casdoor MFA 对 privileged 用户最多作为登录层额外保护或 drift hint，不作为授权真相源。

**读路径**：

- Authorization Service 检查 MFA enrollment 状态时**只以 DB 为授权证据**；
- Casdoor MFA 状态 / `amr` claim 只能作为登录层 MFA hint 或 `mfa_proof` 的来源之一，不替代 DB enrollment；
- 冲突时按 §6.3 一致性矩阵处理（DB 为准；Casdoor 投影漂移触发即时同步 + 告警）；
- Authorization Service 放行 admin 操作时必须另查当前 session 的 `mfa_proof`（见 §3.5.5），不能把 enrollment 当 proof 使用。

**对普通用户 MFA**（非 privileged）：

- 普通用户启用/禁用 MFA 可走 Casdoor profile UI；
- StuHelper DB 不强制为真相源；
- 普通用户的 MFA 状态对业务授权无影响（不触发 step-up 或入口闸门），所以漂移可接受。

**Break-glass / Bootstrap 约束**：

- 系统初始化时**至少**创建 2 个 super_admin 账号；
- 任一 super_admin 不可解除自己的 MFA（防 lockout 同时也防 admin 自我提权后单方逃逸）；
- super_admin MFA reset 需另一名 super_admin 双人复核（§3.5.3）；
- Bootstrap 阶段单人初始化窗口 ≤ 24 小时，超时强制要求绑定第二名 super_admin。

### 3.6 Service Account / Machine 身份

> 现有系统已有非人类调用方（Koishi runtime 通过 `BOT_SERVICE_TOKEN` 调后端，见 `docs/design/security-model.md`）。IAM v2 必须显式建模机器身份，否则会出现"有的用 service token、有的用 Casdoor client、有的绕过 Authorization Service"的漂移。

#### 3.6.1 机器调用方分类

| 调用方 | 推荐身份模型 | 理由 |
|-------|-------------|------|
| Koishi runtime → StuHelper API（已有） | **保留** opaque service token 模式 + 升级路径见 §3.6.4；不上 Casdoor | 内部高频调用；switch 到 Casdoor 增加 IDP 故障域无收益 |
| Outbox worker → Casdoor / OpenFGA | StuHelper 内部凭据（per-worker secret） | 不暴露到外网；不需要 OAuth 流程 |
| Metrics / observability collector | 保留现行 metrics 基本认证 | 与 IDP 解耦 |
| Open Platform server-to-server（v1.1） | StuHelper Identity `client_credentials` grant | 第三方应用走标准 OAuth；仅允许 `resource.read` / `resource.write` app-only scope，避免绕过用户 consent 读取资料 |
| 高敏后台写操作（如运维直接写 OpenFGA） | mTLS 或 sender-constrained token | 强身份 + 不可重放 |

#### 3.6.2 机器身份硬约束

- 机器身份**不进入** §3.3 的 7 个人类角色目录；
- 机器身份**不使用** `school_admin` / `section_admin` 等人类角色；
- 机器身份的授权模型：**capability + audience + scope** + （可选）resource relation；
- 在 `AuthorizationService.Authorize` 中按 `Subject.AppID` 维度走独立决策路径，与人类用户路径分离；
- 每个 service account 的 token 必须带 `aud` 限定可访问的端点集合（例：Koishi service token 只能调 `/api/v1/bot/*`）；
- secret 必须有完整生命周期：创建 → 轮换 → 吊销 → 审计；不允许长期不变。

#### 3.6.3 防滥用

- service account 不允许做"用户身份的代理"——禁止用 service account 模拟真实用户调用敏感端点；
- 真实用户的敏感操作必须走人类 OIDC + MFA，不允许通过 service account 绕过；
- service account 的审计事件保留 3 年（见 §14.3）。

#### 3.6.4 Koishi service token 升级路径

> 已选方案 B。`BOT_SERVICE_TOKEN` 只作为启动 bootstrap / rotation 输入；请求认证路径必须通过 `bot_service_credentials` 的 token hash、audience、scope、revoked/expired 状态校验，不允许回退到裸环境变量常量时间比较。

| 方案 | 优点 | 缺点 |
|------|------|------|
| **A. 内部签名 JWT** — StuHelper 自签发（独立 keypair，不走 Casdoor），含 `aud=/api/v1/bot/*`、`scope`、`exp`、`kid` | 无状态校验；标准 OIDC 思路；与未来 mTLS / DPoP 衔接顺 | 需要 StuHelper 自管签名 key；JWKS endpoint 独立维护 |
| **B. DB-backed opaque API key + 元数据表** — 引入 `bot_service_credentials` 表（`token_hash`、`aud`、`scope[]`、`created_at`、`rotated_at`、`revoked_at`、`last_used_at`）；中间件按 hash 查表 | schema 简单；轮换/吊销直接 SQL；现有 opaque 习惯延续 | 每请求一次 DB lookup（可加 Redis 缓存）；缺乏 OIDC 标准化优势 |

**执行决策**：采用 **B. DB-backed opaque API key + 元数据表**。Koishi 仍发送 Bearer token，但服务端只接受已写入 `bot_service_credentials` 的 HMAC hash；每个 route 绑定最小 scope（如 `bot.qq_binding.consume` / `bot.qq_verification.read`），audience 固定到 `/api/v1/bot/*`，并支持轮换、吊销、过期和 `last_used_at` 记录。

## 4. Casdoor SDK 出口（白名单 + ban-list）

> **核心规则**：Casdoor SDK 只能从 `server/internal/platform/casdoor/` 包内出口；业务模块不得直接 `import casdoorsdk`。

### 4.1 允许出口（platform/casdoor/ 包内）

- `ParseToken(jwt)` → `Subject`、`Roles`、`AppID`
- `VerifyJWT(jwt)` → 标准 OIDC 校验（iss/aud/sig/exp）
- `GetUser(name)` / `GetUsers()`
- `AssignRole(user, role)` / `RemoveRole(user, role)`
- `CreateApplication(spec)` / `UpdateApplication` / `DeleteApplication`（开放平台预留）
- `RefreshToken(refreshToken)`
- `Logout(sessionID)`
- 平台管理需要的其它 admin API

### 4.2 禁止出口（永远不暴露给业务代码）

- `Enforce` / `BatchEnforce`（Casbin 决策）
- `GetPermissions` / Casbin policy 直接读取
- 任何让业务模块跳过 Authorization Service 的 SDK 调用

**理由**：Casdoor 内部使用 Casbin 维护自身平台权限模型；这些对 StuHelper 业务授权是**实现细节**，不是事实源。如果业务代码可以直接 `Enforce`，就会出现两个远程 PDP 同时决定同一件事，违反 §5 单一授权入口原则。

### 4.3 工程检查

CI 增加规则。扫描范围是 `server/internal/` 内除白名单包之外的所有 Go 代码；白名单：

```text
server/internal/platform/casdoor/      # Casdoor SDK 唯一封装层
server/internal/platform/authorization/ # 唯一业务 PDP（可调 OpenFGA）
server/internal/pkg/oidc/              # provider-neutral 标准 OIDC
```

```bash
#!/usr/bin/env bash
set -euo pipefail

SCAN_DIR="server/internal"
[[ -d "$SCAN_DIR" ]] || { echo "ERROR: scan dir $SCAN_DIR not found" >&2; exit 1; }

ALLOWED='^server/internal/(platform/casdoor|platform/authorization|pkg/oidc)/'

run_check() {
  local label="$1" pattern="$2"
  local rc=0 hits=""

  # grep 退出码：0=找到、1=未找到、>=2=错误
  hits=$(grep -rEln "$pattern" "$SCAN_DIR") || rc=$?
  if (( rc > 1 )); then
    echo "ERROR: grep failed (rc=$rc) when scanning for: $label" >&2
    exit 1
  fi

  local violations=""
  if [[ -n "$hits" ]]; then
    violations=$(printf '%s\n' "$hits" | grep -vE "$ALLOWED" || true)
  fi

  if [[ -n "$violations" ]]; then
    echo "ERROR: $label" >&2
    printf '%s\n' "$violations" >&2
    exit 1
  fi
}

run_check "business code must not import casdoorsdk" 'casdoorsdk'
run_check "business code must not call Enforce/BatchEnforce/GetPermissions" '\.(Enforce|BatchEnforce|GetPermissions)\('
run_check "business code must not import openfga client" 'openfga'
```

**鲁棒性要点**：
- `set -euo pipefail` 确保任一子步骤失败立即退出；
- 扫描目录预检（`[[ -d ... ]]`）避免因目录不存在被吞成 "0 hits"；
- 显式分流 grep 退出码：`rc=0`（找到）→ 进入白名单过滤；`rc=1`（未找到）→ 通过；`rc≥2`（IO/权限错误）→ 立即失败而非误判通过；
- 不使用 `grep ... && exit 1 \|\| true` 这种把 IO 错误吞成成功的写法。

## 5. StuHelper Authorization Service

业务模块**唯一**授权入口。

### 5.1 接口

```go
type AuthorizationService interface {
    // Authorize 单次决策。fail-closed：任何依赖不可用都返回 Deny + Error。
    Authorize(ctx context.Context, subject Subject, action Action, resource Resource) Decision

    // BatchAuthorize 列表/批量场景，复用同一份 Casdoor / DB / FGA 上下文。
    BatchAuthorize(ctx context.Context, subject Subject, checks []Check) []Decision
}

type Subject struct {
    UserID string   // StuHelper 内部 ID
    AppID  string   // OAuth client（一方或第三方）
    Roles  []string // 来自 Casdoor JWT，仅作为粗粒度 hint
}

type Decision struct {
    Allow  bool
    Reason string // 拒绝原因，用于审计与诊断
    Error  error  // 依赖故障，区分 401 / 403 / 503
}
```

### 5.2 决策流程

```text
1. 校验 token（本地 JWKS 或 introspection，按 token 类型）
2. 提取 Subject (user_id, app_id, roles claim)
3. 按 action 决定是否需要：
   - 业务 DB 事实（实名 / 学生认证 / 学校 / 手机号验证投影 / QQ 绑定）
   - OpenFGA 资源关系
4. 并行加载所需事实
5. 组合判断（按 §6 一致性矩阵处理冲突）
6. 任一依赖不可用 → fail-closed
```

### 5.3 决策示例

```go
// 学生发布评课
Authorize(subject, "review.create", course)
// → DB: identity_verified=true ∧ student_verified=true ∧ school_id matches course.school
// → OpenFGA: 无（创建不需要资源关系）
// → token role: verified_student（仅作为投影一致性 hint，不作为证据）

// 删除评课
Authorize(subject, "review.delete", review)
// → DB: 加载 review.author / review.school / review.section
// → OpenFGA: subject 是 review author OR section_moderator from review.section OR admin from review.school
// → 无需 DB verified_* 字段

// 查看实名信息
Authorize(subject, "profile.view_identity", profile)
// → DB: profile 状态、profile.school
// → OpenFGA: subject 是 profile.owner OR admin from profile.school
// → token role: super_admin / school_admin 用于粗粒度入口（仍需 OpenFGA 关系细化）
```

## 6. 业务事实权威与一致性

### 6.1 真相源（不可妥协）

| 事实 | 真相源 |
|------|--------|
| 实名认证状态 | StuHelper DB（`user_identities`） |
| 身份类型（学生/教职工/其他） | **派生字段**：默认 `other`；`user_profiles.verification_status='verified'` → `student`；教职工区分推到未来（需新增 `users.identity_type` 列或 `user_staff_profiles` 表）|
| 学生认证状态 | StuHelper DB（`user_profiles`） |
| 学校 ID | StuHelper DB |
| 完整手机号 | Casdoor user profile |
| 手机号验证投影（脱敏展示、已验证状态、更新时间） | StuHelper DB |
| QQ 绑定 | StuHelper DB |
| 课程归属学校 | StuHelper DB |
| 评课/举报/资源 owner | StuHelper DB + OpenFGA（写双份） |

### 6.2 投影同步（DB → Casdoor 角色）

业务事实变更通过 outbox 同步到 Casdoor 扁平角色（如 `verified_student` 标记）：

- 模式：transactional outbox + 后台 worker
- SLA：p95 < 60s，p99 < 5min
- 重试：线性退避（详细参数见 §6.5.3）；超过 max_attempts 后进入"长期 failed"状态，由漂移对账任务（§6.5.4）人工修复
- 死信告警：Prometheus alert，运维介入

### 6.3 一致性冲突矩阵（核心规则）

| 状态 | DB | Casdoor 投影 | 决策 | 理由 |
|------|----|--------------|------|------|
| 一致 ON | ✓ | ✓ | 放行 | 正常路径 |
| 一致 OFF | ✗ | ✗ | 按操作决定 | 正常路径 |
| **DB ON, Casdoor OFF** | ✓ | ✗ | **以 DB 为准放行** + 触发即时同步 + 漂移告警 | DB 是真相源；Casdoor 投影只是缓存 |
| **DB OFF, Casdoor ON** | ✗ | ✓ | **拒绝**敏感操作 + 触发即时撤销 + 漂移告警 | Casdoor 角色可能陈旧；不能让旧 claim 续命 |
| DB unavailable, 操作需 DB 事实 | ? | * | 503 | fail-closed |
| Casdoor 不可用，已签发 JWT 在本地有效期内 | * | * | 身份验证仍可放行（标准 OIDC RS 行为） | 见 §8 |

### 6.4 不可妥协的规则

> **敏感操作绝不信任 JWT claim 的 `verified_*` 字段，必查 DB。**
> Casdoor 角色 claim 仅作为粗粒度入口闸门（"是否进入这块功能"），不作为业务事实证据（"是否真的是已认证学生"）。

具体清单：
- 发布评课、看评课全文、查看实名审核、查看学籍审核、参与板块讨论 → 必查 DB；
- 普通页面浏览、列表页摘要、登录态展示 → 可信任 claim 粗粒度判断；
- 管理面进入闸门 → 信任 claim 角色；具体资源操作 → 仍走 Authorization Service + OpenFGA。

### 6.5 Outbox 与 drift reconciliation 具体化

> 复用现有 `domain_event_outbox` 基础设施（定义在 `server/migrations/000001_initial_schema.up.sql`），**不**新建 `iam_sync_job` 表。

#### 6.5.1 Stream 命名

| Stream | 用途 |
|--------|------|
| `iam_casdoor_role_sync` | DB 业务事实变更 → Casdoor 用户角色 add/remove |
| `iam_casdoor_user_projection` | 用户元数据变更 → Casdoor 用户记录更新 |
| `iam_openfga_tuple_sync` | DB 资源关系变更 → OpenFGA tuple 写入 / 删除 |

现行 `user_profile_projection` job 更新的是 `user_profile:{id}` 的 owner/school OpenFGA tuple，因此必须落在 `iam_openfga_tuple_sync`。`iam_casdoor_user_projection` 仅预留给未来 Casdoor 用户元数据投影；不得承载 profile tuple sync。

#### 6.5.2 Worker 机制

- **基线**：沿用现有 polling worker（按 `pending` → `processing` → `completed` / `failed` 状态机）；
- **轮询周期**：2 秒（统一由 `server/internal/pkg/outbox/streams.go` 的 `IAMWorkerConfig` 配置）；
- **优化（可选）**：未来可引入 PostgreSQL `LISTEN/NOTIFY` 减少延迟，polling 作为兜底；不在 IAM v2 强制范围。

#### 6.5.3 重试与 DLQ 语义

> **重要**：现行 outbox（`server/internal/pkg/outbox/repository.go`）的 `failed` 状态**不是** terminal DLQ。`claimQuery` 通过 `status='failed' AND available_at <= NOW()` 把 failed 行重新 claim（`repository.go:146`）；`markRetryQuery` 把每次失败都写回 `failed` 并更新 `available_at` 与 `attempt_count`（`repository.go:177`）。退避策略是**线性**：`(attemptCount+1) * RetryBaseBackoff`（`worker.go:105`），**不是指数**。本节按现行语义对齐，不臆造新状态。

**v2 表达"已死信"的方式 — 必须改 worker 层（不是纯文档约定）**：

> 现有通用 `ClaimJobs()`（`repository.go:138`）只判断 `status='failed' AND available_at <= NOW()`，不过滤 `attempt_count`。如果只是"claim 后由业务代码跳过"，超阈值行会每 2 秒反复被 claim → reject → alert，造成噪音风暴且永不停止。要让超阈值行真正"停止处理"，必须改 worker 层。

**已选方案 B**：`WorkerConfig.MaxAttempts` 达到阈值时，worker 把 `available_at` 设为远未来（100 年），状态保持 `failed`。后续 `ClaimJobs` 自然跳过。该行为只对显式设置 `MaxAttempts` 的 worker 生效；IAM worker 统一通过 `outbox.IAMWorkerConfig` 设置，非 IAM worker（如 resource cleanup）保持旧语义。

超阈值事件**触发一次** `outbox_job_failures_total{terminal="true"}` 指标增量，并由 Prometheus `StuHelperOutboxTerminalFailures` 告警接管；不重复触发。漂移对账任务（§6.5.4）按夜间扫描 `WHERE status='failed' AND available_at > NOW() + INTERVAL '1 year'` 找出长期 failed 行，纳入人工修复队列。

**v2 退避策略（沿用现行实现）**：

- `RetryBaseBackoff` 配置为 5 秒（统一由 `outbox.IAMWorkerConfig` 设置）；
- 实际退避：`(attempt_count+1) * 5s`（attempt 0→5s、attempt 1→10s、attempt 2→15s、attempt 3→20s、attempt 4→25s）——`worker.go:103` 在 `markRetry` 之前用当前 `AttemptCount`，所以首次失败是 5s 不是 10s；
- `MaxBackoff` cap 5 分钟（统一由 `outbox.IAMWorkerConfig` 设置）；
- 如未来需要**指数退避**或**独立 terminal DLQ status**，必须改 worker 实现 + 加 schema migration，**不在 IAM v2 范围**。

#### 6.5.4 Drift Reconciliation

- **周期**：每日凌晨 3 点全量对账（cron）；
- **对账维度**：
  - DB `user_profiles` ↔ Casdoor `verified_student` 角色；
  - DB `user_identities` ↔ Casdoor 用户元数据投影（如启用）；
  - 业务表 review/profile owner ↔ OpenFGA `author` / `owner` tuple；
- **漂移处理**：
  - 单次漂移条数 < 阈值（建议 100）→ 自动修复（重新写 outbox 事件）；
  - 漂移条数 ≥ 阈值 → 暂停自动修复 + 告警 + 人工确认后再放行；
  - 修复结果落审计。
- **当前落地**：
  - `user_profiles` 每日 reconciliation 重新入队 `verified_student_role` 与 `user_profile_projection` 两类任务；前者修复 Casdoor `verified_student` 角色，后者修复 OpenFGA `user_profile:{id}` 的 `owner` / `school` tuple。
  - `reviews` / `review_reports` 每日 reconciliation 重新入队 `review_relations` 与 `report_relations`，通过现有 worker 修复 OpenFGA `review:{id}` 的 `author` / `course` / `school` tuple 与 `report:{id}` 的 `reporter` / `review` / `school` tuple。

#### 6.5.5 Dedupe 与幂等

- outbox 行使用 `dedupe_key` 字段（已存在）确保同一业务事件不重复入队；
- OpenFGA tuple Write 不天然幂等：worker 必须 Read-before-Write 检查存在性，避免重复 Write 增加 store 体积；删除同理；
- Casdoor role assignment 是天然幂等（重复 add 同一 role 不报错），无需额外 Read。

## 7. OpenFGA

继续作为资源关系权威。

### 7.1 模型（OpenFGA 1.x DSL）

> 概念模型；最终落地以 `infra/openfga/model.fga` 为准。
>
> **OpenFGA DSL 限制**：只支持单层 `X from Y`，不支持 `a.b` 链式 TTU（参考 `infra/openfga/model.fga:11` 注释与 [OpenFGA 配置语言文档](https://openfga.dev/docs/configuration-language)）。本模型按现行仓库风格——每层通过中间 `*_proxy` relation 把上层关系投影下来——避免链式 TTU。

```dsl
model
  schema 1.1

type user

type ecosystem
  relations
    define super_admin: [user]

type school
  relations
    define parent: [ecosystem]
    define admin: [user]
    # 学校生效管理员 = 直接 admin + 生态超管（单层 TTU）
    define effective_admin: admin or super_admin from parent

type section
  relations
    define school: [school]
    # 中间 relation：把学校管理员投影到 section（单层 TTU: from school）
    define school_admin_proxy: effective_admin from school
    define section_admin: [user] or school_admin_proxy
    define section_moderator: [user] or section_admin
    define section_reviewer: [user] or section_admin

type course
  relations
    define school: [school]
    define owner: [user]
    define ta: [user]
    define school_admin_proxy: effective_admin from school
    define can_edit: owner or ta or school_admin_proxy

type review
  relations
    define course: [course]
    # 关键：review 必须直接持有 school（不是 from course 链式取），否则授权会撞链式 TTU 限制
    define school: [school]
    define section: [section]
    define author: [user]
    # 中间投影 relations（全部单层）
    define school_admin_proxy: effective_admin from school
    define section_moderator_proxy: section_moderator from section
    define can_edit: author
    define can_delete: author or section_moderator_proxy or school_admin_proxy
    define can_hide: section_moderator_proxy or school_admin_proxy
    define can_restore: section_moderator_proxy or school_admin_proxy
    define can_admin_delete: section_moderator_proxy or school_admin_proxy
    define can_admin_edit: school_admin_proxy

type report
  relations
    define review: [review]
    define school: [school]
    define section: [section]
    define reporter: [user]
    define school_admin_proxy: effective_admin from school
    define section_moderator_proxy: section_moderator from section
    define can_process: section_moderator_proxy or school_admin_proxy

type user_profile
  relations
    define owner: [user]
    define school: [school]
    define school_admin_proxy: effective_admin from school
    define can_view_identity: owner or school_admin_proxy

# Open Platform v1 预留 — IAM v2 仅写 type/relation 定义，不写业务逻辑
type open_platform_app
  relations
    define developer: [user]
    define approved_by: [user]
```

**校验规则**：上文每一处 `from X` 中的 X 都是**当前 type 的直接 relation**（如 `effective_admin from school` 中 `school` 是 `section` / `course` / `review` / `report` / `user_profile` 上直接持有的关系），不是 `Y from Z` 链式。等价地，要让 review 知道学校管理员，必须直接给 review 加 `school: [school]`，不能写 `admin from school from course`。

**落地文件**：`infra/openfga/model.fga` 是人类可读 DSL；`infra/openfga/model.json` 是 `server/cmd/fga-setup` 导入 OpenFGA 的机器格式。两者必须同步修改，不允许再使用代码内硬编码模型。

**注意**：`scope consent`（用户对应用的 scope 授权）**不在 OpenFGA 中建模**，因为 scope 是字符串属性，不是 OpenFGA 关系目标。Scope consent 由业务 DB 表 `open_platform_user_consent` 承载，由 `AuthorizationService` 在决策时查询。OpenFGA 只承载"应用 → 具体资源"的细粒度关系（详见 [`open-platform-v1.md`](open-platform-v1.md) §9）。

### 7.2 调用边界

- **跨应用 / Open Platform 授权**：由 Authorization Service 调用 OpenFGA，并封装标准 check 接口；
- **StuHelper 内部资源 mutation 授权**：业务模块可通过注入的授权接口执行 request-time OpenFGA `Check`，但不能直接构造具体 fga client；
- 列表、筛选和 UI 范围展示继续使用登录时物化 scope；单条 mutation 以 OpenFGA `Check` 作为资源级权威决策。

## 8. Token 与 JWKS 语义（修正版）

### 8.1 浏览器 Cookie token（Web/Admin）

- 本地 JWKS 校验（iss / aud / sig / exp）；
- 5 分钟 access token TTL（`TOKEN_ACCESS_TTL`）；
- 依赖 refresh 续期；
- 紧急吊销靠 Redis blacklist + 短 TTL；
- **Casdoor 不可用时**：已签发未过期 token 仍可本地 JWKS 校验（标准 OIDC resource server 行为，**不是 silent fallback**）；refresh / userinfo / 未知 kid 的 JWKS 拉取 → 503。

### 8.2 Bearer token（Native/API）

- 本地 JWKS 校验 + 必要时 introspection（按敏感度配置）；
- introspection 失败 → 拒绝当前请求 503；不允许用陈旧 claim 续命。

### 8.3 Casdoor JWT 字段最小化（access + ID token 同 payload）

> **重要**：Casdoor 把 access token 与 ID token 都签成 JWT 且**共用同一份 claim payload**（参考 [Casdoor Token Overview](https://casdoor.org/docs/token/overview/)）。这与 Zitadel 不同——Zitadel 的 access token 通常是 opaque（视配置）、ID token 单独 JWT 可分别配置 claim。本节按 Casdoor 实际语义写：**最小化适用于所有 Casdoor 签发的 JWT，access token 同样不得携带敏感字段**。

JWT（access + ID token 同 payload）允许包含：

```
sub / iss / aud / exp / iat
preferred_username / name / picture
email（仅一方应用且业务确实需要）
```

**条件性允许**（仅在 §3.5.5 capability gate C2/C3 实测 Casdoor 可签发后才使用）：

```
amr        # MFA 方法标识；只能参与 mfa_proof 判断，不能替代 user_mfa_enrollment
auth_time  # 上次认证时间；不可用时 step-up 走 StuHelper 本地 challenge（§3.5.5）
```

> Casdoor 官方 token fields 列表未显示 `amr` / `auth_time`（参考 [Casdoor Token Overview](https://casdoor.org/docs/token/overview/)）。在实测确认前，**不得**把它们作为 Authorization Service 的唯一 step-up 证据。

**默认不包含**（必须通过 Authorization Service + 业务 DB 查询；任何把它们写进 token 的尝试都视为漂移）：
- phone / phone_verified
- identity_verified
- student_verified
- school_id / school_name
- identity_type
- qq_binding

**Authorization Service 不读 token 中的 verified_* 字段**（即使有也忽略），强制走 DB（§6.4）。这一规则与"敏感操作必查 DB"硬规则正交，但提供二次防御——即使 token 配置漂移也不会让陈旧 claim 续命。

### 8.4 Refresh Token Rotation 与 Reuse Detection

参考 [RFC 9700 — OAuth 2.0 Security Best Current Practice](https://www.rfc-editor.org/rfc/rfc9700)。所有 client 类型必须满足：

| 项 | 要求 |
|----|------|
| 单次使用 | 每次 refresh 调用签发**新** access + **新** refresh token；旧 refresh token 立即作废 |
| Reuse detection | 旧 refresh token 再次被提交 → 视为被盗：**立即吊销该 user 的全部 session** + 触发安全告警 |
| Public client（uniapp / mobile / native） | 强制 rotation；refresh token TTL ≤ 7 天；不允许 sender-constrained 之外的长期 token |
| Confidential client（web / admin server-side） | 强制 rotation；可选 sender-constrained（DPoP / mTLS）；refresh token TTL ≤ 30 天 |
| 协同 | rotation 行为由 Casdoor 配置 + StuHelper session/blacklist 双方协同；StuHelper 在 rotation 时同步更新 session store |
| Logout-all / token revoke | 必须调 Casdoor revoke endpoint 吊销 refresh token；不允许仅清 StuHelper session 导致 Casdoor 端仍能 refresh |

#### 8.4.1 Casdoor capability gate（实施前必须验证）

Casdoor 官方 token 文档目前**未明确**说明 single-use rotation + reuse detection 是否原生支持（仅给出 lifecycle、grant type、revocation 语义；参考 [Casdoor Token Overview](https://casdoor.org/docs/token/overview/)）。实施 IAM v2 之前必须验证：

1. **跑实测**：用 Casdoor 测试实例发 refresh token，连续两次以同一 refresh token 调 token endpoint，观察行为；
   - 实测入口：`infra/ops/casdoor-capability-probe.sh`；
   - 因该检查会消耗 refresh token，必须显式设置 `CASDOOR_PROBE_RUN_REFRESH_ROTATION=true` 和一次性 `CASDOOR_PROBE_REFRESH_TOKEN`，脚本不得默认执行破坏性 probe；
   - 第三方 app token claim 最小化的手工 code-flow 实测入口是 `infra/ops/casdoor-token-minimization-probe.sh`；脚本固定请求 `scope=openid`，交换 authorization code 后解码 ID token 与 JWT access token，出现手机号、学生认证、学校、身份类型等业务 claim 时失败；
   - 审批前自动 runtime code-flow 探针入口是 `infra/ops/casdoor-runtime-token-probe-runner.mjs`，backend 生产镜像内置副本为 `/app/casdoor-runtime-token-probe-runner.mjs`；生产必须设置 `OPEN_PLATFORM_TOKEN_PROBE_RUNTIME_REQUIRED=true`、注入专用低权限 Casdoor 探针账号凭据，并用 `infra/ops/casdoor-runtime-token-probe-smoke.sh` 在发布时通过专用 smoke app 实跑一次；
2. **若 Casdoor 原生支持** rotation + reuse detection：直接用 Casdoor 配置；
3. **若 Casdoor 不支持**（默认假设）：StuHelper 在 session store 层包装：
   - 记录每个 refresh token 的 hash + user_id + 颁发时间；
   - refresh 调用时校验旧 hash 仍 active → rotation 后立即作废旧 hash；
   - 旧 hash 在作废后被再次提交 → 触发"reuse detected" → 吊销该 user 全部 session + 安全告警；
   - 当前落地：reuse detection 增加 `auth_refresh_token_reuse_total{token_family}`，由 Prometheus `StuHelperRefreshTokenReuseDetected` 告警接管；label 只允许低基数 token family（如 `self_signed` / `oidc`），不得包含 user / session / token hash；
   - 所有客户端不得直接持有可向 Casdoor token endpoint 使用的 refresh token；refresh 必须走 StuHelper `/auth/refresh` 代理，由 StuHelper 代持、轮换、吊销 Casdoor refresh token；
   - 若无法阻断客户端直连 Casdoor refresh grant，则禁用 refresh token，public client 改为短 session + 重新 OIDC flow；
   - 这层 wrapper 在 Casdoor 之外提供，与 IDP 选型解耦；
4. **Public client 短 TTL 兜底**：在 capability 落地前，public client（uniapp / mobile / native）refresh token TTL ≤ **1 天**，强制每天重新走 OIDC flow，把暴露窗口压到最小。

## 9. SMS / Email 通道

### 9.1 SMS

```text
Casdoor 登录验证码请求
    │
    │ Custom HTTP SMS Provider
    ▼
StuHelper /internal/sms/send  (server/internal/pkg/sms/handler.go)
    │
    │ form POST + internal_key query 鉴权
    ▼
腾讯云 SMS API  (server/internal/pkg/sms/tencent.go)
```

**保持不变的**：`pkg/sms/tencent.go`（TC3-HMAC-SHA256 签名、模板、限流、审计、回滚）继续承载。Casdoor 仅作为登录验证码触发方。

**变更点**：
- 取消 Zitadel Action 触发链路；
- Casdoor 配置 Custom HTTP SMS Provider，URL 指向 `/internal/sms/send`；
- Casdoor Custom HTTP SMS Provider 按 `application/x-www-form-urlencoded` 发送 `phoneNumber` 与验证码字段。StuHelper bootstrap 必须把 provider `title` 固定为 `content`，并在 provider endpoint 注入 `?internal_key=...`，因为 Casdoor Provider 不支持给该回调配置 Bearer Authorization header；
- `/internal/sms/send` handler 保留 `Authorization: Bearer <SMS_INTERNAL_KEY>` + JSON body 作为内部诊断/手动调用入口，但 Casdoor 生产链路使用 query key + form body；
- `sms/handler.go` 第 12 行注释从 "Zitadel Action" 改为 "Casdoor Custom HTTP SMS Provider"。

**禁止**：Casdoor 直连腾讯云 SMS API（避免模板/签名/限流/审计配置散到 IDP）。

### 9.2 Email

类似模式：StuHelper 内部 email service + Casdoor SMTP/Custom HTTP Provider。当前若无 email 发送链路，可延后到 v2 实施时补建。

## 10. 失败语义（修正矩阵）

| 失败场景 | 决策 |
|----------|------|
| Casdoor token 校验失败（签名/过期/iss/aud 错） | 401 |
| **Casdoor 不可用，已签发 JWT 在本地有效期内** | **身份验证放行，进入业务授权检查** |
| Casdoor 不可用，需 introspection（bearer 路径） | 503 |
| Casdoor 不可用，需 refresh / login / userinfo / 未知 kid JWKS 拉取 | 503 |
| Casdoor 不可用，需 application 创建/吊销/角色同步 | 503（管理面） |
| Authorization Service 决策 = Deny | 403 |
| 业务 DB 不可用 + 操作需要业务事实 | 503 |
| OpenFGA 不可用 + 操作需要资源关系 | 503 |
| Outbox 同步失败 | 告警；不阻塞读路径；敏感写路径仍由 Authorization Service 强制 DB 核对 |
| Casdoor SDK 出口 ban-list 被违反（CI 检查） | 构建失败 |

> 受保护操作不允许从已认证静默降级为匿名；不允许用缓存 claim 续命跨 token 自然过期；不允许 mock 成功路径。

## 11. 迁移：从 Zitadel 到 Casdoor

### 11.1 代码替换清单

| 文件 | 操作 | 说明 |
|------|------|------|
| `server/internal/pkg/oidc/management.go` | **删除** | Zitadel Management API client；Casdoor SDK 替代 |
| `server/internal/pkg/oidc/client.go:55` | 修改 | 移除 `urn:zitadel:iam:org:project:id:zitadel:aud` scope；改为 Casdoor scope（按 Casdoor 文档配置）|
| `server/internal/pkg/oidc/client.go:106-115` | 修改 | 移除 `ParseRolesFromRaw` 中的 Zitadel 项目角色解析；改为 Casdoor JWT 的 `roles` claim 解析 |
| `server/internal/pkg/oidc/claims.go` | 修改 | `Claims.OrgScopedRoles` 不再从 Zitadel org-scoped role claim 解析；仅作为 StuHelper 内部 school-scope grant 载体，最终来源必须是 DB/OpenFGA 投影 |
| `server/internal/pkg/oidc/` 其余 | 保留并 provider-neutral 化 | 标准 OIDC discovery / auth code+PKCE / token exchange / refresh / introspection 是协议层，可复用 |
| `server/internal/app/runtime.go:214` | 修改 | `oidc.NewClient(ctx, rt.cfg.Zitadel)` → `oidc.NewClient(ctx, rt.cfg.Casdoor)` |
| `server/internal/app/modules.go:146` | 修改 | issuer 配置切到 Casdoor |
| `server/internal/pkg/middleware/auth.go:74-76` | 修改注释 | "Zitadel introspection" 改为 "Casdoor introspection" |
| `server/internal/pkg/sms/handler.go:11-12` | 修改注释 | "Zitadel Action" 改为 "Casdoor Custom HTTP SMS Provider" |
| `server/internal/pkg/config/config.go` + `validation.go` | 修改 | `ZitadelConfig` → `CasdoorConfig`；`ZITADEL_*` → `CASDOOR_*` env vars |
| `server/internal/platform/casdoor/` | **新增** | Casdoor SDK 封装包；按 §4 出口白名单 |
| `server/internal/platform/authorization/` | **新增** | Authorization Service 实现；按 §5 接口 |
| `server/internal/pkg/capability/capability.go` | 调整 | 保留 capability 概念，但角色来源切到 Casdoor 扁平角色；移除嵌入 ID 的角色 |

### 11.2 配置替换

```bash
# 全局 grep 替换清单（实施时按文件分批）
ZITADEL_ISSUER          → CASDOOR_ENDPOINT
ZITADEL_CLIENT_ID       → CASDOOR_CLIENT_ID
ZITADEL_CLIENT_SECRET   → CASDOOR_CLIENT_SECRET
ZITADEL_REDIRECT_URI    → CASDOOR_REDIRECT_URI
ZITADEL_PROJECT_ID      → CASDOOR_ORGANIZATION  # Casdoor 用 organization 而非 project
ZITADEL_MANAGEMENT_PAT  → 拆分为 CASDOOR_ROLE_SYNC_* / CASDOOR_USER_LOOKUP_* / CASDOOR_APP_PROVISIONING_*（禁止万能 token）
ZITADEL_PUBLIC_SCHEME   → 移除（不需要）
ZITADEL_EXTERNALPORT    → CASDOOR_PORT (按 Casdoor 部署)
```

`.env.example` 全文重写，删除所有 `ZITADEL_*`。

### 11.3 基础设施

| 当前 | 操作 | 目标 |
|------|------|------|
| `infra/zitadel/` (Traefik + zitadel-api + zitadel-login) | **删除** | — |
| Docker compose `zitadel*` services | **删除** | — |
| Volume `stuhelper_zitadel_bootstrap` (PAT 持久化) | **删除** | — |
| PostgreSQL `zitadel` database | **删除** | — |
| `infra/casdoor/` | **新增** | Casdoor 单容器 + 配置 |
| PostgreSQL `casdoor` database + `casdoor_db_user` | **新增** | 隔离 |
| Network `zitadel-internal` | **删除** | — |
| Network `casdoor-internal` | **新增** | 替代 |

### 11.4 数据模型收口（绿地：已落入 baseline schema）

**项目当前状态**：开发中，未部署生产；按"绿地 + 无兼容"原则，`users.casdoor_subject` 已直接落入 baseline schema，不再保留 `external_id` 旧列兼容路径。

#### 11.4.1 Schema 调整

`users` 表直接使用 Casdoor 语义列：

```sql
casdoor_subject character varying(255) NOT NULL
```

#### 11.4.2 数据 truncate

由于项目无生产用户，直接清空所有用户派生数据，由首次 Casdoor 登录重新 provision。

**关键约束**：
- `users` 表的依赖**分两类**——FK 到 `users.id`（CASCADE 自动级联）与字符串 `user_hash` 关联（**不会被 CASCADE 级联**，必须显式 TRUNCATE）；
- 实施 PR 不得依赖本 spec 的硬编码表清单；必须在迁移前用 SQL 查询自动生成完整清单（见下方"清单生成查询"），避免 schema 演进后 spec 过期。

**清单生成查询**（在实施 PR 中实际执行，结果纳入 PR 描述）：

```sql
-- (1) FK 到 users.id 的表
SELECT conrelid::regclass AS table_name
  FROM pg_constraint
 WHERE confrelid = 'users'::regclass AND contype = 'f';

-- (2) 含 user_hash 列的表
SELECT table_name
  FROM information_schema.columns
 WHERE column_name = 'user_hash'
   AND table_schema = current_schema();

-- (3) 含 admin_user_id 等字符串 admin 标识列的表
SELECT table_name, column_name
  FROM information_schema.columns
 WHERE column_name IN ('admin_user_id', 'reviewer_user_id', 'approver_user_id')
   AND table_schema = current_schema();
```

**当前 schema 下的清单（仅作 `server/migrations/000001_initial_schema.up.sql` 的初始化参考；后续 baseline 更新可能改变表结构，例如某些 user_hash 列改为 user_id FK——`不可复制执行`，必须以上方"清单生成查询"实际跑出的结果为准）**：

```sql
BEGIN;

-- 类 1：FK 到 users.id（CASCADE 会自动级联）
TRUNCATE TABLE
  users,
  user_identities,
  user_profiles
RESTART IDENTITY CASCADE;

-- 类 2：通过 user_hash 字符串关联（CASCADE 不覆盖，必须显式列出）
TRUNCATE TABLE
  reviews,
  review_votes,
  review_reports,
  review_drafts,
  review_replies,
  course_favorites,
  notifications
RESTART IDENTITY CASCADE;

-- 类 3：审计事件（旧 admin_operation_logs 已从 baseline schema 移除；
--        现有 audit_events.category 约束只允许 'audit' / 'admin_operation' / 'domain_event'
--        见 server/migrations/000001_initial_schema.up.sql；
--        绿地阶段所有历史审计行为均关联旧 user_id / 旧 admin_user_id 字符串，可一并清空）
TRUNCATE TABLE audit_events RESTART IDENTITY CASCADE;

COMMIT;
```

> 业务实体表（如 `courses`、`teachers`、`departments`、`terms`、`course_categories`、`rating_dimensions`、`course_rating_stats`、`teacher_rating_stats`、`sensitive_words`、`school_configs`、`system_configs`、`academic.buaa_students`）通常**不**清空——这些是业务参考数据，与用户身份无关。但若开发数据库中含污染数据，实施 PR 应单独评估。

#### 11.4.3 代码同步修改

| 文件 | 修改 |
|------|------|
| `server/migrations/000001_initial_schema.up.sql` | `users` 表只保留 `casdoor_subject`，不保留 `external_id` 双列或 rename 兼容路径 |
| `server/internal/modules/user/repository_auth_sync.go` | 所有用户同步查询直接读写 `casdoor_subject` / `CasdoorSubject` |
| `server/internal/modules/user/` 其它文件 | 身份侧外部 subject 统一使用 `casdoor_subject` / `CasdoorSubject`；教务导入域的 `academic_* external_id` 不在此范围 |

#### 11.4.4 运行时缓存清理

- Redis 全量 flush（按 session / token blacklist / OIDC state / OTP 等 prefix）；
- 现有 token blacklist 不迁移；
- shadow user 表：跟随 §11.4.2 truncate，不保留（首次 Casdoor 登录重建）。

#### 11.4.5 禁止做的事

- **禁止**保留旧 Zitadel `external_id` 值并尝试映射到 Casdoor `name`：两个 IDP 的 subject 命名空间不重合，强行映射会引入数据污染；
- **禁止**只清空列值不改名：列名 `external_id` 在 Casdoor 语境下含义错误，且 `NOT NULL UNIQUE` 约束不允许 NULL；
- **禁止**为兼容旧数据保留双列：违反 memory 中的 "no compat shims" 原则。

### 11.5 退役顺序

> **关键时序约束**：以下步骤**必须严格按序**执行；不允许并行。项目已采纳 "no compat shim" 原则，baseline schema 已直接使用 `casdoor_subject`，不再保留 `external_id` 双列、rename SQL 或运行时兼容层。

```
1. 在新分支实施 §11.1 - §11.3 代码 / 配置 / infra 修改
2. Casdoor 容器上线，bootstrap 创建 organization / applications / roles / providers
3. CI 检查：Casdoor SDK 出口 ban-list、env var 全量替换、ZITADEL_* 零引用
4. Staging 环境跑 §15 验证策略全集
5. 切换生产：
   a. Casdoor 部署
   b. 后端切流（蓝绿或维护窗口；项目无生产用户，可大胆切）
   c. 清空 Redis 缓存（session / token blacklist / OIDC state / OTP 等）；按 §11.4.2 truncate 清单清空 user-derived DB 数据
   d. 验证一方应用 OIDC 登录（web / admin / uniapp）
   e. Zitadel 容器停止
   f. infra/zitadel/ + 相关 volume / db / network 删除
6. 文档更新（authorization-model.md / auth-and-session.md / security-model.md / .env.example）
7. 旧 spec 文件归档或删除
```

## 12. Bootstrap

生产 bootstrap 必须**自动化且幂等**。

创建并校验：
- Casdoor organization `stuhelper`；
- Applications：`stuhelper-web`、`stuhelper-admin`、`stuhelper-uniapp`；
- 7 个扁平 role（§3.3）；
- Custom HTTP SMS Provider → `/internal/sms/send`；
- Custom SMTP / HTTP Email Provider（仅当 `CASDOOR_EMAIL_PROVIDER_ENABLED=true`；当前默认延后）；
- Certificate / public key（按 Casdoor 数据初始化文档）；
- 初始管理员账号（密码从 env 读，启动后强制修改提示）。

**生产环境**缺必要 Provider 启动失败；Email Provider 只有在 env 显式启用时才进入必要 Provider 清单。**开发环境**可关闭，但必须 env 显式 + 日志告警。

参考：[Casdoor 数据初始化](https://casdoor.org/docs/deployment/data-initialization/)、[Casdoor SMS Provider](https://casdoor.org/docs/provider/sms/overview/)。

## 13. 包边界

```text
server/internal/platform/casdoor/
  client.go        SDK 初始化 + healthcheck
  applications.go  应用 CRUD（开放平台预留）
  users.go         用户查询 / 投影
  roles.go         角色分配 / 撤销（outbox 消费方）
  providers.go     Provider 配置校验
  bootstrap.go     幂等 bootstrap

server/internal/platform/authorization/
  service.go       AuthorizationService 实现（§5）
  facts.go         业务事实查询（封装 DB）
  fga.go           OpenFGA check 封装
  decision.go      组合决策逻辑
  errors.go        Decision / Error 类型

server/internal/pkg/oidc/   (provider-neutral 标准 OIDC)
  client.go        discovery / auth code+PKCE / token exchange / refresh / introspection
  claims.go        标准 OIDC claim 解析（不含 IDP-specific）
  jwks.go          JWKS 缓存 / 校验

server/internal/pkg/sms/     (保留)
  tencent.go       腾讯云 SMS（不变）
  handler.go       /internal/sms/send（注释更新）

server/internal/modules/*    (业务模块)
  禁止 import casdoorsdk
  禁止 import openfga client
  禁止调用 *.Enforce / *.BatchEnforce / *.GetPermissions
  授权检查仅通过 platform/authorization/AuthorizationService
```

### 13.1 架构硬规则（CI 检查 + code review 强制）

#### 规则 A：`casdoor_subject` ↔ `users.id` 边界

- 业务表外键**只**指向 `users.id`（内部稳定 BIGINT 主键）；
- `casdoor_subject` 是**外部身份键**，仅在以下三层流转：
  - `pkg/oidc/` — token 解析；
  - `platform/casdoor/` — Casdoor SDK 调用；
  - `platform/authorization/` — Subject 解析后立即换成 `users.id`，向下游只暴露 `users.id`；
- **禁止**业务模块（`server/internal/modules/*`）出现 `casdoor_subject` 字段；
- **禁止**新建业务表把 `casdoor_subject` 当业务主键；
- OpenFGA tuple 中 `user:{id}` 的 `id` 部分**统一使用 `users.id`**，不使用 Casdoor subject——否则未来再迁 IDP 会非常痛苦。

CI 增加 grep 检查（与 §4.3 同模式）：业务模块禁止出现 `casdoor_subject` / `CasdoorSubject` 标识符。

#### 规则 B：Casdoor 管理 API 最小权限

- **禁止**使用一个万能 Casdoor admin token 服务所有用途；
- 按用途拆分 service account credential：
  - `casdoor-admin-role-sync` — 仅 add/remove role；
  - `casdoor-admin-app-provisioning` — 仅 create/update/delete application；
  - `casdoor-admin-user-lookup` — 只读 user；
  - `casdoor-admin-bootstrap` — 仅 bootstrap 阶段使用，运行时不挂载；
- 当前落地约束：`verified_student` role outbox worker 必须同时配置 `CASDOOR_ROLE_SYNC_*` 与 `CASDOOR_USER_LOOKUP_*` 两组凭据；role 更新与 user lookup 不共用 secret；
- 每个 credential 配置最小 Casdoor 权限；
- 每个 secret 独立轮换（不联动）；
- 所有 Casdoor admin API 调用落审计：调用方 service account / 操作 / 目标 / 结果 / request_id（保留期见 §14.3）；
- admin credential **禁止**进入业务模块；只在 `platform/casdoor/` 内使用。

## 14. 安全与隐私基线

> 本节列出的是通用安全基线，**与 IDP 选型无关**；保留在 spec 中作为实施清单，不作为 Casdoor 选型论据。

### 14.1 通用基线

1. PKCE 强制（public client 必须，confidential client 推荐）；
2. redirect URI 精确匹配，生产禁 wildcard；
3. client secret 加密存储或仅展示一次；
4. Casdoor JWT（access + ID token 同 payload）默认不暴露 phone / verified status / school / identity type / qq（§8.3）；
5. Cookie：HttpOnly + SameSite=Lax + Secure（生产）；
6. CSRF 防护（refresh / 写操作）；
7. 限流：登录、refresh、SMS 发送、敏感读 API 按 IP + user 限流；
8. 第三方应用最小权限（开放平台 v1 详细规则）；
9. 敏感操作审计：写日志含 request_id / user_id / app_id / decision；
10. 生产 bootstrap 必须验证所有依赖；缺关键依赖启动失败。

### 14.2 登录暴力 / 账号锁定

| 维度 | 阈值（默认值，可配置但不得低于此基线） | 行动 |
|------|--------------------------------------|------|
| 账户级连续失败 | 5 次 / 5 分钟 | 软锁该账户 5 分钟 |
| 账户级累计失败 | 20 次 / 1 小时 | 硬锁；解锁需 super_admin 介入 |
| IP 级连续失败 | 20 次 / 1 小时 | IP 软锁 1 小时 |
| OTP 发送频率（同手机号） | 1 次 / 60 秒；5 次 / 1 小时 | 拒绝并 429 |
| OTP 校验失败 | 5 次 / 同一 OTP 生命周期 | 当前 OTP 作废，需重发 |

- **错误响应防枚举**：账号不存在 / 密码错误 / 账户锁定 → 返回**统一**错误码 + 文案；不在响应中暴露用户存在性；
- **承担方**：Casdoor 内置限流承担账户级 + OTP 级；StuHelper 边缘层（API gateway / middleware）补足 IP 级；
- 锁定事件 + 解锁事件落审计（保留期见 §14.3）。

### 14.3 IAM 自身审计保留

复用现行 `audit_events` 表（定义在 `server/migrations/000001_initial_schema.up.sql`）。**不**新增 `event_category` 字段——现有 schema 已有 `category` (CHECK IN 'audit'/'admin_operation'/'domain_event') + `event_type` (TEXT) + `action` + `resource_type` 四个分类维度，足以表达 IAM 事件类别。IAM 事件按以下映射写入：

| IAM 事件 | category | event_type 前缀 |
|---------|----------|-----------------|
| 登录成功 / 失败 / 锁定 | `audit` | `iam.auth.*` |
| MFA 注册 / 禁用 / reset / step-up | `audit` | `iam.mfa.*` |
| 角色变更 | `admin_operation` | `iam.role.*` |
| Service account credential 创建 / 轮换 / 吊销 | `admin_operation` | `iam.service_account.*` |
| Casdoor application 创建 / secret 轮换 | `admin_operation` | `iam.casdoor_app.*` |
| Token revoke / logout-all | `audit` | `iam.token.*` |
| Casdoor admin API 调用 | `admin_operation` | `iam.casdoor_admin_api.*` |

保留期：

| 事件类型 | 保留期 |
|---------|--------|
| 登录成功 | 90 天（与现行管理员操作审计基线一致）|
| 登录失败 | 1 年（用于安全分析）|
| 账户锁定 / 解锁 | 1 年 |
| MFA 注册 / 禁用 / reset | 1 年 |
| Recovery code 使用 | 1 年 |
| Step-up MFA 失败 / 成功 | 1 年 |
| 角色变更 | **3 年** |
| Service account credential 创建 / 轮换 / 吊销 | **3 年** |
| Service account 调用敏感端点 | **3 年** |
| Casdoor application 创建 / secret 轮换 / 吊销 | **3 年** |
| Token revoke / logout-all | 1 年 |
| Casdoor admin API 调用（含调用方 service account） | 1 年 |

> 高保留期项（3 年）出于合规与争议溯源需要，不得低于此基线。低保留期项可视存储成本调整，但不得低于 90 天。

### 14.4 JWKS / 签名密钥与 Client Secret 存储

#### 14.4.1 Token 签名密钥

- **签名算法**：默认 RS256；可选 ES256；**禁止** HS\*、`none`；
- **StuHelper verifier**：`server/internal/pkg/oidc` 在拉 JWKS 前先解析 JWT header，只允许 `RS256` / `ES256`；`HS*` / `none` 直接拒绝，避免错误 provider 配置被业务路径接受；
- **JWKS rotation**：90 天周期；旧 key 保留 14 天覆盖 token TTL；JWKS endpoint 同时返回新旧 key；
- **JWKS cache（StuHelper 端）**：5 分钟 TTL；遇 unknown `kid` 时立即失效并重新拉取（一次性回源，失败 503，不允许 stale 续命）；
- **StuHelper cache 实现**：`server/internal/pkg/oidc` 在 `go-oidc` RemoteKeySet 外层维护 5 分钟 TTL；TTL 内复用本地 key cache，TTL 后重建 RemoteKeySet 并重新拉取 JWKS；
- **Casdoor 端 key 轮换** 必须在 JWKS endpoint 提前 14 天暴露新 key，再切换签名 key，避免 verifier 端来不及拉新 key。

#### 14.4.2 Client Secret 存储

> 仓库现状：**未引入** PostgreSQL `pgcrypto` 扩展（`grep -r pgcrypto server/migrations/` 为空）；现有 PII 加密在应用层 `server/internal/pkg/crypto/`（cipher / HMAC）。本 spec 据此重写。

- **v1 方案**：复用现有应用层 `pkg/crypto` 接口加密 client secret；密钥从环境变量注入（与现行 PII 加密一致）；
- **未来演进**：若引入 Vault / KMS，迁移路径独立 PR，不在 IAM v2 范围；
- **Open Platform app secret**：注册时只展示**一次**；后续仅显示前 8 位 + `***`；轮换时同样只展示一次；旧 secret 立即失效；
- **Casdoor admin credential**（§13.1 规则 B 拆分的 4 个）：同 `pkg/crypto` 加密存储；运行时通过 secret 注入而非配置文件硬编码。

#### 14.4.3 Secret 不进 Git / log

- CI 增加 secret 扫描规则（gitleaks / trufflehog 二选一）；
- 应用日志 / 审计日志中 token / secret / refresh token 必须**脱敏**（仅记前 8 位 hash）；
- error response 中**绝不**回显 secret / token 内容。

## 15. 验证策略

自动化检查至少覆盖：

1. Casdoor bootstrap 创建 organization / applications / 7 个 role / SMS Provider；当 `CASDOOR_EMAIL_PROVIDER_ENABLED=true` 时还必须创建 SMTP / HTTP Email Provider；
2. 一方应用 OIDC 登录 happy path（web / admin / uniapp）；
3. **投影一致性 case A**：DB `student_verified=true`、Casdoor 角色未同步 → 敏感操作放行（DB 为准）+ 漂移告警触发；
4. **投影一致性 case B**：DB `student_verified=false`、Casdoor 仍有 `verified_student` 角色 → 敏感操作拒绝 + 即时撤销触发；
5. **Token 本地校验**：Casdoor 容器停止后，已签发未过期 JWT 仍可本地 JWKS 校验通过；
6. **Casdoor 不可用 fail-closed**：login / refresh / userinfo / management 全部 503；
7. **OpenFGA 不可用**：资源关系操作 503；
8. **业务 DB 不可用**：业务事实操作 503；
9. **SDK 出口 ban-list**：CI grep 业务模块零 `casdoorsdk` import、零 `Enforce` 调用；
10. **env var 替换完整**：CI grep `ZITADEL_` 在 `server/`、`infra/`、`.env.example` 内零引用；
11. **资源授权**：owner 可编辑自己的 review；
12. **资源授权**：school admin 只能管理本学校 review（OpenFGA tuple 边界）；
13. **资源授权**：section moderator 只能管理被授予的板块；
14. **角色目录**：Casdoor role 列表完全等于 §3.3 的 7 个扁平角色，无嵌入 ID；
15. **SMS 转发链路**：Casdoor 登录验证码 → `/internal/sms/send` → 腾讯云 API；端点鉴权失败 401。
16. **Casdoor capability gate**：`infra/ops/casdoor-capability-probe.sh` 在测试 Casdoor 实例上生成 step-up URL、检查 `amr` / `auth_time` / `acr` claim，并按显式开关验证 refresh token 是否 single-use。

## 16. Open Platform 边界

第三方应用注册、scope 目录、scope 审批、用户 consent、disclosure API、按 app/user 限流、审计、吊销 → 见 [`open-platform-v1.md`](open-platform-v1.md)。

IAM 与 Open Platform 的稳定分工：

- Casdoor 负责账号登录、SMS / Email provider、一方应用 registry 和用户手机号真相源。
- StuHelper Identity / Open Platform 负责第三方 app 元数据、OIDC 授权码/token、scope 审批、用户 consent、disclosure API、审计和撤销。
- 第三方应用必须信任 `iss=https://id.stuhelper.com`；业务字段只按已审批且已授权的 scope 进入 `id` token/UserInfo。
- Scope consent 在业务 DB 而非 OpenFGA 中建模，理由见 [`open-platform-v1.md`](open-platform-v1.md)。
- OpenFGA 只承载未来“应用 → 具体资源”的关系授权。

当前 Open Platform v1 baseline 已落地。后续运营面、撤销 UI、限流、指标、生产准入探针和资源 API v1.1 由 [`current-project-open-items.md`](../internal/exec-plans/active/current-project-open-items.md) 跟踪。

## 17. 后续实施阶段

本文是设计文档，**不是实施计划**。后续实施计划应拆为独立 exec-plan：

1. Casdoor 基础设施 + bootstrap；
2. `platform/casdoor/` SDK 封装（按出口白名单）；
3. `platform/authorization/` 服务实现；
4. `pkg/oidc/` provider-neutral 化；
5. 业务模块迁移到 Authorization Service（移除直接 Casdoor / FGA 调用）；
6. Outbox 同步（DB → Casdoor 角色）；
7. SMS Provider 切换；Email Provider 在 email service 落地后按 `CASDOOR_EMAIL_PROVIDER_ENABLED=true` 单独启用；
8. Zitadel 退役（代码 / infra / env / data）；
9. 文档同步（`authorization-model.md` / `auth-and-session.md` / `security-model.md`）；
10. 验证策略落地；
11. Open Platform 生产运营面与资源 API v1.1 按 [`open-platform-v1.md`](open-platform-v1.md) 的下一步拆独立计划。

## 18. 最终设计立场

Casdoor 承载身份与应用 registry，并提供少量扁平粗粒度角色投影。它**不**承载 StuHelper 业务事实，**不**进入业务授权决策路径。

StuHelper Authorization Service 是业务模块**唯一**授权入口，组合 token 主体、DB 事实、OpenFGA 关系；统一 fail-closed。

OpenFGA 是资源关系权威，但仅作为 Authorization Service 内部依赖，不直接被业务模块调用。

业务 DB 是实名、学生认证、学校归属、手机号验证投影、QQ 绑定与所有业务实体的真相源；完整手机号真相源在 Casdoor。Casdoor 角色 claim 与业务事实冲突时，**敏感操作以 DB 为准**。

开放平台是独立产品域，不把业务 scope 和用户 consent 混进 Casdoor 或 OpenFGA。

## 19. 参考

- Casdoor OIDC 应用与登录：<https://casdoor.org/docs/how-to-connect/oidc-client/>
- Casdoor SMS Provider：<https://casdoor.org/docs/provider/sms/overview/>
- Casdoor Email Provider：<https://casdoor.ai/docs/provider/email/overview/>
- Casdoor 数据初始化：<https://casdoor.org/docs/deployment/data-initialization/>
- Casdoor Go SDK：<https://github.com/casdoor/casdoor-go-sdk>
- 现行授权模型：[`docs/design/authorization-model.md`](authorization-model.md)
- 现行认证模型：[`docs/design/auth-and-session.md`](auth-and-session.md)
- 现行安全模型：[`docs/design/security-model.md`](security-model.md)
- OpenFGA Zanzibar 风格 ReBAC：<https://openfga.dev/docs/concepts>
