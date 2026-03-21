# Component Guidelines

> How components are built in this project.

---

## Use Vue 3 `script setup` with TypeScript

The current web app is consistently built with Vue 3 SFCs using `<script setup lang="ts">`.

Representative files:

- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/business/review/ReviewForm.vue`
- `clients/web/src/App.vue`

Do not introduce Options API components in new code unless there is a very specific compatibility reason.

---

## Component Structure

The most common component structure in this repo is:

1. typed imports
2. `defineProps` / `withDefaults`
3. `defineEmits`
4. refs and computed state
5. small local functions
6. template
7. scoped style only when utility classes are not enough

Examples:

- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/business/review/ReviewForm.vue`

This is not a rigid formatting rule, but it reflects how the strongest current components are written.

---

## Props Conventions

Props are typed explicitly and usually declared near the top of the component.

Two common patterns in the repo:

### `withDefaults` for reusable UI controls

```ts
interface Props {
	modelValue: string;
	placeholder?: string;
	debounce?: number;
}

const props = withDefaults(defineProps<Props>(), {
	placeholder: undefined,
	debounce: 300,
});
```

Example:

- `clients/web/src/components/ui/SearchBar.vue`

### inline generic props for focused business components

```ts
const props = defineProps<{
	courseID: number;
}>();
```

Example:

- `clients/web/src/components/business/review/ReviewForm.vue`

Prefer narrow, explicit prop contracts. Do not pass large loosely typed blobs when the component only needs a few fields.

---

## Emits Conventions

Emits are typed as well.

Examples:

```ts
const emit = defineEmits<{
	"update:modelValue": [value: string];
}>();
```

```ts
const emit = defineEmits<{ posted: [] }>();
```

Representative files:

- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/business/review/ReviewForm.vue`

Do not use untyped string-only emits when the payload can be expressed clearly.

---

## Styling Patterns

Tailwind utility classes are the primary styling surface.

Current behavior in the repo:

- most visual styling is inline in the template
- `scoped` CSS is used only for browser quirks, pseudo-elements, keyframes, shimmer effects, or styles that are awkward in utilities
- component-local styles are scoped when present

Representative examples:

- `clients/web/src/components/ui/SearchBar.vue` — utility-first layout plus a tiny scoped browser fix
- `clients/web/src/components/ui/Loading.vue` — scoped shimmer animation
- `clients/web/src/components/business/review/ReviewForm.vue` — utility-first styling only

### Good pattern

Use utilities for normal state, spacing, color, and layout.

### Good exception

Use scoped CSS when the browser feature or animation is awkward to express in utilities.

Do not create large bespoke CSS blocks for ordinary component layout.

---

## Accessibility

Accessibility is already visible in the stronger reusable components. Follow those patterns.

Common examples in the repo:

- use `aria-label` for icon-only or ambiguous controls
- use semantic navigation roles where appropriate
- use `aria-current` for pagination
- use `role="status"` and `aria-busy="true"` for loading placeholders
- connect validation text to fields with `aria-describedby`

Representative files:

- `clients/web/src/components/ui/Pagination.vue`
- `clients/web/src/components/ui/Loading.vue`
- `clients/web/src/components/ui/SearchBar.vue`
- `clients/web/src/components/business/review/ReviewForm.vue`

If a reusable component is interactive, assume accessibility must be part of the component contract, not a later cleanup task.

---

## Common Mistakes

### Common Mistake: rebuilding shared controls inside pages

**Symptom**: search bars, pagination, or loaders get duplicated in route views.

**Cause**: page work starts quickly and skips the existing `components/ui` layer.

**Fix**: check `components/ui/`, `components/common/`, and `components/business/` before writing a new control.

**Prevention**: if the behavior will appear in more than one view, make it reusable immediately.

---

### Common Mistake: mixing business contracts into weakly typed props

**Symptom**: a component accepts a wide `any`-like object or many optional fields it does not actually need.

**Cause**: skipping explicit props and leaning on a caller-side object shape.

**Fix**: narrow the props to what the component actually consumes.

---

### Common Mistake: overusing CSS when utilities already solve it

**Symptom**: component files collect large style blocks for ordinary layout and spacing.

**Cause**: writing CSS out of habit instead of following the utility-first approach already used by the app.

**Fix**: move normal layout and color decisions back into Tailwind classes.

Keep scoped CSS for the smaller set of things it is already used for well.

---

### Common Mistake: assuming i18n coverage is already complete everywhere

Many shared and polished components use `useI18n()`, but not every page string in the app is fully internationalized yet.

For new reusable components, prefer the i18n-aware pattern already used in `SearchBar.vue`, `Pagination.vue`, and `ReviewForm.vue`.

---

## Examples to follow

- `clients/web/src/components/ui/SearchBar.vue` — typed props/emits, debounce behavior, a11y, tiny scoped CSS
- `clients/web/src/components/ui/Pagination.vue` — computed pagination model, semantic navigation, typed emits
- `clients/web/src/components/ui/Loading.vue` — accessible loading UI with minimal scoped animation CSS
- `clients/web/src/components/business/review/ReviewForm.vue` — business form component with typed props, validation state, and API integration
