# ADR-0004: Dual-selector rule for scoped dark mode CSS

**Date**: 2026-03-31
**Status**: accepted
**Deciders**: Xauryan, Claude

## Context

The StuHelper frontend uses a `data-theme` attribute on `<html>` for dark mode, managed by `useThemeStore`. Three states exist:

- `data-theme="dark"` — user explicitly selected dark
- `data-theme="light"` — user explicitly selected light
- No `data-theme` attribute — follow system preference

The global design system in `tailwind.css` handles all three cases with a dual selector:

```css
[data-theme="dark"],
:root:where(:not([data-theme="light"]):not([data-theme="dark"])) {
  --color-bg-base: #151020;
  /* ... */
}
```

However, scoped component styles (`<style scoped>`) that need dark overrides were inconsistent: some used `[data-theme="dark"]` only (missing the system-dark case), some used `@media (prefers-color-scheme: dark)` (wrong — applies dark when user chose light but OS is dark), and some used Tailwind's `dark:` variant (which maps to `[data-theme="dark"]` via `@custom-variant` but also misses system-dark).

This caused bugs: `FeatureCard.vue` showed dark colors when the app was in light mode (because OS was dark), and `HeroSection.vue` showed a light gradient when the system was dark but no explicit theme was set.

## Decision

When scoped CSS must override styles for dark mode (i.e., when CSS variables alone are insufficient), always use the dual selector:

```css
[data-theme="dark"] .my-element,
:root:where(:not([data-theme="light"]):not([data-theme="dark"])) .my-element {
  /* dark overrides */
}
```

The preferred approach is to avoid scoped dark overrides entirely by using design token CSS variables (`var(--color-text-primary)`, `var(--color-bg-card)`, etc.), which automatically follow the theme. The dual selector is the fallback for cases where variables are insufficient (e.g., `background: linear-gradient(...)` with hardcoded stops).

Rules:
- **Never** use `@media (prefers-color-scheme: dark)` in scoped component styles
- **Never** use hardcoded color values (`#1a1a1a`, `rgba(30,30,30,0.7)`) — use `var(--color-*)` tokens
- **Prefer** CSS variables over scoped dark selectors (zero dark-specific code needed)
- **If** a scoped dark selector is unavoidable, always include both `[data-theme="dark"]` and the `:root:where(...)` fallback

## Alternatives Considered

### Alternative 1: Use only `[data-theme="dark"]` selector
- **Pros**: Simpler, matches the explicit dark choice
- **Cons**: Misses the "system dark, no explicit theme" case entirely
- **Why not**: Users who haven't explicitly chosen a theme but have a dark OS will see light-mode component styling against a dark-mode page. This is a visible regression.

### Alternative 2: Use `@media (prefers-color-scheme: dark)`
- **Pros**: Automatically follows OS preference
- **Cons**: Ignores the user's explicit theme choice. If user selects "light" in the app but OS is dark, the component shows dark styles anyway.
- **Why not**: The app's theme toggle becomes non-functional for any component using this approach.

### Alternative 3: Extend Tailwind's `@custom-variant dark` to include the system fallback
- **Pros**: `dark:` classes would automatically cover both cases, zero extra work per component
- **Cons**: Tailwind v4's `@custom-variant` syntax only accepts a single selector pattern, not a comma-separated list. Would require a workaround or plugin.
- **Why not**: Tailwind v4 limitation. The current `@custom-variant dark (&:where([data-theme="dark"], [data-theme="dark"] *))` cannot be extended to include the system-dark fallback without a custom plugin. Acceptable to use the manual dual selector for the rare cases where scoped dark overrides are needed.

## Consequences

### Positive
- Dark mode works correctly in all three states (explicit dark, explicit light, system preference)
- Design token variables handle 95% of cases with zero dark-specific code
- Consistent rule across all components — easy to audit with `grep`

### Negative
- The dual selector is verbose and easy to forget when writing new scoped styles
- Cannot be enforced at build time (no linter rule exists for this pattern)

### Risks
- Developers may add `@media (prefers-color-scheme: dark)` or single-selector `[data-theme="dark"]` out of habit. Mitigation: code review agents are instructed to flag this pattern.
