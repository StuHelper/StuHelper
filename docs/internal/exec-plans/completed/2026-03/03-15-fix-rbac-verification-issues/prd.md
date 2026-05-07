---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# 修复 RBAC 和实名认证的关键问题

## Goal

修复 MR16 和 MR18 中发现的 4 个关键问题，确保 RBAC 权限控制真正生效，实名认证状态查询语义正确，以及前端文案判断逻辑准确。

## Problems Identified

### 问题 1: RBAC 权限中间件未实际应用到 admin 路由 (Critical)

**现状**:
- `rbac/middleware.go` 已实现 `RequirePermission(...)` 中间件
- `rbac/service.go` 已实现完整的权限检查逻辑（角色、组、个人覆盖、scope）
- 但 `main.go:304-308` 只使用了 `authMW + adminMW`，没有挂载任何细粒度权限检查

**影响**:
- 数据库里能配置角色、权限、用户组、override
- 后台页面也能管理这些数据
- 但运行时访问 `/api/v1/admin/*` 时根本不看这些配置
- RBAC 现在只是"权限配置管理系统"，不是"真实生效的权限系统"

**根本原因**:
- 路由注册时缺少权限中间件的应用

### 问题 2: 实名认证列表的 pending/rejected/verified 语义不完整 (High)

**现状**:
- OpenAPI spec (`admin-user-system.yaml:10-14`) 定义状态为 `pending | verified | rejected | all`
- 数据库 `user_profiles` 表只有 `verification_status` 字段（实际存储的值）
- Repository 查询 (`user/repository.go:244-306`) 只按 `verification_status` 字段过滤
- 但实际业务逻辑中：
  - `pending` = `verification_status='pending'`
  - `verified` = `verification_status='verified'`
  - `rejected` = `verification_status='rejected'`

**需要确认**:
- 数据库 schema 中 `verification_status` 字段的实际定义
- 当前是否已经是 enum 类型还是 varchar
- 如果已经是 enum，值是否包含 pending/verified/rejected

**影响**:
- 如果查询逻辑不完整，管理员会看到混合的结果
- 业务语义不清晰

### 问题 3: 前端实名认证成功文案判断错误 (Medium)

**现状**:
- `IdentityVerificationPage.vue:45-46` 判断 `identity.verifyMethod === 'academic_db_match' || identity.verifyMethod === 'tencent_cloud'`
- 后端实际返回的值是 `academic_db_match`、`manual`、`tencent_cloud`
- 逻辑正确，但需要确认是否有其他遗漏的值

**影响**:
- 文案显示可能不准确

### 问题 4: Admin 路由引用了不存在的 logs 页面 (Low)

**现状**:
- `clients/admin/src/views/dashboard/index.vue:61` 引用了 `admin-logs` 路由
- 需要确认该路由是否存在

**影响**:
- 点击链接可能 404

## Requirements

### R1: 为所有 admin 路由添加细粒度权限控制

- [x] 为每个 admin 路由定义所需的权限名称
- [x] 在路由注册时应用 `RequirePermission(...)` 中间件
- [x] 确保权限检查逻辑真正生效

**权限映射设计**:
```
RBAC 管理:
- GET /admin/roles -> rbac:role:read
- POST /admin/roles -> rbac:role:create
- PUT /admin/roles/:roleID -> rbac:role:update
- DELETE /admin/roles/:roleID -> rbac:role:delete
- GET /admin/roles/:roleID/permissions -> rbac:role:read
- PUT /admin/roles/:roleID/permissions -> rbac:role:update
- GET /admin/permissions -> rbac:permission:read
- GET /admin/users/:userID/roles -> rbac:user:read
- PUT /admin/users/:userID/roles -> rbac:user:update
- GET /admin/users/:userID/permissions -> rbac:user:read
- PUT /admin/users/:userID/permissions -> rbac:user:update
- GET /admin/groups -> rbac:group:read
- POST /admin/groups -> rbac:group:create
- PUT /admin/groups/:groupID -> rbac:group:update
- DELETE /admin/groups/:groupID -> rbac:group:delete
- GET /admin/groups/:groupID/members -> rbac:group:read
- PUT /admin/groups/:groupID/members -> rbac:group:update
- PUT /admin/groups/:groupID/permissions -> rbac:group:update

用户管理:
- GET /admin/identities -> user:identity:read
- PUT /admin/identities/:userID -> user:identity:review
- GET /admin/student-verifications -> user:student:read
- PUT /admin/student-verifications/:userID -> user:student:review
- GET /admin/school-configs -> user:school:read
- PUT /admin/school-configs/:schoolID -> user:school:update
- GET /admin/system-configs -> user:system:read
- PUT /admin/system-configs/:key -> user:system:update
```

### R2: 修复实名认证状态查询逻辑

- [x] 确认数据库 schema 中 `verification_status` 的定义
- [x] 如果已经是正确的 enum，确保查询逻辑正确
- [x] 如果不是，需要添加 migration 或调整查询逻辑

### R3: 修复前端文案判断逻辑

- [x] 确认后端返回的所有可能的 `verifyMethod` 值
- [x] 修正前端判断逻辑
- [x] 确保所有情况都有正确的文案

### R4: 修复 admin logs 路由引用

- [x] 确认 `admin-logs` 路由是否存在
- [x] 如果不存在，移除引用或创建占位页面

## Acceptance Criteria

- [x] 所有 admin 路由都应用了细粒度权限检查
- [x] 权限检查逻辑在运行时真正生效
- [x] 实名认证状态查询返回正确的结果
- [x] 前端文案判断逻辑正确
- [x] Admin logs 路由引用问题已解决
- [x] 所有修改通过 lint 和 typecheck
- [x] 手动测试验证功能正常

## Technical Notes

### 实现策略

**Phase 1: 调研和确认**
1. 检查数据库 schema 确认 `verification_status` 定义
2. 检查后端代码确认所有 `verifyMethod` 可能值
3. 检查 admin 路由确认 logs 页面是否存在

**Phase 2: 修复 RBAC 权限控制**
1. 在数据库中插入所需的权限定义（如果不存在）
2. 修改 `rbac/handler.go` 的 `RegisterAdminRoutes` 方法，为每个路由添加权限中间件
3. 修改 `user/handler.go` 的 `RegisterAdminRoutes` 方法，为每个路由添加权限中间件

**Phase 3: 修复实名认证状态查询**
1. 根据调研结果，修复 repository 查询逻辑或添加 migration

**Phase 4: 修复前端问题**
1. 修正 `IdentityVerificationPage.vue` 的文案判断逻辑
2. 修复或移除 admin logs 路由引用

### 风险和注意事项

1. **权限定义需要预先存在**: 在应用权限中间件之前，需要确保数据库中已经有对应的权限记录
2. **向后兼容**: 如果修改数据库 schema，需要考虑现有数据的迁移
3. **测试覆盖**: 需要手动测试所有受影响的路由

## Development Type

fullstack (backend + frontend)
