---
type: internal
audience: maintainers
status: remediation-reviewed
authoritative-source: code review at develop@6295c6f32a15228c1773be50c6394b5bf2bb86bb
created: 2026-05-08
last-verified: 2026-05-08
remediation-updated: 2026-05-08
---

# 2026-05-08 并行代码库审查记录

本文件记录一次只读并行审查的阶段性结论。它不是长期规范来源；修复前仍应回到当前 checkout 复核源码、配置、OpenAPI 和测试。

## 审查基线

| 项 | 值 |
|----|----|
| 分支 | `develop` |
| HEAD | `6295c6f32a15228c1773be50c6394b5bf2bb86bb` |
| 工作区 | 审查前后均为干净工作区 |
| 执行方式 | 6 个并行代理分模块只读审查，主线程去重和源码抽查 |
| 文件修改 | 未修改、未格式化、未生成、未提交 |

## 复审与修复状态（2026-05-08）

本节是对本文档所有 findings 和外部 Claude 审查合并项的二次复审结果。`✅ 已修复` 表示当前工作区已有代码、配置、生成物或文档变更闭环，并至少有对应的定向测试或 contract 覆盖，可视为本轮已完美修复；`⚠️ 架构决策` 表示不是可以安全地局部修掉的缺陷，需要产品/架构选边；`🟡 部分缓解` 表示已降低风险，但原条目中的长期优化仍存在；`➖ 不成立/无需修` 表示复核后不是当前代码缺陷。

### High Findings

| ID | 状态 | 当前处理 |
|----|------|----------|
| H1 | ✅ 已修复 | 本地 session 撤销已先于 provider revoke 执行；provider revoke 失败只作为显式 partial failure 返回，不再阻断本地封禁。 |
| H2 | ✅ 已修复 | GitLab 新增 `package_scan` stage，容器扫描排在 docker build 之后、package 之前，并由 infra contract 锁定。 |
| H3 | ✅ 已修复 | `websecure` entrypoint 已启用 TLS 和 `letsencrypt` certResolver，Traefik ACME 模式闭环。 |
| H4 | ✅ 已修复 | Admin Docker build context 提升到 `clients`，Dockerfile 显式复制并构建 `clients/shared`。 |
| H5 | ✅ 已修复 | Admin Nginx 增加 `/admin/assets/` 与 `/admin/` rewrite/fallback，匹配当前 Traefik 不 strip + `VITE_BASE=/admin/` 策略。 |
| H6 | ✅ 已修复 | shared API 增加 step-up；Admin 对 `412/A0010205` 跳转 step-up，对 `403/A0010204` 拉取 `/auth/me.accountSettingsUrl` 并跳转 MFA enrollment。 |
| H7 | ✅ 已修复 | session client 只把 401 当 reauth/refresh 触发；403 保留给结构化错误分流。 |
| H8 | ✅ 已修复 | Web Nginx 对 `/admission/` 单独允许 `camera=(self)`，保留其他权限收紧。 |
| H9 | ✅ 已修复 | vote 状态读取增加 `FOR UPDATE`，并用现有 `RowsAffected()` 语义避免并发计数漂移。 |
| H10 | ✅ 已修复 | reply 状态读取增加 `FOR UPDATE`，用户删除和管理员审核都在事务内串行化状态迁移。 |
| H11 | ✅ 已修复 | Koishi 执行动作前做本地 bot/guild/policy 边界校验；服务端 pending action 查询也强制 `platform` + `botSelfID` 必填。 |

### Medium Findings

| ID | 状态 | 当前处理 |
|----|------|----------|
| M1 | ✅ 已修复 | session 持久化 `ProviderAppKey`；refresh/revoke 按应用选择 OAuth config；introspection 使用独立 client credential。 |
| M2 | ✅ 已修复 | 非 file secret backend 读取失败现在直接 `die`，不再吞错继续部署。 |
| M3 | ✅ 已修复 | remote preflight 改用 Compose 网络内 `postgres pg_isready`，失败阻断。 |
| M4 | ✅ 已修复 | 生产 overlay 挂载 `pg_hba.prod.conf`，显式 `hostnossl reject` + `hostssl` 放行。 |
| M5 | ✅ 已修复 | `cadvisor` 加入 prod profile，生产部署服务列表包含 `cadvisor`。 |
| M6 | ✅ 已修复 | Koishi pending action worker 和 shared client 均要求明确 `platform='qq'` / `botSelfID`。 |
| M7 | ✅ 已修复 | freshman material URL 在 `h.image()` 前强制 HTTPS、无凭据、公网 host、allowlist；环境样例补 `STUHELPER_FRESHMAN_MATERIAL_HOSTS`。 |
| M8 | ✅ 已修复 | `check-drift-ts` 显式依赖 `bundle-spec`。 |
| M9 | ✅ 已修复 | `check-drift-all` 纳入 `check-drift-capabilities`。 |
| M10 | ✅ 已修复 | Admin 增加 `unwrapVoid()`，DELETE 204 不再被当失败。 |
| M11 | ✅ 已修复 | 通知 mark-read 改为幂等 no-op，OpenAPI 与实现维持 200/401/403/500 契约。 |
| M12 | ✅ 已修复 | 设计文档改为当前真实 capability：`member_blacklist:manage`。 |
| M13 | ✅ 已修复 | Compose 注释改为“保留 /api 前缀”，Traefik contract 禁止误加 strip。 |

### 外部 Claude 合并项

| ID | 状态 | 当前处理 |
|----|------|----------|
| C1 | ✅ 已修复 | Cookie OIDC auth 通过 `session_id` 读取 session，校验 access token hash 与 `ProviderAppKey`，并调用 `VerifyIDTokenForApplication` 绑定 audience；C11 的 first-party app boundary 一并收口。 |
| C2 | ✅ 已修复 | `ExchangeCodeForApplication` 对空 PKCE verifier 返回 `ErrPKCEVerifierRequired`。 |
| C3 | ✅ 已修复 | introspection endpoint 缺失不再从 token URL 推断 fallback，改为显式配置或 discovery 必须提供。 |
| C4 | ✅ 已修复 | Review/report admin mutation 读侧改为 OpenFGA `Check`；新增 `can_restore` / `can_admin_delete` / `can_admin_edit`，删除未读取的 course/reporter/report tuple 写入和 outbox payload 字段。 |
| C5 | ✅ 已修复 | 删除 `handler_fga.go` 和仅测试引用的 `reviewPermissionRelationForAction`。 |
| C6 | ✅ 已修复 | FGA tuple user/object/relation 增加严格格式正则校验；user/object ID 仅接受 `A-Za-z0-9_-`，覆盖 Check/Write/Delete/List/Read 输入。 |
| C7 | ✅ 已修复 | 静态 school-section tuple 从 review/report hot path 移到 `fga-setup` school bootstrap；`relation_writer.go` 不再懒写静态映射，也不再保留进程级 `sync.Map` 缓存。 |
| C8 | ✅ 已修复 | `RoleScopeResolver` 不再逐 section `ReadTuples`；改用合成 section ID codec 本地校验，FGA 一致性检查降级到 setup/reconcile 侧。 |
| C9 | ✅ 已修复 | 黑名单查询增加 50ms timeout 和 `auth_blacklist_failures_total{reason}` 指标，保留 fail-closed。 |
| C10 | ✅ 已修复 | access/refresh cookie path 抽到 middleware 常量，写入与清理同源。 |
| C11 | ✅ 已修复 / 并入 C1 | 复核后三类 first-party app 都需要 refresh/session 续期，`offline_access` 不是独立缺陷；真实边界是 per-app audience，已由 C1 修复。 |
| C12 | ✅ 已修复 | `reviews.school_id` 与 `review_reports.school_id` 已物化并加索引/FK；创建、举报、列表、FGA reconciliation 和单条 admin 查 school 均改用物化列。 |

### 低风险 / 外部漂移项

| 项 | 状态 | 当前处理 |
|----|------|----------|
| 前端默认校验命令不覆盖 Admin | ✅ 已修复 | `docs/guides/frontend-development.md`、`docs/QUICKSTART.md`、`docs/design/frontend-architecture.md`、`clients/README.md`、`AGENTS.md` 统一改为 `pnpm type-check:all && pnpm lint:all`。 |
| Admin `window.open` 缺 `noopener,noreferrer` | ✅ 已修复 | 账户设置页打开参数已补 `noopener,noreferrer`。 |
| `GetAuthURL` / `GetStepUpAuthURL` panic | ✅ 已修复 | 生产调用面使用 `*ForApplication` 返回 error；兼容 panic wrapper 已移除。 |
| `docs/superpowers/reports` 空目录 | ➖ 不成立/无需修 | 目录无 tracked 文件、无运行时影响；不作为缺陷处理。 |
| AGENTS 命令 target 可能不存在 | ➖ 不成立/无需修 | 根 Makefile 已包含相关 target。 |
| Stateful 容器挂 frontend 网络 | ➖ 不成立/无需修 | `postgres/redis/minio/casdoor/openfga` 均只挂 `backend`。 |
| Koishi 当前启用插件 | ➖ 事实确认 | `stuhelper-core`、`stuhelper-binding`、`stuhelper-group-guard`、`stuhelper-admin` 均为活插件；本轮修复只关闭本文档已确认的 group-guard / platform 边界问题。 |

### 本轮新增闭环项

当前没有未修复的直接 bug 类 finding。原先列为架构/性能边界的 C1/C4/C7/C8/C11/C12 已按长期方案落地；`review_drafts.teacher_id` teacher/course/school 一致性疑点也已补 repository/service 校验和回归测试。

### 本轮验证记录

以下命令在 2026-05-08 的修复工作区内重新执行通过。Admin lint 仍报告 `subcomponents.test.ts` 中测试 stub 的 `vue/one-component-per-file` warning，但退出码为 0，不阻断默认校验。

| 范围 | 命令 | 结果 |
|------|------|------|
| 后端定向包 | `go test -count=1 -timeout=60s ./cmd/casdoor-bootstrap ./internal/pkg/config ./internal/pkg/oidc ./internal/pkg/fga ./internal/pkg/middleware ./internal/pkg/token ./internal/platform/authorization ./internal/modules/auth ./internal/modules/admission ./internal/modules/course/review ./internal/modules/notification` | 通过 |
| FGA 严格校验补测 | `go test -count=1 -timeout=60s ./internal/pkg/fga` | 通过 |
| shared API 单测 | `pnpm --filter @stuhelper/shared test -- --run src/api/__tests__/auth.test.ts src/api/__tests__/session-client.test.ts` | 通过 |
| Web admission 回归 | `pnpm --filter @stuhelper/web test -- src/modules/admission/__tests__/projectionRefresh.test.ts` | 通过 |
| Admin 回归 | `pnpm --filter @vben/web-ele test -- src/api/shared-client.test.ts src/api/shared-result.test.ts src/router/routes/modules/user-system.test.ts src/views/users/member-blacklist/index.test.ts src/views/users/member-blacklist/subcomponents.test.ts src/store/auth.test.ts src/api/admin/admission.test.ts` | 通过 |
| 前端类型 | `pnpm run type-check:all` | 通过 |
| 前端 lint | `pnpm run lint:all` | 通过，保留 3 个测试 stub warning |
| Koishi 回归 | `corepack yarn tsx --test packages/shared/src/platform/index.test.ts plugins/stuhelper-group-guard/src/member-guard.test.ts plugins/stuhelper-group-guard/src/freshman-forward.test.ts plugins/stuhelper-group-guard/src/admission-actions.integration.test.ts` | 通过 |
| Koishi 类型 | `corepack yarn tsc --noEmit` | 通过 |
| Infra contracts | `for test_script in infra/ops/tests/*.sh; do bash "$test_script"; done` | 通过 |
| OpenAPI 生成一致性 | `make generate` 后对 `server/api/openapi.bundled.yaml`、`server/internal/api/gen/server.gen.go`、`clients/shared/src/types/api.gen.ts` 比较前后 diff | 无新增 diff |
| 空白检查 | `git diff --check` | 通过 |

### 追加验证记录（C1/C4/C7/C8/C12 收口）

| 范围 | 命令 | 结果 |
|------|------|------|
| 后端收口验证 | `go test -count=1 -timeout=60s ./internal/pkg/middleware ./internal/modules/auth ./internal/pkg/metrics ./internal/pkg/fga ./internal/platform/authorization ./cmd/fga-setup ./internal/modules/course/review ./internal/modules/course ./internal/app` | 通过 |
| 空白检查 | `git diff --check` | 通过 |

## 覆盖范围

- 后端：认证、会话、OIDC、Capability、RBAC、FGA、评课、通知、部分 admission / user 边界。
- 前端：`clients/shared` API client、Web、Admin、部分 UniAppX 配置。
- Koishi：当前启用插件、admission guard 主链路、平台 client、材料转发。
- Infra：Traefik、Docker Compose、PostgreSQL TLS/HBA、observability、ops 脚本、GitLab CI、Dockerfile。
- 契约与文档：OpenAPI drift 链路、capability 生成物、相关设计和运维文档。

## 高风险 Findings

### H1 Provider 撤销失败会阻断本地 session 撤销

证据：`server/internal/modules/auth/service.go:162`、`service.go:230`、`service_provider_tokens.go:66`、`handler_refresh_reuse.go:32`。

`RevokeSession()` 和 `RevokeAllSessions()` 都先撤销 provider refresh token；provider revoke 或本地解密失败时，函数直接返回，本地 `SessionStore.Revoke()` / `RevokeAll()` 不会执行。refresh-token reuse containment 也调用同一条全设备撤销路径。

影响：Casdoor/OIDC revocation endpoint 临时失败时，本地 Redis session、access/refresh hash 和 session index 仍可能保留。用户登出、全设备登出、refresh token reuse 处置都可能失效。

修复方向：本地 session 黑名单/删除必须独立执行。provider revoke 错误应记录为显式 partial failure，但不能阻断本地封禁。

### H2 GitLab 容器扫描依赖未来 stage 的镜像构建

证据：`.gitlab-ci.yml:8`、`.gitlab-ci.yml:209`、`.gitlab-ci.yml:248`。

`security` stage 排在 `build` 前，但 `container_scan_backend/frontend/admin` 位于 `security` stage，并 `needs` 后续 `build` stage 的 `docker_build_*`。

影响：GitLab pipeline 图可能无效或无法调度，进而阻断 package/deploy。

修复方向：把容器扫描移到镜像构建之后的 stage，或调整 docker build stage 顺序，保留后续 package 对扫描结果的依赖。

### H3 Traefik ACME/HTTPS 模式（已闭环）

证据：`docker-compose.yml:38`、`infra/traefik/services.dynamic.yaml:43`、`infra/traefik/tls.dynamic.yaml:10`、`docs/guides/release-runbook.md:176`。

复核更新：`docker-compose.yml` 的 `websecure` entrypoint 已启用 TLS 并绑定 `letsencrypt` certResolver，仓库声明的 Traefik ACME 模式已闭环。

影响：如果生产最外层由外部 LB、宝塔或 Nginx 终止 TLS，Traefik ACME 会变成冗余路径，但不影响仓库内 Compose 模式正确性。

修复状态：✅ 已修复，并由 Compose/Traefik contract 覆盖。

### H4 Admin Docker build context 不包含 `@stuhelper/shared`

证据：`.gitlab-ci.yml:272`、`clients/admin/scripts/deploy/Dockerfile:19`、`clients/admin/apps/web-ele/package.json:31`。

CI 以 `clients/admin` 为 Docker build context，但 admin app 依赖 `@stuhelper/shared: link:../../../shared`。容器内 `/app/apps/web-ele/../../../shared` 会解析到 `/shared`，该目录不在 build context 中。

影响：干净 CI/Docker 环境中 Admin 镜像无法稳定安装或构建共享 API/类型包。

修复方向：把 build context 提升到 `clients` 或仓库根，并显式复制/构建 `clients/shared`；或调整 workspace 边界，使 Admin Docker 构建与根 `build:shared` 链路一致。

### H5 Admin `/admin/` base path 与 Traefik/Nginx 前缀策略不一致

证据：`infra/traefik/services.dynamic.yaml:56`、`clients/admin/scripts/deploy/nginx.conf:23`、`clients/admin/apps/web-ele/.env.production:1`、`.gitlab-ci.yml:272`。

生产 Admin 以 `VITE_BASE=/admin/` 构建，Traefik 只按 `PathPrefix('/admin')` 转发且不 strip，Admin Nginx 只有 `location /`，没有 `/admin/` alias/rewrite。

影响：`/admin/assets/*` 可能在 Nginx 中回落为 `index.html`，浏览器以 JS/CSS MIME 加载 HTML，导致 Admin 白屏或资源加载失败。

修复方向：选择一种前缀策略：Traefik strip 并把 build base 改 `/`；或 Nginx 增加 `/admin/` location 正确服务静态资源和 SPA fallback。

### H6 MFA step-up 后端契约存在，但前端没有闭环

证据：`server/internal/app/modules_auth.go:67`、`server/internal/modules/rbac/middleware.go:198`、`clients/shared/src/api/auth.ts:28`、`clients/admin/apps/web-ele/src/api/shared-result.ts:16`。

Admin 路由挂 `RequirePrivilegedMFA()`，OpenAPI 已有 `/api/v1/auth/step-up`，但 shared `createAuthApi` 没封装 step-up，admin 错误层也只泛化处理非 401。

影响：管理员遇到 `A0010204` MFA enrollment required 或 `A0010205` step-up required 时，前端无法进入可恢复认证流程，可能卡死、泛化报错或触发错误登录跳转。

修复方向：在 shared API 增加 step-up wrapper；请求层识别 `412/A0010205` 跳转 step-up，识别 `403/A0010204` 展示 MFA enrollment 入口。

### H7 共享 session client 把所有 403 当成未认证

证据：`clients/shared/src/api/session-client.ts:186`、`clients/web/src/api/client.ts:291`、`clients/admin/apps/web-ele/src/api/shared-client.ts:162`。

`isUnauthorizedStatus()` 返回 `status === 401 || status === 403`。当 403 不触发 refresh 时，web 会清 session，admin 会重置 store 并跳 OIDC login。

影响：真实的权限不足、MFA、CSRF、业务 403 都会被误判成登录失效，掩盖授权错误并可能制造 SSO 循环。

修复方向：session 层只把 401 或明确 auth-expired code 当 reauth；403 按结构化错误码分流。

### H8 生产 Web Nginx 禁止 camera，但新生认证依赖摄像头

证据：`clients/web/nginx.conf:16`、`clients/web/Dockerfile:21`、`clients/web/src/modules/admission/cameraCapture.ts:34`、`FreshmanCameraFlow.vue:162`。

生产 Web Nginx 下发 `Permissions-Policy "camera=()"`，而 admission camera flow 调用 `navigator.mediaDevices.getUserMedia()`。

影响：生产环境浏览器会在策略层拒绝摄像头权限，新生材料拍摄流程不可用。

修复方向：至少对 admission 路径允许 `camera=(self)`；如果全站放开 camera，也应继续拒绝 microphone/geolocation/payment。

### H9 评课投票计数在并发取消/切换时可能永久漂移

证据：`service_review_write.go:194`、`repository.go:239`、`repository.go:250`、`repository.go:256`、`repository.go:263`。

`VoteReview` 先读取投票状态，再删除/更新投票并增减冗余计数；读取无锁，`DeleteVote` / `UpdateVoteType` 不检查实际迁移。

影响：同一用户并发取消或切换投票时，`reviews.like_count` / `dislike_count` 可能与 `review_votes` 真实数据永久不一致。

修复方向：使用 `FOR UPDATE` 或 `DELETE/UPDATE ... RETURNING` 做原子状态迁移，只在实际迁移发生后调整计数，并补并发测试。

### H10 回复删除/管理员审核回复会导致 `reply_count` 并发漂移

证据：`service_interaction.go:311`、`repository_interaction.go:403`、`repository_interaction.go:414`、`repository_interaction.go:425`、`service_admin.go:65`。

`GetReplyOwnerAndReviewIDTx` 无 `FOR UPDATE`；`SoftDeleteReply` / `UpdateReplyStatusTx` 只按 id 更新，不校验源状态，也不检查实际迁移。

影响：双击删除、网络重试、两个管理员并发审核同一回复时，`reviews.reply_count` 会与真实 `review_replies.status='published'` 数量不一致。

修复方向：给回复状态读取加锁，或把状态迁移改成带源状态条件的 `UPDATE ... RETURNING`；用户删除和管理员审核共用同一个状态迁移 helper。

### H11 Koishi pending admission action 缺少本地 guild/policy 二次校验

证据：`bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts:61`、`member-guard.ts:185`、`member-guard.ts:238`、`admission-actions.ts:69`。

入群事件路径会 `resolvePolicy(platform, guildID)`；但 pending action 扫描只按 `platform/botSelfID` 拉取，执行时接受后端 action 自带 `guildID/channelID/qqID`，即使找不到本地 record 也能执行。

影响：如果后端 pending queue 返回了不属于当前 Koishi guard policy 的 guild action，Koishi 仍可能解除禁言、踢人或踢人并拒绝再次加群。服务端实际可利用性仍取决于 `/api/v1/bot/admission/sessions/pending` 的实现，但 Koishi 侧缺少 fail-closed 边界是 confirmed。

修复方向：执行前对 action/record 的 guild 重新走本地 `policyStore.resolvePolicy()`；没有本地 record 的 action 必须自带 guild 并通过 policy 校验。

## 中风险 Findings

| ID | 问题 | 证据 | 修复方向 |
|----|------|------|----------|
| M1 | 多 OIDC application 登录后，refresh / revoke / introspection 固定使用 web client | `server/internal/pkg/oidc/client.go:70`, `handler_refresh_oidc.go:62`, `revoke.go:64`, `session.go:73` | session 持久化原始 app/client；按 app 选择 OAuth config，或显式建模专用 resource-server client |
| M2 | 非文件 secret backend 读取失败会被吞掉 | `infra/ops/lib/common.sh:119`, `prod-deploy.sh:14`, `remote-preflight.sh:31` | 配置了非 file backend 时读取失败必须 `die`，保留错误输出 |
| M3 | 远端预检备份 DB 检查走错网络且不阻断 | `remote-preflight.sh:137`, `backup-postgres.sh:46`, `run-scheduled-backup.sh:13` | 复用 `compose run --rm --no-deps postgres pg_isready`，失败升级为阻断 |
| M4 | PostgreSQL 生产 SSL 只靠客户端自觉，HBA 仍允许非 TLS TCP | `.env.prod.example:12`, `docker-compose.yml:85`, `infra/postgres/pg_hba.conf:13` | 生产 HBA 改 `hostssl` 并显式 `hostnossl reject` |
| M5 | Prometheus 固定抓取 cAdvisor，但 prod profile 不启动 cAdvisor | `docker-compose.observability.yml:239`, `prometheus.yml.tmpl:54`, `prod-deploy.sh:245` | 生产启动 cAdvisor，或渲染 Prometheus 时按 mode 去掉该 target |
| M6 | Koishi `platform` 为空时仍拉 pending action，query builder 会省略空字段 | `member-guard.ts:185`, `platform/index.ts:203`, `platform/index.ts:212` | admission worker 要求明确 `platform === 'qq'`，缺失时报错跳过 |
| M7 | 新生材料转发直接把后端 `materialURL` 渲染成图片消息 | `freshman-forward.ts:20` | Koishi 侧校验 `https`、host allowlist、拒绝内网/本地 URL |
| M8 | `check-drift-ts` 单独运行可能消费过期 bundled spec | `server/Makefile:99`, `server/Makefile:121`, `clients/package.json:23` | `check-drift-ts` 显式依赖 `bundle-spec` |
| M9 | `server/Makefile check-drift-all` 不覆盖 capability 生成物 | `server/Makefile:116`, `clients/package.json:22`, `.gitlab/server-ci.yml:53` | 纳入 `check:capabilities-drift` 或统一根 drift 入口 |
| M10 | Admin DELETE 204 成功响应被 `unwrapData()` 当失败 | `openapi.bundled.yaml:3618`, `openapi.bundled.yaml:3770`, `content.ts:108`, `shared-result.ts:27` | 增加 `unwrapVoid` / mutation success helper，或全链路改为 200 envelope |
| M11 | 通知标记已读实现返回 404，但 OpenAPI 没声明 | `notification/handler.go:107`, `notification/repository.go:128`, `review-notification.yaml:87` | OpenAPI 补 404 后重新生成，或实现改成契约声明的状态 |
| M12 | 设计文档列出不存在的 `admission:blacklist:manage` | `docs/design/koishi-admission-verification.md:261`, `server/internal/pkg/capability/catalog.go:20` | 改为当前 `member_blacklist:*` capability，避免复制过期常量 |
| M13 | Compose 注释说 `/api` strip，但 Traefik 实际保留 `/api` | `docker-compose.yml:9`, `infra/traefik/services.dynamic.yaml:3` | 修正文档注释，保留 contract 防线 |

## 低风险和正向确认

- 前端开发文档默认 `pnpm type-check && pnpm lint` 不覆盖 Admin。证据：`docs/guides/frontend-development.md:95`、`clients/package.json:28`。建议改为 `type-check:all` / `lint:all`，或明确默认命令范围。
- Admin 外部账户设置页 `window.open(..., '_blank')` 缺少 `noopener,noreferrer`。证据：`clients/admin/apps/web-ele/src/views/_core/profile/password-setting.vue:16`。
- Koishi platform credential 未发现 tracked 硬编码 secret；`createPlatformClient()` 对空 `baseUrl/serviceToken` 会显式失败。证据：`bots/koishi/koishi.yml:28`、`bots/koishi/packages/shared/src/platform/index.ts:223`。

## 外部审查复核合并

本节合并 2026-05-08 外部 Claude 审查报告中提到、但上文尚未完整覆盖的事项。结论分为 confirmed、duplicate、not-confirmed 和 needs-deep-review；不把外部报告原文当事实源。

### 新增 Confirmed

| ID | 级别 | 结论 | 证据 | 处理建议 |
|----|------|------|------|----------|
| C1 | HIGH | OIDC ID token audience 校验按“任一已配置 client”放行，未绑定当前 application | `server/internal/pkg/oidc/verifier.go:41`, `client.go:281`, `handler_login.go:140`, `handler_login.go:409` | ✅ 已修复：callback/native/refresh/cookie auth 均按 session/application 绑定 expected audience |
| C2 | MEDIUM | `ExchangeCodeForApplication` 在 `codeVerifier == ""` 时无 PKCE 交换，生产调用方若传空串会绕过 PKCE | `server/internal/pkg/oidc/client.go:202`, `client.go:215` | 生产路径强制非空；测试兼容走专用 helper 或 test client |
| C3 | MEDIUM | discovery 缺 `introspection_endpoint` 时会 fallback 到 token URL 同目录 `/introspect` | `server/internal/pkg/oidc/client.go:340`, `client.go:352`, `client.go:413` | 生产应显式配置或 discovery 必须提供 introspection endpoint，避免错误主机/路径 |
| C4 | HIGH | Review/report FGA 关系存在写入、outbox 和 reconciliation，但业务决策读侧没有调用 FGA `Check` | `server/internal/modules/course/review/authorization.go:9`, `service_fga_sync.go:91`, `admin_scope.go:207`, `server/internal/pkg/fga/client.go:83` | ✅ 已修复：admin mutation 使用 FGA `Check`，列表仍用物化 scope；未读取 tuple 和 payload 字段已删除 |
| C5 | HIGH | `reviewPermissionRelationForAction` 仅测试引用，是当前生产死代码 | `server/internal/modules/course/review/handler_fga.go:3`, `authorization_test.go:24` | ✅ 已修复：死代码文件已删除，C4 使用新的 action-relation 映射 |
| C6 | MEDIUM | FGA tuple 字段校验只拦空值和控制字符，未强制 `type:id` / relation 格式 | `server/internal/pkg/fga/client.go:71` | 给 user/object 增加 `type:id` 格式校验，relation 增加 relation-name 校验 |
| C7 | MEDIUM | `WriteMissingTuples` 每次写 review/report 关系都会 read-before-write，且包含静态 `school -> section` 映射 | `server/internal/pkg/fga/relation_writer.go:18`, `relation_writer.go:23`, `relation_writer.go:39` | ✅ 已修复：静态 school/section 关系由 `cmd/fga-setup` bootstrap，review/report hot path 只写动态关系 |
| C8 | LOW | `RoleScopeResolver` 对每个 section 单独 `ReadTuples` 校验学校归属，section 多时登录 RTT 放大 | `server/internal/platform/authorization/role_scope_resolver.go:112`, `role_scope_resolver.go:126` | ✅ 已修复：合成 section ID 本地解析校验，不再登录热路径访问 FGA read |
| C9 | MEDIUM | 黑名单 Redis 查询 fail-closed 是安全取舍，但缺少可见超时/告警约束时会把 Redis 慢调用放大成全站认证慢/503 | `server/internal/pkg/middleware/auth.go:48` | 给黑名单查询设置短超时并补告警；保留 fail-closed 语义 |
| C10 | LOW | `clearAuthCookies` 与 auth handler 的 refresh cookie path 字面量重复，当前值一致且有测试，但后续可能漂移 | `server/internal/pkg/middleware/auth.go:249`, `server/internal/modules/auth/handler_cookies.go:15` | 抽共享常量或用测试继续锁定 |
| C11 | LOW | OIDC scopes 三类 application 统一为 `openid/profile/email/offline_access`，没有按 web/admin/uniapp 分最小 scope | `server/internal/pkg/oidc/client.go:137`, `client.go:142` | ✅ 已复核并并入 C1：三类 first-party app 均需要 refresh；最小权限边界改由 per-app audience 保证 |
| C12 | MEDIUM | 单条 review/report moderation 仍按 ID 反查 schoolID；批量 review 已批量化但单条路径仍有 DB 往返 | `server/internal/modules/course/review/admin_scope.go:207`, `admin_scope.go:224`, `admin_scope.go:265` | ✅ 已修复：review/report 行物化 `school_id`，单条/批量/list/reconciliation 读取改用物化列 |

### Duplicate / 已由上文覆盖

| 外部报告项 | 合并状态 |
|------------|----------|
| 多 OIDC application refresh/revoke/introspection 固定 web client | 已覆盖为 M1 |
| FGA 双轨制 / review 操作未读 Check | 合并为 C4，已按“列表用物化 scope、mutation 用 request-time FGA Check”的分层方案修复 |
| token blacklist Redis 失败导致认证 503 | 合并为 C9；fail-closed 本身不判错，缺少超时/告警约束才是问题 |
| Admin workspace / shared 契约 | 已覆盖为 H4：真实缺陷是 Docker build context；`web-ele` 源码已直接引用 `@stuhelper/shared` |
| Koishi 当前状态需 spot check | 已覆盖 H11、M6、M7；`stuhelper-admin` / `stuhelper-core` 深审仍在待确认 |

### Not Confirmed / 修正

| 外部报告项 | 复核结论 |
|------------|----------|
| “代码中 grep zitadel 0 命中” | 不准确。当前运行代码/infra 主线是 Casdoor，但 archived docs、design 草案和 boundary 脚本仍包含 Zitadel；正确表述是“不要把历史 Zitadel 文档/记忆当当前实现” |
| `GetAuthURL` / `GetStepUpAuthURL` panic 会把生产配置错误变 500/panic | 部分成立。兼容包装函数确实 `panic(err)`，但生产 handler 使用 `GetAuthURLForApplication` / `GetStepUpAuthURLForApplication` 返回 error；当前更像测试兼容 API 的清理项，不是主要生产入口风险 |
| phone 自签名 token 一定会走 RoleScopeResolver | 当前 `needsResolvedScopes` 对普通 `user` 角色短路，功能上安全；可加 issuer 注释/短路，但不作为当前 bug |
| `RoleScopeResolver` nil 会怎样 | `withResolvedRoleScopes` 对 nil resolver 直接返回；不是运行时 bug，但 wire 层没有编译期强约束 |
| phone/OTP 日志可能漏 mask | 已看到 `handler_phone.go` 多处使用 mask；未发现具体泄漏点，作为 audit/PII 深审项处理 |
| Admin 独立 workspace 导致不能复用 shared 类型 | 不成立。`clients/admin/apps/web-ele/package.json` 使用 `@stuhelper/shared: link:../../../shared`，源码也从 shared 导入 API/types/constants |
| `_archived/playground` 未标明用途 | 已有 `clients/admin/_archived/README.md` 标明 inactive/reference；是否删除属于仓库瘦身，不是缺陷 |
| stateful 容器是否挂 frontend 网络 | 已复核，`postgres/redis/minio/casdoor/openfga` 均只挂 `backend`；proxy 和 dev frontend/admin 按预期跨网络 |
| `docs/superpowers/reports` 命名冲突 | 目录存在但为空；无直接代码/文档影响 |
| `AGENTS.md` 中 `make e2e`、`make obs-up` 等 target 可能不存在 | 根 `Makefile` 中这些 target 均存在 |
| `.github/workflows` 未读 | 当前仓库无 `.github/workflows`，CI 入口在 `.gitlab/` |

### 外部上下文漂移

本轮复核确认：当前代码和部署主线是 Casdoor + OpenFGA + 本地 session/JWT；`docker-compose.yml` 仍有 `casdoor` service，`server/cmd/casdoor-bootstrap` 仍是有效入口，网络只有 `frontend/backend/observability` 三个。`docs/internal/exec-plans/archived/`、部分设计草案和本机 memory 中仍有 Zitadel 迁移历史语义。处理原则：这些只能作为历史过程材料，不能作为当前实现事实源；当前事实应以源码、Compose、OpenAPI、migrations、`docs/design/` 当前文档为准。

| 漂移项 | 复核结论 |
|--------|----------|
| Casdoor / Zitadel | 运行主线是 Casdoor；Zitadel 只在历史文档、设计草案或边界脚本中出现，不能代表当前部署事实 |
| OpenFGA | OpenFGA 仍是活依赖；review/report admin mutation 已切到 request-time `Check`，列表仍使用登录时物化 scope |
| Vben Admin | `clients/admin/apps/` 当前主应用是 `web-ele`；历史 playground 已移入 `_archived` 并有 README 标记 |
| RBAC | `server/internal/modules/rbac` 仍由 admin 守卫装配使用，不是可直接删除的死模块 |
| Docker 网络 | Compose 当前定义 `frontend`、`backend`、`observability` 三个网络，不存在 Zitadel-only 内部网络 |
| Bootstrap | 当前有效引导入口是 `server/cmd/casdoor-bootstrap`，不是 Zitadel PAT bootstrap |
| Docs 结构 | 当前目录是 `adr/`、`design/`、`guides/`、`internal/`、`product-specs/`、`reference/`、`superpowers/` 等，不再是旧扁平结构 |

## 待确认项

- `server/internal/modules/user`、`ldap`、完整 `admission`、Koishi `stuhelper-admin` 命令、`stuhelper-core` Console API、`message-guard`、Koishi UI runtime 尚未完成逐文件深审。
- Koishi pending action 的服务端接口 `/api/v1/bot/admission/sessions/pending` 已强制 `platform` 和 `botSelfID` 必填；更细的 credential-derived guild/policy 绑定属于后续服务账号模型设计，不再作为本轮 confirmed bug。
- `notification_preferences` 是否应被 `notification.Service.Send` 读取已有疑点，但是否属于已承诺能力需继续核对产品规格和调用链。
- H3 已闭环；如果生产最外层由外部 LB、宝塔或 Nginx 终止 TLS，Traefik ACME 是冗余路径但不影响仓库 Compose 模式正确性。
- 单独深审项：`pkg/outbox` idempotency / claim race / retry-dead-letter、`pkg/audit` retention 与 PII、`pkg/cache`/`redis`/`singleflightx` 抽象边界、`pkg/metrics` label cardinality、Web/Admin/UniAppX 全量源码、Docker 每服务 limits/healthcheck/read_only/user。
