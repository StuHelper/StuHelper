# Authorization and Policy

This documentation describes the application authorization backbone, focusing on capabilities, access facts, and unified decision flow.

## Code Scope

| Code Location | Purpose |
| --- | --- |
| `server/internal/modules/rbac` | Capability computation and admin authorization |
| `server/internal/modules/course/review/access.go` | Review access facts and content filtering |
| `server/internal/pkg/capability` | Capability constants and admin entry sets |

## Documentation Index

| Document | Description |
| --- | --- |
| [01-hangxiaoban-authorization-model.md](01-hangxiaoban-authorization-model.md) | Capability and access fact model |
| [02-policy-evaluation-order.md](02-policy-evaluation-order.md) | Unified authorization decision flow |
