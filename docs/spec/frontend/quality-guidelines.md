# Quality Guidelines

> Code quality standards for frontend development.

---

## Follow the actual frontend toolchain

The current frontend quality bar is encoded in the repo scripts and config files.

Primary references:

- `clients/package.json`
- `clients/eslint.config.mjs`
- `clients/web/package.json`
- `clients/web/vite.config.ts`
- `clients/web/vitest.config.ts`
- `clients/web/tsconfig.json`

Common commands:

```bash
cd clients
pnpm run lint
pnpm run type-check
pnpm run test:web
```

For web-only work, these are also common:

```bash
cd clients/web
pnpm run lint
pnpm run type-check
pnpm run test
pnpm run build
```

The shared lint entry now lives in `clients/eslint.config.mjs`.

- `clients/web/package.json` and `clients/admin/package.json` both rely on that root flat config
- If linting breaks after an ESLint major upgrade, verify the flat-config entry point before changing scripts or CI jobs
- Do not add package-local `.eslintrc*` files that diverge from the workspace root unless the repo explicitly adopts a multi-config strategy

Do not claim frontend work is done if it only "looks right" in the browser but has not passed the project toolchain.

---

## Forbidden Patterns

### Do not bypass the shared API client

The browser request contract is centralized in `clients/web/src/api/client.ts`.

That file already handles:

- cookie credentials
- CSRF header injection
- refresh flow
- normalized API errors
- auth session clearing on terminal 401s

Avoid direct `window.fetch(...)` in page code when an `api.*` wrapper should be used.

### Do not hand-edit generated API types

Generated types belong to the OpenAPI workflow.

Relevant files:

- `clients/shared/src/types/api.gen.ts`
- `clients/shared/src/api/client.ts`
- `server/api/openapi.yaml`

If the API contract changes, regenerate instead of patching generated files by hand.

### Do not use `any`

Project instructions forbid it, and the current web app is already largely aligned with that rule.

### Do not bypass SFC conventions without a strong reason

Default to `<script setup lang="ts">`, multi-word components, and explicit typing for props and emits.

### Do not replace typed reusable controls with weakly typed page-local copies

Before building a new search, pagination, loading, or shell component, check `components/ui`, `components/common`, and `components/layout` first.

---

## Required Patterns

### Use strict typing and type checks

`clients/web/tsconfig.json` enables:

- `strict`
- `noUnusedLocals`
- `noUnusedParameters`
- `noFallthroughCasesInSwitch`

Treat `pnpm run type-check` as a real gate, not optional cleanup.

### Prefer scoped component styles unless a style is intentionally global

Most component-local styles should stay in `<style scoped>`. Use global theme/layout files only when the style truly belongs to the app shell.

### Use the shared package for cross-client contracts

If a type or endpoint wrapper can be shared, keep it in `@stuhelper/shared`.

Examples:

- `clients/shared/src/api/courses.ts`
- `clients/web/src/types/course.ts`
- `clients/web/src/types/review.ts`

### Keep browser auth and transport behavior centralized

The project already has a strong browser API boundary. New frontend data flows should go through that path.

### Keep route pages aligned with router meta and shell behavior

The router and app shell contract are real runtime dependencies.

Representative files:

- `clients/web/src/router/index.ts`
- `clients/web/src/App.vue`
- `clients/web/src/main.ts`

New pages should fit the existing route-meta, auth-guard, and shell layout behavior.

---

## Testing Requirements

### Minimum expectation

For meaningful frontend changes, aim to keep these green:

- lint
- type-check
- relevant Vitest coverage
- build

### Current test surfaces in the repo

- unit and regression tests with Vitest
- coverage support via V8
- one basic Playwright smoke test
- Storybook with the a11y addon enabled

Representative files:

- `clients/web/vitest.config.ts`
- `clients/web/tests/e2e/home.spec.ts`
- `clients/web/.storybook/main.ts`

### Be honest about current coverage depth

The tooling exists, but coverage is uneven:

- Vitest has real usage and is the main automated regression layer
- Playwright coverage is currently sparse
- Storybook exists, but story coverage is also sparse

Do not document a stronger test matrix than the repo actually has today.

---

## Accessibility expectations

Accessibility is not fully standardized everywhere, but reusable components already set a clear bar.

Look for these patterns:

- `aria-label`
- `aria-current`
- `role="status"`
- `aria-busy`
- semantic navigation roles
- keyboard-safe button behavior

Representative files:

- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/ui/Loading.vue`
- `clients/web/src/components/common/CommandPalette.vue`

For new reusable components, follow the stronger existing a11y patterns rather than the weakest page-level examples.

---

## Code Review Checklist

Review frontend changes for these points:

- Does the change use `api.*` instead of ad hoc browser fetch code?
- Are shared contracts coming from `@stuhelper/shared` where appropriate?
- Are props, emits, and state typed explicitly?
- Does the route integrate with existing router meta, auth, and layout behavior?
- Are interactive components accessible enough to match existing reusable controls?
- Is new global state really global, or should it stay local/composable-scoped?
- If API contracts changed, was the OpenAPI generation flow updated?
- Did lint, type-check, and relevant tests run successfully?

---

## Common frontend gotchas worth preserving

### Session bootstrap order matters

`clients/web/src/main.ts` restores auth state before installing the router:

```ts
const authStore = useAuthStore(pinia);
await authStore.bootstrapSession();
app.use(router);
```

Do not reorder this casually. The current app depends on that startup contract.

### Router error handling is defensive for chunk failures

`clients/web/src/router/index.ts` includes special chunk-load fallback behavior.

Avoid route-loading changes that break this protection during deployments.

### Browser security behavior already lives in the API client

CSRF, cookie credentials, refresh retries, and normalized API errors are all centralized in `clients/web/src/api/client.ts`.

New code should plug into that existing path, not reimplement it inconsistently.

---

## What is still evolving

A few quality layers are present but still thin:

- Storybook usage is real, but limited
- Playwright is present, but coverage is currently minimal
- not every screen has equally polished i18n or accessibility treatment

Document that truthfully. The goal of this spec is to help future work match the repo as it exists today, while nudging new code toward the stronger patterns that are already visible.
