# RBAC Role Permission Flow

> Keep role permission assignment safe across OpenAPI, backend, and admin UI.

## Scope

This guide applies when changing any of:

- `GET /api/v1/admin/roles/{roleID}/permissions`
- `PUT /api/v1/admin/roles/{roleID}/permissions`
- admin role-permission dialogs
- `AssignRolePermissionsRequest`

Primary files:

- `server/api/components/schemas/user-system.yaml`
- `server/api/paths/admin-user-system.yaml`
- `server/internal/modules/rbac/middleware.go`
- `server/internal/modules/rbac/handler.go` (及 `handler_*.go` 拆分文件)
- `server/internal/modules/rbac/service.go` (及 `service_*.go` 拆分文件)
- `server/internal/modules/rbac/repository.go` (及 `repository_*.go` 拆分文件)
- `server/internal/pkg/capability/capability.go`
- `clients/shared/src/api/rbac.ts`
- `clients/shared/src/constants/capabilities.ts`
- `clients/admin/src/views/rbac/RoleManage.vue`

## Contract

### Read current role permissions

`GET /api/v1/admin/roles/{roleID}/permissions`

Success payload:

```json
{
  "success": true,
  "data": {
    "permissionIDs": [1, 2, 3]
  }
}
```

Rules:

- Path parameter is always `roleID`
- Response returns permission IDs, not permission names
- Order should be stable enough for testing (`ORDER BY permission_id ASC`)

### Update role permissions

`PUT /api/v1/admin/roles/{roleID}/permissions`

Request payload:

```json
{
  "permissionIDs": [1, 2, 3],
  "clearAll": false
}
```

Rules:

- `permissionIDs` is required as a field
- `permissionIDs: []` is only allowed when `clearAll: true`
- System roles cannot have permissions reassigned through this endpoint
- Duplicated permission IDs should be deduplicated in service logic
- Unknown permission IDs must be rejected before write

## Validation Matrix

| Case | Input | Expected Result |
| --- | --- | --- |
| Good | `permissionIDs=[1,2]`, `clearAll=false` | Replace assigned permissions |
| Base | `permissionIDs=[]`, `clearAll=true` | Clear all permissions |
| Bad | missing `permissionIDs` | `400 Bad Request` |
| Bad | `permissionIDs=[]`, `clearAll=false` | `400 Bad Request` |
| Bad | `permissionIDs` contains invalid ID | `400 Bad Request` |
| Bad | role is a system role | `403 Forbidden` |
| Bad | role does not exist | `404 Not Found` |

## Layer Responsibilities

### OpenAPI

- Defines `RolePermissionIDsResponse`
- Defines `AssignRolePermissionsRequest.permissionIDs`
- Defines `AssignRolePermissionsRequest.clearAll`

### Handler

- Parse `roleID`
- Ensure `permissionIDs` field is present
- Map service errors to `400` / `403` / `404` / `500`
- Use `response.Success(...)` and `response.BadRequest(...)`

### Service

- Check role existence
- Reject system-role mutations
- Deduplicate permission IDs
- Reject invalid IDs
- Reject empty assignments without explicit `clearAll`

### Repository

- `GetRolePermissionIDs(...)` reads current assignment IDs
- `SetRolePermissions(...)` remains the transactional replace operation

### Admin UI

- Load current permission IDs before enabling save
- If loading fails, keep save disabled
- Require a confirmation dialog before sending `clearAll: true`
- Use the shared typed `api.rbac` client instead of local raw wrappers

## Required Tests

Backend:

- handler: get role permission IDs success
- handler: missing `permissionIDs` returns `400`
- handler: empty `permissionIDs` without `clearAll` returns `400`
- handler: system role assignment returns `403`
- service: empty assignment requires explicit `clearAll`
- service: system role assignment is rejected
- service: invalid permission IDs are rejected

Frontend:

- permission dialog does not allow save after load failure
- clear-all action requires explicit confirmation

## Common Regression Pattern

The dangerous regression is:

1. dialog fails to load current permissions
2. local selection stays empty
3. save still remains available
4. backend interprets empty array as a valid replacement
5. all role permissions are deleted

Never rely on frontend state alone to prevent this. The backend must require explicit `clearAll`.

## Admin 路由权限中间件缓存

`RequireAnyPermission`（挂在 admin 路由组）和 `RequirePermission`（挂在每条路由）
共享 Gin context 缓存。详见 `authorization-architecture.md` § 9。

关键规则：
- 新增 admin 路由时 **必须** 添加 `RequirePermission`，否则仅受组级 `RequireAnyPermission` 保护（持有任一 admin 能力即可访问）
- `capability.go` 和 `capabilities.ts` 中的常量必须保持一致
- seed SQL 中的权限名必须与 `capability.go` 常量完全匹配
- 已被细粒度权限取代的粗粒度权限（如 `admin:users:manage`）不应再保留在 seed SQL 中
