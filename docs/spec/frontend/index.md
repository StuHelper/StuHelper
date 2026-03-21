# Frontend Development Guidelines

> Best practices for frontend development in this project.

---

## Overview

This directory contains guidelines for frontend development. Fill in each file with your project's specific conventions.

## Frontend Entry Checklist

Read this index first for any frontend task, then continue to the detailed docs that match your change:

1. `component-guidelines.md`
2. `hook-guidelines.md`
3. `state-management.md`
4. `type-safety.md`
5. `quality-guidelines.md`

For cross-layer work, also read `../guides/index.md` and the relevant backend index before implementation.

## Frontend Rules That Always Apply

- Use `<script setup lang="ts">` for Vue SFCs.
- Keep components multi-word and file names in `PascalCase`.
- Use `camelCase` in props, emits, and TypeScript APIs; use `kebab-case` in templates where Vue expects it.
- Avoid `any` unless there is a strong, documented reason.
- Prefer local component/composable state unless the data is truly shared across screens.
- Keep browser transport, auth, and API behavior behind the shared client instead of ad hoc fetch calls.

---

## Guidelines Index

| Guide                                             | Description                             | Status |
| ------------------------------------------------- | --------------------------------------- | ------ |
| [Directory Structure](./directory-structure.md)   | Module organization and file layout     | Filled |
| [Component Guidelines](./component-guidelines.md) | Component patterns, props, composition  | Filled |
| [Hook Guidelines](./hook-guidelines.md)           | Custom hooks, data fetching patterns    | Filled |
| [State Management](./state-management.md)         | Local state, global state, server state | Filled |
| [Quality Guidelines](./quality-guidelines.md)     | Code standards, forbidden patterns      | Filled |
| [Type Safety](./type-safety.md)                   | Type patterns, validation               | Filled |

---

## How to Fill These Guidelines

For each guideline file:

1. Document your project's **actual conventions** (not ideals)
2. Include **code examples** from your codebase
3. List **forbidden patterns** and why
4. Add **common mistakes** your team has made

The goal is to help AI assistants and new team members understand how YOUR project works.

---

**Language**: All documentation should be written in **English**.
