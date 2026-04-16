# ADR-0005: UniApp X Shadow File (.js/.ts pair) Cleanup

**Date**: 2026-04-16
**Status**: accepted
**Deciders**: Xauryan

## Context

`clients/uniappx/src/` contained 15 `.js` files co-existing with identically named `.ts` source files. These shadow files were transpiled/compiled outputs that had been accidentally committed. Their presence created:

- Runtime ambiguity: bundlers may resolve the `.js` file instead of the `.ts` source
- Maintenance drift: edits to `.ts` files were not reflected in `.js` copies
- Developer confusion: which file is authoritative?

## Decision

Delete all `.js` shadow files, enforce `.ts`-only via `allowJs: false` in tsconfig, and add a CI gate to prevent recurrence.

## Shadow File Inventory and Comparison

All 15 pairs were audited via `diff`. Results:

| File (relative to `src/`) | Verdict | Diff Summary |
|---|---|---|
| `main.ts` / `main.js` | **Pure copy** — delete `.js` | Semicolons, whitespace only |
| `config/pagination.ts` / `.js` | **Pure copy** — delete `.js` | Trailing semicolons only |
| `config/featureSurface.ts` / `.js` | **Semantic fork** — `.ts` is authoritative | `.js` had `getHomeFeatures()` inlined; `.ts` has typed `UniappxFeatureLink` interface + same logic |
| `config/__tests__/featureSurface.test.ts` / `.js` | **Pure copy** — delete `.js` | Type annotations removed |
| `stores/auth.ts` / `.js` | **Semantic fork** — `.ts` is authoritative | `.ts` has full typed store (NativeTokens, BOOTSTRAP_STALE_MS, etc.); `.js` was older partial transpile |
| `utils/format.ts` / `.js` | **Pure copy** — delete `.js` | Type annotations removed, same logic |
| `api/result.ts` / `.js` | **Semantic fork** — `.ts` is authoritative | `.ts` has `StructuredApiError`/`ApiEnvelope` types + `parseApiError` from shared; `.js` had inline reimplementation |
| `api/shared-client.ts` / `.js` | **Semantic fork** — `.ts` is authoritative | `.ts` has typed `ApiClient`, CSRF handling, native token refresh; `.js` was older transpile |
| `api/__tests__/shared-client.test.ts` / `.js` | **Pure copy** — delete `.js` | Type annotations removed |
| `api/__tests__/result.test.ts` / `.js` | **Pure copy** — delete `.js` | Type annotations removed |
| `api/index.ts` / `.js` | **Pure copy** — delete `.js` | Type annotations and `export type` removed |
| `i18n/zh-CN.ts` / `.js` | **Content drift** — `.ts` is authoritative | `.ts` has 207 keys; `.js` had 187 (missing 20 newer keys) |
| `i18n/en-US.ts` / `.js` | **Content drift** — `.ts` is authoritative | Same: `.ts` 207 keys, `.js` 187 |
| `i18n/index.ts` / `.js` | **Semantic fork** — `.ts` is authoritative | `.ts` has typed `SupportedLocale`, `TranslationParams`, uni runtime abstraction |
| `i18n/__tests__/index.test.ts` / `.js` | **Pure copy** — delete `.js` | Type annotations removed |

**Summary**: 9 pure copies, 4 semantic forks (`.ts` is the superset in all cases), 2 content drifts (`.ts` has more keys). All 15 `.js` files deleted; zero behavioral regression expected because TypeScript sources were already the active code path.

## Consequences

### Positive
- Single source of truth for all runtime code
- `allowJs: false` prevents future accidental `.js` in `src/`
- CI gate (`check_shadow_files` job) blocks merge if pairs reappear

### Negative
- None identified — `.js` files were not imported anywhere directly

### Risks
- If a future contributor needs raw `.js` in `src/` for a uni-app native plugin, they must explicitly opt in by updating tsconfig and the CI script. This is intentional friction.
