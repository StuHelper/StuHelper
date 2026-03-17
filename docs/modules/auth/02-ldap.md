# LDAP 验证客户端

LDAP 客户端位于 `server/internal/modules/ldap/client.go`，在学生认证流程中提供基于 LDAP 的凭据验证和用户信息查询。

对外产品口径统一使用统一身份认证。本文面向开发实现，继续使用 `LDAP` 这个技术名。

## 客户端方法

| 方法                        | 用途                                    |
| --------------------------- | --------------------------------------- |
| `NewClient(cfg Config)`     | 校验必填配置项，创建 LDAP 客户端实例    |
| `Login(ctx, uid, password)` | 使用 uid + password 进行 LDAP bind 认证 |
| `QueryUserByUID(ctx, uid)`  | 使用系统账号查询 LDAP 用户信息          |

## 集成点

学生认证服务（`user.Service.VerifyStudent`）根据学校配置的 `verificationMethod` 选择验证方式。当 `verificationMethod` 为 `ldap` 时执行 LDAP 验证。

## 验证流程

1. 校验请求参数：`schoolID`、`studentID`、`password`、`consent` 状态
2. 调用 `ldap.Client.Login` 进行 LDAP bind 认证
3. 认证通过后调用 `ldap.Client.QueryUserByUID` 获取用户详细信息
4. 更新 `user_profiles` 表：写入 `phone`、`student_ids`、`active_student_id`，将 `verification_status` 设为 `verified`

`Login` 方法内部流程：

- 校验 uid 格式（正则 `^[a-zA-Z0-9._-]+$` 防止 LDAP 注入）
- 将 uid 转为小写
- 建立 LDAP 连接（根据配置决定是否启用 StartTLS）
- 构造用户 DN（`uid=xxx,BaseDN`）进行 bind
- bind 成功即认证通过

`QueryUserByUID` 方法内部流程：

- 使用系统账号 bind
- 在 `BaseDN` 下搜索 `(uid=xxx)`
- 返回 `uid`、`cn`、`sn`、`employeeNumber`、`departmentNumber`、`mail`、`mobile`、`employeeType` 字段

## 环境变量配置

| 变量                        | 说明                                                                |
| --------------------------- | ------------------------------------------------------------------- |
| `LDAP_URL`                  | LDAP 服务器地址（如 `ldap://host:389` 或 `ldaps://host:636`），必填 |
| `LDAP_BASE_DN`              | 搜索基础 DN，必填                                                   |
| `LDAP_SYSTEM_BIND_DN`       | 系统绑定账号完整 DN，必填                                           |
| `LDAP_SYSTEM_BIND_PASSWORD` | 系统绑定账号密码，必填                                              |
| `LDAP_USE_TLS`              | 启用 StartTLS 升级（仅对 `ldap://` 连接生效）                       |
| `LDAP_INSECURE_SKIP_VERIFY` | 跳过 TLS 证书验证（开发/测试环境）                                  |

## 数据流

```text
school_configs (verification_method: "ldap")
    |
    v
前端: 根据学校配置显示 LDAP 登录表单
    |
    v
POST /api/v1/user/profile/verify (schoolID, studentID, password)
    |
    v
user.Service.VerifyStudent
    |
    +---> ldap.Client.Login (验证凭据)
    |
    +---> ldap.Client.QueryUserByUID (获取用户信息)
    |
    v
更新 user_profiles (student_ids, active_student_id, verification_status, phone)
```
