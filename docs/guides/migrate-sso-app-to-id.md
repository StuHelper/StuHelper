---
type: guide
audience: integrator
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-05-22
---

# Casdoor 直连应用迁移到 StuHelper Identity

本指南用于把原来直接接入 `https://sso.stuhelper.com` 的应用迁移到 `https://id.stuhelper.com`。迁移后，应用不再把 Casdoor 当作 StuHelper 的对外 issuer，而是通过 StuHelper Identity 完成 OIDC authorization code flow、UserInfo、token introspection 和 revoke。

## 目标状态

| 项 | 迁移前 | 迁移后 |
|----|--------|--------|
| Issuer / discovery | `https://sso.stuhelper.com` | `https://id.stuhelper.com` |
| Authorization endpoint | Casdoor discovery 返回值 | `https://id.stuhelper.com/oauth2/authorize` |
| Token endpoint | Casdoor discovery 返回值 | `https://id.stuhelper.com/oauth2/token` |
| UserInfo endpoint | Casdoor discovery 返回值 | `https://id.stuhelper.com/oidc/userinfo` |
| JWKS | Casdoor JWKS | `https://id.stuhelper.com/.well-known/jwks.json` |
| Token issuer claim | `iss=https://sso.stuhelper.com` | `iss=https://id.stuhelper.com` |
| 用户主体 claim | Casdoor subject | StuHelper Identity subject，格式为 `sub=stuhelper:<internal-user-id>` |
| 用户字段来源 | Casdoor 用户 API 或 Casdoor token | StuHelper Identity 按已审批 scope 和用户 consent 披露 |

`sso.stuhelper.com` 仍然负责账号密码、注册、短信和上游登录会话。应用侧不再直接信任它作为 StuHelper Open Platform issuer。

## 迁移前清单

先从原 Casdoor 应用和应用配置中记录：

- Casdoor application name。
- 当前 `client_id`。
- 当前 `client_secret`，如果应用侧仍持有。
- 所有生产、预发、本地回调地址。
- 当前请求的 OIDC scope。
- 应用实际读取的用户字段，例如用户名、邮箱、手机号、学生认证状态、学校。
- 应用是否使用 refresh token、introspection、revoke 或 Casdoor 用户 API。
- 应用是否把 Casdoor `sub` 当作本地账号主键；如果是，迁移前必须准备按已验证邮箱、手机号或其他可信业务键做账号关联 / 回填，不能假设 `sub` 在迁移后保持不变。

回调地址必须是精确 URI。StuHelper Identity 不接受 wildcard、fragment、regex、通配子域或运行时自由 return URL。

## Scope 映射

迁移时按应用实际需要申请最小 scope。

| 原 OIDC scope / 字段 | StuHelper Identity scope |
|----------------------|--------------------------|
| `openid` | `openid` |
| `profile`、用户名、头像 | `profile.basic.read` |
| `email` | `email.read` |
| `phone`、手机号 | `phone.read` |
| 实名/身份状态 | `stu.identity.status.read` |
| 身份类型 | `stu.identity.type.read` |
| 学生认证状态 | `stu.student.status.read` |
| 学校信息 | `stu.student.school.read` |

标准 scope `profile`、`email`、`phone` 会在服务端映射到对应业务 scope，但迁移申请里建议显式写业务 scope，便于审批和审计。

## 管理员导入应用

管理员使用具备 `open_platform:manage` capability 的账号调用导入接口。接口定义以 `server/api/openapi.yaml` 为准。

```bash
curl -X POST "https://stuhelper.com/api/v1/admin/open-platform/apps/import-casdoor" \
  -H "Authorization: Bearer <admin-access-token>" \
  -H "Content-Type: application/json" \
  -d '{
    "casdoorApplicationName": "legacy-client-app",
    "displayName": "Legacy Client App",
    "description": "Migrated from direct Casdoor OIDC",
    "homepageURL": "https://client.example.com",
    "privacyPolicyURL": "https://client.example.com/privacy",
    "redirectURIs": [
      "https://client.example.com/oauth/callback"
    ],
    "clientSecret": "<optional-current-client-secret>",
    "scopes": [
      {
        "scope": "profile.basic.read",
        "reason": "Display the signed-in user name and avatar"
      },
      {
        "scope": "email.read",
        "reason": "Bind account email"
      }
    ]
  }'
```

响应中的 `clientSecretSource` 决定应用侧是否需要更新密钥：

| `clientSecretSource` | 含义 | 应用侧动作 |
|----------------------|------|------------|
| `provided` | 使用请求里的 `clientSecret` 重新 hash 入库 | 保持原 secret |
| `casdoor` | 从 Casdoor application 读取到 secret 并导入 | 保持原 secret |
| `generated` | 服务端生成了新 secret，并只在本次响应展示 | 立即更新应用 secret |

如果响应携带 `clientSecret`，它只展示一次，应立即写入应用密钥管理系统。

## 应用配置修改

应用侧把 OIDC discovery 或 issuer 改为：

```text
https://id.stuhelper.com/.well-known/openid-configuration
```

如果应用不支持 discovery，直接配置：

```text
issuer=https://id.stuhelper.com
authorization_endpoint=https://id.stuhelper.com/oauth2/authorize
token_endpoint=https://id.stuhelper.com/oauth2/token
userinfo_endpoint=https://id.stuhelper.com/oidc/userinfo
jwks_uri=https://id.stuhelper.com/.well-known/jwks.json
introspection_endpoint=https://id.stuhelper.com/oauth2/introspect
revocation_endpoint=https://id.stuhelper.com/oauth2/revoke
```

授权请求使用 authorization code flow：

```text
GET https://id.stuhelper.com/oauth2/authorize
  ?response_type=code
  &client_id=<client_id>
  &redirect_uri=<exact-registered-redirect-uri>
  &scope=openid%20profile%20email
  &state=<opaque-state>
  &code_challenge=<pkce-s256-challenge>
  &code_challenge_method=S256
```

token 请求使用 client secret 和 PKCE verifier：

```bash
curl -X POST "https://id.stuhelper.com/oauth2/token" \
  -u "<client_id>:<client_secret>" \
  -H "Content-Type: application/x-www-form-urlencoded" \
  --data-urlencode "grant_type=authorization_code" \
  --data-urlencode "code=<authorization-code>" \
  --data-urlencode "redirect_uri=https://client.example.com/oauth/callback" \
  --data-urlencode "code_verifier=<pkce-code-verifier>"
```

## 验收

迁移完成后必须验证：

- discovery 的 `issuer` 等于 `https://id.stuhelper.com`。
- `id_token` 和 `access_token` 的 `iss` 等于 `https://id.stuhelper.com`。
- `sub` 使用 StuHelper Identity subject，格式为 `stuhelper:<internal-user-id>`；应用能把新 subject 正确关联到既有本地账号。
- `aud` 或 `client_id` 等于导入后的 `client_id`。
- 未登记的 `redirect_uri` 被拒绝。
- 未审批或未授权的 scope 不会出现在 UserInfo。
- UserInfo 只返回应用申请且用户同意的字段。
- 请求 `phone.read` 时，如果本地加密手机号投影缺失或不可解密，应拒绝披露而不是返回空手机号伪装成功。
- introspection 对有效 token 返回 `active=true`，revoke 后返回 `active=false`。
- 应用不再调用 Casdoor 用户 API 读取 StuHelper 业务字段。

## 切换与回滚

推荐按以下顺序切换：

1. 管理员导入应用并确认 scope 已审批。
2. 在预发环境把 issuer 切到 `https://id.stuhelper.com`，跑完整授权码登录、UserInfo、revoke 验收。
3. 在生产应用灰度切换 issuer 和 endpoint。
4. 确认新 token 的 `iss`、`aud`、UserInfo 字段和审计事件都正确。
5. 停止给直连 Casdoor 应用新增 scope 或新增回调地址。
6. 确认没有生产流量依赖 Casdoor 直连后，吊销或冻结原 Casdoor 第三方应用。

回滚只允许回到迁移前已登记的 Casdoor 配置。回滚期间不要扩大 Casdoor 应用权限；问题修复后应重新切回 `id.stuhelper.com`。
