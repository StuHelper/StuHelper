# Authorization Architecture

> Canonical target architecture for ecosystem identity, app authorization, and open-platform data access.

---

## Scenario: Ecosystem Identity And Application Authorization

### 1. Scope / Trigger

- Trigger: any change that touches SSO, admin gating, roles, permissions, groups, verification-derived access, third-party app onboarding, or resource-scoped moderation.
- This spec is mandatory for:
  - `server/internal/modules/auth`
  - `server/internal/modules/rbac`
  - admin route middleware
  - app capability APIs
  - open-platform scopes / claims / userinfo surfaces
- This project must not treat `Casdoor isAdmin` or any other upstream platform-admin flag as the business-authorization source for Hangxiaoban.

### 2. Decision Summary

- `sso.stuhelper.com` is the ecosystem identity plane.
- There is no single "StuHelper application" in Casdoor. Each real app such as `hangxiaoban`, `developer-portal`, or a third-party client must have its own Casdoor Application.
- `航小伴` is an application inside the ecosystem, not the ecosystem itself.
- Casdoor is authoritative for:
  - authentication
  - OAuth/OIDC application onboarding
  - platform-level administrator identity
  - ecosystem-level scopes and consent
  - coarse app-access grants when needed
- Hangxiaoban is authoritative for:
  - business facts
  - verification facts
  - course / resource / category ownership and moderation relationships
  - final business authorization decisions
- Preferred long-term authorization model for Hangxiaoban:
  - coarse app/module roles
  - resource relationships
  - business attributes
  - optional dedicated relationship engine such as OpenFGA / SpiceDB once resource-scoped rules grow beyond simple module RBAC

### 3. Signatures

#### 3.1 Identity plane input

Casdoor-issued session or token data must be normalized into a principal model with this shape:

```go
type IdentityPrincipal struct {
    Subject          string
    Organization     string
    EcosystemGroups  []string
    EcosystemRoles   []string
    OAuthScopes      []string
    IsPlatformAdmin  bool
    ApplicationID    string
}
```

Required rule:

- `IsPlatformAdmin` means platform / identity-administration power.
- `IsPlatformAdmin` must not be used as the Hangxiaoban business-admin gate.

#### 3.2 Hangxiaoban authorization check

Business authorization checks must converge on a single app-owned decision path:

```go
type AuthorizationInput struct {
    SubjectID   string
    Action      string
    Resource    AuthorizationResource
    Facts       AuthorizationFacts
}

type AuthorizationResource struct {
    Kind string
    ID   string
}

type AuthorizationFacts struct {
    SchoolID             *string
    ActorType            *string
    StudentVerified      bool
    IdentityVerified     bool
    IsResourceOwner      bool
    IsCourseTeacher      bool
}
```

Expected decision kinds:

```go
const (
    DecisionAllow   = "allow"
    DecisionDeny    = "deny"
    DecisionPartial = "partial"
)
```

`partial` exists because Hangxiaoban has content-shaping rules such as:

- authenticated but not student-verified users can view only a truncated review
- unverified publishers may be shown with warning labels instead of being fully blocked

#### 3.3 Open-platform data release

Third-party applications must not receive raw high-sensitivity identity data by default.

Minimal profile / fact release surfaces must be scope-gated and support values such as:

```json
{
  "identityVerified": true,
  "studentVerified": true,
  "actorType": "student",
  "schoolID": "10006"
}
```

These surfaces must not include:

- real name
- student ID
- certificate / ID-card data
- phone number

### 4. Contracts

#### 4.1 What belongs in Casdoor

Use Casdoor for:

- login and session issuance
- app registration and callback management
- OAuth scopes and consent
- platform-level administrators
- first-party vs third-party app onboarding policy
- optional coarse app-entry grants such as `hangxiaoban.access`

Casdoor officially supports authorization models built on Casbin, including ACL, RBAC, ABAC, and RESTful access control. It also supports `JWT-Custom` token fields that can emit `roles`, `groups`, and `permissions` as claims. Those capabilities are valid for identity-plane policy, coarse app access, and open-platform scopes.

Do not use Casdoor as the storage layer for:

- course-specific moderators
- category-specific resource managers
- owner-only content rights
- teacher-of-course relationships
- school-specific visibility rules
- verification-dependent posting / viewing policies

Reason:

- high-cardinality business relations belong to the application domain
- Hangxiaoban business facts such as `schoolID`, verification state, teacher-of-course, and content ownership live in Hangxiaoban data, not in the identity plane
- synchronous per-request calls to Casdoor management APIs are not an acceptable runtime authorization path

#### 4.2 What belongs in Hangxiaoban

Use Hangxiaoban-owned authorization logic for:

- app-wide admin (`hangxiaoban.admin`)
- module admin (`review.manage`, `resource.manage`)
- resource-scoped relations:
  - `course.manager`
  - `course.intro_editor`
  - `course.resource_editor`
  - `course.review_moderator`
  - `resource_category.manager`
  - `content.owner`
  - `course.teacher`
- attribute-driven checks:
  - `schoolID` is included in the configured review-access school whitelist
  - actor type is `student`
  - student verification is complete
  - identity verification is complete

#### 4.3 Platform-admin boundary

- Casdoor built-in super user is a bootstrap / break-glass account only.
- Day-to-day platform administration must use a normal account granted a platform-admin role or group in Casdoor.
- Platform-admin power automatically covers:
  - `sso.stuhelper.com`
  - first-party app lifecycle management
  - ecosystem-level operational controls
- Platform-admin power must not implicitly grant unrestricted business-data operations inside third-party applications unless those apps explicitly opt into that trust model.

#### 4.4 First-party admin gating

Wrong:

- frontend route guard checks `isAdmin`
- backend `/admin` group checks only `isAdmin`

Correct:

- `/api/v1/auth/me` returns:
  - `capabilities`: all granted capability names, including scoped grants
  - `globalCapabilities`: only grants with no school / role scope restrictions
  - `capabilityGrants`: full grant records with `name`, `scopeSchoolIDs`, `scopeRoles`, `global`
  - `canAccessAdmin`: derived from `globalCapabilities`, not from scoped grants
- frontend renders current admin menus and route entry from `globalCapabilities`
- backend authorizes each business route from Hangxiaoban-owned authorization decisions

Executable contract:

```json
{
  "id": "casdoor-user-id",
  "name": "alice",
  "displayName": "Alice",
  "isPlatformAdmin": false,
  "capabilities": ["rbac:role:read", "user:identity:review"],
  "globalCapabilities": ["rbac:role:read"],
  "capabilityGrants": [
    {
      "name": "rbac:role:read",
      "global": true
    },
    {
      "name": "user:identity:review",
      "scopeSchoolIDs": ["10006"],
      "global": false
    }
  ],
  "canAccessAdmin": true
}
```

Current frontend rule:

- `clients/admin/src/stores/auth.ts` and `clients/web/src/stores/auth.ts` may persist full `capabilityGrants`
- current route/menu gating must use `globalCapabilities`
- scoped grants are retained for future scoped UI, but must not unlock current full-page admin entry

### 5. Validation & Error Matrix

| Scenario | Expected Decision | Reason |
| --- | --- | --- |
| Unauthenticated request to protected Hangxiaoban API | `401` | Authentication failure |
| Authenticated, no matching app permission / relation | `403` | Authorization failure |
| Authenticated, partial-view policy applies | `200` with truncated / labeled payload | Content shaping, not hard deny |
| Third-party app requests verification facts without scope | `403` or `insufficient_scope` | Scope boundary |
| `/auth/me` includes only scoped grant for an admin page and no matching global grant | `canAccessAdmin=false` and menu stays hidden | Current frontend cannot safely enter full admin pages with scoped grants only |
| First-party admin route uses `isAdmin` only | Invalid architecture | Must be rejected in review |
| Platform admin accesses third-party business data without explicit delegation | Invalid architecture | Platform/app boundary violation |

### 6. Good / Base / Bad Cases

#### Good

- Casdoor authenticates the user.
- Hangxiaoban resolves:
  - app role
  - module role
  - resource relationship
  - verification facts
- backend returns `allow`, `deny`, or `partial` based on app-owned rules.

#### Base

- Casdoor token provides only subject, org, groups, scopes, and platform-admin status.
- Hangxiaoban reads business facts from its own datastore.
- external applications obtain only minimal verification facts through scope-gated endpoints.

#### Bad

- `isAdmin == true` means full Hangxiaoban admin
- course moderators are stored as Casdoor global roles
- student-verification facts are encoded only in frontend state
- every request calls Casdoor management APIs synchronously for permission checks

### 7. Tests Required

- Authentication tests:
  - unauthenticated requests fail before authorization
- Authorization tests:
  - app admin can access all Hangxiaoban admin modules
  - module admin can access only its own module
  - course manager can edit only assigned courses
  - owner can edit only self-created content
- Current-user contract tests:
  - `/api/v1/auth/me` returns `capabilities`, `globalCapabilities`, `capabilityGrants`, and `canAccessAdmin`
  - scoped-only grants appear in `capabilityGrants` and `capabilities`, but not in `globalCapabilities`
  - `canAccessAdmin` becomes `true` only when at least one `globalCapabilities` item matches `capability.AdminEntryCapabilities`
- Attribute-policy tests:
  - protected review publishing requires `studentVerified=true`, `identityVerified=true`, and `schoolID` allowed by the current review-access policy
  - partial-view policy returns truncated payload based on backend-configured preview title length, content length, and content percentage
- Open-platform tests:
  - third-party app without verification scope cannot read verification facts
  - granted scopes expose only minimal fact fields, never full PII
- Boundary tests:
  - `isAdmin` alone must not pass Hangxiaoban business-admin routes
  - platform-admin override must be explicit and limited to approved first-party scopes

### 8. Wrong vs Correct

#### Wrong

```go
admin := api.Group("/admin")
admin.Use(authMW, RequireAdmin(ssoClient))
```

```ts
if (!currentUser.isAdmin) {
  router.replace("/login")
}
```

This treats an identity-platform flag as the business authorization source.

#### Correct

```go
// 组级纵深防御：认证 + 至少持有一个 admin 能力
admin := api.Group("/admin")
admin.Use(authMW, rbac.RequireAnyPermission(rbacService, capability.AdminEntryCapabilities...))

// 路由级细粒度：具体操作权限 + scope 约束
admin.GET("/roles", rbac.RequirePermission(rbacService, capability.RBACRoleRead), handler.ListRoles)
```

```ts
const canManageReviewModule = capabilities.includes("admin:reviews:manage")
```

This keeps authentication in the identity plane and business authorization in the application plane.

### 9. Middleware Caching Pattern

`RequireAnyPermission`（组级中间件）解析 internal user ID 和 effective permissions
后，将它们缓存在 Gin context 中。下游的 `RequirePermission`（路由级中间件）直接
从缓存读取，避免重复 DB 查询。

```
请求 → authMW → RequireAnyPermission → RequirePermission → Handler
                 ↓ 缓存                   ↓ 读缓存
                 ctxKeyInternalUserID      (跳过 GetInternalUserID)
                 ctxKeyEffectivePerms      (跳过 GetEffectivePermissions)
```

**PermissionService 接口设计**：

| 方法 | 用途 | 何时使用 |
| --- | --- | --- |
| `GetInternalUserID` | Casdoor external_id → 内部 user.id | 首次解析时（后续从缓存读取） |
| `GetEffectivePermissions` | 加载用户全部生效权限 | 首次加载时（后续从缓存读取） |
| `CheckPermission` | 加载权限 + 名称匹配 + scope 验证 | 中间件链外部（如 access.go） |
| `CheckPermissionScope` | 仅 scope 验证（跳过权限加载） | 中间件链内部（有缓存时） |

**Wrong**: `RequireAnyPermission` 调用 `GetUserCapabilities`（内部又调 `GetInternalUserID`
+ `GetEffectivePermissions`），`RequirePermission` 再次调用 `GetInternalUserID` +
`CheckPermission`（又调 `GetEffectivePermissions`），每次 admin 请求 5-6 次 DB 查询。

**Correct**: 两个中间件共享 `resolveInternalUserID` / `resolveEffectivePermissions`
辅助函数，首次调用时解析并缓存，后续调用直接读取。每次 admin 请求 3-4 次 DB 查询。

### 10. Admin Route Defense-in-Depth Pattern

Admin route groups must use layered authorization:

```go
// Group-level: require at least one admin capability
admin := api.Group("/admin")
admin.Use(authMW, rbac.RequireAnyPermission(rbacService, capability.AdminEntryCapabilities...))

// Route-level: require specific permission + scope constraints
admin.GET("/roles", rbac.RequirePermission(rbacService, capability.RBACRoleRead), handler.ListRoles)
```

**Why both layers?**

- Group-level guard ensures admin capability is required even if a per-route guard is missed
- Route-level guard enforces fine-grained permission and scope constraints
- Defense-in-depth prevents accidental exposure of admin endpoints

**Wrong**:

```go
// Only route-level guards, no group-level protection
admin := api.Group("/admin")
admin.Use(authMW)
admin.GET("/roles", rbac.RequirePermission(rbacService, capability.RBACRoleRead), handler.ListRoles)
// If a new route forgets RequirePermission, any authenticated user can access it
```

**Correct**:

```go
// Both layers protect admin routes
admin := api.Group("/admin")
admin.Use(authMW, rbac.RequireAnyPermission(rbacService, capability.AdminEntryCapabilities...))
admin.GET("/roles", rbac.RequirePermission(rbacService, capability.RBACRoleRead), handler.ListRoles)
// Even if RequirePermission is missed, RequireAnyPermission still blocks non-admin users
```

**Required for**:

- All admin route groups
- Any route group that should only be accessible to users with specific capabilities

**Not required for**:

- Public routes
- User-facing routes that use attribute-based or resource-scoped authorization

### 11. Evolution Path

- Short term:
  - introduce relation-backed authorization checks for course/category/content ownership
- Long term:
  - adopt OpenFGA / SpiceDB if relation cardinality and delegation flows outgrow simple local RBAC tables

### 10. External References

- Casdoor permission overview: <https://casdoor.org/docs/permission/overview>
- Casdoor permission configuration: <https://casdoor.org/docs/permission/permission-configuration>
- Casdoor token overview: <https://casdoor.org/docs/token/overview>
- Casdoor user roles: <https://casdoor.org/docs/user/roles>
- Casdoor user permissions: <https://casdoor.org/docs/user/permissions>
