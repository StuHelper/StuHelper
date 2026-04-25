---
type: adr
audience: frontend-dev
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# ADR-0001: Self-built ScrollReveal over GSAP ScrollTrigger

**Date**: 2026-03-31
**Status**: accepted
**Deciders**: Xauryan, Claude

## Context

The frontend motion overhaul (Plan B) required scroll-triggered entrance animations for page sections, feature cards, and stat counters. GSAP is already a project dependency (used by `ParticleBackground.vue` for particle tweening), and GSAP ScrollTrigger is the industry-standard plugin for scroll-driven animations. However, ScrollTrigger is a separate 15KB+ plugin that introduces a global scroll listener, pin/scrub mechanics, and a complex lifecycle that must be carefully cleaned up in SPA route transitions.

Our actual need is simple: fade/slide elements in when they enter the viewport, with configurable direction, delay, and threshold. No pinning, no scrub, no timeline-driven scroll sequences.

## Decision

We build a lightweight `ScrollReveal.vue` component using the native `IntersectionObserver` API. It accepts `direction`, `delay`, `duration`, `distance`, `threshold`, and `once` props, and applies inline CSS transitions when the element intersects the viewport.

## Alternatives Considered

### Alternative 1: GSAP ScrollTrigger plugin
- **Pros**: Battle-tested, supports pinning/scrubbing/parallax, large community
- **Cons**: 15KB+ additional bundle, global scroll listener conflicts with Vue router, requires manual `ScrollTrigger.kill()` on route change, complex lifecycle management in SPA
- **Why not**: Overkill for our use case (simple viewport-triggered fade-ins). The plugin's power comes with complexity we don't need, and improper cleanup in SPAs causes memory leaks and stale scroll bindings.

### Alternative 2: @vueuse/motion `v-motion` directive with `visibleOnce`
- **Pros**: Already installed, Vue-native, declarative
- **Cons**: Only used in one component (`FadeIn.vue`), the `visibleOnce` feature requires additional configuration per element, no built-in stagger support, harder to apply uniformly across page sections
- **Why not**: Viable but less ergonomic for our pattern of wrapping entire page sections. ScrollReveal as a wrapper component with slot is cleaner for our layout composition style.

## Consequences

### Positive
- Zero additional bundle cost (IntersectionObserver is native)
- Simple component API matches our composition pattern (`<ScrollReveal :delay="100"><Content /></ScrollReveal>`)
- No global scroll listener; each instance manages its own observer
- Built-in `prefers-reduced-motion` support (content visible immediately, no observer)
- Easy to remove or replace later without changing consuming components

### Negative
- Cannot do scroll-pinning or scrub-driven parallax effects (if needed later, would require GSAP ScrollTrigger or CSS `animation-timeline`)
- Each instance creates its own IntersectionObserver (not shared), which is fine for <50 instances but would need a shared observer pattern at scale

### Risks
- If we later need advanced scroll effects (parallax heroes, scroll-linked progress bars), we would need to either adopt ScrollTrigger or use the emerging CSS `scroll-timeline` / `view-timeline` APIs. This ADR does not preclude that.
