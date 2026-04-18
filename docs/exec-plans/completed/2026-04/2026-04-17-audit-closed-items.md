# 2026-04-17 已关闭审计项与文档归档

## 说明

- 依据当前代码、测试与配置实况复核。
- 仅保留仍未完成的问题在 active 文档中；已完成/已核销项从 active 移出。
- 以下内容为历史闭环记录，不再占用 active 审计清单。

## 归档文档

- `docs/exec-plans/archived/merged-audit-2026-04-17-codex-second-review.md`
- `docs/exec-plans/archived/iam-architecture-migration.md`

## 已删除的原始 review 文档（已被 merged/completed 接管）

- `docs/reviews/2026-04-16-full-parallel-agents-audit.md`
- `docs/reviews/2026-04-16-client-front-audit-round2.md`
- `docs/reviews/2026-04-16-full-parallel-maintainability-audit-round3.md`
- `docs/reviews/2026-04-16-backend-quality-audit-round4.md`
- `docs/reviews/2026-04-16-fallback-duplication-audit-round4.md`
- `docs/reviews/2026-04-16-api-boundary-doc-structure-audit-round4.md`
- `docs/reviews/2026-04-16-all-agents-audit-consolidated-round4.md`

说明：
- 这些文件是原始并行审计输入，内容与 `merged-audit(active)`、`completed` 长期重复。
- 所有**仍未完成**的问题只保留在 `/Users/zxy/Code/StuHelper/docs/exec-plans/active/merged-audit-2026-04-16.md`。
- 所有**已闭环/已核销**的问题只保留在本完成文档。
- 原始逐条证据仍可通过 Git 历史回溯；当前工作树不再保留重复 review 稿。

## 已关闭条目（自 active 审计移出）

### ✅ [FIXED] user / review 域错误到 HTTP 状态映射已收敛到集中层
（来源: doc1 ARCH-M4@278, doc5 #5@170）
已修复：`server/internal/pkg/response/mapped_error.go` 新增 `RespondMappedError` / `RespondMappedErrorGroups` 作为通用映射基座，`server/internal/modules/user/http_errors.go` 与 `server/internal/modules/course/review/http_errors.go` 现在集中维护各自领域的 sentinel error → HTTP status / error code / message 映射。`handler_self.go`、`handler_admin.go`、`review.go`、`review_draft.go`、`review_reply.go`、`review_interaction.go`、`admin.go`、`admin_review.go`、`handler_teacher_admin.go`、`handler_sensitive_word_admin.go`、`review_read.go`、`handler_content_flag.go` 已切到统一映射 helper，不再在每个 handler 手写一套 `switch errors.Is(...)` 逻辑；相关单元测试已补到 `internal/pkg/response/mapped_error_test.go`、`internal/modules/user/http_errors_test.go`、`internal/modules/course/review/http_errors_test.go`，并通过 `go test ./internal/pkg/response ./internal/modules/user ./internal/modules/course/review -count=1` 验证。

### ✅ [FIXED] 管理端身份审核列表不再 N+1 presign
（来源: doc5 #7@240）
已修复：`server/internal/modules/user/handler_admin.go` 的实名认证审核列表不再逐条调用 `ResolveIdentityReviewItemAssets()` 做对象存储 presign；列表接口现只返回审核元数据，不再携带 `docPhotoFront/docPhotoBack/docPhotoSelfie`。OpenAPI `AdminIdentityReviewItem`、Go/TS 生成类型与 `identityReviewItemResponse` 已同步收敛，admin 前端现状本就未消费这些照片字段，因此无行为回退且热路径 20 条分页不再触发 60 次 presign I/O。

### ✅ [FIXED] `CourseDetailPage.vue` 已收敛到可维护规模
（来源: doc1 FE-M3@376）
已修复：`clients/web/src/modules/review/views/CourseDetailPage.vue` 已从 798 行降到 705 行，并把回复状态机与管理端审核/恢复/编辑动作分别收敛到现有 `useReviewReplies()`、`useReviewAdmin()` composable；页面文件不再同时维护 reply/admin 的内联副作用逻辑，职责边界明显收窄。

### ✅ [FIXED] “预检查 + 事务内重复检查”规则已收口到单一 validator
（来源: doc5 #13@396）
已修复：`server/internal/modules/user/service_student_verify.go` 新增 `validateStudentVerificationTransition()`，`VerifyStudent` 事务外预检查与事务内最终校验现复用同一套状态规则，不再在两个位置手写 `StatusVerified/StatusPending` 分支。`review.PostReview` 当前实现已只保留事务内权威校验，因此这条关于“规则逻辑重复漂移”的活跃风险已闭环；回归测试已补到 `server/internal/modules/user/service_student_verify_test.go`。

### ✅ [FIXED] `ListLatest()` 不再使用 `COUNT(*) OVER()` 全量窗口扫描
（来源: doc1 ARCH-L2@307）
已修复：`server/internal/modules/course/review/repository.go` 的 `ListLatest()` 已改为分离的 `SELECT COUNT(*)` + 数据查询，不再在热门最新评论列表上使用 `COUNT(*) OVER()` 让每次分页都扫描全部命中行；对应 `go test ./internal/modules/course/review` 已通过。剩余分页风格统一问题继续单独在 active 跟踪。

### ✅ [FIXED] WAL 归档目录不再位于项目树内
（来源: doc1 INFRA-M2@461）
已修复：运行中的 PostgreSQL WAL 归档已从仓库内 bind mount 迁移到 Docker named volume `postgres_wal_archive`；`sync-postgres-backups.sh` / `run-scheduled-backup.sh` 改为直接从 volume 读取并清理 WAL，`fetch-postgres-backups.sh` 的 restore cache 默认移到本机 state 目录，不再把活动 WAL 写回仓库树。

### ✅ [FIXED] PostgreSQL WAL 归档 / 备份不再给 dev 强加多余对象存储变量
（来源: doc1 XC-M3@1142）
已修复：开发环境默认复用主对象存储端点 / 凭据，`BACKUP_OBJECT_STORAGE_*` 在 dev 已全部变为可选；`init-dev-env.sh` 不再生成额外 backup MinIO 凭据，`minio-init` 与备份脚本会在缺省时回退到主对象存储配置，dev 环境变量面显著收窄。

### ✅ [FIXED] cAdvisor 不再进入生产默认拓扑
（来源: doc1 SEC-M3@184）
已修复：`docker-compose.yml` 中 `cadvisor` 已收敛为 `observability` 本地/临时 profile 专用，`infra/ops/prod-deploy.sh` 不再在 production 默认拉起该服务；生产默认观测面保留 node-exporter / postgres-exporter / redis-exporter / blackbox-exporter，避免在生产主机上长期暴露 `/:/rootfs:ro`、`/var/run`、`/sys`、`/var/lib/docker` 广泛只读挂载。

### ✅ [FIXED] `review` 不再通过 `user.*` 具体类型“假解耦”
（来源: doc1 ARCH-M1@255）
已修复：新增 `server/internal/pkg/reviewaccess/types.go` 作为最小 DTO 边界，`review.ReviewAccessReader` 现只暴露 `reviewaccess.SchoolConfig / SystemConfig / Subject`；`user.Repository` 提供专用投影适配方法，`review` 包已不再导入 `internal/modules/user` 的具体类型，模块边界恢复为稳定的只读 DTO 契约。

### ✅ [FIXED] UniApp JS/TS shadow files
已修复: 删除 UniApp X JS/TS 影子文件并加上 `check_shadow_files` 门禁；测试副本与语义分叉风险随单一 TS 源一并清理。
（命中次数: 4；来源: doc2 3.1@146, doc3 P1-03@38, doc4 P3@96, doc6 S0-2@45, doc3 P3-02@141, doc3 P3-03@152）
双轮复核核实：`clients/uniappx/src` 中无 `.ts/.js` 同名 shadow pair；`.gitlab-ci.yml:53` 存在 `check_shadow_files` job。保持门禁即可。


### ✅ [FIXED] Admin playground 归档
已修复: `clients/admin/playground/` 已迁入 `_archived/`，活动代码与脚本引用已切断。
（命中次数: 2；来源: doc6 S2-9@218, doc7 S1-3@107）
双轮复核核实：`clients/admin/_archived/playground/` 存在，`clients/admin/playground/` 已消失。后续需保持活动代码不引用 `_archived/` 路径，可加 CI 规则禁止。


### ✅ [FIXED] Metrics 测试路由脱靶
已修复: metrics 测试已改用真实挂载路径常量，并补充 route contract tests。
（命中次数: 2；来源: doc2 4.1@195, doc4 P2@86）
基于 commit `8a2fc42/610e3aa` 的声明与 route contract 补齐路径，修复成立。保持 route contract test 作为回归屏障。


### ✅ [FIXED] govulncheck in CI
已核销: 审计报告误报；实际 `.gitlab/server-ci.yml` 中已存在 `backend_vulnerability_scan` 执行 `govulncheck ./...`。
（命中次数: 1；来源: doc1 SEC-L4@216, doc1 SEC-L6@228）
双轮复核核实：`.gitlab/server-ci.yml:33-34` 存在 `go install golang.org/x/vuln/cmd/govulncheck@v1.1.4` + `govulncheck ./...`。无需再修。


### ✅ [FIXED/误报] `payloadOrEmptyJSON` 在 notification service 被调用但未定义
（命中次数: 1；来源: doc1 GO-M1@96）
双轮复核核销：grep 确认 `payloadOrEmptyJSON` 定义于 `server/internal/modules/notification/repository.go:100`，同包 `service.go:135`、`templates.go:97`、`repository.go:59` 均正常调用。原审计条目不成立。

---

## 本轮新增闭环（2026-04-17，代码与测试复核）

### ✅ [FIXED] `SessionStore.Touch` 非原子 read-modify-write 竞态
（来源: doc1 GO-C1, doc5 #1, doc6 S0-3）
已修复：session store touch 路径完成原子化闭环，相关认证/会话测试已通过；不再保留“读旧值再回写”的竞态窗口。

### ✅ [FIXED] 手机验证码登录的 session ID / JWT `sid` 双源不一致
（来源: doc2 1.4, doc5 #2, doc6 S0-1）
已修复：手机登录先生成单一 `sessionID`，签 JWT 与创建服务端 session 均复用同一值；`refresh/revoke` 不再落到不同 sid。

### ✅ [FIXED] OptionalAuthMiddleware 对无效/吊销 token 默认放行
（来源: doc3 P2-01, doc6 S1-4）
已修复：改为四分支策略——无 token 匿名继续；坏 cookie 清理后匿名继续；坏 bearer 直接 401；后端故障注入诊断标记而非静默伪装成匿名。

### ✅ [FIXED] Refresh 端点 CSRF bypass 边缘场景
（来源: doc1 SEC-M1@171）
已修复：refresh 路径已按当前认证模型加固，相关认证测试通过。

### ✅ [FIXED] `errors.Is` 未用于 `redis.Nil` 比较
（来源: doc1 GO-C3@41）
已修复：统一改为 `errors.Is(..., redis.Nil)`，避免包装错误后比较失效。

### ✅ [FIXED] 公共搜索 / 批量端点缺少渐进式限流
（来源: doc1 SEC-M3@184）
已修复：新增 progressive endpoint rate limit，中匿名用户与已认证用户额度分离，已接入 review search / batch 入口并有中间件测试。

### ✅ [FIXED] Metrics 端点缺少来源验证
（来源: doc1 SEC-L5@222）
已修复：新增 metrics origin validation middleware；未配置 allowlist 时生产默认拒绝，开发使用明确本地白名单。

### ✅ [FIXED] 并发校验后再插入，竞态下返回 500
（来源: doc1 GO-M5, doc5 #5）
已修复：repository 层将唯一约束冲突映射为既有业务错误（如已评/已举报），不再把竞态重复写入暴露成 500。

### ✅ [FIXED] 认证双轨：session store 与 legacy token tracking 并存
（来源: doc1 GQ-H6, doc5 #11, doc6 S0-1）
已修复：`RevokeAllSessions` 等会话撤销路径只保留 session store 权威链路，legacy token tracking 已从活跃控制流移除。

### ✅ [FIXED] OpenFGA 可选化导致资源级授权 fail-open
（来源: doc5 #9, doc6 S1-3）
已修复：review 授权改成 fail-closed provider；未配置 FGA 时显式注入拒绝型 stub，而非 runtime `nil => skip auth`。

### ✅ [FIXED] 学生评课能力判断未按能力粒度闭环
（来源: doc1 XC-H4, doc5 #3）
已修复：review access facts 已细化到 `ReviewCreate / ReviewEditOwn / ReviewDeleteOwn`，服务层按能力与事实前置条件同时判定。

### ✅ [FIXED] Zitadel 角色缺少组织 scope
（来源: doc2 1.2@27）
已修复：认证中间件不再把 `school_admin` 扁平化成全局管理员能力；`OrgScopedRoles` 会展开成带 `scopeSchoolIDs` 的 capability grants，`globalCapabilities` 仅保留真正全局授权。用户管理后台中仅保留已落地 school scope 的学生认证 / 学校配置能力，系统配置与实名审核改为全局能力；school-scoped 入口会校验请求 `schoolID` 或目标资料所属学校，杜绝“任一 school_admin 管所有学校”。

### ✅ [FIXED] 请求层 / 鉴权层在 3 个前端分支重复实现
（来源: doc3 P1-01@18, doc3 P3-01@129, doc7 S1-2@67）
已修复：`clients/shared/src/api/session-client.ts` 现为三端共用会话运行时核心，统一了 schema path 归一化、path/query 序列化、CSRF header 装配、refresh singleflight 与 401/refresh 处理。`clients/admin/apps/web-ele/src/api/shared-client.ts`、`clients/uniappx/src/api/shared-client.ts`、`clients/web/src/api/client.ts` 均改为基于 shared 会话核心实现；web 保留浏览器 `fetch` 适配层，admin / uniappx 分别注入各自 transport，但不再各自维护一套 refresh/重试状态机。

### ✅ [FIXED] 路由层直接操作组件/弹层状态
（来源: doc3 P1-05@59）
已修复：`clients/web/src/router/index.ts` 不再直接触发 `useReviewPost().closePostModal()`；关闭写评弹层的职责已移到 `/Users/zxy/Code/StuHelper/clients/web/src/App.vue` 中基于 `route.fullPath` 的 watch，路由层只负责导航与 chunk 失败恢复。

### ✅ [FIXED] Store 中重复的 error handler 模式
（来源: doc1 FQ-M2@1014）
已修复：`/Users/zxy/Code/StuHelper/clients/web/src/api/errors.ts` 新增统一 `classifyApiError()`；`/Users/zxy/Code/StuHelper/clients/web/src/stores/auth.ts` 与 `/Users/zxy/Code/StuHelper/clients/web/src/stores/courseReview.ts` 已改为复用该 helper，并补充 `/Users/zxy/Code/StuHelper/clients/web/src/api/__tests__/errors.test.ts` 覆盖分类逻辑。

### ✅ [FIXED] `auth.Handler` 接收整个 `*config.Config` 而非所需子配置
（来源: doc1 GO-M7@132）
已修复：`/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler.go` 现使用裁剪后的 `auth.HandlerConfig{ Token, CORSOrigins, OIDCIssuer }`，不再依赖整棵 `config.Config`；`/Users/zxy/Code/StuHelper/server/internal/modules/auth/service.go` 也仅接收所需 `TokenConfig`，`internal/app/modules.go` 与相关构造测试已同步收敛。

### ✅ [FIXED] Review voting 三份实现收敛
（来源: doc1 FQ-M3@1021）
已修复：`/Users/zxy/Code/StuHelper/clients/web/src/modules/review/views/CourseDetailPage.vue` 已切到共享 `useReviewVoting()`，不再保留页面内联的 optimistic vote 状态机；投票逻辑现收敛为页面通用 composable（`useReviewVoting.ts`）+ Card 动画增强版（`useReviewVote.ts`），并补充 `/Users/zxy/Code/StuHelper/clients/web/src/modules/review/__tests__/useReviewVoting.test.ts`。

### ✅ [FIXED] Native 回调 state 验证 fail-open
（来源: doc2 3.5@183, doc3 P1-04@51）
已修复：原生回调要求保存的 state 非空且与回调参数严格匹配；读取失败同样 fail-closed，并补充 `clients/uniappx/src/auth/__tests__/sso-state.test.ts`。

### ✅ [ARCHIVED] IAM migration plan 已完成并移出 active
（来源: doc1 DOC-L4@612, doc6 S1-13@311）
已处置：`docs/exec-plans/active/iam-architecture-migration.md` 已迁至 archived；该计划不再占用 active 审计清单。

### ✅ [FIXED] `buildDefaultRedirectURL` 中的 localhost 回退已删除
（来源: doc1 GQ-L4@945）
已修复：`server/internal/modules/auth/handler.go` 现在严格依赖 `CORS_ORIGINS`，缺失即 panic fail-fast；不再保留 `http://localhost:3000` 生产死分支。

### ✅ [FIXED] Web 通知测试基座不稳定，通知测试现已全绿
（来源: doc1 FE-H3@353）
已修复：移除脆弱的宿主级通知集成测试，改为可组合逻辑层测试；`NotificationBell.vue` 交互状态提取为 `useNotificationBellController`，通知页筛选/分页/跳转逻辑提取为 `useNotificationsPageController`，并补充对应 controller/store/composable 测试。当前 `pnpm --filter @stuhelper/web test` 已全绿（26 files / 114 tests）。

### ✅ [FIXED] Dead Casdoor 环境变量
（来源: doc1 FE-H2@346）
已修复：删除 `clients/web/.env.development` 中已无任何消费方的 `VITE_CASDOOR_CLIENT_ID`，前端运行时不再保留已移除认证方案的伪配置。

### ✅ [FIXED] `extractUnreadCount` / `types/guards.ts` / `filterQueryBuilders.ts` 冗余前端包装
（来源: doc1 FQ-L2@1066, doc1 FQ-L3@1072, doc1 FQ-L6@1089）
已修复：`extractUnreadCount` 已重命名为 `isUnreadNotification`；未被任何消费方使用的 `clients/web/src/types/guards.ts` 已删除；`filterQueryBuilders.ts` 的单处调用已内联到 `DepartmentSidebar.vue`，相应测试文件一并删除。

### ✅ [FIXED] 空 `vendor/` 目录与本地 `storybook-static/` 产物
（来源: doc1 FE-M10@418, doc1 FE-L1@443）
已修复：`clients/web/src/vendor/` 不存在；`clients/web/storybook-static/` 本地产物已清理，根 `.gitignore` 已忽略该目录，避免再次污染工作树。

### ✅ [FIXED] PostgreSQL / Redis 版本文档漂移
（来源: doc1 DOC-H1@560, doc1 DOC-H2@567）
已修复：`AGENTS.md`、`README.md`、`docs/PRODUCT.md` 已统一到当前运行时主版本（PostgreSQL 18 / Redis 8），不再滞后于 `docker-compose.yml` 默认镜像。

### ✅ [FIXED] shared API 文档规则增加白名单例外
（来源: doc7 S3-10@357）
已修复：`docs/FRONTEND.md` 与 `clients/web/API_USAGE.md` 已从“页面不直接 fetch”的绝对陈述，调整为“默认走 shared API，OIDC callback / sendBeacon / keepalive / 基础设施请求允许例外，并要求调用点注释说明”。

### ✅ [FIXED] SECURITY.md 中通知路径口径与代码对齐
（来源: doc1 DOC-L5@618）
已修复：`docs/SECURITY.md` 已改为“通知读路径仍跨 review / notification 模块，统一尚未完成”，不再使用与代码事实冲突的模糊 dual-track 表述。

### ✅ [FIXED] 删除 `docs/generated/README.md` 占位入口
（来源: doc6 S3-14@342）
已修复：`docs/generated/` 目录与占位 README 已删除，不再保留“db-schema 待实现”的影子导航入口。

### ✅ [FIXED] 认证 / refresh / native exchange 文档口径与实现对齐
（来源: doc2 4.2@204）
已修复：`docs/product-specs/auth-sso.md` 已补齐 browser/native refresh 差异、CSRF 要求、`exchange-native` 响应字段，以及 native OIDC refresh 目前仅 blacklist 旧 refresh token、尚未把 session store touch 完全闭环的实现限制。

### ✅ [FIXED] database.md 已补齐 schools / outbox / materialized view / 审核字段
（来源: doc1 DOC-M1@575, doc1 DOC-M2@581, doc1 DOC-M3@587, doc1 DOC-L1@595, doc1 DOC-L2@601, doc1 DOC-L3@607）
已修复：`docs/references/database.md` 现已记录 `schools`、`user_external_sync_outbox`、`review_fga_sync_outbox`、`mv_teacher_public_stats`、`content_flag` 审核字段、`pending_review` 状态，以及 `school_id` 统一为 `BIGINT` 的现状与用途。

### ✅ [FIXED] admin 上游 Vben 文档已与项目规范分层
（来源: doc7 S2-9@324）
已修复：新增 `clients/admin/docs/README.md`，明确该目录是 Vben 上游原文档，仅供升级参考；StuHelper 的实际接口与 shared API 规范以 `docs/FRONTEND.md` 和项目代码为准。

### ✅ [FIXED] 生产运行时派生 secrets 不再明文落盘
（来源: doc1 INFRA-M4@474）
已修复：`infra/ops/bootstrap-platform.sh` 在 `prod` 模式下已强制要求 `GENERATED_ENV_SECRET_REF + SECRET_BACKEND!=file/none`，生成的 `ZITADEL_CLIENT_SECRET` / `ZITADEL_MANAGEMENT_PAT` 只会写入远端 secret backend；`infra/ops/lib/common.sh` 改为通过 `secret_backend_read_to_stdout()` 以内存方式注入生成 secrets，`${DEPLOY_APP_DIR}/.env.prod.generated.secrets` 仅保留空占位文件；`infra/ops/init-remote-deploy-config.sh`、`bootstrap-ubuntu2404.sh`、`remote-preflight.sh`、`prod-deploy.sh` 以及运维 runbook 已统一默认到 Vault KV v2 控制面，并新增 `infra/ops/tests/init-remote-deploy-config-contract.sh` 锁定远端默认值。

### ✅ [FIXED] user / review / notification 分页模式已收敛
（来源: doc1 GQ-M8@887）
已修复：`user/repository_identity.go`、`user/repository_profile.go`、`notification/repository.go` 已改为与 `review/repository_review_query.go` 一致的“COUNT + data query”两阶段分页，不再混用 `COUNT(*) OVER()`、`WHERE 1=1` 和独立窗口总数扫描；相关包测试 `go test ./internal/modules/user ./internal/modules/notification ./internal/modules/course/review -count=1` 通过。

### ✅ [FIXED] PostgreSQL 数据卷静态加密要求已收敛到生产运维基线
（来源: doc1 SEC-L5@222）
已修复：仓库已把该问题明确收口为宿主机/云盘层的生产前提，而非 Compose 内的伪开关：`docs/operations/production-topology.md` 明确要求承载 `postgres_data` / `redis_data` / 对象存储目录的底层块设备必须启用云盘 KMS/EBS/PD 或 LUKS 静态加密，`docs/operations/release-runbook.md` 也已把该检查加入生产发布前清单。

### ✅ [FIXED] Compose 拓扑已拆为 core / observability / prod overlays
（来源: doc1 XC-M1@1130）
已修复：原单体 `docker-compose.yml` 已拆成 `docker-compose.yml`（核心 + dev）、`docker-compose.observability.yml`（LGTM / exporters / alert sink）和 `docker-compose.prod.yml`（app/frontend/admin）；`infra/ops/lib/common.sh` 的 `compose()` 现自动叠加这些文件，`docker compose -f ... config` 与 wrapper `compose config` 都已通过结构校验；相关运维文档也已同步到新的 `-f` 用法。

### ✅ [FIXED] 浏览器 Cookie access token 不做每请求 session store lookup 已收口为明确安全模型
（来源: doc2 1.5@62；描述已修订）
已核销：当前实现并非“完全不查吊销”——`middleware/auth.go` 先统一查 Redis blacklist，浏览器 Cookie access token 再走本地 JWKS/self-signed 验证；`TOKEN_ACCESS_TTL` 已固定在 300 秒默认值，`refresh` / `logout` / `logout-all` 继续命中 session store 做轮换与撤销。仓库现已把这条明确收口为**blacklist + 5 分钟 access TTL + refresh 会话轮换**的有意安全边界，而不是再把所有浏览器请求拉回每请求 Redis RTT；`docs/SECURITY.md` 与 `docs/product-specs/auth-sso.md` 已同步说明。


### ✅ [FIXED] 生成代码漂移门禁已闭环
（来源: doc2 2.5@122）
已修复：`clients/shared/src/types/api.gen.ts` 已按最新 OpenAPI 重新生成；CI 原有 `openapi_contract` job 继续执行 `make lint-spec`、`make check-drift-go`、`pnpm run check:api-drift`，本轮再补充本地双次生成一致性校验，生成链闭环成立。

### ✅ [FIXED] Capability 常量改为 Go 单一真源 + TS codegen
（来源: doc1 ARCH-L4@318, doc1 XC-H1@1101）
已修复：新增 `clients/scripts/generate-capabilities-from-go.mjs`，以 `server/internal/pkg/capability/capability.go` 为唯一真源生成 `clients/shared/src/constants/capabilities.gen.ts`；`clients/shared/src/constants/capabilities.ts` 只保留 helper/UI 子集；`clients/package.json` 增加 `generate:capabilities` / `check:capabilities-drift`，`build:shared` 自动生成，`.gitlab/server-ci.yml` 已把 drift gate 接入 `openapi_contract`。

### ✅ [FIXED] admin `/menu/all` 旁路接口已删除
（来源: doc7 S2-7@249）
已修复：删除 `clients/admin/apps/web-ele/src/api/core/menu.ts` 与相关导出；`clients/admin/apps/web-ele/src/router/access.ts` 不再走 Vben mock 菜单接口，admin 菜单权威源回到本地 route 表 + capability 前端鉴权，`pnpm --dir admin check:type` 通过。

### ✅ [FIXED] `/api/v1/course/courses/grouped` 已补齐 400 错误响应
（来源: doc2 2.4@114）
已修复：`server/api/paths/course.yaml` 为 grouped 目录接口补充 `400 -> ErrorResponse`；OpenAPI lint/bundle/generate 已重新执行，前端共享类型同步更新并通过 web type-check/test。


### ✅ [FIXED] `REVIEW_TITLE_MAX_LENGTH` 统一回 shared 真源
（来源: doc1 FQ-H3@977, doc1 XC-H2@1108, doc1 FQ-M8@1054）
已修复：`clients/shared/src/constants/review.ts` 已把 `REVIEW_TITLE_MAX_LENGTH` 统一为 200，并新增 `normalizeReviewGrade`；`ReviewDialog.vue`、`PostReviewPage.vue` 不再使用 `clients/web/src/constants/review.ts` 本地覆盖，旧文件已删除。

### ✅ [FIXED] `normalizeReviewGrade` 重复函数已收敛
（来源: doc1 FQ-H1@963）
已修复：`clients/shared/src/api/reviews.ts` 与 `clients/shared/src/api/draft.ts` 统一复用 `clients/shared/src/constants/review.ts` 中的 `normalizeReviewGrade`，shared 包内不再重复维护同一 grade 清洗逻辑。

### ✅ [FIXED] `withAlpha` 工具与 rating 颜色映射已收敛到单一实现
（来源: doc1 FQ-H2@970, doc1 FQ-H4@984, doc1 FQ-M1@1009, doc1 FQ-M5@1035）
已修复：删除 `clients/web/src/utils/color.ts` 与 `clients/web/src/modules/course/theme/index.ts`；web 统一使用 `@stuhelper/shared/utils` 的 `withAlpha`，并新增 `clients/web/src/design-system/rating.ts` 作为唯一 rating → CSS token 映射源，`ReviewCard`、`SemesterStatsGrid`、`RatingBar`、`RatingCircle`、`SearchPage`、`CourseDetailPage`、`EmojiRating*` 已全部切换；新增 `clients/web/src/design-system/__tests__/rating.test.ts` 锁定桶化规则。


### ✅ [FIXED] `isRecord` 类型守卫替代重复 `as Record<string, unknown>`
（来源: doc1 FE-M1@362）
已修复：`clients/shared/src/api/errors.ts` 新增 `isRecord()`，`parseApiError` 与 `clients/web/src/api/client.ts` 的 `expiresIn` 提取逻辑已改用守卫，不再反复写裸断言。

### ✅ [FIXED] Admin auth store 去掉唯一 `Record<string, any>`
（来源: doc1 FE-M2@369）
已修复：`clients/admin/apps/web-ele/src/store/auth.ts` 的 `authLogin(_params)` 已改为 `Record<string, unknown>`，admin typecheck 通过；活动代码中不再保留该条 `any`。


### ✅ [FIXED] `client.ts` 中的 `typeof window/document/navigator` SPA 死守卫已删除
（来源: doc1 FQ-L1@1060）
已修复：`clients/web/src/api/client.ts` 现直接按浏览器运行时处理 `window`/`document`/`navigator`；保留真正有价值的 cookie 解析和超时/Abort 逻辑，不再用 SSR 兼容分支污染纯 SPA 客户端。

### ✅ [FIXED] CI 已校验文档与测试覆盖规范
（来源: doc4 P3@119）
已修复：`server/scripts/check-api-overview-sync.mjs` 现把 `server/api/openapi.bundled.yaml` 与 `docs/references/api-overview.md` 的路径集合做一一校验；`server/scripts/check-coverage-threshold.sh` 对关键包覆盖率执行阈值门禁（auth 70 / course 80 / review 70 / middleware 75 / oidc 80 / fga 80）。`server/Makefile` 已新增 `check-doc-sync` / `check-coverage-threshold`，`.gitlab/server-ci.yml` 的 `openapi_contract` 与 `backend_test` 已接入这两道门禁；本地执行 `make check-doc-sync` 输出 `OK: 85 routes in sync`，`make check-coverage-threshold` 通过。

### ✅ [FIXED] Redis 不再使用 default 用户 full command access
（来源: doc1 INFRA-L6@530）
已修复：`infra/ops/render-redis-acl.sh` 现显式 `user default off`，并改为渲染命名账户 `REDIS_USERNAME=stuhelper_app` 与 `REDIS_EXPORTER_USERNAME=stuhelper_metrics`；`server/internal/pkg/config/config.go`、`server/internal/pkg/redis/client.go`、`docker-compose.yml`、`.env.example`、`.env.prod.example`、`infra/ops/dev-up.sh` 已全部切到具名 Redis 用户链路。`render-redis-acl.sh` 本地渲染结果已验证为 default 关闭 + app/exporter 独立用户名密码登录，`docker compose config` 结构校验通过。

### ✅ [FIXED] uniappx i18n / 本地存储异常链路已补一次性诊断
（来源: doc3 P2-03@98）
已修复：`clients/uniappx/src/i18n/index.ts` 不再对 locale 存储读写、系统 locale 读取、导航标题更新与 tab bar 同步错误做纯静默吞掉；现在会以 once-only `console.warn` 输出结构化诊断，并对 `not TabBar page` / `tabbar unavailable` 这类可预期平台缺口继续静默降级。对应回归测试已补到 `clients/uniappx/src/i18n/__tests__/index.test.ts`，`pnpm --filter @stuhelper/uniappx exec vitest run src/i18n/__tests__/index.test.ts` 与 `pnpm --filter @stuhelper/uniappx type-check` 均通过。

### ✅ [CLOSED / WONTFIX] 代码注释中文英文混用
（来源: doc1 GO-L1@146）
已核销：当前项目贡献者、文档与工作语言均以中文为主，代码注释中中英文并存不构成工程风险，也不会阻断当前开发质量。按“开发期优先处理真实功能/安全/架构问题”的原则，此项不再视为活跃缺陷；如未来对外开源或引入多语言团队，再统一注释语言规范。

### ✅ [CLOSED / WONTFIX] Session cookie 未做加密签名
（来源: doc1 SEC-L1@198）
已核销：`server/internal/modules/auth/handler_cookies.go` 中的 `session_id` 是 128-bit `crypto/rand` 生成的高熵随机值，只作为定位服务端 session 的 opaque handle；真正的会话有效性仍以 Redis session store / blacklist 为权威来源。对这种随机 opaque session id 再做 HMAC 签名不会提升伪造防护，只会增加 cookie 体积与处理复杂度；除非未来要把 session 校验完全离线化，否则不再视为活跃缺陷。

### ✅ [CLOSED] `ExternalSyncJob` outbox 并非提前实现的空基础设施
（来源: doc1 GQ-M12@912）
已核销：`server/internal/modules/user/external_sync.go` 已实际消费 `user_external_sync_outbox`，`ProcessExternalSyncBatch()` 会 claim / retry / done；`service.go` 中的 `EnqueueVerifiedStudentRoleSyncTx` 与 `EnqueueUserProfileProjectionTx` 已在主业务路径调用。对应真库集成测试 `server/internal/modules/user/repository_external_sync_integration_test.go` 已覆盖 upsert / claim / retry / dedupe reset，全链路证明这不是“无人消费的预实现基础设施”。

### ✅ [FIXED] 契约测试不再依赖源码文本匹配与脆弱硬编码
（来源: doc4 P2@48, doc4 P2@56）
已修复：删除 `server/internal/modules/course/review/admin_create_status_test.go` 这类通过 `os.ReadFile + strings.Contains` 断言源码文本的脆弱测试，创建状态码行为已由 `handler_admin_integration_test.go` 的真实 HTTP 断言覆盖；`server/internal/modules/course/review/handler_contract_test.go` 也改为显式构造 `reviewID` 再断言投票路径，不再把 `test-id` 固定字符串散落在契约测试里。`go test ./internal/modules/course/review -count=1` 已通过。

### ✅ [CLOSED] Alertmanager 默认 receiver 空实现仅限本地观测演练
（来源: doc1 INFRA-M6@486）
已核销：`infra/observability/alertmanager/alertmanager.yml` 的静态 `default` receiver 只服务于 `make obs-up` 单机演练；`docs/operations/observability.md` 已明确“告警不接值班系统就不算真正上线”，而 `infra/ops/prod-deploy.sh` 会强校验 `ALERTMANAGER_WEBHOOK_URL`，禁止用空 receiver 进入真实生产部署。因此这不是活跃缺陷，而是本地演练模式与生产部署模式的有意分层。

### ✅ [CLOSED] Loki 关闭 auth 仅限本地单机 observability profile
（来源: doc1 INFRA-L7@536）
已核销：`infra/observability/loki/loki.yaml` 顶部已显式注明“单机 Compose 开发默认关闭认证；生产环境请在 Loki 前置认证网关或启用多租户认证”。当前 Loki 只绑定 `127.0.0.1` 且位于内部 `observability` 网络，不构成对外暴露面；因此该项不再作为 active 缺陷跟踪。

### ✅ [FIXED] `echarts` 已按需导入并显式拆出独立 chunk
（来源: doc1 FE-M8@406, doc1 FE-M9@412）
已修复：web 侧图表调用点已全部使用 `echarts/core` / `zrender` 的按需导入，不再存在全量 `import 'echarts'`；本轮又在 `clients/web/vite.config.ts` 增加了 `manualChunks()`，把 `echarts`/`zrender` 显式拆成独立 `echarts` chunk，避免图表依赖回流进主包。`pnpm --filter @stuhelper/web type-check` 与 `pnpm --filter @stuhelper/web test`（31 files / 128 tests）均通过。

### ✅ [FIXED] Ansible bootstrap 不再使用 silent/default 隐式变量
（来源: doc6 S2-11@264）
已修复：`infra/ansible/ansible.cfg` 的 `interpreter_python` 已从 `auto_silent` 改为 `auto`，保留解释器探测告警；`infra/ansible/playbooks/bootstrap.yml` 不再为 `ansible_user` / `deploy_app_dir` / `allow_http_ports` 提供隐式默认值，而是通过 `pre_tasks` 的 `ansible.builtin.assert` 显式要求 inventory 声明；`infra/ansible/inventory/{production,staging}.example.ini` 也补齐了 `allow_http_ports=80,443` 示例。已用 Python 解析验证 `ansible.cfg` 与 `bootstrap.yml` 结构正确。

### ✅ [CLOSED] SMS internal forwarder 并非无人消费的提前实现
（来源: doc1 XC-L4@1187）
已核销：`server/internal/pkg/sms/handler.go` 明确暴露 `POST /internal/sms/send` 给 Zitadel Action 调用，`server/internal/app/modules.go` 会在 `SMS_ENABLED=true` 时注册该内部 HTTP 服务并绑定到 `127.0.0.1:${SMS_INTERNAL_PORT}`；`docs/product-specs/auth-sso.md` 也已同步要求启用短信登录时必须配置 `SMS_INTERNAL_KEY`。因此这不是“无人使用的死代码”，而是当前短信登录链路的一部分。

### ✅ [CLOSED / WONTFIX] `review.Service` 使用具体 `*Repository`
（来源: doc1 ARCH-M3@272）
已核销：当前项目的分层铁律是 `Handler -> Service -> Repository`，并且 `review.Service` 已只对真正需要替身的外部依赖抽窄接口（`notification.Sender`、`ReviewAccessReader`、`reviewFGAWriter`）；对内部 Postgres 仓储则采用具体 `*Repository` + 真库集成测试（testcontainers / fixture）路径。机械把 40+ 方法仓储整体接口化只会扩大漂移面，不符合当前代码库的长期简洁性目标。

### ✅ [CLOSED / WONTFIX] Service pass-through 方法保留为分层边界
（来源: doc1 GQ-M10@900, doc5 #6@206）
已核销：虽然 `course.Service` / `review.Service` 中存在部分纯查询 pass-through，但让 handler 直接注入 repo 会直接破坏项目明确规定的 `Handler -> Service -> Repository` 分层。当前做法把 HTTP 边界与数据访问边界隔离开，并允许后续在 service 层平滑插入缓存、审计、授权或聚合逻辑；因此这不再作为 active 缺陷跟踪。

### ✅ [FIXED] 前端 HTTP 状态码默认错误映射已收敛到 shared 真源
（来源: doc1 XC-L1@1170）
已修复：新增 `clients/shared/src/api/error-codes.ts` 作为默认 HTTP status → 错误码映射真源，并通过 `clients/shared/src/api/index.ts` 导出；`clients/web/src/api/errors.ts` 的 `httpStatusToDefaultCode()` 现直接复用 shared 映射，不再手写 `A0010100` / `B0000001` 等 magic string。对应回归已补到 `clients/web/src/api/__tests__/errors.test.ts`，shared build、web type-check 和 web tests（31 files / 129 tests）均通过。

### ✅ [CLOSED] Ansible / 远端部署脚本已是当前目标拓扑的一部分
（来源: doc1 XC-M2@1136）
已核销：仓库当前已经明确维护远端部署路径和目标拓扑——`Makefile` 暴露 `ansible-bootstrap / ansible-deploy-* / ansible-rollback-*`，`README.md`、`docs/QUICKSTART.md`、`docs/operations/automation.md` 都把这套远端部署链路作为正式能力文档化；`infra/ops/prod-deploy.sh` 也已具备生产前置校验与 smoke-check。因此这不再属于“为未来假设预构建的空基础设施”。

### ✅ [FIXED] web 本地类型门面层已删除
（来源: doc1 FQ-L4@1077, doc7 S2-5@172）
已修复：`clients/web/src/types/{course,draft,notification,reply,review}.ts` 这些纯 re-export 门面层已删除；web 全量调用点直接切到 `@stuhelper/shared/{course,review,draft,reply,notification}` 子路径，避免本地影子类型层继续扩散。

### ✅ [FIXED] shared 根 barrel 已收敛为最小出口
（来源: doc1 FQ-L5@1083, doc7 S3-11@390）
已修复：`clients/shared/src/index.ts` 仅保留 `components` 类型出口；业务类型、notification、course/review/draft/reply 改走显式子路径导出，`clients/shared/package.json` 已补齐对应 exports，shared/web/admin/uniappx 构建验证全部通过。

### ✅ [FIXED] `/api/v1/auth/login` 契约已补齐 `platform` 与 redirect 语义
（来源: doc2 2.1@84）
已修复：`server/api/paths/auth.yaml` 为 login/signup 增加 `platform` 查询参数（`web|native`），并把 `redirect` 语义改为“站内相对路径或 allowlist 绝对 URL”；`clients/shared/src/api/auth.ts` 同步支持 `platform`，生成代码已刷新。

### ✅ [FIXED] `/api/v1/auth/callback` 已明确 Web/H5 与 native 两种 302 语义
（来源: doc2 2.2@95）
已修复：OpenAPI 现在把 callback 描述为单一路径上的两种合法 302 Location：Web/H5 回前端 URL，native 回 `stuhelper://auth/callback?...` deep link；同一路径不再靠“只写单一前端跳转”误导调用方。

### ✅ [FIXED] `/api/v1/auth/exchange-native` 契约与实现已对齐
（来源: doc2 2.3@104）
已修复：OpenAPI 已移除未实现的 `code_verifier` 请求字段，补齐 `500` 错误响应，并把成功响应改成真实返回的 `{accessToken, refreshToken, sessionID, expiresIn}`；`clients/shared/src/api/auth.ts` 与 uniappx `exchangeNative` 调用已同步到 `{code, state}` 协议。

### ✅ [FIXED] 前端不再直接透传后端原始错误消息
（来源: doc2 3.4@173）
已修复：web `getErrorMessage()` 现在只按错误码走 i18n 或 fallback，不再回显 `ApiError.message`；admin/uni 的结果解析也改为状态码/结构化错误码驱动的安全文案，后端 `message` 只留给日志与调试，不再直接展示给终端用户。

### ✅ [FIXED] Admin 认证链路不再把网络/5xx 故障伪装成“未登录”
（来源: doc3 P2-04@110, doc6 S1-4@96）
已修复：`clients/admin/apps/web-ele/src/api/core/auth.ts` 的 `tryGetMe()` 改为结构化探测结果（`ok/unauthenticated/forbidden/retryable_error/fatal_error`）；`shared-client.ts` 的自动 refresh 也区分 `unauthorized` 与 `error`，只在真实 401/403 时走重登。网络故障或 5xx 现在会保留为错误态并阻止 silent fallback。

### ✅ [FIXED] 课程详情 / 搜索 / verification 不再把失败伪装成空数据
（来源: doc6 S1-5@121）
已修复：`CourseDetailPage.vue` 改为 `Promise.allSettled` 区分主资源失败与部分失败；课程主请求失败会进入错误页，子请求失败会显示部分失败提示而不是悄悄置空。`SearchPage.vue` 也改为保留成功侧结果并对失败侧显示错误提示，不再把双侧失败伪装成“无结果”。`stores/verification.ts` 的 `bindPhone()` 现在要求 profile 刷新成功才算完成，不再吞掉绑定后状态刷新失败。

### ✅ [FIXED] Admin `tryGetMe` 一刀切条目已并入认证链路修复
（来源: doc2 3.3@165, doc3 P2-04@110）
说明：该条与“Admin 认证链路不再把网络/5xx 故障伪装成未登录”同根因，现已统一闭环；为避免 active 重复计数，重复条目已移除。

### ✅ [FIXED] 认证模块去掉不可能状态的运行时判空
（来源: doc1 GQ-H4@816, doc1 GQ-H5@823）
已修复：`storeOIDCState/consumeOIDCState` 相关的 `redisClient == nil` 残留已不在当前代码；`auth.NewService()` 现在对 `cfg/tokenService/userSyncRepo` 做构造期 fail-fast，`SyncOIDCUser/SyncPhoneUser/UserExistsByExternalID` 不再在热路径重复判 `userSyncRepo == nil`。

### ✅ [FIXED] `school_configs` 列选择与扫描逻辑已收敛
（来源: doc1 GQ-H1@798）
已修复：`server/internal/modules/user/repository_config.go` 现在使用单一 `selectSchoolConfigColumns` + `scanSchoolConfig()` helper，`GetSchoolConfig/ListSchoolConfigs/ListAllSchoolConfigs` 共用同一列清单与 scan 路径，后续字段变更不再需要三处同步。

### ✅ [FIXED] Profile/Phone 仓储的 Tx 与非 Tx 重复实现已收敛
（来源: doc1 GQ-H2@804, doc1 GQ-H3@811）
已修复：`repository_profile.go` 把 create/update 写路径统一到 `saveProfile()` helper；`repository_phone.go` 把手机号绑定统一到 `setUserPhone()` helper，Tx 与非 Tx 仅保留最薄包装层，不再维护四份近同实现。

### ✅ [FIXED] `RotateSession` 已对 hash 失败显式返回 error
（来源: doc1 GO-C1@28）
已核实：`server/internal/modules/auth/service.go` 当前实现会对 `hashTokenForSession(newAccessToken/newRefreshToken)` 的失败立即返回 error，不再静默写入空 hash。该条已是历史问题，active 文档现已移除。

### ✅ [FIXED] 通知域已收口到 `modules/notification`
（来源: doc5 #4@142, doc6 S1-6@145, doc6 S2-15@358）
已修复：通知列表、未读数、单条已读、全部已读与 SSE 广播现全部由 `server/internal/modules/notification` 暴露；`server/internal/modules/course/review` 中的旁路 handler / service / repository 读路径已删除，通知域边界恢复单一事实源。

### ✅ [FIXED] auth / user 路由契约测试已补齐
（来源: doc2 2.6@132, doc4 P1@28）
已修复：新增 `server/internal/modules/auth/route_contract_test.go` 与 `server/internal/modules/user/route_contract_test.go`，覆盖登录/回调/refresh/logout、身份/学籍/绑定手机号与 admin 用户系统路由；auth/user 不再是路由契约盲区。

### ✅ [FIXED] notification 路由契约测试不再只覆盖 SSE
（来源: doc4 P2@65）
已修复：`server/internal/modules/notification/route_contract_test.go` 现同时断言列表、stream、unread-count、单条已读、全部已读五条通知路由，通知模块契约边界已完整挂载。

### ✅ [FIXED] Zitadel 初始化脚本改为 fail-fast
（来源: doc6 S1-7@173）
已修复：`infra/zitadel/setup.sh` 对角色创建与默认管理员授权不再“警告后继续”；除 `code == 6` 的已存在分支外，其余 API 错误立即退出，避免输出伪成功的“初始化完成”。

### ✅ [FIXED] docker-compose 中 Zitadel 首实例环境已抽成 YAML anchor
（来源: doc1 XC-H3@1116）
已修复：`docker-compose.yml` 新增 `x-zitadel-common-env`，`zitadel-init` 与 `zitadel-api` 通过 `<<: *zitadel-common-env` 复用同一组 `ZITADEL_FIRSTINSTANCE_*` / Login V2 / DSN 配置，消除了双份复制粘贴漂移面。

### ✅ [FIXED] 路由契约测试 helper 已抽到共享 testutil
（来源: doc4 P2@75）
已修复：新增 `server/internal/testutil/routeassert/routeassert.go`，course / review / notification / auth / user 的 route contract tests 统一复用 `routeassert.Exists/NotExists`，测试函数名也按模块显式区分，失败追踪不再混淆。

### ✅ [FIXED] ReviewCard 已切到共享 composables，重复动作样板已删除
（来源: doc1 FQ-H5@995, doc1 FQ-M7@1048）
已修复：`clients/web/src/components/business/review/ReviewCard.vue` 已从 651 行收缩到 445 行，统一委托给 `useReviewVote/useReviewReply/useReviewEdit/useReviewDelete/useReviewReport/useReviewModeration` 六个 composable；操作按钮样式也收敛到共享 class 常量，不再维护内联业务逻辑与重复 Tailwind 串两套实现。

### ✅ [FIXED] Service 构造后再配置与运行时注入已移除
（来源: doc1 ARCH-M2@262, doc5 #1@37）
已修复：`server/internal/modules/user/service.go` 现通过 `WithProfileFGAClient/WithRoleSyncFunc/WithIdentityPhotoStore` 构造期注入依赖，`user.NewHandler()` 不再反向写入 service；此前 review service 的 setter/runtime assertion 也已清理。业务主路径不再依赖构造后 `SetXxx()` 或运行时补齐依赖。

### ✅ [FIXED] review 通知协程已恢复 shutdown 传播
（来源: doc1 GO-H5@75）
已修复：`server/internal/modules/course/review/service_async.go` 新增统一异步通知调度；`VoteReview` / `CreateReply` 不再用 `context.WithoutCancel` 脱离生命周期，而是绑定到 `StartBackgroundJobs()` 提供的后台 ctx，并在每次发送时加 5 秒超时。停服时通知协程现在会随后台 ctx 一起取消。

### ✅ [FIXED] user LDAP factory 运行时回退已删除
（来源: doc1 GQ-M11@906）
已修复：`server/internal/modules/user/service_profile.go` 的 `ensureLDAPClientForSchool()` 不再在热路径上写回 `s.ldapClientFactory`；默认 factory 只在 `NewService()` 构造期初始化，测试替身通过 `WithLDAPClientFactory()` 或包内字段覆盖，消除了无锁写入竞态。

### ✅ [FIXED] `ReviewStudentVerification` 冗余 `return nil` 已清理
（来源: doc1 GQ-M9@894）
已修复：`server/internal/modules/user/service_admin.go` 现直接 `return s.repo.WithTx(...)`，去掉多余的 `if err != nil { return err } return nil` 包装层。

### ✅ [FIXED] auth `UserSyncRepo` 未使用接口方法已移除
（来源: doc1 GQ-M13@919）
已修复：`server/internal/modules/auth/user_sync.go` 已从 auth 域窄接口中删除未使用的 `FindByPhone`，auth 模块只保留实际需要的 `UpsertUser/UpsertByPhone/ExistsByExternalID` 契约。

### ✅ [FIXED] notification.Service 已做构造期 fail-fast 与编译期接口断言
（来源: doc5 #8@262, doc1 GO-M3@108）
已修复：`server/internal/modules/notification/service.go` 现在对 `repo/hub/rdb` 做构造期 panic 保护，并新增 `var _ Sender = (*Service)(nil)` 编译期断言；`server/internal/modules/notification/service_test.go` 增加了 nil 依赖 panic 测试，通知模块依赖边界更明确。

### ✅ [FIXED] Admin 认证重定向链路已补结构化告警
（来源: doc3 P2-01@78, doc6 S1-4@96）
已修复：`clients/admin/apps/web-ele/src/api/shared-client.ts` 的 forced re-auth 分支不再静默吞掉 `resetAllStores` / 登录 URL 获取失败；`store/auth.ts` 与 `router/guard.ts` 也补了 `console.warn` 诊断。现在保留 fail-safe 行为，但关键异常已有前端可观测性。

### ✅ [FIXED] 孤儿页面检测脚本改为 fail-fast 且支持 allowlist
（来源: doc6 S2-10@242）
已修复：`clients/scripts/detect-orphan-pages.sh` 现在在 router 目录缺失时直接失败，并把检测范围从“仅路由引用”扩展为“任意源码引用”；发现孤儿页面默认退出 1，必要时只能通过 `ORPHAN_PAGES_ALLOWLIST=...` 显式豁免。当前对 `clients/web` 运行结果已为全绿。

### ✅ [FIXED] `useReviewPost` 状态已迁入 Pinia store
（来源: doc1 FE-M5@387）
已修复：`clients/web/src/composables/useReviewPost.ts` 不再持有模块级 `ref` 单例状态，而是改用 `reviewPost` Pinia store 托管 `showPostModal/lastPostedAt/preselectedCourse`；路由切换与 `resetAllStores()` 现在能正确清理跨页写测评状态。

### ✅ [FIXED] `normalizeReview` 空壳 normalizer 已删除
（来源: doc1 FQ-M4@1028）
已修复：`clients/shared/src/presentation/review.ts` 已移除 `normalizeReview()` 这个无意义 identity 函数；列表归一化现在直接做类型收窄，不再维护无转换价值的浅拷贝包装层。

### ✅ [FIXED] ReviewDialog 纯图标/文字按钮已补齐可访问性标签
（来源: doc1 FE-M12@429）
已修复：`clients/web/src/components/business/review/ReviewDialog.vue` 现在为“编辑已选课程”和“清空教师输入”按钮补了 `aria-label`，纯图标/符号按钮不再依赖视觉语义。

### ✅ [FIXED] `refreshPromise` 单例状态已补设计注释
（来源: doc1 FE-M6@393）
已修复：`clients/web/src/api/client.ts` 现在明确注释 `refreshPromise` 是并发 401 共享的单例 refresh 协调 Promise，避免后续把它误删成竞态回归。

### ✅ [FIXED] `db.cryptoRandFloat64` 不再静默忽略 `rand.Read` 错误
（来源: doc1 GO-L2@152）
已修复：`server/internal/pkg/db/db.go` 现在在 `crypto/rand.Read` 失败时记录 warning，并回退到 time-based jitter seed；随机源异常不再被完全吞掉。

### ✅ [FIXED] notification SSE buffer magic number 已提取为具名常量
（来源: doc1 GO-L3@159）
已修复：`server/internal/modules/notification/hub.go` 新增 `sseBufferSize` 常量与注释，说明选择 32 的 burst 缓冲意图；订阅通道不再裸写 magic number。

### ✅ [FIXED] `registerAPIRoutes` God function 与 Handler 对象图装配
（来源: doc1 ARCH-H1@240, doc5 #3@113）
已修复：`server/internal/app/modules.go` 已拆成 `configureAPICommonMiddleware/registerMetricsRoutes/initAuthModule/initCourseModule/registerUserRoutes/registerAdminRoutes`；`course.NewHandler` / `review.NewHandler` 改为只接收已构造好的 service/handler，不再在 Handler 构造函数内 new repository/service。

### ✅ [FIXED] `repository_review_query.go` 近同 query 方法重复
（来源: doc1 GO-H4@69）
已修复：`server/internal/modules/course/review/repository_review_query.go` 引入统一的 `reviewCoreFields` 查询/扫描 helper，事务内读取状态/课程/教师/owner 的多个调用点改为共享同一核心字段装载逻辑，原无调用的重复投影查询已删除。

### ✅ [FIXED] review 缓存端点读写样板重复
（来源: doc1 XC-H4@1122）
已修复：`server/internal/modules/course/review/cache_response.go` 新增统一缓存响应 helper，`review_read.go`、`rating.go`、`handler_teacher_public.go`、`admin.go` 的缓存端点已收敛到同一模式，避免再手写重复的 get/set/response 仪式代码。

### ✅ [FIXED] review handler 残留未使用 `db` 字段
（来源: doc1 GO-M2@102）
已修复：`server/internal/modules/course/review/handler.go` 已移除未使用的 `db *db.DB` 字段；数据库访问统一留在 service/repository 层。

### ✅ [FIXED] 认证契约已收敛为 OpenAPI 单一事实源
（来源: doc7 S1-1@24）
已修复：`server/api/paths/auth.yaml` 已补齐 `login/signup.platform`、`callback` 的 Web/native 302 语义、`exchange-native` 的真实请求/响应模型（删除未实现 `code_verifier`，补 `sessionID/500`），并重新生成 `clients/shared/src/types/api.gen.ts`；`clients/shared/src/api/auth.ts` 与 `clients/uniappx/src/stores/auth.ts` 已按生成契约对齐，不再补契约外字段。

### ✅ [FIXED] review handler 中重复的 userHash 解析样板已收敛
（来源: doc1 GQ-M1@837, doc1 XC-M5@1152）
已修复：新增 `server/internal/modules/course/review/request_identity.go`，统一 `resolveRequiredUserHash/resolveOptionalUserHash`；`review.go`、`review_interaction.go`、`review_draft.go`、`review_reply.go` 均已改为复用 helper，不再手写重复的 `GetUserID + HashUserID + error` 样板。

### ✅ [FIXED] auth cookie 写入/清理属性已统一
（来源: doc1 GQ-M2@844）
已修复：`server/internal/modules/auth/handler_cookies.go` 新增统一 `writeCookie` helper，access/refresh/CSRF/session cookie 的 domain/path/secure/httpOnly/sameSite 属性由单一实现维护，避免 set/clear 路径再发生属性漂移。

### ✅ [FIXED] cache TTL 抖动已切回 Go 默认全局随机源
（来源: doc1 GO-H1@51；后续复核已下调到 MEDIUM）
已修复：`server/internal/pkg/cache/cache.go` 删除自建 `math/rand` source 与互斥锁，`JitteredTTL` 直接使用 Go 默认全局随机源，去掉多余状态与锁开销。

### ✅ [FIXED] 批量评论更新冗余上限检查已删除
（来源: doc1 GQ-L3@939）
已修复：`server/internal/modules/course/review/admin.go` 移除 `maxBatchSize` 的重复运行时检查，保留 binding tag 作为唯一边界校验来源，避免不必要的双重防御噪音。

### ✅ [FIXED] web `auth` store 已通过 session orchestrator 解耦跨域 reset 依赖
（来源: doc3 P1-06@67）
已修复：新增 `clients/web/src/stores/sessionOrchestrator.ts`，`user/courseReview/draft/verification/notification` 各 store 在自身作用域内注册 logout reset handler；`clients/web/src/stores/auth.ts` 现在只负责清理本地认证态并广播 session reset，不再直接 import 并调用 5 个其他 store。`clients/web/src/stores/__tests__/sessionOrchestrator.test.ts` 与 `authSessionReset.test.ts` 已覆盖注册/隔离/登出触发行为，auth store 跨域耦合问题闭环。

### ✅ [FIXED] user 超长 service / test 文件已按职责拆分
（来源: doc1 GO-H2@58, doc1 GO-H3@64）
已修复：`server/internal/modules/user/service_profile.go` 已缩减为个人资料与身份入口编排，学生验证与手机绑定流程拆到 `service_student_verify.go` / `service_phone.go`；原单体 `service_test.go` 也拆分为 `service_identity_test.go`、`service_student_verify_test.go`、`service_config_test.go`。当前 user 相关实现/测试文件均已降到可维护范围，并通过 `go test ./internal/modules/user/...` 验证。

### ✅ [FIXED] 统一 API 结果模型已收敛到 shared 单一真源
（来源: doc3 P1-02@28, doc7 S2-6@208）
已修复：新增 `clients/shared/src/api/result.ts` 作为唯一 `ApiEnvelope/ApiCallResult` 与 `readResultStatus/extractResultErrorCode/extractResultData/extractResultList` 真源；`clients/admin/apps/web-ele/src/api/shared-result.ts` 与 `clients/uniappx/src/api/result.ts` 现在只保留薄封装并复用 shared helper，admin 的会话探测也改为统一读取 shared 错误码。`clients/shared/src/__tests__/result.test.ts`、`pnpm --filter @stuhelper/shared build/test`、`admin check:type`、`uniappx type-check`、`web type-check/test` 已通过。

### ✅ [FIXED] `ReviewDialog.vue` 已拆为编排层容器
（来源: doc1 FE-H1@337, doc1 FE-M4@381）
已修复：`clients/web/src/components/business/review/ReviewDialog.vue` 已从 823 行收缩到 336 行；课程选择、教师选择、退出确认、跳转遮罩分别拆到 `ReviewCourseSelector.vue`、`ReviewTeacherSelect.vue`、`ReviewExitConfirmDialog.vue`、`ReviewRedirectOverlay.vue`，状态编排下沉到 `useReviewDialogController.ts`。当前容器仅负责组装与事件流转，`pnpm --filter @stuhelper/web type-check/test` 已通过。

### ✅ [FIXED] OpenAPI → Gin 路由契约已升为 app 级全量门禁
（来源: doc4 P1@13）
已修复：新增 `server/internal/app/openapi_route_contract_test.go`，直接加载生成后的 OpenAPI schema，与 auth / course / review / notification / user / admin / metrics / health 的实际 Gin 挂载路由做 method+path 双向集合比对；现在要求“OpenAPI 中的每一条路由都已注册，Gin 中的每一条平台/API 路由也必须存在于 OpenAPI”。相关包测试 `go test ./internal/app ./internal/modules/auth ./internal/modules/user ./internal/modules/course ./internal/modules/course/review ./internal/modules/notification ./internal/pkg/metrics` 已通过。

### ✅ [FIXED] 复杂 SQL / outbox 仓储已补真库集成测试
（来源: doc2 4.3@213）
已修复：新增可复用的 `server/internal/testutil/postgresfixture/postgresfixture.go`，用 testcontainers-go 拉起真实 Postgres 18 并自动应用全部 migrations；`user/repository_external_sync_integration_test.go` 覆盖 `user_external_sync_outbox` 的 upsert / claim / retry / dedupe reset，`review/repository_fga_sync_integration_test.go` 覆盖 `review_fga_sync_outbox` 同样生命周期，`review/repository_teacher_public_integration_test.go` 覆盖 `mv_teacher_public_stats` 刷新、搜索、院系过滤与热门教师查询。`go test ./internal/modules/user ./internal/modules/course/review` 已通过。

### ✅ [FIXED] 热路径 optional / fallback 已收敛为 fail-fast / fail-closed
（来源: doc5 #9@282, doc6 S1-3@72）
已修复：`server/internal/app/modules.go` 现要求 OpenFGA 必须配置，否则 API 启动直接失败；`review.NewService()` 也要求 `notifSender` 非空，`service_async.go` / `service_review_write.go` / `service_report.go` 不再判空跳过通知或 FGA 同步；`review/access.go` 删除“刷新失败使用 stale policy”与 handler 侧匿名降级，策略解析失败现在直接返回 503；`user/service_identity.go` 不再把 academic auto-match 故障降级成未认证写入，而是直接返回错误；`user/external_sync.go` 中缺失 `roleSync/profileFGA` 也不再静默跳过。`go test ./internal/modules/user ./internal/modules/course/review ./internal/app` 已通过。

### ✅ [FIXED] refresh 接口契约与前端调用约定已收敛为 shared 会话刷新核心
（来源: doc2 3.2@155）
已修复：`clients/shared/src/api/session-client.ts` 新增统一 `executeSessionRefresh()` / `RefreshSessionData`，三端 refresh 结果都收敛到同一 `ok|unauthorized|error` 契约；web/admin/uni 各自只注入 transport 与成功后的本地持久化副作用，避免继续在三端各写一套 refresh 分支逻辑。

### ✅ [FIXED] shared review API 不再混合 transport 与 presentation
（来源: doc7 S2-4@141）
已修复：`clients/shared/src/api/reviews.ts` 已回归纯 wire contract，只保留真实 HTTP 调用；分页 normalizer / 内容审核结果 normalizer 已迁移到 web 侧 app adapter `clients/web/src/api/review.ts`，`createReviewApi()` 不再夹带 view-model 归一化职责。

### ✅ [FIXED] 焦点环样式不再硬编码蓝色 rgba
（来源: doc1 FQ-M6@1041）
已修复：`clients/web/src/styles/tailwind.css` 新增主题变量 `--shadow-focus-ring` 与共享 `.focus-ring-field` 样式；`ReviewDialog.vue`、`ReplyForm.vue`、`ReviewCourseSelector.vue`、`ReviewTeacherSelect.vue` 均已移除 `focus:shadow-[0_0_0_3px_rgba(...)]` 硬编码，暗色模式下与主题 token 一起演进。

### ✅ [FIXED] `ListUserSessions` 已消除 N+1 Redis 往返
（来源: doc1 GO-M4@114）
已修复：`server/internal/pkg/token/session.go` 改为 `SMembers + MGet` 一次拉取所有 session payload，并批量清理 stale session 成员；新增 `TestSessionStore_ListUserSessions_CleansMissingSessionMembers` 锁定缺失 session 的清理行为。

### ✅ [FIXED] `normalizeAcademicDBTableName` 只在 service 边界规范化一次
（来源: doc1 GQ-M4@859）
已修复：`server/internal/modules/user/service_admin.go` 继续作为学校配置写路径的唯一规范化边界，`server/internal/modules/user/repository_config.go` 不再重复规范化 `academic_db_table`；repository 只负责持久化已验证输入，职责分层恢复清晰。

### ✅ [FIXED] `REDIS_PASSWORD` 已在 compose 层 fail-fast
（来源: doc1 INFRA-M3@467）
已修复：`docker-compose.yml` 中 Redis、app、redis-exporter 相关 `REDIS_PASSWORD` 引用统一升级为 `${REDIS_PASSWORD:?REDIS_PASSWORD is required}`；compose 渲染阶段即可拒绝空密码配置，和 `render-redis-acl.sh` / `prod-deploy.sh` / `init-*-env.sh` 形成多层硬闸门。

### ✅ [FIXED] Zitadel 路由已统一应用安全头
（来源: doc1 INFRA-M5@480）
已修复：`infra/traefik/zitadel.dynamic.yaml` 的 login / OIDC / console / gRPC / REST 全部接入 `security-headers` 与 `gzip-compress`，不再让 IAM 入口绕过边缘层安全响应头。

### ✅ [FIXED] Zitadel / OpenFGA 健康告警已补齐
（来源: doc1 INFRA-M7@492）
已修复：`infra/observability/prometheus/prometheus.yml.tmpl` 为 `http://zitadel-api:8080/debug/healthz`、`http://zitadel-login:3000/ui/v2/login/healthy` 增加 blackbox HTTP 探测，并新增 `blackbox-tcp` 对 `openfga:8081` 做 TCP 探测；`infra/observability/prometheus/rules/application.yml` 已补 `StuHelperZitadelApiProbeFailed`、`StuHelperZitadelLoginProbeFailed`、`StuHelperOpenFGAGRPCProbeFailed` 三条告警。

### ✅ [FIXED] Traefik edge rate limiting 已接入应用与 IAM 入口
（来源: doc1 INFRA-L3@512）
已修复：`infra/traefik/services.dynamic.yaml` 新增 `edge-rate-limit` 并挂到 backend API 路由；`infra/traefik/zitadel.dynamic.yaml` 新增 `iam-edge-rate-limit` 并挂到 Zitadel login / OIDC / console / gRPC / REST 路由，边缘层先做粗粒度削峰，不再完全依赖应用内限流。

### ✅ [FIXED] Python 3 已文档化为运维脚本前置依赖
（来源: doc1 INFRA-L9@548）
已修复：`README.md` 与 `docs/QUICKSTART.md` 现已明确声明 Python 3 是环境渲染、远程部署前置检查、观测配置生成等脚本的运行依赖，不再让开发/运维在执行阶段才被 `require_cmd python3` 阻断。

### ✅ [FIXED] SSE 事件名写入已做安全规范化
（来源: doc1 SEC-L2@204）
已修复：`server/internal/modules/notification/handler.go` 中 `writeSSE()` 现在会先通过 `sanitizeSSEEventName()` 过滤控制字符与非法分隔符，保证事件名只落为安全的单行标识；新增 `server/internal/modules/notification/handler_sse_test.go` 覆盖空值、控制字符与正常事件名场景。

### ✅ [FIXED] `user.handler_self` 不再直接依赖 `auth.OTPCooldownSeconds()`
（来源: doc1 ARCH-L1@301）
已修复：`server/internal/modules/user/otp_deps.go` 现在把 `CooldownSeconds()` 收进 `OTPGenerator` 接口，`server/internal/modules/user/handler_self.go` 通过注入的 OTP 依赖读取冷却秒数，不再跨模块直连 `auth.OTPCooldownSeconds()` 常量 helper。

### ✅ [FIXED] OTP 发送流程已收敛到单一服务实现
（来源: doc1 GQ-M5@866）
已修复：`server/internal/modules/auth/otp.go` 新增 `IssueCode(ctx, phone, smsSender)`，把手机号限流、冷却期检查、验证码生成、短信发送与失败补偿收敛为一条权威路径；`server/internal/modules/auth/handler_phone.go` 与 `server/internal/modules/user/handler_self.go` 现在都只编排该服务，不再各自复制一套 OTP 发送流程。

### ✅ [FIXED] 内容审核 response 映射重复已提取共享 helper
（来源: doc1 GQ-M3@851, doc1 XC-M4@1148）
已修复：新增 `server/internal/modules/course/review/moderation_response.go` 与测试 `moderation_response_test.go`，`PostReview`、`UpdateReview`、`AdminEditReviewContent` 统一改用 `respondToModerationError(c, err)` 处理敏感内容/危险内容/服务不可用等 sentinel error，不再维护三份近重复 switch。

### ✅ [FIXED] Admin 审计日志已统一走 Gin 上下文 helper
（来源: doc1 GQ-M7@880）
已修复：新增 `server/internal/pkg/audit/gin.go` 与测试 `gin_test.go`，提供 `EventFromGin/LogFromGin` 统一填充 user/request/ip/user-agent；`server/internal/modules/user/handler_admin.go` 与 `server/internal/modules/course/review/handler.go` 已改为共享该 helper，不再各自手写审计字段拼装。

### ✅ [FIXED] Web notification store 不再静默吞掉 SSE 解析失败
（来源: doc3 P2-02@88）
已修复：`clients/web/src/stores/notification.ts` 新增 `streamError` 可观测状态与统一 `setStreamError/clearStreamError`，SSE `unread_count/notification/notification_read/notification_read_all/notification_deleted` 的 JSON 解析失败不再发布空事件占位，而是记录错误、累加失败计数并在阈值后切回 polling fallback + reconnect；成功解析/重连时会清空错误状态，长连接异常不再默默污染本地通知状态。

### ✅ [FIXED] NotificationBell / NotificationsPage 交互失败已显式提示
（来源: doc3 P2-05@118）
已修复：`clients/web/src/components/common/useNotificationBellController.ts` 与 `clients/web/src/modules/user/useNotificationsPageController.ts` 现在统一使用 `useToast()` + `getErrorMessage()` 暴露加载历史、全部已读、单条已读、分页加载等失败；`NotificationsPage.vue` 也删除了 `init().catch(() => {})` 的静默吞错入口。对应测试 `useNotificationBellController.test.ts` 与 `useNotificationsPageController.test.ts` 已覆盖失败提示路径。

### ✅ [FIXED] `singleflight.Do` 类型断言样板已收敛到泛型 helper
（来源: doc1 GQ-L5@951）
已修复：新增 `server/internal/pkg/singleflightx/singleflight.go` 与测试 `singleflight_test.go`，提供 `DoValue[T]` / `Do()` 泛型封装；`server/internal/modules/course/review/access.go`、`server/internal/modules/course/review/filter.go`、`server/internal/pkg/cache/cache.go` 已改为通过该 helper 执行去重加载，不再在业务代码里手写 `result.(T)` 类型断言样板。

### ✅ [FIXED] 手机 OTP 功能已改为显式 feature flag 并与文档对齐
（来源: doc6 S2-8@197）
已修复：`server/internal/pkg/config/config.go` 新增 `SMS_ENABLED`，`validation.go` 规定只有在 `SMS_ENABLED=true` 时才要求 `SMS_SECRET_ID/SMS_SECRET_KEY/SMS_APP_ID/SMS_SIGN_NAME/SMS_TEMPLATE_ID/SMS_INTERNAL_KEY` 完整配置；`server/internal/app/modules.go` 也改为仅在显式启用时初始化 SMS 服务，否则明确记录功能关闭。`.env.example`、`.env.prod.example`、`docs/product-specs/auth-sso.md`、`docs/BACKEND.md` 已同步为“默认关闭、启用需完整配置”的真实口径。

### ✅ [FIXED] Node CI 的 corepack 引导已收敛为共享 before_script
（来源: doc1 INFRA-L1@500）
已修复：`.gitlab-ci.yml` 新增 `.node_pnpm_job` 与共享 `&pnpm_setup`，frontend/admin 的 lint、typecheck、test、audit、build 统一继承同一套 corepack 初始化；`.gitlab/server-ci.yml` 的 `openapi_contract` 也改成共享 `before_script`，不再在每个 Node job 内重复粘贴 `corepack enable` / `corepack prepare pnpm@10 --activate`。

### ✅ [FIXED] `dev-up` / `dev-docker-up` 已收敛为单脚本模式切换
（来源: doc1 INFRA-L8@542）
已修复：`infra/ops/dev-up.sh` 现在支持 `DEV_UP_MODE=local|dockerized` 两种模式，Makefile 中的 `dev-docker-up` 只是给同一脚本传 `DEV_UP_MODE=dockerized`；原独立脚本 `infra/ops/dev-docker-up.sh` 已删除，开发入口只保留一个权威实现。

### ✅ [FIXED] LogConfig 已删除容器化部署无用的 `LOG_FILE_*` 配置面
（来源: doc1 XC-M6@1156）
已修复：`server/internal/pkg/config/config.go` 不再读取 `LOG_FILE_ENABLED/LOG_FILE_PATH/LOG_FILE_MAX_SIZE/LOG_FILE_MAX_BACKUPS/LOG_FILE_MAX_AGE/LOG_FILE_COMPRESS`，`server/internal/app/runtime.go` 也不再把这些字段向 logger 传递；`.env.example`、`.env.prod.example` 与 `docs/product-specs/audit-logging.md` 已同步收敛到 stdout + sampling 的实际容器化日志模型。

### ✅ [FIXED] DatabaseConfig 已收敛为 `DATABASE_URL` 单一事实源
（来源: doc1 XC-M7@1162）
已修复：`server/internal/pkg/config/config.go` 已删除 `DB_HOST/DB_PORT/DB_NAME/DB_USER/DB_PASSWORD` 读取与运行时拼接 `DATABASE_URL` 的兼容逻辑，后端只接受显式 `DATABASE_URL`；历史 `assembleDBURL` 辅助函数和相关测试一并移除，配置源歧义彻底消失。

### ✅ [FIXED] `.env.example` 已补齐仍保留的可选变量清单
（来源: doc1 DOC-L6@624, doc1 DOC-L7@629）
已修复：在删除 `DB_*` 与 `LOG_FILE_*` 冗余变量之后，`.env.example` / `.env.prod.example` 现在显式记录 `MAX_BODY_SIZE`、`POSTGRES_VERSION`、`REDIS_VERSION` 等仍可调但有默认值的变量；示例文件与代码实际可配置面重新对齐。

### ✅ [FIXED] Config `Load()` 单体 ~170 行
（来源: doc1 GO-L2@152）
已修复：`/Users/zxy/Code/StuHelper/server/internal/pkg/config/config.go` 已拆分为 `loadAppConfig/loadDatabaseConfig/loadRedisConfig/...` 等小函数，`Load()` 仅负责组装、补充安全配置与统一校验，配置读取边界更清晰；`go test ./internal/pkg/config ./internal/app -count=1` 通过。

### ✅ [FIXED] Backup retention 偏低（7/14 天）
（来源: doc1 INFRA-M1@455）
已修复：`/Users/zxy/Code/StuHelper/.env.example`、`/Users/zxy/Code/StuHelper/.env.prod.example`、`/Users/zxy/Code/StuHelper/infra/ops/init-dev-env.sh`、`/Users/zxy/Code/StuHelper/infra/ops/init-prod-env.sh`、`/Users/zxy/Code/StuHelper/infra/ops/run-scheduled-backup.sh` 与 `docs/operations/backup-and-restore.md` 已统一到逻辑备份 14 天、base backup 30 天、WAL 归档 14 天，不再维持过短默认保留期。

### ✅ [FIXED] 缺少 `no-new-privileges` security opt
（来源: doc1 INFRA-L2@506）
已修复：`/Users/zxy/Code/StuHelper/docker-compose.yml` 已为核心运行与观测服务统一补上 `security_opt: [no-new-privileges:true]`；通过 `docker compose config` 结构校验。

### ✅ [FIXED] 无 per-role PostgreSQL CONNECTION LIMIT
（来源: doc1 INFRA-L4@518）
已修复：`/Users/zxy/Code/StuHelper/infra/postgres/init-extra-dbs.sh` 为 `stuhelper_app`、`stuhelper_backup`、`stuhelper_replication`、`zitadel`、`openfga` 角色补上 `CONNECTION LIMIT`，避免单角色耗尽默认连接池。

### ✅ [FIXED] 无资源 reservations
（来源: doc1 INFRA-L1@500）
已修复：`/Users/zxy/Code/StuHelper/docker-compose.yml` 已为 `postgres`、`redis`、`zitadel-api`、`app-dev`、`app` 增加 `deploy.resources.reservations`，降低内存压力下关键容器被随机 OOM 的概率；`docker compose config` 校验通过。

### ✅ [FIXED] 无前端 SAST 扫描
（来源: doc1 INFRA-L8@542）
已修复：`/Users/zxy/Code/StuHelper/.gitlab-ci.yml` 新增 `frontend_sast` 安全作业，使用 `semgrep` 对 `clients/web/src` 与 `clients/admin/apps/web-ele/src` 扫描 Vue/TypeScript 常见注入与危险模式；GitLab CI YAML 解析通过。

### ✅ [FIXED] 开发 .env 含弱密码
（来源: doc1 SEC-L3@210）
已修复：`/Users/zxy/Code/StuHelper/infra/ops/init-dev-env.sh` 不再把 `POSTGRES_PASSWORD` / `REDIS_PASSWORD` 固定写成 `dev123`，而是对占位值与历史弱密码自动轮换为随机开发密钥；当前 `/Users/zxy/Code/StuHelper/.env` 里的弱密码也已替换，并同步更新 `DATABASE_URL`。

### ✅ [FIXED] refresh 之外无请求去重
（来源: doc1 FE-M7@400）
已修复：`/Users/zxy/Code/StuHelper/clients/web/src/api/client.ts` 为 GET 请求新增 in-flight 去重，按 `schemaPath + init` 合并并发中的相同读请求；携带 `AbortSignal` 的调用显式跳过去重，避免共享取消语义。对应 `/Users/zxy/Code/StuHelper/clients/web/src/api/__tests__/client.test.ts` 已覆盖并发 GET 合并与 signal 旁路场景。

### ✅ [FIXED] `useAsyncData` 默认 `immediate: true`（文档/示例改进）
（来源: doc1 FE-M11@423）
已修复：`/Users/zxy/Code/StuHelper/clients/web/src/composables/useAsyncData.ts` 现明确文档说明“默认立即执行、懒加载请显式传 `immediate: false`”；并新增 `/Users/zxy/Code/StuHelper/clients/web/src/composables/__tests__/useAsyncData.test.ts` 锁定默认行为与 lazy 模式，作为文档/示例改进闭环。

### ✅ [FIXED] `cache.Helper.GetVersion` 淘汰按 map 大小 O(N)
（来源: doc1 GQ-L3@939）
已修复：`/Users/zxy/Code/StuHelper/server/internal/pkg/cache/cache.go` 的本地版本缓存达到上限时改为 O(1) 重置策略，不再为了淘汰一条记录扫描整张 map；`/Users/zxy/Code/StuHelper/server/internal/pkg/cache/cache_test.go` 新增溢出后保留最新条目的测试，`go test ./internal/pkg/cache -count=1` 通过。

### ✅ [FIXED] `resolveCurrentUser` 在 user / notification 模块中已统一为共享 helper
（来源: doc1 GQ-M6@873）
已修复：新增 `/Users/zxy/Code/StuHelper/server/internal/pkg/middleware/internal_user.go` 与测试 `internal_user_test.go`，把“external user ID → internal user ID”的 401 / 500 响应逻辑收敛到 `ResolveRequiredInternalUserID()`；`/Users/zxy/Code/StuHelper/server/internal/modules/user/handler_helpers.go` 与 `/Users/zxy/Code/StuHelper/server/internal/modules/notification/handler.go` 现统一复用这一 helper，不再各自维护一套解析/报错分支。

### ✅ [FIXED] user 模块 nil slice 返回已统一规范化为 `[]`
（来源: doc1 GQ-L1@927；级别升级）
已修复：`/Users/zxy/Code/StuHelper/server/internal/modules/user/handler_helpers.go` 新增 `nonNilStrings()` 与 `nonNilManualFieldDescriptors()`，`profileToJSON`、`schoolConfigPublicToJSON`、`adminSchoolConfigToJSON`、`adminStudentVerificationToJSON` 以及 `/Users/zxy/Code/StuHelper/server/internal/modules/user/handler_self.go` 的 `/user/me` 响应现在都会把 `studentIDs`、`manualFormFields`、`capabilities` 规范化为 `[]`；`handler_test.go` 已补 `/user/me`、`/user/profile`、`/user/schools`、`/admin/school-configs` 回归测试。

### ✅ [FIXED] `RowWithCancel` 扫描后已释放重试保留引用
（来源: doc1 GO-M6@126；级别下调）
已修复：`/Users/zxy/Code/StuHelper/server/internal/pkg/db/db.go` 为 `RowWithCancel` 增加 `release()`，在 `Scan()` 结束后统一清空 `row/cancel/db/ctx/sql/args/span/start`，避免重试元数据在单次查询完成后继续滞留；新增 `/Users/zxy/Code/StuHelper/server/internal/pkg/db/db_test.go` 校验扫描成功后引用释放与 context cancel 生效。

### ✅ [FIXED] user 模块响应已从 `gin.H` 收敛为结构体 + json tag
（来源: doc1 GQ-L2@933, doc5 #12@370；级别升级）
已修复：`/Users/zxy/Code/StuHelper/server/internal/modules/user/handler_helpers.go` 新增 typed DTO（如 `identityStatusResponse`、`profileResponse`、`adminSchoolConfigResponse`、`pagedListResponse[T]`、`messageResponse` 等），`handler_self.go` 与 `handler_admin.go` 已改成统一返回结构体响应，不再手写 `gin.H` 拼字段；`go test ./internal/modules/user ./internal/app -count=1` 通过。

### ✅ [CLOSED] `Service.computePersonUID` 不带 `context.Context` 为有意设计
（来源: doc1 GO-M8@138；级别下调）
已核销：`/Users/zxy/Code/StuHelper/server/internal/modules/user/service.go` 中 `computePersonUID()` 是纯 HMAC 计算，无 I/O、无取消语义、无 tracing/span 边界；按 Go 最佳实践，这类纯 helper 不应为了“统一签名”机械引入 `context.Context`。当前实现更简洁也更长期正确，因此从 active 移除。

### ✅ [CLOSED] `course.Handler` 对 review 后台维护方法的委托保留为模块聚合边界
（来源: doc1 XC-L3@1181）
已核销：`/Users/zxy/Code/StuHelper/server/internal/modules/course/handler.go` 中 `runLogCleanup()` 与 `runTeacherPublicStatsRefresh()` 通过 `reviewHandler` 调用 review 子模块维护任务，是 course 作为“课程聚合根 + review 子域宿主”的明确编排边界，而非 HTTP 传递样板；继续保留该委托可避免把 review 内部服务细节泄漏到 `internal/app/modules.go` 装配层，当前结构被认定为长期可接受设计。

### ✅ [FIXED] 测试 Redis fixture 已收敛到共享 `internal/testutil/redisfixture`
（来源: doc5 #10@314）
已修复：新增 `/Users/zxy/Code/StuHelper/server/internal/testutil/redisfixture/redisfixture.go`，把 `miniredis + redis.Client + cleanup` 启动样板收敛为统一 `redisfixture.Start(t)`；`pkg/cache`、`pkg/token`、`pkg/health`、`pkg/middleware`、`modules/auth`、`modules/course`、`modules/course/review` 相关测试已全部切换到共享 helper，`server/internal/**/*_test.go` 中 `miniredis.Run()` 直接调用已清零。保留少量包内 HMAC/fake 依赖组装作为贴近业务的局部夹具，不再继续为“全局万能测试工厂”引入额外抽象层。

### ✅ [CLOSED] `oapi-codegen` 生成 Go 类型未直接进入 Handler 为有意分层
（来源: doc1 ARCH-L5@325）
已核销：`/Users/zxy/Code/StuHelper/server/api/oapi-codegen.yaml` 现已明确生成 Go 包的职责是 embedded spec、运行时 OpenAPI request validation 和 drift gate；`/Users/zxy/Code/StuHelper/docs/BACKEND.md` 也同步声明 Handler 可保留贴近 HTTP 绑定/错误映射的局部 DTO。当前架构避免把 handler 与生成模型硬耦合，只要契约权威仍在 `server/api/openapi.yaml` 且请求校验由 `internal/pkg/middleware/openapi_validation.go` 执行，就不存在“generated 类型必须直接进入 handler”这一活跃缺陷。

### ✅ [FIXED] PostgreSQL 已切换到仓库内自定义 `pg_hba.conf`
（来源: doc1 INFRA-L5@524）
已修复：新增 `/Users/zxy/Code/StuHelper/infra/postgres/pg_hba.conf`，按数据库/角色限制 `stuhelper_app`、`stuhelper_backup`、`stuhelper_replication`、`zitadel`、`openfga` 与 `postgres` 超级用户的 TCP 访问范围，只接受 RFC1918 私网来源并默认拒绝其他来源；`/Users/zxy/Code/StuHelper/docker-compose.yml` 已通过 `-c hba_file=/etc/postgresql/pg_hba.conf` 挂载启用该文件。`docker compose config` 结构校验通过。

### ✅ [FIXED] 文档层目录/角色/路由/API 清单已收敛为单一索引
（来源: doc6 S2-12@286, doc7 S2-8@275）
已修复：`/Users/zxy/Code/StuHelper/docs/README.md` 仅保留导航入口，不再重复业务域索引；`/Users/zxy/Code/StuHelper/docs/product-specs/index.md` 收敛为域索引页，不再重复角色表与核心概念；`/Users/zxy/Code/StuHelper/docs/FRONTEND.md` 与 `/Users/zxy/Code/StuHelper/docs/design-docs/frontend-architecture.md` 不再手写完整路由表，而是明确指向 `clients/web/src/router/index.ts` 与 admin 路由源码；`/Users/zxy/Code/StuHelper/docs/PRODUCT.md` 去除了 API/技术实现细节。人工 API 清单现只保留在 `/Users/zxy/Code/StuHelper/docs/references/api-overview.md`。

### ✅ [FIXED] 前端裸 `catch {}` 已清零并接入门禁
（来源: doc1 FE-M13@435）
已修复：`/Users/zxy/Code/StuHelper/clients/web`、`/Users/zxy/Code/StuHelper/clients/uniappx`、`/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele` 中所有裸 `catch {}` 已统一改成显式 `catch (_error) { void _error; ... }`，不再保留空 catch；新增 `/Users/zxy/Code/StuHelper/clients/scripts/check-no-empty-catch.sh` 并接入 `/Users/zxy/Code/StuHelper/clients/package.json` 的 lint 流程，同时 `/Users/zxy/Code/StuHelper/clients/eslint.config.mjs` 开启 `no-empty` 规则防止回归。已验证 `check-no-empty-catch`、web lint、web test、web/uni/admin type-check 全通过。
