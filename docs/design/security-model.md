---
type: design
audience: backend-dev, ops
status: current
authoritative-source: this file
last-verified: 2026-05-18
---

# 安全实践

## 认证与会话

### OIDC 登录

- 后端生成授权地址，包含服务端生成的 `state`
- `state` 存 Redis（一次性验证后销毁）+ HttpOnly cookie 双重校验
- 回调时 `GetDel` 原子验证并销毁 state，防伪造和重放

### Token

| 令牌 | TTL | 存储 | 用途 |
|------|-----|------|------|
| Access Token | 5 分钟（默认 300 s，可通过 `TOKEN_ACCESS_TTL` 配置） | HttpOnly Cookie | API 访问 |
| Refresh Token | 7 天 | Path `/api/v1/auth` HttpOnly Cookie | 续期 |
| CSRF Token | 随 refresh | 普通 Cookie | 前端读取后放 `X-CSRF-Token` |

access / refresh token 已显式区分 `typ`。

### 吊销

- Redis 黑名单用于紧急吊销
- `logout` 撤销当前设备
- `logout-all` 撤销全部已跟踪 token
- OIDC provider refresh token 由后端加密代持；`refresh` 轮换、`logout`、`logout-all` 都会先调用 Casdoor revocation endpoint 吊销 provider refresh token
- provider revoke 失败时返回错误并保留本地 session，避免 Casdoor 端仍可 refresh 但 StuHelper 误报登出成功
- 浏览器 Cookie 中的 Casdoor JWT access token 走 **本地 JWKS 验证**，不会为每个请求额外查询 session store
- 即时吊销模型明确收口为：**blacklist + 5 分钟 access TTL + refresh 轮换**；`refresh` 仍会触达 session store，浏览器写请求不引入每请求 Redis RTT

## 手机号与 SMS

公开手机号验证码登录由 Casdoor SMS Provider 承载。StuHelper 不暴露公开 phone OTP 登录端点，也不签发独立 phone-login token。

- Casdoor 使用受控 SMS Provider 回调 StuHelper 内部 SMS 发送入口，再由 StuHelper 调腾讯云 SMS。
- 完整手机号真相源在 Casdoor；StuHelper 本地只保存脱敏投影、验证状态和业务展示需要的更新时间。
- StuHelper 个人中心可以触发补绑 / 更换手机号，但必须写入 Casdoor；本地投影只作为缓存和业务状态展示。
- Open Platform 的 `phone.read` 披露必须实时读取 Casdoor 手机号；Casdoor 不可用、手机号未绑定或格式不可标准化时 fail-closed。

## Open Platform 安全

- 第三方应用的 Casdoor OIDC scope 固定为 `openid`；业务字段不进入第三方 token。
- `redirect_uri` 必须精确匹配 `open_platform_apps.redirect_uris`，禁止 wildcard、fragment、regex、通配子域和通配路径。
- 业务 scope 审批和用户 consent 是两道独立检查；缺任一项都拒绝披露。
- Consent challenge 存 Redis，TTL 为 5 分钟，绑定 app、user、scope、redirect URI 和 state。
- Consent 页面由 StuHelper 承载，显示应用信息、当前登录身份和每个 scope 实际读取字段。
- 应用密钥只在批准或轮换时展示一次，服务端仅保存 hash。
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

`doc_photo_front` / `doc_photo_back` / `doc_photo_selfie` 只存对象存储 key，审核时后端签发短时 URL。

## 匿名与隐私

- 评课作者对外只暴露 `userHash`（`HMAC-SHA256(userID, secret)`）
- 内部用真实 `userID` 做所有权归属

## 内容安全

### 文本净化

实体解码、零宽字符移除、危险标签过滤、危险内容拦截。

### 敏感词

- `block`：直接拦截
- `warn`：允许提交但添加标记，后台复核
- 词表有签名保护，检查接口不回传词表

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
- TLS 证书验证在所有环境强制启用（已移除 `LDAP_INSECURE_SKIP_VERIFY`、`REDIS_TLS_INSECURE`、`sslmode=require` 选项）
- 外部依赖统一通过受控 client 调用，接入指标和 trace

## CI 安全门禁

- Go：`gosec`（版本固定 v2.22.4，零 issue 零 nolint 注释）+ `govulncheck`
- Node：`pnpm audit`
- 镜像：`Trivy`
- CI SSH 部署使用固定 host key（`DEPLOY_TARGET_SSH_KNOWN_HOSTS` CI 变量），禁止 TOFU
- 部署前自动执行数据库备份（`backup-postgres.sh`），失败阻断发布
- 这些门禁在 GitLab CI 中会阻塞后续构建 / 发布

## 日志脱敏

禁止原样记录：token / password / code / secret / state、完整手机号、完整 IP、过长 User-Agent。

## 待改进

1. 证件照片仍经后端中转上传（未实现浏览器直传）
2. 生产环境仍需真实启用 HTTPS、强密码和值班告警
