# State Management

> How state is managed in this project.

---

## Use Pinia for cross-page state, not for everything

The current frontend uses Pinia setup stores, but it does not force every piece of state into global storage.

Representative stores:

- `clients/web/src/stores/auth.ts`
- `clients/web/src/stores/notification.ts`
- `clients/web/src/stores/user.ts`
- `clients/web/src/stores/theme.ts`
- `clients/web/src/stores/locale.ts`

The real pattern is mixed:

- Pinia for shared app state
- composables for shared UI state
- local refs for page-specific state
- router and URL params for navigation state

---

## State Categories

### Global app state

Use Pinia when the state is shared across pages, app boot, or multiple unrelated components.

Current examples:

- auth session in `stores/auth.ts`
- notifications and polling in `stores/notification.ts`
- user favorites in `stores/user.ts`
- theme preference in `stores/theme.ts`
- locale preference in `stores/locale.ts`

### Shared UI state

Use a singleton composable when the state is cross-component but lighter than a full store.

Current examples:

- `clients/web/src/composables/useToast.ts`
- `clients/web/src/composables/useCommandPalette.ts`
- `clients/web/src/composables/useReviewPost.ts`

### Page-local state

Keep state local when it only belongs to one route or one view flow.

Examples:

- form fields in `clients/web/src/components/business/review/ReviewForm.vue`
- page fetch state in user-center tabs and many route views

Do not promote local page state to Pinia without a real sharing need.

---

## When to Use Global State

Promote state to Pinia when one or more of these are true:

- the app needs it during bootstrap
- more than one route depends on it
- multiple unrelated components read or mutate it
- it needs a clear reset lifecycle on logout or session loss
- it represents persistent user preference or session state

A strong reference example is `clients/web/src/stores/auth.ts`.

That store handles:

- bootstrap session recovery
- login and callback flow
- refresh behavior
- logout cleanup
- resetting other stores when the session is cleared

This is global state because the whole app depends on it.

---

## Server State

This project does not currently use a dedicated server-state library.

Current server-state strategy:

- fetch through the shared API layer
- keep many request results local to the page or composable
- use Pinia only when server-derived state needs to persist across screens or background behavior

Examples:

- `clients/web/src/stores/notification.ts` polls unread counts and owns cross-page notification state
- `clients/web/src/stores/auth.ts` bootstraps current user state from `/auth/me`
- route views often keep one-off lists and filters as local refs

Server state is therefore a mixed model, not a centralized cache framework.

---

## Store conventions already in use

### Use setup stores

Current stores use:

```ts
export const useAuthStore = defineStore("auth", () => {
  ...
})
```

Do not introduce Options-style Pinia stores without a clear reason.

### Add explicit `reset()` methods

Because setup stores do not get `$reset()` automatically, the repo uses explicit reset functions.

Examples:

- `clients/web/src/stores/auth.ts`
- `clients/web/src/stores/notification.ts`

### Clean up long-lived side effects

If a store owns polling or timers, it must also own stopping them.

Example:

- `clients/web/src/stores/notification.ts` uses `onScopeDispose(stopPolling)` and exposes `startPolling()` / `stopPolling()`

---

## Common Mistakes

### Common Mistake: putting one-screen state into a store too early

**Symptom**: a page-only list or form ends up in Pinia with no reuse benefit.

**Cause**: assuming "shared state tool" means all state must be global.

**Fix**: keep route-specific state local until another screen truly needs it.

---

### Common Mistake: forgetting logout reset paths

**Symptom**: stale favorites, notifications, or drafts remain visible after session loss.

**Cause**: a new store was added but not wired into auth cleanup.

**Fix**: update the auth reset path, following `resetAllStores()` in `clients/web/src/stores/auth.ts`.

---

### Common Mistake: adding polling without stop logic

**Symptom**: polling continues after navigation or repeated page entry.

**Cause**: timer ownership is unclear.

**Fix**: keep polling start/stop and cleanup in the store that owns the data.

---

### Common Mistake: bypassing the shared browser API client

**Symptom**: state updates behave differently around CSRF, cookies, or session refresh.

**Cause**: requests were added with raw `fetch` instead of `api.*`.

**Fix**: fetch through the shared client pipeline so state transitions stay consistent with auth behavior.

---

## Examples to follow

- `clients/web/src/stores/auth.ts` — app-wide auth lifecycle and reset orchestration
- `clients/web/src/stores/notification.ts` — polling state with cleanup and retry caps
- `clients/web/src/composables/useToast.ts` — shared UI state without a full store
- `clients/web/src/components/business/review/ReviewForm.vue` — local form state kept inside the component

---

## What is still evolving

The current state model is deliberately mixed:

- global store for session and preferences
- singleton composables for lightweight shared UI state
- local refs for many page screens

Document and follow that mixed model. Do not rewrite the docs as if the app already uses a single all-purpose state management strategy.
