# StuHelper IAM 架构设计

> 版本：1.0
> 日期：2026-03-21
> 状态：大部分已完成。本文保留迁移设计背景、取舍和历史术语，不等同于当前运行手册。

---

## 一、架构全景

```
┌─────────────────────────────────────────────────────────────┐
│                     StuHelper 生态                           │
│                                                              │
│  ┌─────────────────┐  ┌──────────────┐  ┌───────────────┐  │
│  │    Zitadel       │  │   OpenFGA    │  │   航小伴       │  │
│  │   (身份层)       │  │  (授权层)    │  │  (应用层)      │  │
│  │                  │  │              │  │                │  │
│  │ • 认证           │  │ • 关系存储   │  │ • 业务逻辑     │  │
│  │ • 组织管理       │  │ • 关系查询   │  │ • 认证流程     │  │
│  │ • 角色分配       │  │ • 权限继承   │  │ • 内容管理     │  │
│  │ • Token 签发     │  │              │  │ • 数据存储     │  │
│  │ • 用户元数据     │  │              │  │                │  │
│  └────────┬─────────┘  └──────┬───────┘  └───────┬────────┘  │
│           │                    │                   │           │
│           │      JWT Token     │   Check API       │           │
│           └────────────────────┼───────────────────┘           │
│                                │                               │
└────────────────────────────────┼───────────────────────────────┘
                                 │
                    ┌────────────┴────────────┐
                    │       PostgreSQL         │
                    │  • Zitadel 数据          │
                    │  • OpenFGA 关系          │
                    │  • 航小伴 业务数据       │
                    └─────────────────────────┘
```

---

## 二、Zitadel 组织结构

```
Zitadel Instance: sso.stuhelper.com
│
├── Organization: stuhelper-platform (生态管理组织)
│   └── 用户: 平台超级管理员 (A)
│
├── Organization: buaa (北京航空航天大学)
│   └── 用户: 北航学生、北航管理员 (B)、北航志愿者
│
├── Organization: tsinghua (清华大学)  ← 未来
│   └── 用户: 清华学生、清华管理员
│
└── Project: hangxiaoban (航小伴)
    ├── Application: web (OIDC, 授权码+PKCE)
    │   └── redirect: https://hangxiaoban.com/callback
    ├── Application: admin (OIDC, 授权码+PKCE)
    │   └── redirect: https://admin.hangxiaoban.com/callback
    ├── Application: mobile (OIDC Native, PKCE)
    │   └── redirect: com.stuhelper.hangxiaoban://callback
    └── Roles:
        ├── super_admin      (生态超级管理员)
        ├── school_admin     (学校管理员)
        ├── moderator        (内容审核志愿者)
        ├── verified_student (已认证学生)
        └── user             (普通用户)
```

### 角色分配示例

| 用户 | 组织 | 角色 | 含义 |
|------|------|------|------|
| A | stuhelper-platform | super_admin | 生态超级管理员，拥有所有权限 |
| B | buaa | school_admin | 北航管理员，管理北航所有内容 |
| C | buaa | verified_student | 北航已认证学生 |
| D | buaa | moderator | 北航志愿者，可删帖但不可看实名 |
| E | buaa | user | 北航普通用户，已注册但未认证 |

### Token Claim 结构

```json
{
  "sub": "user-B-id",
  "name": "李四",
  "email": "lisi@buaa.edu.cn",
  "iss": "https://sso.stuhelper.com",
  "urn:zitadel:iam:org:project:roles": {
    "school_admin": {
      "buaa-org-id": "buaa.stuhelper.com"
    }
  }
}
```

> 认证状态（identity_verified、student_verified）**不放 Token claim**。
> 真相源在应用 DB，需要时查应用接口。Token 只携带粗粒度角色用于中间件粗筛。

---

## 三、角色→能力静态映射

### 映射定义（Go 常量，不需要 DB）

```go
package capability

// RoleCapabilities 角色→能力静态映射
// 这是权限模型的核心配置，修改需要 code review
var RoleCapabilities = map[string][]string{
    "super_admin": {
        // 超级管理员拥有所有能力
        AdminDashboardView, AdminReviewsManage, AdminReportsManage,
        AdminTeachersManage, AdminSensitiveWordsManage, AdminLogsView,
        UserIdentityRead, UserIdentityReview,
        UserStudentRead, UserStudentReview,
        UserSchoolRead, UserSchoolUpdate,
        UserSystemRead, UserSystemUpdate,
        RBACRoleRead, RBACRoleCreate, RBACRoleUpdate, RBACRoleDelete,
        RBACPermissionRead, RBACUserRead, RBACUserUpdate,
        RBACGroupRead, RBACGroupCreate, RBACGroupUpdate, RBACGroupDelete,
    },
    "school_admin": {
        AdminDashboardView, AdminReviewsManage, AdminReportsManage,
        AdminTeachersManage, AdminSensitiveWordsManage, AdminLogsView,
        UserIdentityRead, UserIdentityReview,
        UserStudentRead, UserStudentReview,
        UserSchoolRead, UserSystemRead,
        RBACRoleRead, RBACPermissionRead, RBACUserRead, RBACGroupRead,
    },
    "moderator": {
        AdminReviewsManage, AdminReportsManage, AdminTeachersManage,
    },
    "verified_student": {
        ReviewListFull, ReviewCreate, ReviewEditOwn, ReviewDeleteOwn,
    },
    "user": {
        ReviewListBrief,
    },
}
```

### 中间件工作流程

```
HTTP 请求进入
  ↓
Auth 中间件: 从 Cookie/Bearer 提取 JWT → 标准 OIDC 验证
  ↓
Role 中间件: 从 Token claim 读角色 + 组织
  → 内存 map 展开: roles → capabilities
  → 注入 Gin context: capabilities, orgID
  ↓
路由中间件: RequireCapability("admin:reviews:manage")
  → 从 context 读 capabilities → 检查是否包含
  ↓
资源级中间件 (需要时): 调 OpenFGA
  → Check("user:B", "can_delete", "review:100")
  ↓
Handler 处理业务逻辑
```

---

## 四、OpenFGA 授权模型

### 关系模型定义

```
model
  schema 1.1

# StuHelper 生态
type ecosystem
  relations
    define super_admin: [user]

# 学校
type school
  relations
    define parent: [ecosystem]
    define admin: [user]
    define reviewer: [user]
    define volunteer: [user]
    # 有效管理员 = 直接管理员 + 生态超管
    define effective_admin: admin or super_admin from parent

# 课程
type course
  relations
    define school: [school]
    define owner: [user]
    define teaching_assistant: [user]
    # 能编辑课程信息
    define can_edit: owner or teaching_assistant or effective_admin from school
    # 能查看课程
    define can_view: owner or teaching_assistant or effective_admin from school

# 评课
#
# review 上增加 school 中间 relation，避免链式 TTU。
# 创建评课时同时写入：review:100 | school | school:buaa
type review
  relations
    define course: [course]
    define school: [school]
    define author: [user]
    # 能编辑（仅作者）
    define can_edit: author
    # 能删除（作者 + 志愿者 + 学校有效管理员）
    define can_delete: author or volunteer from school or effective_admin from school
    # 能隐藏（志愿者 + 学校有效管理员）
    define can_hide: volunteer from school or effective_admin from school
    # 能看作者实名信息（仅学校有效管理员，志愿者不行）
    define can_view_author_identity: effective_admin from school

# 举报
#
# 同理：report 上增加 school 中间 relation。
# 创建举报时同时写入：report:200 | school | school:buaa
type report
  relations
    define review: [review]
    define school: [school]
    define reporter: [user]
    # 能处理举报（志愿者 + 学校有效管理员）
    define can_process: volunteer from school or effective_admin from school

# 用户档案（实名信息访问控制）
type user_profile
  relations
    define owner: [user]
    define school: [school]
    # 能看自己的档案
    define can_view_own: owner
    # 能看实名信息（学校有效管理员）
    define can_view_identity: effective_admin from school
    # 能审核学生认证（学校有效管理员）
    define can_review_verification: effective_admin from school
```

### 关系写入时机

| 事件 | 写入的关系 |
|------|-----------|
| 平台初始化 | `ecosystem:stuhelper \| super_admin \| user:A` |
| 学校创建 | `school:buaa \| parent \| ecosystem:stuhelper` |
| 管理员任命 | `school:buaa \| admin \| user:B` |
| 志愿者任命 | `school:buaa \| volunteer \| user:D` |
| 课程创建 | `course:42 \| school \| school:buaa` + `course:42 \| owner \| user:teacher` |
| 评课发布 | `review:100 \| course \| course:42` + `review:100 \| school \| school:buaa` + `review:100 \| author \| user:C` |
| 举报创建 | `report:200 \| review \| review:100` + `report:200 \| school \| school:buaa` + `report:200 \| reporter \| user:E` |

> **注意**：review 和 report 上的 `school` 是冗余 relation，因为 OpenFGA DSL 只支持单层 `x from y`，
> 不支持 `volunteer from course.school` 这种链式 TTU。创建资源时必须同时写入 school 关系。

### 权限查询示例

```go
// 检查用户能否删除某条评课
allowed, err := fga.Check(ctx, openfga.CheckRequest{
    User:     fmt.Sprintf("user:%s", userID),
    Relation: "can_delete",
    Object:   fmt.Sprintf("review:%s", reviewID),
})

// 检查用户能否查看评课作者的实名信息
allowed, err := fga.Check(ctx, openfga.CheckRequest{
    User:     fmt.Sprintf("user:%s", userID),
    Relation: "can_view_author_identity",
    Object:   fmt.Sprintf("review:%s", reviewID),
})
```

---

## 五、权限判断流程（完整示例）

### 场景：用户请求删除评课 `DELETE /api/v1/reviews/:id`

```
1. Auth 中间件
   Token 验证 → 确认身份是 user:D

2. Role 中间件
   Token claim → roles: ["moderator"], org: "buaa"
   展开能力 → ["admin:reviews:manage", "admin:reports:manage", ...]

3. RequireCapability("admin:reviews:manage")
   检查通过 ✅（moderator 有这个能力）

4. Handler 内调 OpenFGA
   fga.Check("user:D", "can_delete", "review:100")
   → review:100 的 course 是 course:42
   → course:42 的 school 是 school:buaa
   → school:buaa 的 volunteer 包含 user:D
   → can_delete 包含 volunteer
   → ✅ 允许

5. 执行删除
```

### 场景：志愿者 D 想看评课#100 作者的实名信息

```
1-3. 同上（身份确认 + 能力检查通过）

4. Handler 内调 OpenFGA
   fga.Check("user:D", "can_view_author_identity", "review:100")
   → can_view_author_identity 只包含 effective_admin
   → user:D 不是 effective_admin
   → ❌ 拒绝

5. 返回评课数据，但不包含作者实名信息字段
```

---

## 六、认证流程设计

### 设计原则：真相源在应用，Zitadel 只存最小 coarse fact

认证状态（identity_verified、student_verified）的**真相源是应用 DB**，不是 Zitadel。
原因：认证可能被撤销、过期、复审驳回，这些状态变更必须实时生效，
不能受 Token 生命周期影响。

Zitadel 角色（verified_student）是**派生数据**，用于 Token claim 携带粗粒度角色。
但业务授权的最终判断必须查应用 DB 的实时状态。

```
真相源层级：
  应用 DB (user_profiles.verification_status)  ← 实时，权威
    ↓ 同步
  Zitadel 角色 (verified_student)              ← 派生，用于 Token 粗筛
    ↓ 读取
  Token claim (roles: ["verified_student"])     ← 缓存，有 TTL 延迟

业务判断顺序：
  1. 中间件读 Token 角色 → 粗筛（没有 verified_student 角色直接拒绝）
  2. 需要精确判断时 → 查应用 DB 实时状态（认证被撤销时立即生效）
```

### 实名认证

```
用户在航小伴发起实名认证
  │
  ├─ 前端收集：姓名 + 身份证号
  │
  ├─ 航小伴后端：
  │   ├─ 调腾讯云二要素核验 API
  │   ├─ 通过 → PII 加密存储到 user_identities 表（真相源）
  │   ├─ 更新 user_identities.verified = true（真相源）
  │   └─ 失败 → 返回错误
  │
  └─ 注意：不需要写 Zitadel 元数据。实名状态由应用 DB 管理。
     只有粗粒度角色（verified_student）需要同步到 Zitadel。
```

### 学生认证

```
用户在航小伴发起学生认证
  │
  ├─ 前端：根据学校配置显示 LDAP 表单 或 手动上传表单
  │
  ├─ LDAP 模式：
  │   ├─ 航小伴后端查学校 LDAP
  │   ├─ 匹配成功 → 更新 user_profiles（真相源）
  │   └─ 同步到 Zitadel: AddUserRole("verified_student", buaa_org_id)
  │
  ├─ 手动模式：
  │   ├─ 用户提交表单 → 存 user_profiles 表（真相源）
  │   ├─ 管理员审核通过 → 更新 user_profiles（真相源）
  │   ├─ 同步到 Zitadel: AddUserRole("verified_student", buaa_org_id)
  │   └─ 审核拒绝 → 记录拒绝原因
  │
  ├─ 撤销/过期：
  │   ├─ 管理员撤销认证 → 更新 user_profiles（真相源，立即生效）
  │   ├─ 同步到 Zitadel: RemoveUserRole("verified_student")
  │   └─ 下次 Token 刷新前：中间件粗筛可能仍通过，
  │      但业务判断查 DB 时立即拒绝
  │
  └─ 效果：
      ├─ Token claim 带 roles=["verified_student"]（粗筛用）
      ├─ 精确判断查 user_profiles.verification_status（实时）
      └─ OpenFGA 无需额外操作（角色级权限已通过 Token/DB 处理）
```

---

## 七、数据存储分布

| 数据 | 存储位置 | 说明 |
|------|----------|------|
| 用户身份（登录凭证） | Zitadel | 唯一认证源 |
| 组织成员关系 | Zitadel | 用户属于哪个学校 |
| 角色分配 | Zitadel | 用户在哪个组织有什么角色（派生数据） |
| **本地 shadow user** | **航小伴 PostgreSQL `users` 表** | **保留。** Zitadel subject → `users.external_id` → `users.id` (BIGSERIAL)，所有业务表的外键仍挂在 `users.id` 上 |
| 实名认证状态+材料 | 航小伴 PostgreSQL | `user_identities` 表（真相源），PII 加密 |
| 学生认证状态+材料 | 航小伴 PostgreSQL | `user_profiles` 表（真相源） |
| 学校配置 | 航小伴 PostgreSQL | `school_configs` 表 |
| 资源关系（谁拥有什么） | OpenFGA (PostgreSQL) | 关系 tuples |
| 评课/课程/举报数据 | 航小伴 PostgreSQL | 业务表 |
| 角色→能力映射 | Go 常量 | 不需要数据库 |

### Shadow User 策略（修正）

**保留 `users` 表和 `users.id` (BIGSERIAL) 作为业务表的外键锚点。**

原因：`user_identities.user_id`、`user_profiles.user_id`、评课 `author_id` 等
全部挂在 `users.id` 上。把外键链改成 Zitadel subject (TEXT) 需要改动所有业务表的
主键类型和索引，收益不大，风险高。

```
Zitadel user subject (TEXT, e.g. "user-abc-123")
  ↓ 登录/回调时 upsert
users.external_id (TEXT, UNIQUE)
  ↓
users.id (BIGSERIAL) ← 所有业务表的外键继续挂在这里
```

`user_sync.go` 的 UpsertUser 逻辑**保留并简化**（只同步 external_id + username + email），
不再从 Casdoor 拉完整用户画像。

### Token 吊销策略（修正）

**不能只靠本地 JWT 校验 + 删除黑名单。** Zitadel JWT access token 被 revoke 后，
本地校验仍会认为有效直到过期。需要分场景处理：

| 客户端 | Token 类型 | 校验方式 | 吊销生效 |
|--------|-----------|----------|----------|
| Web/Admin | Cookie (短 TTL, 如 5min) | 本地 JWT 校验 | Token 过期即失效（最长 5min 延迟） |
| Web/Admin | Refresh Token | Zitadel 管理 | 撤销后无法刷新，access token 过期即踢出 |
| Mobile/API | Bearer Token | **Zitadel Introspection** | 每次请求调 introspection，即时生效 |
| 紧急吊销 | 所有 | Zitadel revoke + 应用侧 Redis 黑名单 | 即时生效 |

**实现**：
- `token/blacklist.go` **不删除**，改为仅用于紧急吊销场景（管理员踢人）
- Cookie 场景：access token TTL 设短（5 min），靠 refresh 续期，不需要 introspection
- Bearer 场景：auth 中间件检测到 Bearer token 时走 Zitadel introspection endpoint
- 普通登出：Zitadel revoke refresh token，access token 自然过期

---

## 八、删除清单

迁移完成后可删除的代码和数据：

### 代码
| 文件/目录 | 行动 |
|-----------|------|
| `server/internal/pkg/sso/` | 删除，替换为标准 OIDC 库 |
| `server/internal/pkg/jwt/validator.go` | 删除，Zitadel 标准 OIDC 验证 |
| `server/internal/pkg/token/blacklist.go` | **保留并简化**，仅用于紧急吊销 |
| `server/internal/pkg/token/service.go` | 简化为 OIDC token 验证 + introspection |
| `server/internal/modules/auth/user_sync.go` | **保留并简化**，只同步 external_id + 基本信息 |
| `server/internal/modules/rbac/repository_users.go` | 删除（角色在 Zitadel） |
| `server/internal/modules/rbac/service_permissions.go` | 删除（能力在内存展开） |
| `server/internal/pkg/sso/cache.go` | 删除（不再需要 Casdoor 用户缓存） |

### 数据库表
| 表 | 行动 |
|----|------|
| `users` | **保留**，作为业务表的外键锚点（shadow user） |
| `roles` | 删除（角色在 Zitadel） |
| `permissions` | 删除（能力在 Go 常量） |
| `role_permissions` | 删除（映射在 Go 常量） |
| `user_roles` | 删除（角色分配在 Zitadel） |
| `user_groups` + `user_group_members` + `user_group_permissions` | 删除 |
| `user_permissions` | 删除（不再需要个人覆盖） |

### 基础设施
| 组件 | 变化 |
|------|------|
| Redis | **保留**：紧急 Token 吊销 + 业务缓存。移除 L1+L2 用户缓存和 OAuth state 用途 |

---

## 九、新增组件

| 组件 | 用途 | 部署方式 |
|------|------|----------|
| Zitadel | SSO / 身份管理 | Docker Compose, ~200MB RAM |
| OpenFGA | 关系型授权 | Docker Compose, ~100MB RAM |
| SMS 转发服务 | Zitadel → 腾讯云短信 | 可内嵌到航小伴，或独立 ~10MB |

---

## 十、实施优先级

```
Phase 1: 架构设计文档 ← 当前阶段
Phase 2: Zitadel 部署 + SSO 迁移（最大工作量，核心依赖）
Phase 3: RBAC 简化（依赖 Phase 2 完成）
Phase 4: OpenFGA 集成（可与 Phase 3 并行）
Phase 5: 认证流程迁移（依赖 Phase 2 + 3）
Phase 6: 清理和文档
```
