# OpenAPI Tooling

> Source-of-truth rules for keeping StuHelper on OpenAPI 3.1 while still generating Go code with `oapi-codegen`.

---

## Scenario: OpenAPI 3.1 source specs with Go code generation

### 1. Scope / Trigger

- Trigger: Any change under `server/api/`, `server/Makefile`, `server/internal/api/gen/`, or `clients/shared/src/types/api.gen.ts`
- Trigger: Any failure in `make lint-spec`, `make generate`, or `make check-drift`
- Trigger: Any proposal to "just downgrade the spec to 3.0" to satisfy `oapi-codegen`

StuHelper is spec-first. The source contract stays on **OpenAPI 3.1**. We do **not** hand-author a parallel 3.0 spec just to satisfy a generator.

### 2. Signatures

Primary commands:

```bash
cd server
make lint-spec
make generate
make check-drift
```

Go code generation entrypoint:

```bash
cd server
go generate ./internal/api/gen/...
```

Resolved generator command:

```bash
cd server
go tool oapi-codegen --config api/oapi-codegen.yaml api/openapi.bundled.yaml
```

Frontend type generation entrypoint:

```bash
cd clients
pnpm run api:generate
```

### 3. Contracts

#### 3.1 Source spec contract

- `server/api/openapi.yaml` must stay at `openapi: 3.1.0`
- Author source schemas using **OpenAPI 3.1 / JSON Schema** rules
- Nullable values in source specs must use type unions, not `nullable: true`

Correct source shape:

```yaml
verifiedAt:
  type:
    - string
    - 'null'
  format: date-time
```

Forbidden source shape:

```yaml
verifiedAt:
  type: string
  format: date-time
  nullable: true
```

#### 3.2 Go codegen compatibility contract

`oapi-codegen` still does not natively support OpenAPI 3.1 source documents. StuHelper handles this with an **OpenAPI Overlay compatibility layer**, not with a downgraded source spec.

Files:

- `server/api/oapi-codegen.yaml`
- `server/api/oapi-codegen-overlay.yaml`
- `server/internal/api/gen/generate.go`

Rules:

- Bundle the real 3.1 spec first
- Apply an overlay only for Go generation
- Downgrade only the pieces `oapi-codegen` cannot parse yet
- Keep `clients/shared/src/types/api.gen.ts` generated directly from the 3.1 source spec

#### 3.3 Toolchain version contract

Current contract:

- `oapi-codegen` tool: `v2.6.0`
- `oapi-codegen/runtime`: `v1.2.0`
- `Redocly CLI`: `2.21.1`
- `openapi-typescript`: `7.13.0`
- `openapi-fetch`: `0.17.0`

Important compatibility pin:

- `github.com/getkin/kin-openapi` stays aligned with the version expected by the current `oapi-codegen` release
- For `oapi-codegen v2.6.0`, that means `kin-openapi v0.133.0`
- Do not blindly bump `kin-openapi` past what the generator release currently compiles against

#### 3.4 Overlay contract

The overlay must be:

- explicit
- minimal
- tied to the current source spec

StuHelper currently uses the overlay for:

- top-level `openapi: 3.1.0 -> 3.0.3` during Go generation only
- `type: [T, 'null'] -> type: T + nullable: true` for supported nullable property shapes

Do not use the overlay as a speculative future bucket. `oapi-codegen` applies overlays strictly, so an action that matches nothing can fail generation.

### 4. Validation & Error Matrix

| Symptom | Likely Cause | Correct Fix |
| --- | --- | --- |
| `nullable` rejected by Redocly under OpenAPI 3.1 | Source spec is still using OpenAPI 3.0 nullable syntax | Convert the source field to `type: [X, 'null']` |
| `unhandled Schema type: &[string null]` from `oapi-codegen` | Source 3.1 union reached the generator without a matching overlay action | Add or fix the relevant overlay action |
| Overlay action "did not match any targets" | Overlay contains a stale or over-generalized selector | Remove or retarget the action so it matches the real spec |
| `oapi-codegen` tool compile errors inside `kin-openapi` or `oasdiff/yaml` | Main module dependency graph drifted away from the generator's supported versions | Re-align the Go module to the official `oapi-codegen` release dependency set |
| Generated Go/TS files drift after spec edits | Source spec changed but generation was not rerun | Run `make generate` and commit generated outputs |
| Need OpenAPI 3.1 examples but Go generation breaks | Source spec uses 3.1-only constructs that the overlay does not yet downgrade | Extend the overlay deliberately; do not downgrade the source spec |

### 5. Good / Base / Bad Cases

#### Good

- Source spec remains 3.1
- Nullable source fields use union types
- Redocly lints the 3.1 source spec directly
- `oapi-codegen` consumes the bundled 3.1 spec plus overlay
- TypeScript types are generated directly from the 3.1 source spec

#### Base

- A new nullable string field is added
- Source schema uses `type: [string, 'null']`
- Overlay already has a matching generic nullable-string action
- `make generate` succeeds without any overlay edits

#### Bad

- Developer changes `openapi: 3.1.0` back to `3.0.3`
- Developer reintroduces `nullable: true` into source YAML
- Developer hand-edits `openapi.bundled.yaml` or `server.gen.go`
- Developer bumps `kin-openapi` independently and breaks `go tool oapi-codegen`

### 6. Tests Required

For OpenAPI/tooling changes, run:

```bash
cd server
make lint-spec
make generate
go test ./...
go build ./...
```

And for frontend contract consumers:

```bash
cd clients
pnpm run api:generate
pnpm run type-check
```

Assertion points:

- `server/api/openapi.yaml` stays on `3.1.0`
- No source YAML under `server/api/components/schemas/` uses `nullable: true`
- `server/internal/api/gen/server.gen.go` regenerates with the expected `oapi-codegen` version
- `clients/shared/src/types/api.gen.ts` regenerates cleanly from the same source contract

### 7. Wrong vs Correct

#### Wrong

```yaml
# server/api/openapi.yaml
openapi: 3.0.3

# server/api/components/schemas/user-system.yaml
verifiedAt:
  type: string
  format: date-time
  nullable: true
```

Why it is wrong:

- Downgrades the real source contract
- Mixes authoring syntax and compatibility syntax
- Forces the repository to pretend it is still 3.0-authored

#### Correct

```yaml
# server/api/openapi.yaml
openapi: 3.1.0

# server/api/components/schemas/user-system.yaml
verifiedAt:
  type:
    - string
    - 'null'
  format: date-time
```

```yaml
# server/api/oapi-codegen-overlay.yaml
overlay: 1.0.0
actions:
  - target: "$.openapi"
    update: "3.0.3"
  - target: "$.components.schemas.*.properties[?(@.type && length(@.type) == 2 && ((@.type[0] == 'string' && @.type[1] == 'null') || (@.type[0] == 'null' && @.type[1] == 'string')))]"
    update:
      type: string
      nullable: true
```

Why it is correct:

- Keeps the source contract honest
- Makes the generator workaround explicit and reviewable
- Localizes compatibility debt to one overlay file

---

## Design Notes from the external references

The current StuHelper approach is based on two upstream realities:

1. `oapi-codegen` officially supports OpenAPI 3.0, not 3.1, and explicitly points users to an overlay-based downgrade workflow in the meantime.
2. Jamie Tanna's May 4, 2025 write-up shows the practical pattern: keep the real 3.1 spec, generate an overlay that downgrades only the incompatible pieces, and wire that overlay into `oapi-codegen`.

Key takeaways we preserve in this repo:

- Keep the real spec on 3.1
- Prefer overlay-based compatibility over maintaining a forked 3.0 source spec
- Start with explicit conversions, then reduce duplication with carefully scoped generic JSONPath selectors
- Be cautious with 3.1-only constructs beyond nullable unions, especially `examples`

External references:

- Jamie Tanna, "Tricking `oapi-codegen` into working with OpenAPI 3.1 specs" (May 4, 2025): [https://www.jvt.me/posts/2025/05/04/oapi-codegen-trick-openapi-3-1/](https://www.jvt.me/posts/2025/05/04/oapi-codegen-trick-openapi-3-1/)
- `oapi-codegen` README, FAQ and overlay documentation: [https://github.com/oapi-codegen/oapi-codegen](https://github.com/oapi-codegen/oapi-codegen)
