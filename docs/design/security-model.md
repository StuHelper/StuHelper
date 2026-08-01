---
type: design
audience: backend-dev, ops
status: current
authoritative-source: this file
last-verified: 2026-07-31
---

# 安全实践

## 认证与会话

### OIDC 登录

- 后端生成授权地址，包含服务端生成的 `state`
- `state` 存 Redis（一次性验证后销毁）+ HttpOnly cookie 双重校验
- 回调时 `GetDel` 原子验证并销毁 state，防伪造和重放

### Token

| 凭据 / 状态 | TTL 口径 | 存储 | 用途 |
|-------------|----------|------|------|
| Casdoor ID Token（access credential） | 以已验证 provider `exp` 为真值；托管 application 默认 1 小时 | HttpOnly Cookie / native 安全存储 | API 访问 |
| Access Cookie / `expiresIn` 策略 | `TOKEN_ACCESS_TTL` 默认 300 s，不改变 provider `exp` | HttpOnly Cookie / 响应字段 | 缩短客户端持有与刷新窗口 |
| Provider Access / Refresh Token / 本地 session | access 默认 1 小时、refresh 默认 24 小时；本地 session lease 默认 7 天 | refresh 位于 Path `/api/v1/auth` HttpOnly Cookie；Redis 分别保存两份 provider token 密文 | 续期与 provider token-family 撤销 |
| CSRF Token | 随本地 refresh/session lease | 普通 Cookie | 前端读取后放 `X-CSRF-Token` |

Casdoor access / refresh 由 provider `tokenType` 区分；遗留 StuHelper 自签 token 才使用
`typ`，且已不进入公开登录链路。

### 吊销

- Redis 黑名单用于紧急吊销
- `logout` 撤销当前设备
- `logout-all` 撤销全部已跟踪 token
- OIDC provider access/refresh token 由后端分别加密代持；本地 session/blacklist 撤销与
  provider token-family 撤销是两层边界。
- 固定 Casdoor 镜像没有 RFC 7009 `revocation_endpoint`。只有 discovery 中与 issuer
  同源且路径精确为 `/api/logout` 的 endpoint 才进入 Casdoor adapter：服务端发送
  `id_token_hint=<provider access token>`，并同时校验 HTTP 状态与 JSON
  `status=ok`。旧的 `token=<refresh>&token_type_hint=refresh_token` 对该 endpoint
  是无浏览器 session 时的 HTTP 200 no-op，禁止恢复。
- Casdoor logout 把 access/refresh 所属同一 token row 的 `expires_in` 置零，因此成功后
  access introspection 与 refresh grant 必须同时失效。正常 refresh 已删除旧 row，不重复
  logout；尚未提交的新 family 才补偿撤销。
- 旧 session 缺 provider access 密文时，固定 Casdoor 当前返回同值的 `id_token` /
  `access_token`，故当前设备可使用已经匹配 session hash 的原始 access；
  logout-all 用加密 refresh 先 rotation、再立即撤销替代 family。只有 Casdoor
  `invalid_grant` 可判为 family 已失效，配置错误、其他 4xx、5xx、网络失败或业务
  `status=error` 均 fail-closed；两种 provider 凭据都缺失也不能静默成功。logout-all
  在外部调用前先完成本地 session/blacklist 撤销，避免 provider 延迟耗尽本地清理预算。
- refresh reuse detection 不把“存在历史 attribution”单独当成攻击证据。只有 referenced
  session 仍存在，且其当前 refresh hash 非空、与提交 token hash 不同，才撤销整个用户的
  session family 并记录 reuse metric/audit；session 已删除或 hash 相同只表示 token 已被
  logout 撤销，返回 401 而不影响其他设备。
- 浏览器 Cookie 中的 Casdoor JWT access credential 会先查 blacklist 和 tracked session
  hash，再做本地 JWKS / audience / `exp` 验证；Bearer 继续走 provider introspection，
  不新增每请求 session 绑定。
- 登录保存已验证 `exp`，refresh 原子更新 access hash + expiry。单设备和全设备登出按
  每个 token 的真实剩余寿命写 blacklist，不再使用 5 分钟
  `TOKEN_ACCESS_TTL` 代替 provider 自然寿命，也不会把超过 30 天硬上限的 TTL 静默截断。
- 新 access token 的剩余寿命必须不大于 session lease。旧 session 缺少 expiry 时，只在
  托管 Casdoor access TTL 不超过 session lease 的发布约束下使用实际 Redis PTTL；无 TTL
  的旧 session 撤销 fail-closed。
- 每请求 blacklist 查询使用保留 request values、忽略客户端取消的 50 ms 有界 context。
  `context.Canceled` 是 neutral outcome：当前调用仍 fail-closed，但不增加共享 breaker 的
  失败数，并必须释放 half-open probe；内部 deadline exceeded 与真实 Redis 错误仍记作
  backend failure。没有生产 Redis 延迟和 breaker 遥测证据时，不调整该 budget、失败阈值
  或 30 秒 open window。

## 手机号与 SMS

公开手机号验证码登录由 Casdoor SMS Provider 承载。StuHelper 不暴露公开 phone OTP 登录端点，也不签发独立 phone-login token。

- Casdoor 使用受控 SMS Provider 回调 StuHelper 内部 SMS 发送入口，再由 StuHelper 调腾讯云 SMS。
- 完整手机号真相源在 Casdoor；StuHelper 本地只保存脱敏投影、验证状态和业务展示需要的更新时间。
- StuHelper 个人中心可以触发补绑 / 更换手机号，但必须写入 Casdoor；本地投影只作为缓存和业务状态展示。
- Open Platform 的 `phone.read` 披露必须实时读取 Casdoor 手机号；Casdoor 不可用、手机号未绑定或格式不可标准化时 fail-closed。

## Open Platform 安全

- 第三方应用直接接入 `sso.stuhelper.com` OIDC；业务字段默认不进入第三方 token，第三方业务数据通过 StuHelper Open API 获取。
- `redirect_uri` 必须精确匹配 `open_platform_apps.redirect_uris`，禁止 wildcard、fragment、regex、通配子域和通配路径。
- 业务 scope 审批和用户 consent 是两道独立检查；缺任一项都拒绝披露。
- `sso.stuhelper.com` 的 token scope 保留第三方被授予的 OAuth scope，例如 `openid profile email`；StuHelper Open API 在校验时再映射为业务 disclosure scope 做审批、consent 和字段披露。不要定义笼统的 `properties` scope，也不要让 Casdoor `User.Properties` 成为第三方业务数据契约。
- Consent challenge 存 Redis，TTL 为 5 分钟，绑定 app、user、scope、redirect URI 和 state。
- 业务 consent 页面由 StuHelper 承载，或在当前 Casdoor 版本能稳定表达业务 custom scope 时由 Casdoor consent 展示 scope display name / description；无论 UI 在哪里，业务授权真源都是 StuHelper 数据库里的 app、scope approval、user consent 和审计记录。
- 应用密钥只在批准或轮换时展示一次，服务端仅保存 hash。
- StuHelper Open API 必须在每次 disclosure 前反映当前授权状态：除 JWT 签名、过期时间、JTI revoke 外，还要重新检查 app 仍为 approved、token scope 映射出的业务 scope 仍被批准、用户 disclosure consent 仍 active，且 token 中记录的 consent 指纹仍等于当前 active consent 指纹。用户撤销授权、重新确认授权、管理员定向撤销授权或管理员暂停 / 吊销 app 后，后续 disclosure 必须 fail-closed；管理员恢复 suspended app 后只恢复 app active 状态，不自动恢复用户已撤销的 consent，也不会让旧 token 在用户重新授权后复活。
- `resource.read` / `resource.write` 只表示应用可申请资源类能力；具体资源授权必须落在 OpenFGA tuple。管理员吊销 app 时会同步删除该 app 的 `resource_item` / `user_profile` tuple，并写入 `open_platform.resource_access.revoked` 审计，避免留下陈旧资源授权。
- 敏感 disclosure、consent 决策、异常拒绝和手机号读取失败必须进入 `open_platform_audit_events` 或等价审计链路。

## 机器人内部接口

Koishi 与主站后端跨机通信时，不复用用户态会话，而是使用独立的服务令牌：

- 接口范围：`/api/v1/bot/*`
- 鉴权方式：`Authorization: Bearer <Koishi service credential>`
- 后端配置项：`BOT_SERVICE_TOKEN` 只作为启动 bootstrap / rotation 输入；服务端会将其 HMAC 写入 `bot_service_credentials`
- 校验方式：服务端按 token hash 查询 DB 凭据，并校验 audience `/api/v1/bot/*` 与路由 scope
- 失败行为：未配置返回 `503`，令牌错误或已吊销返回 `401`，audience/scope 不匹配返回 `403`

这类接口只提供给机器人运行时，不对浏览器、移动端或普通用户开放。

## PII 保护

### 证件号

- `doc_number_enc`：AES-256-GCM 密文
- 信封格式：`version | keyID | nonce | ciphertext+tag`
- 支持多密钥并存与轮转

### 稳定标识

- `person_uid`：`HMAC-SHA256(doc_type + ":" + doc_number)`
- 用于同人匹配和去重，不可逆

### 证件照片

- `doc_photo_front` / `doc_photo_back` / `doc_photo_selfie` 只保存私有对象存储 key，不接受客户端提交的 `data:`、HTTP(S) URL 或其他外部引用。
- 上传只接受内容声明与实际探测结果一致的 JPEG、PNG 或 WebP，单文件上限 5 MiB；对象 key 绑定当前用户和 `front` / `back` / `selfie` 槽位。提交时再次校验 key 所属用户、槽位和对象可读性。
- 非大陆证件和所有不能自动通过的大陆身份证申请都必须同时具备证件正面照与手持/自拍照，背面照可选。大陆身份证只有在事务内锁定并复核“已认证学生账号、同校、绑定学号、姓名和证件记录”仍全部一致时，才允许无照片自动通过。
- 管理列表不读取也不返回照片 key。审核员通过 `GET /api/v1/admin/identities/{userID}` 按需获取短时签名 URL；该入口同时要求全局 `user:identity:read` capability 和 step-up MFA，并为每次成功查看写入 `data.access` 审计事件。直接批准缺失、越权或对象存储不可验证的材料会被拒绝。

## 匿名与隐私

- 评课作者对外只暴露 `userHash`（`HMAC-SHA256(userID, secret)`）
- 内部用真实 `userID` 做所有权归属

## 内容安全

### 文本净化

实体解码、零宽字符移除、危险标签过滤、危险内容拦截。

### 敏感词

- `block`：直接拦截
- `warn`：允许提交但添加标记，后台复核
- `review`：进入 `pending_review`，等待后台人工审核
- 词表有签名保护，检查接口不回传词表
- 管理端创建、更新或删除成功后，当前应用进程立即把内存词表标记为过期；下一次内容检查从
  PostgreSQL 重载。失败的 mutation 不失效现有快照，重载失败则内容检查 fail-closed。
- 当前受支持的生产 Compose 拓扑是单应用实例，因此不引入 Redis version/pub/sub。扩展为多副本前必须增加
  跨实例失效机制；在此之前不能把本地失效语义误写成多副本强一致。

### 导出防护

CSV 加 UTF-8 BOM，公式注入字符（`=`、`+`、`-`、`@`）添加前缀转义。

## 速率限制

| 操作 | 默认 |
|------|------|
| 发布评课 | 5/min |
| 投票 | 30/min |
| 举报 | 10/min |
| 回复 | 10/min |
| 刷新 token | 10/min |
| 实名材料上传 | 每个已登录用户 12 次/24 h |
| Casdoor SMS Provider 回调 | 按 IP + provider key + 手机号维度限流 |
| Open Platform consent | 按 user + app 维度限流 |
| Open Platform disclosure | 按 app + user + endpoint 维度限流 |

## 安全响应头

- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy`
- 生产环境额外：`Strict-Transport-Security`

前端静态页和后端 API 统一禁止被 `iframe` 嵌入：静态页使用 `X-Frame-Options: DENY` + `frame-ancestors 'none'`，后端 API 也保持同样策略。

## 数据库与外部依赖

- SQL 全部参数化
- 动态排序使用白名单
- 学生认证所需 LDAP 连接信息只保存在学校级 `ldap_config`，由管理端配置更新和服务层校验负责收口
- 生产 TLS 连接只接受完整证书链与主机名校验；LDAP、PostgreSQL、Redis、对象存储和外部 Oracle 学籍源均不得使用跳过验证或仅加密不验身份的模式
- PostgreSQL 服务端 CA 私钥/私钥证书目录只挂载到 PostgreSQL；Redis 服务端 CA 私钥、服务端私钥和仅含密码哈希的 ACL 目录只挂载到 Redis。两者启动时分别复制到 UID 70/999 可读的私有 tmpfs，源文件保持 0600。Redis 应用与 exporter 使用独立密码和显式命令白名单，exporter 无应用 key 访问权。
- 应用、迁移、OpenFGA 和 exporter 只挂载 `postgres-client-ca` / `redis-client-ca` 中的公开 `ca.crt`；部署脚本会拒绝客户端 CA 目录中的额外文件、符号链接和私钥。
- Oracle 学籍源只允许 TCPS `verify-full`，默认端口为 `2484`；应用只挂载独立的 `/external-student-source-tls/ca.crt` 公共 CA，不挂载服务端密钥、CA 私钥或数据库数据目录
- Oracle 运行账号必须与源 schema owner 不同，且不得是 `SYS`、`SYSTEM`、`SYSBACKUP`、`SYSDG`、`SYSKM` 或 `SYSRAC`。账号不得继承任何 role 或列级权限；只允许直接授予无 `ADMIN OPTION` 的 `CREATE SESSION`，以及目标表上无 `GRANT OPTION`、无 `HIERARCHY OPTION` 的 `SELECT`。学号和姓名都经过长度、字符集与控制字符校验，冲突重复行和非法源记录按数据完整性故障关闭
- 外部依赖统一通过受控 client 调用，记录固定低基数的延迟/结果指标并启用熔断；外部学籍源不可用时 User 与 Admission API 返回 503，不回退为“未匹配”，避免把基础设施故障误判为学生身份失败

## CI 安全门禁

- Go：`gosec`（版本固定 v2.22.4，零 issue 零 nolint 注释）+ `govulncheck`
- Node：Web、Admin、UniAppX 通过 npm 官方审计端点执行全依赖 `pnpm audit`；Koishi 执行全工作区 `yarn npm audit`；两者均阻断 `MODERATE` 及以上风险，不使用通告忽略项。`brace-expansion` 统一锁定到含 CVE-2026-14257 修复的 `5.0.8`，仓库补丁仅恢复旧版 `minimatch` 的 callable CommonJS/default export，并由 `check:dependency-compat` 验证安全版本、命名导出和实际 brace 匹配行为
- 应用候选镜像：固定 digest 的 `Trivy` 阻断 `HIGH` / `CRITICAL`
- 第三方运行时镜像：`infra/security/runtime-images.json` 管理完整 `tag@sha256` 清单，Trivy 同时检查 `HIGH` / `CRITICAL` / `UNKNOWN`；`CRITICAL` 只接受带证据且最长 30 天复核周期的 `not_affected` VEX，`HIGH` / `UNKNOWN` 只接受逐包逐版本、最长 30 天的显式例外
- 生产部署：实际基础设施镜像引用必须与已扫描策略一致；可变标签即使固定 digest 也必须在 30 天内重新核对上游
- CI SSH 部署使用固定 host key（`DEPLOY_TARGET_SSH_KNOWN_HOSTS` CI 变量），禁止 TOFU
- 部署前自动执行数据库备份（`backup-postgres.sh`），失败阻断发布
- 这些门禁在 GitHub Actions 中汇总到 `CI / Required`，失败会阻塞后续镜像发布

## 日志脱敏

禁止原样记录：token / password / code / secret / state、完整手机号、完整 IP、过长 User-Agent。

- 匹配到 Gin route 的结构化访问日志、panic 日志和请求体超限告警只记录
  `c.FullPath()` 路由模板，例如 `/api/v1/admission/sessions/:token`；不得回退到含动态
  参数值的 `Request.URL.Path`。
- 404/405 等未匹配请求固定记录 `path=unmatched`，不保留 raw path。query string 继续按
  敏感参数名脱敏；不能用 path 参数黑名单或字符串替换器代替路由模板边界。

## 待改进

1. 证件照片仍经后端中转上传（未实现浏览器直传）
2. 证件材料的业务保留期限尚未形成可由仓库自动验证的对象存储 lifecycle 策略；生产 bucket 必须在数据治理确定保留期后配置到期删除，并把策略与删除审计纳入上线证据
3. 生产环境仍需用真实环境证据验收 HTTPS、强密码和值班告警
