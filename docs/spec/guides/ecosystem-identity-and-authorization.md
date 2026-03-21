# Ecosystem Identity And Authorization Guide

> Thinking checklist for deciding whether a rule belongs in the identity platform, the application, or a dedicated authorization engine.

---

## Start Here

Use this guide when a change touches any of these:

- `sso.stuhelper.com`
- Hangxiaoban admin gating
- app onboarding for first-party or third-party applications
- scopes, claims, or userinfo payloads
- course/category/content-level moderation rules
- student / teacher / school / verification dependent features

Then read:

1. `../backend/authorization-architecture.md`
2. `../backend/openapi-tooling.md` if contracts or claims change
3. the relevant module docs under `docs/modules/auth/` and `docs/modules/policy/`

## 1. Decide Which Plane Owns The Rule

### Put it in Casdoor if the rule is about:

- login
- logout
- token issuance
- OAuth/OIDC application onboarding
- user consent
- app scopes
- platform-level administrator identity
- coarse app access such as "can sign in to Hangxiaoban"

### Put it in Hangxiaoban if the rule is about:

- course-level admin
- category-level resource management
- content ownership
- teacher-of-course defaults
- verification-dependent visibility or publishing
- school-specific feature access
- warning labels or partial rendering

### Escalate to a relation engine if the rule needs:

- many-to-many delegation
- resource-specific admins at scale
- per-course or per-category editors
- composable ownership and inheritance
- auditable relation tuples instead of one-off condition branches

## 2. Do Not Use Identity Flags As Business Admin Gates

Check for these anti-patterns:

- frontend route guards based on `isAdmin`
- backend `/admin` groups gated only by `isAdmin`
- app menus rendered from platform-admin state instead of app capabilities

If you see any of these, stop and move the decision back into app authorization.

## 3. Classify The Authorization Rule Correctly

### RBAC

Use for:

- Hangxiaoban full admin
- review module admin
- resource module admin

### ReBAC

Use for:

- course manager
- course intro editor
- course resource editor
- review moderator for one course
- resource-category manager
- content owner
- teacher of course

### ABAC

Use for:

- `schoolID == 10006`
- actor type is `student`
- student verification completed
- identity verification completed
- visibility / publish rules based on multiple facts

If a feature needs all three, do not force it into a single flat role table.

## 4. Third-Party App Data Rules

Before exposing identity or verification data to external apps, check:

- Is there an explicit OAuth scope?
- Is the returned data the minimum necessary?
- Could the same use case work with a boolean or status field instead of raw PII?
- Does the app really need the data at login time, or can it call a scoped profile API later?

Default answer:

- expose statuses and coarse identity facts
- do not expose names, student IDs, phone numbers, or raw identity documents

## 5. Platform Admin vs App Admin

Use these meanings consistently:

- Platform admin: manages `sso.stuhelper.com`, app registration, ecosystem operations
- App admin: manages a single first-party application's business behavior
- Resource admin: manages only a module, course, category, or content subset

Do not collapse these into one `admin` concept.

## 6. Review Checklist

- [ ] Did we treat Casdoor as the identity plane rather than the Hangxiaoban business-auth source?
- [ ] Did we avoid using `isAdmin` as the business-admin gate?
- [ ] Did we model course/category/content delegation as relations instead of fake global roles?
- [ ] Did we keep school / verification / actor-type checks in the application fact layer?
- [ ] If external apps receive user facts, are scopes and field minimization explicit?
- [ ] If the rule is high-cardinality or many-to-many, did we consider OpenFGA / SpiceDB?

## 7. Common Wrong Turns

### Wrong

- adding a new global Casdoor role for every course moderator
- putting course ownership into SSO groups
- solving partial visibility with frontend-only checks
- assuming platform admin should automatically read third-party app business data

### Better

- keep identity and app authorization separate
- keep business facts in the application domain
- use relation checks for high-cardinality moderation and ownership rules
- use scopes and minimal claims for open-platform data sharing
