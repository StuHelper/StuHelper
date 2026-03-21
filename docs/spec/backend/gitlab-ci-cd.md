# GitLab CI/CD

> Executable contract for repository CI/CD after the GitLab migration.

---

## Scenario: Repository CI/CD after migrating from legacy Gitea workflows

### 1. Scope / Trigger

- Trigger: you change repository-level CI/CD behavior, pipeline job names, deploy secrets, Docker image publishing, or build/test entry points.
- Trigger: you rename or move `.gitlab-ci.yml`, `.gitlab/server-ci.yml`, or `.gitlab/cd.yml`.
- Trigger: you change any script in `clients/package.json`, `clients/web/package.json`, `clients/admin/package.json`, or `server/Makefile` that a pipeline job invokes.

This is a cross-layer infra contract. Treat it as blocking documentation, not optional prose.

### 2. Signatures

Primary pipeline files:

- `/Users/zxy/Code/StuHelper/.gitlab-ci.yml`
- `/Users/zxy/Code/StuHelper/.gitlab/server-ci.yml`
- `/Users/zxy/Code/StuHelper/.gitlab/cd.yml`

Local verification commands:

```bash
cd /Users/zxy/Code/StuHelper/clients && pnpm lint
cd /Users/zxy/Code/StuHelper/clients && pnpm type-check
cd /Users/zxy/Code/StuHelper/clients && pnpm test:web
cd /Users/zxy/Code/StuHelper/server && make lint-spec
cd /Users/zxy/Code/StuHelper/server && make build
cd /Users/zxy/Code/StuHelper/server && make check-drift
```

Pipeline structure:

- `.gitlab-ci.yml`
  - `frontend_lint`
  - `frontend_typecheck`
  - `frontend_test`
  - `frontend_build`
  - `docker_build_backend`
  - `docker_build_frontend`
- `.gitlab/server-ci.yml`
  - `backend_lint`
  - `backend_security`
  - `openapi_contract`
  - `backend_test`
  - `backend_build`
- `.gitlab/cd.yml`
  - `package_backend`
  - `package_frontend`
  - `deploy_production`

### 3. Contracts

#### Workflow rules

`.gitlab-ci.yml` is the single entry point and must:

- run for merge requests
- run for `develop`
- run for `main`
- include `.gitlab/server-ci.yml`
- include `.gitlab/cd.yml`

#### Stage contract

The repository pipeline uses these stages in order:

1. `lint`
2. `test`
3. `build`
4. `package`
5. `deploy`

Do not invent a new stage without updating this spec and the root pipeline file together.

#### Frontend job contract

These jobs must stay aligned with the actual frontend toolchain:

- `frontend_lint`
  - command: `cd clients && pnpm lint`
- `frontend_typecheck`
  - command: `cd clients && pnpm type-check`
- `frontend_test`
  - command: `cd clients && pnpm test:web`
- `frontend_build`
  - command: `cd clients && pnpm --filter @stuhelper/web build && pnpm --filter @stuhelper/admin build`

Lint contract:

- `clients/eslint.config.mjs` is the workspace lint entry point
- `clients/web/package.json` and `clients/admin/package.json` must call `eslint src`
- do not rely on missing package-local `.eslintrc*` files

#### Backend job contract

These jobs must stay aligned with `server/Makefile`:

- `backend_lint`
  - command: `cd server && make lint`
- `backend_security`
  - command: `cd server && make security`
- `openapi_contract`
  - command: `cd server && make lint-spec && make check-drift`
- `backend_test`
  - command: `cd server && go test -v -race -timeout=10m -coverprofile=coverage.out ./...`
- `backend_build`
  - command: `cd server && make build`

Test service contract:

- PostgreSQL service alias: `postgres`
- Redis service alias: `redis`
- `DATABASE_URL=postgres://test:test@postgres:5432/test?sslmode=disable`
- `REDIS_HOST=redis`
- `REDIS_PORT=6379`
- `REDIS_PASSWORD=""`

#### Release/deploy contract

`package_backend` and `package_frontend` only run for `main` push pipelines.

Required environment variables:

- `REGISTRY`
- `REGISTRY_USERNAME`
- `REGISTRY_PASSWORD`
- `BACKEND_IMAGE`
- `FRONTEND_IMAGE`

Deploy job required environment variables:

- `DEPLOY_HOST`
- `DEPLOY_USER`
- `DEPLOY_SSH_KEY`
- `DEPLOY_APP_DIR`
- `DEPLOY_PORT` (optional, defaults to 22 in the shell command)

Image tagging contract:

- publish `:latest`
- publish `:${CI_COMMIT_SHORT_SHA}`

### 4. Validation & Error Matrix

| Area | Validation | Failure Symptom | Required Response |
| --- | --- | --- | --- |
| Frontend lint | `pnpm lint` | ESLint exits non-zero, often due to missing `clients/eslint.config.mjs` or parser/plugin mismatch | Fix lint config or scripts first; do not disable the job silently |
| Frontend types | `pnpm type-check` | `vue-tsc` errors | Fix types or generated contracts before merging |
| Frontend tests | `pnpm test:web` | Vitest failure | Treat as regression unless test was intentionally updated |
| OpenAPI contract | `make lint-spec` | Redocly validation error | Fix source spec, not generated files |
| Drift detection | `make check-drift` | generated Go/TS files changed | Run generation and commit the updated artifacts |
| Backend tests | `go test -race` | DB/Redis connection errors or test failures | Verify service aliases and env vars before touching tests |
| Registry push | `docker login` or `buildx build --push` fails | auth error or image push failure | Check registry variables and credentials |
| SSH deploy | `ssh` or remote `docker compose` fails | connection denied or remote script error | Fix deploy secrets or remote host state; do not mark deployment successful |

### 5. Good/Base/Bad Cases

#### Good

- You change `server/Makefile` and update the matching GitLab job command in `.gitlab/server-ci.yml`.
- You change a frontend lint script and verify it still works through `clients/eslint.config.mjs`.
- You add a new deploy variable and document it here plus the deployment guide in the same change.

#### Base

- You only change prose in deployment docs and do not alter any pipeline behavior.
- You only refactor job names inside GitLab YAML but keep stage order, commands, and rules identical.

#### Bad

- You update README to say "GitLab CI/CD" but leave `.gitea` workflow files as the real entry point.
- You add a new pipeline command but do not verify it locally with the matching repo command.
- You keep `pnpm lint` in CI while the workspace has no working ESLint 9 flat config.
- You change package scripts used by CI without updating the pipeline files and this spec together.

### 6. Tests Required

At minimum, after changing any of the pipeline files or the commands they call, run:

```bash
cd /Users/zxy/Code/StuHelper/clients && pnpm lint
cd /Users/zxy/Code/StuHelper/clients && pnpm type-check
cd /Users/zxy/Code/StuHelper/clients && pnpm test:web
cd /Users/zxy/Code/StuHelper/server && make lint-spec
cd /Users/zxy/Code/StuHelper/server && make build
cd /Users/zxy/Code/StuHelper/server && make check-drift
cd /Users/zxy/Code/StuHelper && git diff --check
```

Assertion points:

- `pnpm lint` proves the flat ESLint config exists and is discoverable from `web` and `admin`
- `make check-drift` proves OpenAPI generation and drift detection still align with CI
- `git diff --check` catches YAML/doc formatting regressions before commit

### 7. Wrong vs Correct

#### Wrong

- Move to GitLab in docs only
- Delete legacy workflow files and create unrelated replacements without preserving command parity
- Leave frontend lint enabled in CI while no valid ESLint config exists

#### Correct

- Migrate the existing workflow entry points into GitLab files, then adapt them in place
- Keep pipeline jobs anchored to real repo commands in `server/Makefile` and `clients/package.json`
- Treat CI/CD changes as executable contracts and update this spec when commands, env vars, or stage semantics move
