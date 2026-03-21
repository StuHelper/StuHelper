# Hook Guidelines

> How hooks are used in this project.

---

## Use composables, not React-style hooks

This project is Vue-based, so the real pattern is Vue composables under `clients/web/src/composables/`.

Use `useXxx.ts` naming for reusable stateful logic.

Representative files:

- `clients/web/src/composables/useAsyncData.ts`
- `clients/web/src/composables/useToast.ts`
- `clients/web/src/composables/useCommandPalette.ts`
- `clients/web/src/composables/useReviewPost.ts`

When writing Trellis docs, describe them as Vue composables even if the template uses the word "hook".

---

## Custom composable patterns

The strongest current composables share a few traits:

- they expose a small flat API
- they return refs and functions, not classes
- they clean up timers, observers, or listeners with `onScopeDispose`
- they centralize repeated UI or async behavior instead of copying it across views

Example from `useAsyncData.ts`:

```ts
return { data, loading, error, execute, reset };
```

Example from `useToast.ts`:

```ts
return {
	toasts,
	show,
	remove,
	success,
	error,
	info,
	warning,
	clearAll,
};
```

Prefer this style over deeply nested return objects or class-like wrappers.

---

## Data Fetching

There is no React Query or SWR layer here.

Current fetching patterns are:

- page-local fetch logic for one-off screens
- reusable composables for repeated async state patterns
- API access through the shared client layer, not ad hoc `fetch` calls inside every component

### Reusable async state pattern

`clients/web/src/composables/useAsyncData.ts` is the clearest shared pattern for:

- loading state
- error state
- reset behavior
- safe no-op writes after scope disposal

Example:

```ts
const data = shallowRef<T | null>(options?.initialData ?? null);
const loading = ref(false);
const error = ref<Error | null>(null);
```

Use this when multiple views or admin pages need the same `data/loading/error/execute` flow.

### API boundary pattern

Use the project API layer:

- shared endpoint wrappers in `clients/shared/src/api/`
- browser transport and auth behavior in `clients/web/src/api/client.ts`

Do not introduce direct `window.fetch(...)` calls in random pages when `api.*` already covers the operation.

---

## Naming Conventions

Follow the existing naming style:

- file names start with `use`
- the rest of the name is a noun or verb phrase describing the reusable behavior
- the file exports one primary composable with the same name

Examples:

- `useAsyncData`
- `useToast`
- `useCommandPalette`
- `useOptimisticUpdate`
- `useIntersectionObserver`

Do not create generic names like `helpers.ts` for stateful logic that belongs in a composable.

---

## Single-instance composables are allowed when the state is app-wide

Not every composable creates isolated state per call.

Current repo examples intentionally use module-level refs for shared UI state:

- `clients/web/src/composables/useToast.ts`
- `clients/web/src/composables/useCommandPalette.ts`
- `clients/web/src/composables/useReviewPost.ts`

This is acceptable when the goal is one shared app-level state source.

If you do this, make the singleton behavior obvious in the file design.

---

## Cleanup is part of the contract

Current composables already protect against stale updates and leaked resources.

Examples:

- `useAsyncData.ts` prevents writing to refs after scope disposal
- `useToast.ts` cleans timers in `onScopeDispose`
- `useCommandPalette.ts` manages listener lifetime carefully
- `useIntersectionObserver.ts` cleans up observers

If your composable owns a timer, event listener, observer, or async callback that can outlive the caller, add cleanup.

---

## Common Mistakes

### Common Mistake: duplicating loading and error state in many views

**Symptom**: each page recreates the same `loading/error/data` boilerplate.

**Cause**: skipping `useAsyncData` or another reusable composable.

**Fix**: extract the repeated state flow into a composable when the pattern repeats.

---

### Common Mistake: forgetting cleanup for timers or listeners

**Symptom**: stale callbacks fire after navigation, or polling keeps running unexpectedly.

**Cause**: composable sets up browser resources without `onScopeDispose`.

**Fix**: mirror the cleanup patterns already used in `useToast.ts` and related files.

---

### Common Mistake: hiding app-wide singleton behavior

**Symptom**: a composable looks local, but it actually shares module-level state across callers.

**Cause**: the file structure does not make the singleton intent obvious.

**Fix**: keep shared refs near the top-level module scope and expose a clear API.

---

### Common Mistake: inventing a fetch abstraction that bypasses the project API client

**Symptom**: browser requests skip CSRF handling, cookie credentials, or refresh behavior.

**Cause**: new composable calls `window.fetch(...)` directly.

**Fix**: use `api.*` wrappers and `clients/web/src/api/client.ts` so auth and error handling stay consistent.

---

## Examples to follow

- `clients/web/src/composables/useAsyncData.ts` — generic async state contract
- `clients/web/src/composables/useToast.ts` — singleton UI state plus timer cleanup
- `clients/web/src/composables/useCommandPalette.ts` — shared UI state plus listener management
- `clients/web/src/api/client.ts` — the real browser fetch boundary that composables should build on

---

## What is still evolving

The app uses a hybrid approach:

- some pages keep fetch logic inline
- some repeated patterns are extracted into composables
- not every screen has been refactored into the same level of reuse yet

Document that reality. Do not pretend every page-level side effect must already be hidden behind a composable.

---

## Admin Client Typed API Pattern

### Scope

This section applies when writing or modifying admin panel pages in `clients/admin/`.

### Contract

Admin pages must use the shared typed client layer instead of raw `fetchJSON` wrappers.

The admin API client is composed from domain-specific factory functions:

```ts
// clients/admin/src/api/index.ts
import { apiClient } from "./client";
import {
    createAuthApi,
    createAdminApi,
    createUserAdminApi,
    createRbacApi,
} from "@stuhelper/shared/api";

export const api = {
    auth: createAuthApi(apiClient),
    admin: createAdminApi(apiClient),
    userAdmin: createUserAdminApi(apiClient),
    rbac: createRbacApi(apiClient),
};
```

Each factory function lives in `clients/shared/src/api/` and uses `openapi-fetch` with generated DTOs from `clients/shared/src/types/api.gen.ts`.

### Usage in admin views

Import `api` from `@/api` and call the typed methods:

```ts
import { api } from '@/api'
import type { components } from '@stuhelper/shared'

type IdentityReviewItem = components['schemas']['AdminIdentityReviewItem']

// Typed request with generated DTOs
const res = await api.userAdmin.listIdentityVerifications({
  status: filterStatus.value,
  page: page.value,
  pageSize: pageSize.value,
})
const data = res.data?.data
```

### Adding a new admin domain

1. Create a factory function in `clients/shared/src/api/<domain>.ts`
2. Use generated DTO types from `components['schemas']`
3. Export the factory from `clients/shared/src/api/index.ts`
4. Register it in `clients/admin/src/api/index.ts`

### Wrong

```ts
// Raw fetchJSON wrapper with hand-written types
async function listIdentities(params: { status?: string }) {
  return fetchJSON('/api/v1/admin/identities', { params })
}
```

```ts
// Inline fetch in a view
const res = await fetch('/api/v1/admin/identities?status=pending')
```

### Correct

```ts
// Shared typed client using openapi-fetch + generated DTOs
export const createUserAdminApi = (client: ApiClient) => ({
  listIdentityVerifications: (params?: {
    status?: 'pending' | 'verified' | 'rejected' | 'all';
    page?: number;
    pageSize?: number;
  }) =>
    client.GET('/api/v1/admin/identities', { params: { query: params } }),
})
```

### Benefits

- Type safety from OpenAPI-generated DTOs
- Consistent auth/CSRF handling through the shared `apiClient`
- Single source of truth for endpoint paths and parameter shapes
- Compile-time errors when the API contract changes
