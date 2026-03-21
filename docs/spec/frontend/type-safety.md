# Type Safety

> Type safety patterns in this project.

---

## Use TypeScript strict mode and keep the shared package as the contract source

Both the web app and the shared package run with strict TypeScript settings.

Representative files:

- `clients/web/tsconfig.json`
- `clients/shared/tsconfig.json`

The current contract hierarchy is:

- OpenAPI-generated transport types in `clients/shared/src/types/api.gen.ts`
- business-facing shared types in `clients/shared/src/types/business/`
- thin web re-exports or adapters in `clients/web/src/types/`

Do not create a second hand-maintained source of truth inside `clients/web` if the type belongs in `clients/shared`.

---

## Type Organization

### Generated API types

Generated endpoint shapes come from OpenAPI.

Examples:

- `clients/shared/src/types/api.gen.ts`
- `clients/shared/src/api/client.ts`

The shared API client is typed like this:

```ts
return createClient<paths>({
	baseUrl: options.baseUrl,
	credentials: options.credentials,
	fetch: options.fetch,
});
```

That means endpoint request and response shapes should flow from generated types first.

### Shared business types

Business-facing types live in `clients/shared/src/types/business/`.

Example:

- `clients/shared/src/types/business/course.ts`

That file defines things like:

- `RatingValue = 1 | 2 | 3 | 4 | 5`
- `Term`
- `Course`
- `FavoriteCourse`

### Web-local types

Web-local files are usually thin wrappers, re-exports, or bridging adapters.

Examples:

- `clients/web/src/types/course.ts`
- `clients/web/src/types/review.ts`
- `clients/web/src/types/guards.ts`

Prefer re-exporting or adapting existing types over redefining them.

---

## Validation

There is no dominant runtime schema library like Zod in the current frontend.

The real validation style is a mix of:

- TypeScript types for static safety
- union types and const arrays for constrained values
- manual guards for runtime narrowing
- component-level checks for UI form constraints
- API-layer and backend validation for final authority

Examples:

- `clients/shared/src/types/business/course.ts` uses `RatingValue` and `isValidRating`
- `clients/web/src/types/guards.ts` defines runtime type guards like `isValidSortType`
- `clients/web/src/components/business/review/ReviewForm.vue` uses computed checks for title, content length, `termID`, and rating bounds

If runtime narrowing is needed in the browser, prefer a guard or centralized adapter over scattered type assertions.

---

## Common Patterns

### Prefer re-exports when the web app just needs the shared type

Example from `clients/web/src/types/course.ts`:

```ts
export type { Course, Term, FavoriteCourse } from "@stuhelper/shared";
```

This keeps one source of truth.

### Use narrow unions for constrained values

Example from `clients/shared/src/types/business/course.ts`:

```ts
export type RatingValue = 1 | 2 | 3 | 4 | 5;
```

Example from `clients/web/src/types/guards.ts`:

```ts
export const SORT_TYPES = ["time", "likes", "rating"] as const;
export type SortType = (typeof SORT_TYPES)[number];
```

### Centralize unavoidable adapter assertions

Sometimes generated transport types are looser than business types.

The project already centralizes one such bridge in `clients/web/src/types/review.ts`:

```ts
export function toReviews<T extends { ratings: Record<string, number> }>(
	apiReviews: T[],
): Review[] {
	return apiReviews as unknown as Review[];
}
```

This is better than scattering `as Review[]` throughout many views.

---

## Forbidden Patterns

### Do not use `any`

Project-level instructions already forbid it, and the current frontend code reflects that rule.

If you are tempted to add `any`, first try:

- a generated OpenAPI type
- a shared business type
- a local narrow interface
- a runtime guard
- an adapter function that isolates one boundary cast

### Do not duplicate shared contracts in web-local files

Wrong direction:

- defining a second copy of `Course`, `Review`, or API payload types in `clients/web`

Correct direction:

- re-export from `@stuhelper/shared`
- add a local adapter only when the web layer genuinely needs a stricter or UI-specific model

### Do not scatter unchecked assertions across pages

If one API shape needs bridging, centralize it in a dedicated adapter file like `clients/web/src/types/review.ts`.

---

## Wrong vs Correct

### Wrong

```ts
const reviews = response.data as any[];
return reviews as Review[];
```

### Correct

```ts
import { toReviews } from "@/types/review";

return toReviews(response.data);
```

One narrow adapter is easier to audit than many local assertions.

---

## Common Mistakes

### Common Mistake: redefining generated or shared types in the web app

**Symptom**: API and UI drift because similar interfaces exist in two places.

**Fix**: move the shared shape back into `clients/shared` or re-export the existing type.

---

### Common Mistake: widening constrained values to plain `string` or `number`

**Symptom**: invalid rating values or sort keys spread through the UI.

**Fix**: use union types and guards like the existing rating and sort helpers.

---

### Common Mistake: using broad assertions where a guard would be clearer

**Symptom**: future refactors silently break runtime assumptions.

**Fix**: use a centralized adapter or a type guard instead of repeated `as ...` casts.

---

## What is still evolving

A few type boundaries are still intentionally imperfect:

- generated OpenAPI types are sometimes looser than business-facing types
- web-local adapters still exist to bridge those gaps
- some route views still define view-only extension types for presentation needs

Document that honestly. The goal is not "zero local typing," but "shared contracts first, local adaptations only when justified."
