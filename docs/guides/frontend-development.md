# Frontend Development Guide

This guide is for developers who already have the project running. It focuses on the frontend monorepo's development entry points and collaboration sequence.

> For environment setup, see [Quick Start](../tutorials/quick-start.md).

## The Four Packages

| Package | Purpose | What You Typically Change |
| --- | --- | --- |
| `clients/web` | Main web app | Pages, routes, browser adaptation, user experience |
| `clients/admin` | Independent admin console | Admin pages, menus, RBAC management flows |
| `clients/shared` | Shared API and types | OpenAPI-generated types, shared client, capability constants |
| `clients/uniappx` | Cross-platform experimental entry | Experimental pages and platform adaptation |

Most feature changes touch `clients/web` or `clients/admin`, plus `clients/shared`.

## Adding a New Page

- Main app pages go in `clients/web/src/modules/<module>/views/`
- Admin pages follow each app's `views/` and `router/` structure
- Business components go in `clients/web/src/components/business/<domain>/` or the corresponding admin module
- Shared contracts and capabilities go in `clients/shared`

## Routing and Access Control

The project uses object-oriented routing conventions:

- List pages use plural nouns: `/courses`
- Detail pages nest under the object: `/courses/:id`
- Actions on an object continue nesting: `/courses/:id/reviews/post`

Login state is handled by existing route guards. Admin or restricted pages use three layers of access control:

1. Route declares `requiresAuth`
2. Route or menu declares `requiredCapabilities`
3. Page buttons further check `capabilities` and `canAccessAdmin`

Both `clients/web` and `clients/admin` follow these rules.

Example route definition:

```typescript
{
  path: '/admin/reviews',
  component: () => import('@/views/admin/ReviewList.vue'),
  meta: {
    requiresAuth: true,
    requiredCapabilities: ['admin:reviews:manage']
  }
}
```

Example button visibility:

```vue
<template>
  <el-button
    v-if="hasCapability('admin:reviews:manage')"
    @click="handleModerate"
  >
    Moderate
  </el-button>
</template>
```

## Adding a New API Endpoint

Follow this sequence when adding a new endpoint:

1. Edit `server/api/openapi.yaml` and its split files
2. Run `make generate` in `server/`
3. Verify `server/internal/api/gen/` and `clients/shared/src/types/api.gen.ts` are updated
4. Add domain API wrappers in `clients/shared/src/api/`
5. Implement page logic in `web` or `admin`

The shared pipeline is: OpenAPI -> `clients/shared` -> `web/admin`.

Example API wrapper:

```typescript
// clients/shared/src/api/course.ts
import { client } from './client'

export const courseApi = {
  getCourse: (courseID: string) =>
    client.GET('/api/v1/course/courses/{courseID}', {
      params: { path: { courseID } }
    }),

  searchCourses: (query: string, page = 1) =>
    client.GET('/api/v1/course/courses/search', {
      params: { query: { q: query, page } }
    })
}
```

## Authentication Notes

- Access Token and Refresh Token are written to cookies by the backend
- Browser requests use `credentials: 'include'`
- Mutation requests automatically include `X-CSRF-Token`
- On 401, the client first tries `/api/v1/auth/refresh`; if that fails, it clears the local session
- After login, user state and admin accessibility are determined by the `/api/v1/auth/me` response: `capabilities`, `canAccessAdmin`, `isPlatformAdmin`

The `isPlatformAdmin` field represents the platform admin identity from Casdoor. Application-level admin access is controlled by capabilities.

## Pre-Commit Checks

Run at minimum before committing frontend changes:

```bash
cd clients
pnpm type-check
pnpm lint
```

For changes to critical main app flows, also run:

```bash
cd clients
pnpm test:web
pnpm test:e2e:web
```

## Key Files

| File | Purpose |
| --- | --- |
| `clients/web/src/router/index.ts` | Main app routes and guards |
| `clients/admin/src/router/index.ts` | Admin console routes and access control |
| `clients/web/src/api/client.ts` | Browser Cookie, CSRF, refresh logic |
| `clients/admin/src/api/` | Admin API wrappers |
| `clients/shared/src/api/` | Shared base client |
| `clients/shared/src/types/api.gen.ts` | OpenAPI-generated types |

## Related Documentation

- [Frontend Architecture](../architecture/frontend-architecture.md)
- [OpenAPI Development Guide](openapi-development-guide.md)
- [API Overview](../reference/api-overview.md)
