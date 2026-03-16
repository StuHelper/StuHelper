# Application RBAC

The RBAC module manages admin capabilities, role relationships, and personal permission overrides.

## Code Scope

| Code Location | Purpose |
| --- | --- |
| `server/internal/modules/rbac` | Roles, permissions, user groups, capability computation |
| `server/internal/pkg/capability` | Capability constants and admin entry capability sets |

## Data Model

| Entity | Description |
| --- | --- |
| **Role** | Coarse-grained business role (e.g., "Content Moderator", "System Admin") |
| **Permission** | Fine-grained capability string (e.g., `admin:reviews:manage`) |
| **User Role** | User-to-role binding |
| **User Group** | User group and membership relationships |
| **User Permission Override** | Explicit permission grants or denials for individual users |

Final capabilities are computed by merging role permissions, user group permissions, and personal overrides.

## Capability Computation

```go
// Pseudocode for capability computation
func ComputeUserCapabilities(userID string) []string {
    capabilities := []string{}

    // 1. Get capabilities from user roles
    roles := GetUserRoles(userID)
    for role in roles {
        capabilities += GetRolePermissions(role.ID)
    }

    // 2. Get capabilities from user groups
    groups := GetUserGroups(userID)
    for group in groups {
        capabilities += GetGroupPermissions(group.ID)
    }

    // 3. Apply user-specific overrides
    overrides := GetUserPermissionOverrides(userID)
    for override in overrides {
        if override.Type == "grant" {
            capabilities += override.Permission
        } else if override.Type == "deny" {
            capabilities -= override.Permission
        }
    }

    return unique(capabilities)
}
```

## API Endpoints

All admin endpoints are under `/api/v1/admin`:

- `/roles` - List and create roles
- `/roles/{roleID}` - Update and delete roles
- `/roles/{roleID}/permissions` - View and set role permissions
- `/permissions` - List all available permissions
- `/users/{userID}/roles` - View and set user roles
- `/users/{userID}/permissions` - View and set user permission overrides
- `/groups` - List and create user groups
- `/groups/{groupID}` - Update and delete user groups
- `/groups/{groupID}/members` - View and set group members
- `/groups/{groupID}/permissions` - Set group permissions

## Usage Pattern

### Frontend: Check Capability

```typescript
import { useAuth } from '@/composables/useAuth'

const { hasCapability } = useAuth()

// In template
<el-button v-if="hasCapability('admin:reviews:manage')">
  Moderate Reviews
</el-button>

// In script
if (hasCapability('admin:reviews:manage')) {
  // Show admin UI
}
```

### Backend: Require Capability

```go
// In handler registration
adminGroup := router.Group("/api/v1/admin")
adminGroup.Use(rbacMiddleware.RequireCapability("admin:dashboard:view"))

reviewsGroup := adminGroup.Group("/reviews")
reviewsGroup.Use(rbacMiddleware.RequireCapability("admin:reviews:manage"))
reviewsGroup.GET("", handler.ListReviews)
reviewsGroup.PUT("/:id", handler.UpdateReview)
```

## Common Capabilities

| Capability | Purpose |
| --- | --- |
| `admin:dashboard:view` | Access admin dashboard |
| `admin:reviews:manage` | Moderate reviews and content |
| `admin:reports:manage` | Handle user reports |
| `admin:teachers:manage` | Manage teacher records |
| `admin:sensitive_words:manage` | Manage sensitive word list |
| `admin:logs:view` | View operation logs |
| `user:identities:review` | Review identity verification requests |
| `user:students:review` | Review student verification requests |
| `user:schools:manage` | Manage school configurations |
| `user:system_configs:manage` | Manage system configurations |
| `rbac:roles:manage` | Manage roles |
| `rbac:permissions:manage` | Manage permissions |
| `rbac:groups:manage` | Manage user groups |

## Consumption Pattern

- `/auth/me` returns the capability set
- Admin routes and admin pages read capabilities
- `isPlatformAdmin` maintains platform administrator semantics (separate from application capabilities)

## Related Documentation

- [Authorization Model](../policy/01-hangxiaoban-authorization-model.md)
- [Policy Evaluation Order](../policy/02-policy-evaluation-order.md)
- [Identity and Authorization](../../architecture/ecosystem-identity-and-authorization.md)
