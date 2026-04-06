# ADR-0003: Mandatory three-guard safety pattern for JS animation composables

**Date**: 2026-03-31
**Status**: accepted
**Deciders**: Xauryan, Claude

## Context

During the motion overhaul, we created `use3DTilt.ts` and `useMagneticCursor.ts` — Vue 3 composables that register DOM event listeners (`mousemove`, `mouseenter`, `mouseleave`) and compute CSS transforms in JavaScript. Code review (3 parallel Opus agents) identified three categories of bugs that CSS-only animations avoid:

1. **prefers-reduced-motion ignored**: The global CSS rule `* { animation-duration: 0.01ms !important }` only affects CSS animations. JS-computed `transform` values still update on every event, causing elements to flicker rather than remain still.
2. **Touch device phantom activation**: `mousemove`/`mouseleave` fire inconsistently on touch devices (emulated after tap), causing effects to stick in non-zero states.
3. **Ref null during cleanup**: Vue 3 sets template refs to `null` before `onUnmounted` runs when the element is inside `v-if`, silently leaking event listeners.

These are not edge cases — they affect real users on mobile devices and those with vestibular disorders.

## Decision

All JS-driven animation composables in this project must implement a three-guard pattern:

1. **Guard 1 — Reduced motion**: Check `matchMedia('(prefers-reduced-motion: reduce)')` at setup time. If true, skip all listener registration and return a static identity style.
2. **Guard 2 — Pointer capability**: Check `matchMedia('(hover: hover)')` at setup time. If false (touch-only device), skip listener registration.
3. **Guard 3 — Captured DOM reference**: In `onMounted`, capture `elementRef.value` into a local `let mountedEl` variable. In `onUnmounted`, use `mountedEl` (not `elementRef.value`) for `removeEventListener`.

This pattern applies to: `use3DTilt`, `useMagneticCursor`, and any future composable that registers DOM listeners for animation (parallax, magnetic buttons, cursor effects, gyroscope tilt, etc.).

## Alternatives Considered

### Alternative 1: CSS-only animations everywhere (no JS composables)
- **Pros**: Automatically respects `prefers-reduced-motion` via the global CSS rule, no cleanup needed
- **Cons**: Cannot do mouse-position-tracking effects (3D tilt, magnetic cursor) in pure CSS
- **Why not**: Some effects fundamentally require JS (reading `clientX`/`clientY`, computing perspective transforms).

### Alternative 2: `@vueuse/core` `useEventListener` for automatic cleanup
- **Pros**: Auto-removes listeners on unmount, less boilerplate
- **Cons**: Does not solve reduced-motion or touch-device detection, still needs the ref-null guard because `useEventListener` also reads from the ref
- **Why not**: Solves only 1 of 3 problems. We still need the other two guards, so using `useEventListener` saves one line but doesn't eliminate the pattern.

## Consequences

### Positive
- Users with `prefers-reduced-motion` see no JS-driven motion — elements are static
- Touch-only devices have zero wasted event listeners
- No event listener leaks regardless of `v-if` / `KeepAlive` / rapid mount-unmount cycles
- Pattern is easy to audit: grep for `onMounted.*addEventListener` and verify all three guards are present

### Negative
- Slightly more boilerplate per composable (~8 extra lines)
- Reduced-motion and hover checks run once at setup — if the user changes OS settings mid-session, the composable won't react (acceptable trade-off; page reload picks it up)

### Risks
- Developers may forget the pattern when writing new composables. Mitigation: the learned skill at `~/.claude/skills/learned/vue3-animation-composable-safety.md` will remind Claude to enforce it during code review.
