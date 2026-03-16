# Authorization Model

The authorization model consists of three parts: capabilities, access facts, and ownership checks.

## Capability Layer

Admin management capabilities come from `roles`, `permissions`, `role_permissions`, `user_roles`, `user_group_*`, and `user_permissions`. Common capabilities include:

| Capability | Purpose |
| --- | --- |
| `admin:dashboard:view` | Admin dashboard access |
| `admin:reviews:manage` | Review moderation and content management |
| `admin:reports:manage` | Report handling |
| `admin:teachers:manage` | Teacher management |
| `admin:sensitive_words:manage` | Sensitive word management |
| `admin:logs:view` | Operation log viewing |
| `user:*` | User system admin operations |
| `rbac:*` | Role, permission, and user group management |

## Access Fact Layer

Review and user system modules read these business facts:

| Fact | Source | Purpose |
| --- | --- | --- |
| `studentVerified` | `user_profiles` | Full content visibility |
| `identityVerified` | `user_identities` / `user_profiles` | Publishing eligibility and verification flow |
| `schoolID` | `user_profiles` | School-scoped filtering |
| `canManageReviews` | Capability check | Hidden content visibility and admin moderation view |

## Ownership Layer

Review editing, deletion, reply deletion, and similar actions check content ownership and resource status. Ownership checks happen within service-layer transactions.

## Authorization Outcomes

| Scenario | Rule |
| --- | --- |
| Anonymous review browsing | Returns publicly visible content |
| Authenticated user browsing | Content range determined by verification status |
| Post review | Requires student verification and identity verification |
| Admin management | Capabilities determine entry points and available actions |

## Related Documentation

- [Policy Evaluation Order](02-policy-evaluation-order.md)
- [RBAC Module](../rbac/README.md)
- [Identity and Authorization](../../architecture/ecosystem-identity-and-authorization.md)
