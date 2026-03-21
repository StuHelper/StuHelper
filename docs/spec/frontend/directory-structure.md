# Directory Structure

> How frontend code is organized in this project.

---

## Organize the web app by runtime role and feature boundary

The current frontend is a Vue 3 app under `clients/web`, with shared cross-client code under `clients/shared`.

Start from these boundaries:

- `clients/web/src/main.ts` boots the app, Pinia, and i18n.
- `clients/web/src/router/index.ts` is the route map and route-meta contract.
- `clients/web/src/modules/<domain>/views/` holds page-level route targets.
- `clients/web/src/components/` holds reusable UI, layout, common, and business components.
- `clients/web/src/stores/` holds Pinia stores for cross-page state.
- `clients/web/src/composables/` holds reusable stateful logic.
- `clients/shared/src/` holds API wrappers, generated types, business types, constants, and shared utilities.

Do not describe the frontend as a flat `views/components/utils` app. It is already a mixed module-oriented structure with a separate shared package.

---

## Directory Layout

```text
clients/
├── shared/
│   └── src/
│       ├── api/
│       ├── constants/
│       ├── types/
│       ├── types/business/
│       └── utils/
└── web/
    └── src/
        ├── api/
        ├── components/
        │   ├── animated/
        │   ├── business/
        │   ├── common/
        │   ├── layout/
        │   └── ui/
        ├── composables/
        ├── constants/
        ├── i18n/
        ├── modules/
        │   ├── admin/
        │   ├── auth/
        │   ├── course/
        │   ├── errors/
        │   ├── home/
        │   ├── review/
        │   └── user/
        ├── router/
        ├── stores/
        ├── styles/
        ├── types/
        └── utils/
```

Representative files:

- `clients/web/src/router/index.ts`
- `clients/web/src/main.ts`
- `clients/shared/src/index.ts`

---

## Put route views in `modules/<domain>/views`

Current route targets are page-oriented and live under module directories.

Examples from `clients/web/src/router/index.ts`:

- `@/modules/auth/views/LoginPage.vue`
- `@/modules/course/views/CourseListPage.vue`
- `@/modules/review/views/CourseDetailPage.vue`
- `@/modules/admin/views/DashboardPage.vue`
- `@/modules/errors/views/NotFoundPage.vue`

Use `modules/<domain>/views` for route-level components.

This is the clearest current pattern in the web app.

---

## Keep reusable components in `components/`, grouped by reuse level

The project does not put every component under a feature folder.

Instead, it uses reuse-oriented buckets:

- `components/ui/` for generic controls like `SearchBar.vue`, `Pagination.vue`, `Loading.vue`
- `components/common/` for app-wide components such as overlays, error boundaries, and shared UX pieces
- `components/layout/` for shells and navigation wrappers
- `components/business/<domain>/` for reusable domain-specific pieces, especially review flows

Representative examples:

- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/layout/AppShell.vue`
- `clients/web/src/components/business/review/ReviewForm.vue`

Do not force a fake rule like "all feature components must live inside `modules/<feature>/components`". That is not the current reality.

---

## Put shared runtime logic where it is already expected

Current placement rules in the repo:

- router behavior lives in `src/router/`
- composables live in `src/composables/`
- Pinia stores live in `src/stores/`
- web-only type adapters and guards live in `src/types/`
- app bootstrap styles live in `src/styles/`
- API client behavior lives in `src/api/` for web-specific transport concerns and `shared/src/api/` for shared endpoint wrappers

Examples:

- `clients/web/src/api/client.ts`
- `clients/web/src/composables/useAsyncData.ts`
- `clients/web/src/stores/auth.ts`
- `clients/web/src/types/review.ts`

---

## Use `clients/shared` as the cross-client source of truth

Shared code already has a clear role:

- generated API types in `clients/shared/src/types/api.gen.ts`
- business types in `clients/shared/src/types/business/`
- endpoint wrappers in `clients/shared/src/api/`

Web-local type files often re-export from `@stuhelper/shared` instead of redefining the contract.

Examples:

- `clients/web/src/types/course.ts`
- `clients/web/src/types/review.ts`
- `clients/shared/src/api/courses.ts`

If a type or API wrapper is meant to be reused across clients, put it in `clients/shared`, not `clients/web`.

---

## Naming Conventions

Observed frontend naming patterns:

- Vue components use PascalCase file names: `SearchBar.vue`, `ReviewForm.vue`, `AppShell.vue`
- composables use `useXxx.ts`: `useAsyncData.ts`, `useToast.ts`
- stores use domain-focused names: `auth.ts`, `notification.ts`
- module folders use lowercase names: `auth`, `course`, `review`, `admin`
- local type files use short domain names: `course.ts`, `review.ts`, `guards.ts`

Representative files:

- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/composables/useToast.ts`
- `clients/web/src/stores/auth.ts`

---

## Examples to follow

Use these files as structure references:

- `clients/web/src/router/index.ts` — route-to-module organization
- `clients/web/src/main.ts` — application bootstrap order
- `clients/web/src/components/business/review/ReviewForm.vue` — reusable domain component outside route views
- `clients/web/src/types/course.ts` — thin re-export from shared types
- `clients/shared/src/api/courses.ts` — shared endpoint wrapper placement

---

## Avoid these structural mistakes

- Do not put route pages back into one flat `views/` directory.
- Do not redefine shared API contracts inside `clients/web` if the type belongs in `clients/shared`.
- Do not move one-off page logic into `components/` just because it uses JSX-like template complexity.
- Do not assume every feature owns a complete self-contained folder; this app currently uses a hybrid module + central component library structure.
- Do not hand-edit generated type files in `clients/shared/src/types/api.gen.ts`.

---

## What is still evolving

The structure is already usable, but not fully uniform:

- modules own route views, but many business components still live in centralized `components/business/`
- some web-local type adapters still bridge generated types to stricter business types
- Storybook and some style entry points still reflect older layout decisions

Document the hybrid structure as it exists today rather than forcing a cleaner pattern that the codebase has not actually adopted yet.
