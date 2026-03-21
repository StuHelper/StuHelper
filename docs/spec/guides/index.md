# Thinking Guides

> **Purpose**: Expand your thinking to catch things you might not have considered.

---

## Why Thinking Guides?

**Most bugs and tech debt come from "didn't think of that"**, not from lack of skill:

- Didn't think about what happens at layer boundaries → cross-layer bugs
- Didn't think about code patterns repeating → duplicated code everywhere
- Didn't think about edge cases → runtime errors
- Didn't think about future maintainers → unreadable code

These guides help you **ask the right questions before coding**.

---

## Start Here Every Session

Before implementation, read these project-level entry points:

1. `/Users/zxy/Code/StuHelper/.trellis/workflow.md`
2. `/Users/zxy/Code/StuHelper/.trellis/spec/guides/index.md`
3. `/Users/zxy/Code/StuHelper/.trellis/spec/backend/index.md`
4. `/Users/zxy/Code/StuHelper/.trellis/spec/frontend/index.md`
5. `/Users/zxy/Code/StuHelper/.trellis/workspace/index.md`
6. Your developer workspace index and active journal

This directory is the project-wide rule entry, not just a list of thinking prompts.

## Project-Wide Working Agreements

### Communication and language

- Use Chinese for user-facing collaboration and for explanatory code comments when comments are needed.
- Keep code identifiers, filenames, environment variables, and commit messages in English.

### Git and PR expectations

- `main` and `develop` are protected integration branches. Do not push directly.
- Prefer focused, single-responsibility PRs.
- Prefer squash merge unless a branch explicitly needs preserved commit history.
- Use Conventional Commits for commit messages.
- Do not mention AI vendor names in commit subjects.

### Engineering quality baseline

- Treat the project as pre-launch by default.
- Prefer the most thorough, enterprise-grade design instead of migration-minimizing patchwork.
- Do not commit secrets or hardcode configuration values.
- Do not silently ignore errors.
- Keep functions reasonably small; split large flows instead of accumulating 200+ line functions.
- When changing config, constants, contracts, or shared types, search the repo first for coupled references.

### Cross-layer naming conventions

| Layer | Convention | Example |
| --- | --- | --- |
| PostgreSQL | `snake_case` | `user_id`, `created_at` |
| Go struct fields | `PascalCase` | `UserID`, `CreatedAt` |
| Go JSON tags | `camelCase` | `userID`, `createdAt` |
| Vue / TypeScript | `camelCase` | `userID`, `createdAt` |
| HTTP request/response fields | `camelCase` | `userID`, `createdAt` |

Common initialisms stay uppercase in identifiers, such as `ID`, `URL`, `HTTP`, `API`, and `JSON`.

### Documentation and record-keeping

- Keep code, generated contracts, and docs in sync.
- Follow the `tutorials / guides / reference / architecture / modules` split under `docs/`.
- Record project-wide architecture, workflow, and convention changes in `workspace/<developer>/journal-1.md` under the `Project Archive` section.
- Record completed implementation sessions with `python3 ./.trellis/scripts/add_session.py`.

## Where Specific Rules Live

| Topic | Primary Location |
| --- | --- |
| Backend structure and workflow | `spec/backend/index.md` |
| Backend quality bar | `spec/backend/quality-guidelines.md` |
| Frontend structure and workflow | `spec/frontend/index.md` |
| Frontend quality bar | `spec/frontend/quality-guidelines.md` |
| Cross-layer thinking and reuse checks | `spec/guides/` |
| Developer journals | `workspace/<developer>/journal-N.md` |
| Project archive | `workspace/<developer>/journal-1.md` |

---

## Available Guides

| Guide | Purpose | When to Use |
|-------|---------|-------------|
| [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md) | Identify patterns and reduce duplication | When you notice repeated patterns |
| [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md) | Think through data flow across layers | Features spanning multiple layers |
| [Ecosystem Identity And Authorization](./ecosystem-identity-and-authorization.md) | Decide what belongs in Casdoor, what belongs in Hangxiaoban, and when to use relation-based auth | SSO, admin gating, app onboarding, verification-gated features, or resource-scoped moderation |
| [Manual Student Verification Flow](./manual-student-verification-flow.md) | Keep manual student verification aligned across spec, backend, DB, and frontend | When changing `POST /api/v1/user/profile/verify`, school config fields, or admin student verification review |
| [RBAC Role Permission Flow](./rbac-role-permission-flow.md) | Keep role permission assignment safe across spec, backend, and admin UI | When changing role permission APIs, permission dialogs, or clear-all behavior |

---

## Quick Reference: Thinking Triggers

### When to Think About Cross-Layer Issues

- [ ] Feature touches 3+ layers (API, Service, Component, Database)
- [ ] Data format changes between layers
- [ ] Multiple consumers need the same data
- [ ] You're not sure where to put some logic

→ Read [Cross-Layer Thinking Guide](./cross-layer-thinking-guide.md)

### When to Think About Code Reuse

- [ ] You're writing similar code to something that exists
- [ ] You see the same pattern repeated 3+ times
- [ ] You're adding a new field to multiple places
- [ ] **You're modifying any constant or config**
- [ ] **You're creating a new utility/helper function** ← Search first!

→ Read [Code Reuse Thinking Guide](./code-reuse-thinking-guide.md)

---

## Pre-Modification Rule (CRITICAL)

> **Before changing ANY value, ALWAYS search first!**

```bash
# Search for the value you're about to change
grep -r "value_to_change" .
```

This single habit prevents most "forgot to update X" bugs.

---

## How to Use This Directory

1. **Before coding**: Skim the relevant thinking guide
2. **During coding**: If something feels repetitive or complex, check the guides
3. **After bugs**: Add new insights to the relevant guide (learn from mistakes)

---

## Contributing

Found a new "didn't think of that" moment? Add it to the relevant guide.

---

**Core Principle**: 30 minutes of thinking saves 3 hours of debugging.
