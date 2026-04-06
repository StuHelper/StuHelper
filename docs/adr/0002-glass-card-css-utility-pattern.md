# ADR-0002: Glass-card as global CSS utility, not component wrapper

**Date**: 2026-03-31
**Status**: accepted
**Deciders**: Xauryan, Claude

## Context

The motion overhaul introduced a glassmorphism (frosted glass) design language across the application: review cards, course cards, skeleton loaders, CTA sections, search panels, and the login card all use `backdrop-filter: blur() saturate()` with semi-transparent backgrounds. We needed a consistent way to distribute this effect.

The project already had a `Card.vue` component (`components/ui/Card.vue`) with a scoped `.glass-card` class, and the design tokens in `tokens.ts` defined `glassEffects.card/navbar/modal` objects. However, the `Card.vue` scoped definition conflicted with the global one (different blur radius, hardcoded colors that broke in dark mode), and wrapping every element in `<Card>` added unnecessary component nesting.

## Decision

We use `.glass-card` as a global CSS utility class defined in `tailwind.css`, not as a component. Any element can apply the glass effect by adding `class="glass-card"`. The `Card.vue` component's scoped `.glass-card` override is removed; `Card.vue` only retains `.glass-navbar` and `.glass-modal` variant overrides.

## Alternatives Considered

### Alternative 1: `Card.vue` wrapper component for all glass surfaces
- **Pros**: Single source of truth, encapsulated styling, can add shared behavior (click handlers, aria roles)
- **Cons**: Adds a wrapper `<div>` to every glass element, increases component tree depth, forces refactoring of `<router-link>` and `<button>` elements that need the glass effect, scoped CSS specificity conflicts with global utilities
- **Why not**: We need glass on `<router-link>`, `<button>`, `<div>`, and `<section>` elements. Wrapping each in `<Card>` would add DOM bloat and fight with the existing Tailwind utility-first approach.

### Alternative 2: Tailwind plugin generating glass-* utilities from tokens
- **Pros**: Fully integrated into Tailwind's purge/JIT pipeline, configurable variants
- **Cons**: Tailwind v4 `@theme` doesn't support custom utility generation from arbitrary tokens, would require a custom plugin
- **Why not**: Unnecessary complexity. A plain CSS class in the global stylesheet achieves the same result with zero configuration.

## Consequences

### Positive
- Any HTML element can become a glass card with one class — no component wrapping needed
- Single definition in `tailwind.css` using design token CSS variables — automatically follows dark/light theme
- `hover:` state is built into the class (background densifies, border appears, shadow increases)
- `-webkit-backdrop-filter` prefix included once, not duplicated per component

### Negative
- No encapsulation: the class name `.glass-card` must remain stable as a public API
- If we need per-card behavior (e.g., click analytics, intersection tracking), we still need a component wrapper

### Risks
- Browser support: `backdrop-filter` is not available in Firefox < 103. The fallback is a slightly more opaque `background` color, which is acceptable degradation. No polyfill exists.
