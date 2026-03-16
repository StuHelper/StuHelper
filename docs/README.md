# StuHelper Documentation

`docs/` organizes content by use case with stable entry points and clear topics. All documentation directly corresponds to the current codebase, APIs, and data structures.

## Source of Truth Priority

Technical facts are read in this order:

1. `server/api/openapi.yaml` - API contracts
2. `server/scripts/init.sql` - Database schema
3. Current code and tests - Implementation
4. `docs/` - Organized documentation

## Key Concepts

| Term | Definition |
| --- | --- |
| **Capability** | A permission string (e.g., `admin:reviews:manage`) that grants access to specific backend features. Capabilities are computed from roles, groups, and user-specific overrides. |
| **Access Facts** | Business conditions (e.g., `studentVerified`, `identityVerified`, `schoolID`) used to determine content visibility and action eligibility beyond capabilities. |
| **Platform Admin** | A user marked as admin in Casdoor (the SSO provider). This is separate from application-level capabilities. |
| **Student Verification** | The process of verifying a user is a student, either via LDAP or manual review. |
| **Identity Verification** | Real-name verification using government ID documents. |

## Documentation Structure

| Directory | Purpose | Entry Point |
| --- | --- | --- |
| `tutorials/` | Step-by-step getting started | [tutorials/README.md](tutorials/README.md) |
| `guides/` | Task-oriented development guides | [guides/README.md](guides/README.md) |
| `reference/` | API routes, database schema, error codes | [reference/README.md](reference/README.md) |
| `architecture/` | System layers, frontend structure, identity and authorization boundaries | [architecture/README.md](architecture/README.md) |
| `modules/` | Business domain and module documentation | [modules/README.md](modules/README.md) |

## Quick Navigation

| Goal | Document |
| --- | --- |
| Set up the project locally | [tutorials/quick-start.md](tutorials/quick-start.md) |
| Continue backend development | [guides/backend-quickstart.md](guides/backend-quickstart.md) |
| Continue frontend development | [guides/frontend-development.md](guides/frontend-development.md) |
| Work with the OpenAPI workflow | [guides/openapi-development-guide.md](guides/openapi-development-guide.md) |
| Look up APIs and data | [reference/api-overview.md](reference/api-overview.md), [reference/database.md](reference/database.md) |
| Understand module boundaries | [modules/README.md](modules/README.md) |
| Understand system architecture | [architecture/README.md](architecture/README.md) |

## Modules

| Module | Code Entry Points | Documentation |
| --- | --- | --- |
| Authentication and Sessions | `server/internal/modules/auth`, `server/internal/pkg/sso`, `server/internal/pkg/token` | [modules/auth/](modules/auth/) |
| Course Review Community | `server/internal/modules/course`, `server/internal/modules/course/review` | [modules/course/](modules/course/) |
| User System | `server/internal/modules/user`, `server/internal/modules/ldap` | [modules/user-system/](modules/user-system/) |
| Application RBAC | `server/internal/modules/rbac` | [modules/rbac/](modules/rbac/) |
| Authorization Policy | `server/internal/modules/rbac`, `server/internal/modules/course/review/access.go` | [modules/policy/](modules/policy/) |
| Logging and Audit | `server/internal/pkg/logger`, `server/internal/pkg/middleware`, `server/internal/pkg/audit` | [modules/logging/](modules/logging/) |

## Project Rules

Project workflow and documentation rules are maintained under `.trellis/`:

- `.trellis/workflow.md` - Development workflow
- `.trellis/spec/guides/index.md` - Thinking guides
- `.trellis/spec/backend/index.md` - Backend guidelines
- `.trellis/spec/frontend/index.md` - Frontend guidelines
