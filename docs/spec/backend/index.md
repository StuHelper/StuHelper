# Backend Development Guidelines

> Best practices for backend development in this project.

---

## Overview

This directory contains guidelines for backend development. Fill in each file with your project's specific conventions.

## Backend Entry Checklist

Read this index first for any backend task, then continue to the detailed docs that match your change:

1. `database-guidelines.md`
2. `error-handling.md`
3. `logging-guidelines.md`
4. `quality-guidelines.md`
5. `type-safety.md`
6. `openapi-tooling.md` for OpenAPI/spec-generation work
7. `authorization-architecture.md` for ecosystem identity, Hangxiaoban authorization, or open-platform access design

For cross-layer work, also read `../guides/index.md` and the relevant frontend index before implementation.

## Backend Rules That Always Apply

- Keep the architecture layered as `Handler -> Service -> Repository`.
- Keep SQL in repositories, HTTP concerns in handlers, and business rules in services.
- Use constructor-based dependency injection.
- Use `response.*` helpers for HTTP responses instead of ad hoc `c.JSON(...)`.
- Wrap errors with context instead of dropping them.
- Keep JSON fields in `camelCase`, database fields in `snake_case`, and Go fields in `PascalCase`.
- Respect the project-wide ban on hardcoded config values and silent error ignoring.

---

## Guidelines Index

| Guide                                           | Description                         | Status |
| ----------------------------------------------- | ----------------------------------- | ------ |
| [Directory Structure](./directory-structure.md) | Module organization and file layout | Filled |
| [Database Guidelines](./database-guidelines.md) | ORM patterns, queries, migrations   | Filled |
| [Error Handling](./error-handling.md)           | Error types, handling strategies    | Filled |
| [Type Safety](./type-safety.md)                 | OpenAPI-driven DTOs, optional-field semantics, and typed boundaries | Filled |
| [Authorization Architecture](./authorization-architecture.md) | Ecosystem identity boundary, Hangxiaoban authz, and open-platform access contracts | Filled |
| [GitLab CI/CD](./gitlab-ci-cd.md)               | Repo pipeline contracts and deploy rules | Filled |
| [Quality Guidelines](./quality-guidelines.md)   | Code standards, forbidden patterns  | Filled |
| [Logging Guidelines](./logging-guidelines.md)   | Structured logging, log levels      | Filled |
| [OpenAPI Tooling](./openapi-tooling.md)         | OpenAPI 3.1 authoring and codegen compatibility workflow | Filled |

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
