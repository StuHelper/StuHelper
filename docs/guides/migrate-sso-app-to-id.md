---
type: guide
audience: backend-dev, ops
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-05-24
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

应用侧继续按 OIDC 习惯请求 scope。若授权请求为 `openid profile email`，token response、access token claim 和 introspection 里的 `scope` 仍返回 `openid profile email`；服务端内部才把它们映射为 `profile.basic.read` 和 `email.read` 做审批、用户 consent 和字段披露。直接请求业务 scope 也被支持，例如 `openid profile.basic.read email.read resource.read` 会原样保留在 token scope 中。需要登录身份或 UserInfo 时必须请求并获批 `openid`；纯 `resource.read` / `resource.write` 授权码流只返回 access token，不返回 `id_token`，UserInfo 也会拒绝该 token。

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

只消费 OAuth2 authorization server metadata 的网关或资源服务器，也可以使用等价的 RFC 8414
元数据地址：

```text
https://id.stuhelper.com/.well-known/oauth-authorization-server
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
end_session_endpoint=https://id.stuhelper.com/oauth2/logout
code_challenge_methods_supported=S256
prompt_values_supported=none,login,consent
grant_types_supported=authorization_code,refresh_token,client_credentials
authorization_response_iss_parameter_supported=true
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

成功授权回调会返回 `code`、原始 `state` 和 `iss=https://id.stuhelper.com`。支持
RFC 9207 的客户端应校验授权响应中的 `iss` 与 discovery `issuer` 一致，用于降低
多 issuer 或迁移期并存接入时的 authorization code mix-up 风险；暂不支持该参数的客户端必须按
OAuth 兼容要求忽略未知 query 参数。

需要静默探测登录态的应用可以在同一授权请求上追加 `prompt=none`。当用户未在
`id.stuhelper.com` 建立会话时，Identity Server 会在校验 `client_id`、精确匹配的
`redirect_uri` 和 S256 PKCE 参数后，回调应用并携带
`error=login_required`、`iss=https://id.stuhelper.com` 与原始 `state`；当用户已登录但仍需补资料或重新同意 scope 时，
回调会携带 `error=interaction_required` 或 `error=consent_required`，同样包含 `iss`。这些场景都不会展示
StuHelper 登录、资料补全或授权同意页面。

授权响应只支持 query response mode。大多数客户端不需要传 `response_mode`；如果库会显式传参，
只能使用 `response_mode=query`。Identity Server 会把授权码、错误、`state` 和 RFC 9207 `iss`
写入 redirect URI query，并拒绝 `fragment`、`form_post` 或带空白的 response mode。

需要强制重新认证的应用可以在授权请求中使用 `prompt=login`，或使用 `max_age=0`。Identity
Server 会把当前授权请求转接到 StuHelper 登录页并触发上游 Casdoor `prompt=login&max_age=0`
授权，完成重新认证后继续原授权请求。转接时会消费掉原始授权请求中的 `prompt=login` 和
`max_age=0`，避免重新认证成功后再次进入 reauth 循环；正数 `max_age` 会按当前会话
`auth_time` 校验，缺少可信 `auth_time` 时 fail-closed 到重新认证。

需要让用户重新确认已授权的资料披露 scope 时，可以使用 `prompt=consent`。如果本次授权
请求包含 `email`、`profile`、`phone`、`offline_access` 或 StuHelper 业务 disclosure
scope，即使用户此前已经同意，Identity Server 也会重新展示 StuHelper consent 页并刷新
对应 consent 审计；`prompt=login consent` 会先完成上游重新认证，再保留 `prompt=consent`
继续授权确认。`prompt=none` 不能与 `consent` 或 `login` 组合。

需要长期保持登录态的服务端应用，可以额外申请并请求 `offline_access` scope：

```text
GET https://id.stuhelper.com/oauth2/authorize
  ?response_type=code
  &client_id=<client_id>
  &redirect_uri=<exact-registered-redirect-uri>
  &scope=openid%20profile%20offline_access
  &state=<opaque-state>
  &code_challenge=<pkce-s256-challenge>
  &code_challenge_method=S256
```

`offline_access` 是需要开发者申请、管理员审批、用户同意并可由用户撤回的 scope。只有授权码中包含
`offline_access` 时，`/oauth2/token` 才返回 `refresh_token`。刷新请求使用同一个 confidential client
认证，并执行 refresh token rotation：

```text
POST https://id.stuhelper.com/oauth2/token
grant_type=refresh_token
refresh_token=<current-refresh-token>
```

每次刷新都会重新校验 app 仍为 approved、`offline_access` 与相关业务 scope 仍被批准、用户 consent
仍 active；用户撤回授权、管理员撤销 scope 或暂停 / 吊销 app 后，旧 refresh token 不能再换取新 token。
Identity access token 与 refresh token 会绑定发行时的 disclosure consent 指纹；用户撤回后再重新授权、
或通过 `prompt=consent` 重新确认授权，都会生成新的 consent 指纹，旧 token 的 UserInfo、introspection
和 refresh grant 继续返回 inactive / invalid，而不会因为当前授权再次存在而复活。
刷新请求可以额外携带 `scope` 来收窄长期授权，例如从 `openid profile email offline_access`
收窄为 `openid profile offline_access`；请求的 scope 必须是原 refresh grant 的子集，并保留
`openid offline_access`，否则返回 `invalid_scope`。收窄成功后，token response、access token、
ID token 和下一代 refresh token 都只携带收窄后的 scope，后续刷新不会静默恢复已移除字段。
refresh token 有效期由服务端 `IDENTITY_REFRESH_TOKEN_TTL` 控制，默认 2592000 秒（30 天），生产可在
3600 到 2592000 秒之间收紧。服务端只用 refresh token 的哈希作为 Redis 索引，避免 keyspace
或备份直接暴露可使用的 token；每次 refresh grant 和 refresh token introspection 还会校验该
token hash 仍是 refresh token family 的当前 token，family 缺失、已撤销或指向其他 token 时返回
无效。服务端在签发新 access token / ID token 前会先消费当前 refresh token；已被 rotation 消费过的 refresh token 如果被同一个 client 再次提交到 refresh grant，
服务端会撤销当前 refresh token family，写入 `iam.token.revoked` 审计事件，并要求用户重新授权。同一个 client 把已被 rotation 消费的 refresh token 提交到
`/oauth2/revoke` 时，也会撤销当前 refresh token family，用于覆盖应用登出与后台刷新并发时只拿到旧 token 的场景。

`/oauth2/token`、`/oauth2/introspect` 和 `/oauth2/revoke` 都支持 `client_secret_basic`
和 `client_secret_post`。同一个请求只能选择一种 client authentication method；如果同时发送
Authorization Basic 头和 body `client_id` / `client_secret`，Identity Server 会返回
`invalid_client`，并携带 `WWW-Authenticate: Basic realm="StuHelper Identity"`。
introspection 和 revoke 请求可以携带标准 `token_type_hint=access_token|refresh_token`；Identity
Server 会把 hint 作为查找优化而不是安全决策，hint 错误或未知时仍按所有支持 token 类型继续处理。

服务端到服务端的资源类集成使用 `client_credentials` grant。该 grant 只接受应用级
`resource.read` / `resource.write` scope，不接受 `openid`、`profile`、`email`、`phone`
或 `offline_access`，也不会返回 `id_token` / `refresh_token`：

```text
POST https://id.stuhelper.com/oauth2/token
grant_type=client_credentials
scope=resource.read
Authorization: Basic base64(client_id:client_secret)
```

返回的 app-only access token 使用 `sub=client:<client_id>`，introspection 会返回
`grant_type=client_credentials`，并在每次校验时重新检查应用仍为 approved 且对应资源 scope
仍被批准。该 token 不能调用 `/oidc/userinfo`。

检查具体资源访问时，推荐把该 access token 作为 Bearer token 传给资源访问 API；旧版
`clientID` / `clientSecret` body 认证仍保留用于兼容，但两种认证方式互斥。携带
`Authorization: Bearer ...` 时不要再在 body 里发送 `clientID` 或 `clientSecret`，否则会按无效请求拒绝。
使用 body 兼容认证时不要发送 `Authorization` 头；非 Bearer 的 Authorization scheme 会被拒绝：

```text
POST https://api.stuhelper.com/api/v1/open-platform/resources/access/check
Authorization: Bearer <client_credentials-access-token>

{
  "resourceType": "resource_item",
  "resourceID": "42",
  "action": "read"
}
```

需要从第三方应用触发单点登出的场景，使用 `end_session_endpoint`：

```text
GET https://id.stuhelper.com/oauth2/logout
  ?client_id=<client_id>
  &post_logout_redirect_uri=<exact-registered-redirect-uri>
  &state=<opaque-state>
```

`post_logout_redirect_uri` 必须精确匹配该应用已注册的 redirect URI。浏览器当前已有
StuHelper 会话时，Identity Server 会撤销当前 session、清理认证 cookie，然后重定向回应用；
没有当前会话时只执行 redirect URI 校验和回跳。客户端也可以用当前 `id_token` 作为
`id_token_hint` 省略 `client_id`；一旦请求携带 `id_token_hint`，即使没有
`post_logout_redirect_uri`，该 hint 也必须是 StuHelper Identity 签发给该 client 的 ID token，
不能用 access token 或任意字符串代替。

StuHelper Identity 强制使用 S256 PKCE。`code_verifier` 必须是 43 到 128 个字符，只能包含字母、数字、`-`、`.`、`_`、`~`；`code_challenge` 必须是 `BASE64URL(SHA256(code_verifier))`，不接受 `plain` 或缺失 PKCE 的授权请求。token 请求必须提交与授权请求完全一致的 `redirect_uri`，省略、替换或附加空白都会返回 `invalid_grant`。

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

`/oauth2/introspect` 和 `/oauth2/revoke` 也必须使用该 token 所属应用的 client credential。其他已批准应用即使提供有效 client credential，也只能得到 `active=false`，且 revoke 会被视为 no-op，避免跨应用 token 探测或误撤销。所属 client 成功撤销 access token 或 refresh token 时会写入 `iam.token.revoked` 审计事件，审计只记录 JTI 或 refresh token family hash，不记录原始 token。标准 `token_type_hint` 只是查找提示，不改变所属 client 校验或 revoke no-op 语义；未知 token / 跨 client token 仍返回 `200`，但撤销黑名单或 refresh token family 查找 / 删除失败时会返回 `503 {"error":"server_error"}`，应用应将其视为可重试的服务端失败而不是已撤销成功。
refresh token introspection 也只允许所属 client 看到 `active=true`；已 rotation 消费、被 revoke、用户撤权或 app / scope 失效后都会返回 `active=false`。

`/oauth2/token`、`/oidc/userinfo`、`/oauth2/introspect` 和 `/oauth2/revoke` 的成功与错误响应都会携带 `Cache-Control: no-store` 和 `Pragma: no-cache`。迁移后的应用不应依赖代理、浏览器或本地 HTTP 调试缓存保存 token、UserInfo 或 introspection 结果。
`invalid_client` 401 响应会携带 Basic challenge；UserInfo 的 `invalid_token` 401 响应会携带 `WWW-Authenticate: Bearer realm="StuHelper Identity", error="invalid_token"`。客户端应按 challenge 区分 client credential 失败和 Bearer token 失败。

## 验收

迁移完成后必须验证：

- discovery 的 `issuer` 等于 `https://id.stuhelper.com`；同时验证 `/.well-known/openid-configuration` 和 `/.well-known/oauth-authorization-server` 返回同一套 endpoint 基线。
- discovery 的 `authorization_response_iss_parameter_supported` 为 `true`；授权成功回调和可回调的授权错误响应都包含 `iss=https://id.stuhelper.com`。
- discovery 的 `response_modes_supported` 包含且只依赖 `query`；授权请求显式携带 `response_mode=query` 时仍通过 query 回调，`fragment` / `form_post` 被拒绝。
- discovery 的 `prompt_values_supported` 同时包含 `none`、`login` 和 `consent`；`prompt=login` 或 `max_age=0` 会触发上游 SSO 重新认证后再继续授权码流程；`prompt=consent` 对已有 consent 的 disclosure scope 仍会重新展示授权页。
- `id_token` 和 `access_token` 的 `iss` 等于 `https://id.stuhelper.com`。
- `sub` 使用 StuHelper Identity subject，格式为 `stuhelper:<internal-user-id>`；应用能把新 subject 正确关联到既有本地账号。
- access token 的 `client_id` 等于导入后的 `client_id`，`aud` 包含该 `client_id`，且 `azp` 与该 `client_id` 一致；ID token 的 `aud` 也必须指向该 client，存在 `azp` 时必须落在 `aud` 内。
- token、UserInfo、introspection 和 revoke 响应都带有 `Cache-Control: no-store` 与 `Pragma: no-cache`。
- token / introspection / revoke 的 `invalid_client` 401 响应带有 Basic `WWW-Authenticate` challenge；UserInfo 的 `invalid_token` 401 响应带有 Bearer `WWW-Authenticate` challenge。
- token response、access token claim 和 introspection 的 `scope` 与本次被授予的 OAuth scope 一致，不会把 `profile`、`email`、`phone` 改写成内部业务 scope。
- 授权 scope 不含 `openid` 时，token response 不包含 `id_token`，UserInfo 返回 `invalid_token`；资源类授权应通过 access token introspection 或资源访问 API 校验。
- 未登记的 `redirect_uri` 被拒绝。
- 缺失 PKCE、`code_challenge_method=plain` 或错误 `code_verifier` 被拒绝。
- 未审批或未授权的 scope 不会出现在 `id_token` 或 UserInfo。
- `id_token` 和 UserInfo 只返回应用申请且用户同意的字段，并使用同一套字段名，例如 `identityVerified`、`identityType`、`studentVerified`、`phoneVerified` 和 `school`；需要标准 OIDC 手机号字段时使用 `phone_number` / `phone_number_verified` 别名。
- `resource.read` / `resource.write` 是应用级资源能力 scope，不进入用户 consent，也不会产生 UserInfo 字段；具体资源访问必须走资源访问 API，并由 OpenFGA tuple 判断。
- `client_credentials` token 只允许携带 `resource.read` / `resource.write`，用于服务端应用证明自己当前拥有资源类能力；资源访问 API 会按 token scope、app 已审批 scope 和 OpenFGA tuple 共同判断。用户资料 disclosure 仍必须走授权码 flow 和用户 consent。
- 请求 `phone.read` 时，如果本地加密手机号投影缺失或不可解密，应拒绝披露而不是返回空手机号伪装成功。
- introspection 对有效且当前仍被授权的 token 返回 `active=true`；校验时会把 token 中的 OAuth scope 映射为业务 scope 后检查 app approval、用户 consent 和 token 发行时的 consent 指纹。revoke、用户撤销对应 disclosure scope / 整应用授权、`prompt=consent` 刷新授权、管理员暂停或吊销应用后返回 `active=false`；重新授权只会让新授权码发行的新 token active，不会复活旧 token。若应用用已被 rotation 消费的旧 refresh token 调用 revoke，Identity Server 会撤销该旧 token 所在 family 的当前 refresh token。
- 使用其他应用的 client credential introspect 当前应用 token 时返回 `active=false`，使用其他应用 revoke 当前应用 token 后，原应用 introspection 仍为 `active=true`。
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
