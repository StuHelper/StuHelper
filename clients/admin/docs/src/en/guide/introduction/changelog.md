---
outline: deep
---

# Changelog

This page records notable changes that are already implemented in the current repository and affect how the admin app is integrated and maintained. It reflects the current codebase reality rather than a separate release stream.

## Authentication And Sessions

- The admin app now uses the shared API session model for login-state probing, refresh, and logout flows.
- Probe failures, forced re-login failures, and explicit logout failures keep their real error semantics instead of being disguised as "logged out" states.
- Regression coverage has been added around route guards and session initialization to reduce auth-chain regressions.

## Authorization And Page Access

- The access-control component now treats an empty permission set as "no restriction" so valid page content is not hidden by mistake.
- The admin app continues to consume shared capability constants and shared request primitives instead of duplicating authorization strings in page code.

## Contract And Data Alignment

- OpenAPI remains the single source of truth for backend contracts, and the admin app reuses generated types and request helpers from `clients/shared`.
- Shared pagination parsing now supports both `items` and `list` payloads to make list pages more resilient to contract evolution.
- Review governance, report handling, identity review, and student verification pages continue to be aligned with the implemented backend behavior.

## Documentation Maintenance

- This docs site now reflects implemented repository behavior instead of leaving blank placeholder pages.
- When implementation moves ahead of docs, code and tests win first, then the docs are updated to match.
