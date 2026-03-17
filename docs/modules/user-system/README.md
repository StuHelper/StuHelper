# 用户系统模块

用户系统覆盖实名认证、学生认证、学校配置、系统配置和学籍信息查询。

## 代码范围

| 代码位置                         | 职责                                             |
| -------------------------------- | ------------------------------------------------ |
| `server/internal/modules/user`   | 实名认证、学生认证、学校配置、系统配置、学籍查询 |
| `server/internal/modules/ldap`   | LDAP 登录验证和用户信息查询                      |
| `server/internal/pkg/crypto/pii` | 证件号 AES-256-GCM 加密                          |

## 两级认证体系

### 1. 实名认证（Identity Verification）

用户提交证件类型和证件号码进行实名认证。

- 证件类型：大陆身份证（`MAINLAND_ID`）、港澳居民来往内地通行证（`HK_MACAU`）、台湾居民来往大陆通行证（`TW`）、护照（`PASSPORT`）
- 证件号码使用 AES-256-GCM 加密存储为 `doc_number_enc`
- 通过 HMAC-SHA256(doc_type:doc_number) 派生 `person_uid`，用于跨记录匹配同一自然人
- 非大陆身份证需上传证件照片（正面、背面、手持）
- 大陆身份证提交时会尝试学籍数据库自动匹配（`tryAcademicDBMatch`）：如果证件号和姓名命中学籍记录，`verify_method` 写为 `academic_db_match`，认证状态自动设为 `verified`
- 未自动匹配的记录需要管理员审核后批准或拒绝

### 2. 学生认证（Student Verification）

学生认证依赖实名认证通过。当前学校配置只支持两种认证方式。

| 方式         | 说明                                                                |
| ------------ | ------------------------------------------------------------------- |
| 统一身份认证 | 对外口径。当前实现名为 `ldap`，用学号和密码对学校目录服务做凭据校验 |
| 表单人工认证 | 当前实现名为 `manual`，用户填写学校配置的动态表单，管理员审核       |

产品、后台和面向用户的文案统一使用统一身份认证这个名称。`ldap` 只在代码、接口字段和开发文档里作为实现名出现。

## 学校配置

`school_configs` 表的关键字段：

| 字段                  | 说明                                        |
| --------------------- | ------------------------------------------- |
| `verification_method` | 验证方式，`ldap` 或 `manual`                |
| `ldap_config`         | LDAP 连接配置（加密存储，BYTEA）            |
| `manual_form_fields`  | 人工审核模式的动态表单字段定义（JSONB）     |
| `consent_text`        | 向用户展示的同意协议文本                    |
| `academic_db_table`   | 学籍数据表名（如 `academic.buaa_students`） |
| `enabled`             | 是否启用该学校的认证                        |

动态表单字段（`ManualFieldDescriptor`）支持 `text`、`select`、`textarea` 等类型，每个字段可配置 `key`、`label`、`type`、`required`、`options`、`placeholder`。

## 当前审批行为

当前认证方式和审批结果是绑定的：

- `ldap` 验证成功后，学生认证直接写成 `verified`
- `manual` 提交后，学生认证写成 `pending`
- 管理员可以在管理端把学生认证改成 `verified` 或 `rejected`

当前学生认证记录里还没有独立的审批策略字段，也没有完整的复审元数据。

## 后续目标

后续要把认证方式和审批策略拆开配置。

建议补充学校级审批策略字段，例如：

- 统一身份认证提交后自动通过
- 统一身份认证提交后进入人工复核
- 表单人工认证提交后自动通过
- 表单人工认证提交后进入人工复核

无论是否自动通过，都要保留一条完整认证记录。管理员需要能看到自动通过记录，支持复审、撤销和打回，并保留审核人、审核时间、拒绝原因和复审轨迹。

## 系统配置

`system_configs` 表存储全局配置项，每项包含 `key`、`value`、`description`，通过管理后台维护。

## 业务规则

- 实名认证通过是学生认证的前置条件
- 证件号码以密文存储，`person_uid` 用于稳定匹配
- 学校配置更新使用合并写入语义（可选字段，未提供的字段保留原值）
- 管理端审核端点需要对应的能力字符串
- 当前 `ldap` 验证成功后自动将 `verification_status` 设为 `verified`
- 当前 `manual` 提交后 `verification_status` 设为 `pending`，等待管理员批准
- 拒绝操作必须提供拒绝原因

## 用户端 API 端点

| 端点                                 | 方法 | 说明                           |
| ------------------------------------ | ---- | ------------------------------ |
| `/api/v1/user/identity`              | GET  | 查看实名认证状态               |
| `/api/v1/user/identity`              | POST | 提交实名认证                   |
| `/api/v1/user/profile`               | GET  | 查看学生认证档案               |
| `/api/v1/user/profile/verify`        | POST | 提交学生认证                   |
| `/api/v1/user/profile/bind-phone`    | POST | 绑定手机号                     |
| `/api/v1/user/profile/academic-info` | GET  | 查看学籍信息（需学生认证通过） |
| `/api/v1/user/schools`               | GET  | 列出学校（无需认证）           |

## 管理端 API 端点

| 端点                                          | 方法 | 所需能力               | 说明             |
| --------------------------------------------- | ---- | ---------------------- | ---------------- |
| `/api/v1/admin/identities`                    | GET  | `user:identity:read`   | 列出实名认证请求 |
| `/api/v1/admin/identities/:userID`            | PUT  | `user:identity:review` | 审核实名认证     |
| `/api/v1/admin/student-verifications`         | GET  | `user:student:read`    | 列出学生认证请求 |
| `/api/v1/admin/student-verifications/:userID` | PUT  | `user:student:review`  | 审核学生认证     |
| `/api/v1/admin/school-configs`                | GET  | `user:school:read`     | 列出学校配置     |
| `/api/v1/admin/school-configs/:schoolID`      | PUT  | `user:school:update`   | 更新学校配置     |
| `/api/v1/admin/system-configs`                | GET  | `user:system:read`     | 列出系统配置     |
| `/api/v1/admin/system-configs/:key`           | PUT  | `user:system:update`   | 更新系统配置     |

## 数据库表

| 表                | 用途                                             |
| ----------------- | ------------------------------------------------ |
| `users`           | 本地用户记录（从 Casdoor 同步）                  |
| `user_identities` | 实名认证数据（加密证件号、person_uid、审核状态） |
| `user_profiles`   | 学生认证数据（学校、学号、认证状态、手机号）     |
| `school_configs`  | 学校认证配置                                     |
| `system_configs`  | 全局系统配置                                     |

## 相关文档

- [LDAP 验证](../auth/02-ldap.md)
- [会话与安全](../auth/04-security.md)
- [RBAC 权限控制](../rbac/README.md)
