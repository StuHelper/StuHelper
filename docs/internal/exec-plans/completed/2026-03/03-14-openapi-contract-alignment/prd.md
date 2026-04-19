---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# OpenAPI 契约全量对齐：user-system + RBAC

## Goal

消除 OpenAPI spec、后端实现、前端类型三者之间的全部偏差，恢复 Spec-First 流程的可信度。

## Fixed Decisions

- OpenAPI 是唯一真相源
- JSON 字段统一大写初始缩写：`ID` / `IDs` / `UID` / `URL`，不做兼容别名
- bind-phone 当前不实现 OTP，删掉无效的验证码字段
- `ReviewStudentVerificationRequest` 和 `ReviewIdentityRequest` 同构：`{ approved, rejectionReason }`
- service 层引入 `ReviewDecision` DTO，不再暴露字符串状态
- 前端 typed client 按职责拆模块

## Scope

### Phase 1: OpenAPI spec 全量修正

**1a. user-system.yaml ID 大小写统一**

| Before | After |
|--------|-------|
| userId | userID |
| personUid | personUID |
| schoolId | schoolID |
| studentId | studentID |
| studentIds | studentIDs |
| activeStudentId | activeStudentID |
| scopeSchoolIds | scopeSchoolIDs |
| permissionId | permissionID |
| permissionIds | permissionIDs |
| roleIds | roleIDs |
| userIds | userIDs |
| avatarUrl | avatarURL |

**1b. admin-user-system.yaml**
- query param `schoolId` → `schoolID`
- `PUT /admin/roles/{roleID}` 返回 `Role`（非 MessageData）
- `PUT /admin/groups/{groupID}` 返回 `UserGroup`（非 MessageData）
- `GET /admin/groups/{groupID}/members` 保持 `GroupMember[]`

**1c. ReviewStudentVerificationRequest**
- 保持 `{ approved: boolean, rejectionReason?: string }` 同构设计

**1d. BindPhoneRequest**
- 删除 `verificationCode` / `code` 字段，只保留 `phone`

**1e. 验证**: `make lint-spec`

### Phase 2: 后端 user 模块全量对齐

**2a. 响应扁平化（8 端点）**

| Endpoint | Before | After |
|----------|--------|-------|
| GET /user/identity | `{identity:{...}}` | 直接 UserIdentity |
| POST /user/identity | `{identity:{...}}` | 直接 UserIdentity (201) |
| GET /user/profile | `{profile:{...}}` | 直接 UserProfile |
| POST /user/profile/verify | `{profile:{...}}` | 直接 UserProfile |
| GET /user/profile/academic-info | `{student:{...}}` | 直接 AcademicStudentInfo |
| GET /user/schools | `{schools:[...]}` | 直接 SchoolConfig[] |
| GET /admin/school-configs | `{configs:[...]}` | 直接 AdminSchoolConfig[] |
| GET /admin/system-configs | `{configs:[...]}` | 直接 SystemConfig[] |

**2b. academic-info 补齐字段**: 新建 `academicInfoToJSON`，含全部 10 字段

**2c. ReviewStudentVerification**: handler 改回 `{ approved, rejectionReason }`

**2d. bind-phone**: 删除 handler/service 中的 code 字段

**2e. schoolID query**: 消费点统一

### Phase 3: 后端 RBAC 全量对齐

**3a. 响应扁平化（10 端点）**

| Endpoint | Before | After |
|----------|--------|-------|
| GET /admin/roles | `{roles:[...]}` | Role[] |
| POST /admin/roles | `{role:{...}}` | Role (201) |
| PUT /admin/roles/:id | `{role:{...}}` | Role |
| GET /admin/permissions | `{permissions:[...]}` | Permission[] |
| GET /admin/users/:id/roles | `{roles:[...]}` | Role[] |
| GET /admin/users/:id/permissions | `{permissions:[...]}` | EffectivePermission[] |
| GET /admin/groups | `{groups:[...]}` | UserGroup[] |
| POST /admin/groups | `{group:{...}}` | UserGroup (201) |
| PUT /admin/groups/:id | `{group:{...}}` | UserGroup |
| GET /admin/groups/:id/members | `{memberIds:[...]}` | GroupMember[] |

**3b. JSON tag 大写 ID**: permissionIDs, roleIDs, userIDs, permissionID

**3c. Gin 路由参数**: `:userId` → `:userID`, `:id` → `:roleID`/`:groupID`

**3d. GetGroupMembers**: service/repo 新增成员详情查询

### Phase 4: 领域层收敛

- `ReviewStudentVerification(ctx, userID, status, reason)` → `ReviewStudentVerification(ctx, userID, decision ReviewDecision)`
- `ReviewIdentity` 也统一用 `ReviewDecision`
- `BindPhoneRequest` 删除 Code 字段

### Phase 5: 前端 typed client

- 新建 `identity.ts`: user identity/profile/schools
- 新建 `user-admin.ts`: admin identities/verifications/school-configs/system-configs
- 新建 `rbac.ts`: roles/permissions/groups/members/user-roles/user-permissions
- 更新 barrel exports

### Phase 6: 生成 + 验证

```
make lint-spec && make generate && make check-drift
go build ./... && go test ./...
pnpm run type-check
```

## Acceptance Criteria

- [ ] `make lint-spec` 通过
- [ ] `make check-drift` 无 diff
- [ ] `go build && go test` 通过
- [ ] `pnpm type-check` 通过
- [ ] 所有 user/rbac 端点的 data 直接是对象/数组，无多余包裹
- [ ] 所有 JSON 字段遵循大写初始缩写
- [ ] ReviewStudentVerification 走 approved bool 全链路
- [ ] bind-phone 无残留验证码字段
- [ ] GroupMembers 返回完整 GroupMember[]
- [ ] 前端 3 个新 typed client 覆盖全部用户系统端点
