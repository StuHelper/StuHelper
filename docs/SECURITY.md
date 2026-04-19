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
- 浏览器 Cookie 中的 Zitadel OIDC access token 走 **本地 JWKS 验证**，不会为每个请求额外查询 session store
- 即时吊销模型明确收口为：**blacklist + 5 分钟 access TTL + refresh 轮换**；`refresh` 仍会触达 session store，浏览器写请求不引入每请求 Redis RTT

## 手机号登录

补充登录方式，不授予管理权限。

- 仅限中国大陆手机号：`^1[3-9]\d{9}$`
- 有冷却和频率限制
- 登录成功只签发 `user` 角色

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
| 手机验证码 | 5/min |
| 手机验证码（单号码） | 5/hour |

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
2. 通知读路径仍跨 `review` / `notification` 模块，统一尚未完成
3. 生产环境仍需真实启用 HTTPS、强密码和值班告警
