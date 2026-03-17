# 账号同步

账号模型围绕 Casdoor 外部用户和本地 `users` 表展开。Casdoor 提供外部身份；应用在本地存储业务所需的用户记录。

## 数据同步

登录回调（`HandleCallback`）和 `/auth/me`（`GetCurrentUser`）都通过 `buildUserInfo` 调用 `UpsertUser`，将 Casdoor 用户信息同步到本地 `users` 表：

| 同步字段 | 来源 |
| --- | --- |
| `external_id` | Casdoor `id`（JWT claims 中的 `Id`） |
| `username` | Casdoor `name` |
| `email` | Casdoor `email`（空值写入 NULL） |
| `avatar_url` | Casdoor `avatar`（空值保留原值） |

同步使用 `INSERT ... ON CONFLICT (external_id) DO UPDATE` 语义：首次登录创建记录，后续登录更新 `username`、`email`、`avatar_url` 和 `updated_at`。

## 数据库表关系

| 表 | 用途 |
| --- | --- |
| `users` | 本地用户基础记录（id, external_id, username, email, avatar_url） |
| `user_identities` | 实名认证数据（doc_type, doc_number_enc, person_uid, real_name, verified） |
| `user_profiles` | 学生认证数据（school_id, student_ids, active_student_id, verification_status, phone） |
| `user_roles` / `user_permissions` / `user_group_members` | 应用授权关系 |

## 用户标识符

| 标识符 | 说明 |
| --- | --- |
| `users.external_id` | 与 Casdoor 对齐的稳定外部 ID，全局唯一 |
| `users.username` | 登录后的用户名 |
| `user_identities.person_uid` | 通过 HMAC-SHA256(doc_type:doc_number) 派生的稳定匹配标识 |
| `user_profiles.active_student_id` | 当前激活的学号 |

## 账号流程

```text
Casdoor 用户
    |
    v
认证回调 (HandleCallback)
    |
    v
users 表 upsert (external_id 唯一约束)
    |
    v
/auth/me (GetCurrentUser)
    |
    v
前端状态初始化 (用户信息 + 能力集)
```

每次调用 `buildUserInfo` 都会：
1. 通过 `GetCachedUserByID` 获取 Casdoor 最新用户信息（L1 本地缓存 + L2 Redis 二级缓存）
2. 调用 `UpsertUser` 同步本地记录
3. 调用 `GetUserCapabilities` 计算能力集
4. 组装 UserInfo 响应
