# 安全存储与会话管理

> 状态：已部分实现

## 当前敏感数据存储策略

| 数据类型 | 敏感级别 | 当前存储方式 | 说明 |
| -------- | -------- | ------------ | ---- |
| 用户密码 | 极高 | **禁止存储**，仅内存中转 | Casdoor 和 LDAP 认证后即丢弃 |
| 证件号 | 极高 | AES-256-GCM 加密 | 写入 `user_identities.doc_number_enc` |
| 证件号派生标识 | 高 | HMAC-SHA256 | 写入 `user_identities.person_uid`，用于稳定去重和学籍匹配 |
| 手机号 | 高 | 明文存储 | 当前在 `user_profiles.phone` 中保存，后续如需提升等级应独立改造 |
| 学号 | 中 | 明文存储 | 当前在 `user_profiles.student_ids` / `active_student_id` 中保存 |
| 昵称 | 低 | 明文存储 | 普通展示字段 |

当前已经落地的重点是实名认证证件号保护：

- `user.Service` 不再持有裸 AES key。
- 证件号通过共享 `internal/pkg/crypto/pii` 组件加密后再写库。
- 普通用户状态查询、管理员列表查询、审核状态更新默认不再读取 `doc_number_enc`。
- 业务层不暴露公开的解密接口，避免把可逆 PII 的访问面扩大。

## 当前证件号加密方案

实名认证证件号使用 AES-256-GCM，密文采用版本化信封格式：

```text
version(1 byte) | keyID(1 byte) | nonce(12 bytes) | ciphertext+tag
```

这个格式的目的不是“为了复杂而复杂”，而是给后续密钥轮换留出空间：

- `version`：当前为 `0x01`，后续如果切换算法或信封格式，可以显式升级。
- `keyID`：标记当前用的是哪把密钥，支持未来 key rotation。
- `nonce`：AES-GCM 随机 nonce，同一明文每次加密结果都不同。
- `ciphertext+tag`：实际密文和认证标签。

当前实现只对“写入证件号”开放加密能力，不对业务模块开放解密能力。

## 配置 PII 加密密钥

后端启动时必须提供以下环境变量：

```bash
DOC_AES_ACTIVE_KEY_ID=1
DOC_AES_KEYS=1:<64位hex密钥>
```

建议直接生成 32 字节 AES 密钥：

```bash
openssl rand -hex 32
```

还需要同时配置：

```bash
HMAC_SECRET=<至少 32 字符>
```

原因：

- `DOC_AES_KEYS` 用于可逆加密证件号。
- `HMAC_SECRET` 用于生成稳定的 `person_uid`，避免把原始证件号拿去做业务匹配。

只要 `DOC_AES_ACTIVE_KEY_ID` 或 `DOC_AES_KEYS` 缺失、格式错误、长度不对，服务就会直接启动失败，不区分开发环境和生产环境。

## 为什么不直接在业务层解密

实名认证记录里已经有 `doc_number_enc`，但当前设计刻意不在 `user.Service` 暴露 `Decrypt`。

这样做是为了减少误用风险：

- 普通查询接口不应该看到证件号明文。
- 管理端列表和审核流也不应该默认接触证件号明文。
- 如果未来真的需要合规审计读取，应单独设计受审计的内部链路，而不是复用业务接口。

## 会话管理

### Token 设计

采用 JWT + Redis 双重验证：

```
┌─────────────────────────────────────────────────────┐
│                   JWT Token 结构                     │
├─────────────────────────────────────────────────────┤
│ Header:  { "alg": "RS256", "typ": "JWT" }           │
├─────────────────────────────────────────────────────┤
│ Payload: {                                          │
│   "sub": "user_id",                                 │
│   "school": "buaa",                                 │
│   "exp": 1704067200,                                │
│   "jti": "unique_token_id"                          │
│ }                                                   │
├─────────────────────────────────────────────────────┤
│ Signature: RSA-SHA256(...)                          │
│ 仅允许 RS256/RS384/RS512 算法，HMAC 算法被显式拒绝 │
└─────────────────────────────────────────────────────┘
```

### Token 生命周期

| Token 类型    | 有效期                                 | 存储位置        |
| ------------- | -------------------------------------- | --------------- |
| Access Token  | 15 分钟（默认 900s，可配置 60-86400s） | HttpOnly Cookie |
| Refresh Token | 7 天                                   | HttpOnly Cookie |
| 会话状态      | 7 天                                   | Redis           |

### Token 刷新流程

1. Access Token 过期
2. 客户端携带 Refresh Token 请求刷新
3. 服务端验证 Refresh Token 有效性
4. 检查 Redis 中会话状态
5. 签发新的 Access Token
6. 轮换 Refresh Token（旧 token 加入黑名单，签发新的 token 对，清理用户 token 集合中的旧 token）

## 监控指标

| 指标                      | 说明              | 告警阈值  |
| ------------------------- | ----------------- | --------- |
| `auth_sso_success_rate`   | SSO 认证成功率    | < 95%     |
| `auth_sso_latency_p99`    | SSO 认证 P99 延迟 | > 5s      |
| `auth_token_refresh_rate` | Token 刷新频率    | 异常波动  |
| `auth_failed_attempts`    | 认证失败次数      | > 100/min |
