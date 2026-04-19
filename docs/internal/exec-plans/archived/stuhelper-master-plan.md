---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# StuHelper Master Plan

> 状态：已归档。当前计划范围的 Phase 0-7 已闭环，文档保留作历史记录。

## overall goal

将当前完整平台中已较成熟的认证、用户、课程评课能力持续收口，并补齐教务展示、资源共享、存储驱动等关键缺口。当前直接执行的重点仍然落在平台中的后端域模型、OpenAPI/TS 共享契约、运行时接线、测试与配套文档上。核心方向固定为：

- 可靠的认证与会话
- 可靠的用户与学校身份体系
- 成熟的课程目录与评课体系
- 外部教务数据的导入、标准化、展示与查询
- 资源共享模块
- 可插拔存储驱动与挂载层
- 清晰的授权、审计与外部依赖治理

## product scope

- `auth`：OIDC、手机号 OTP、会话、refresh rotation、logout
- `user`：实名认证、学生认证、手机号绑定、学校配置、系统配置
- `course` / `review`：课程目录与评课社区
- `academics`：外部教务数据导入、标准化、查询展示
- `resource`：资源共享、资源绑定、标签、检索、下载
- `storage`：挂载点、驱动注册、能力声明、统一对象访问
- `notification`：站内通知与 SSE
- `authorization`：capability、本地策略、FGA 投影
- `audit`：统一审计事件与领域事件基础设施

## out of scope

- 完整教务写侧
- 实验系统
- 作业系统
- 提交 / 批改 / 评分 / 申诉系统
- 选课 / 退课 / 调课 / 排课写侧
- 第三方教务系统真实模拟登录连接器
- `bhpan` 等第三方网盘真实驱动

## current phase

Phase 0-7（当前计划范围已闭环）：产品边界、模块边界、native OIDC session 根链路、`academics` / `storage` / `resource` 一级模块、统一审计与 outbox 基础设施、外部依赖 typed errors、黑盒验证与文档回写已完成。当前剩余事项主要是受外部上下文约束的真实连接器与真实第三方驱动。

## done

- 完成 `todo.md` 与仓库现实对照，并把平台边界固定为“完整校园信息平台；本轮重点在平台中的后端/契约/基础设施”，不再把项目错误收窄成“后端项目”。
- 新增边界与规格文档：
  - `docs/exec-plans/archived/stuhelper-master-plan.md`
  - `docs/architecture/0001-stuhelper-target-scope-and-module-boundaries.md`
  - `docs/product-specs/academics-data-integration.md`
  - `docs/product-specs/resource-sharing.md`
  - `docs/product-specs/storage-driver-architecture.md`
- 修复 native OIDC `sessionID` 根链路：
  - 新增 `X-Stuhelper-Session-ID`
  - `refresh` / `logout` 支持原生 OIDC 客户端显式回传 session
  - OpenAPI、生成代码与 auth 黑盒测试同步收口
- 落地 `academics` 一级模块闭环：
  - migration：`000021_academics_module`
  - fixture connector、source/import job、标准化读模型
  - API：terms / offerings / offering detail / my courses / my schedule / admin sources / import jobs
  - 新增 HTTP 黑盒测试，覆盖导入触发、公开查询、我的课程、我的课表
- 落地 `storage` 一级模块闭环：
  - migration：`000022_storage_resource_module` 中的 `storage_mounts`
  - driver registry、capability model、S3/MinIO driver、default mount upsert
  - API：admin mounts list/create/health-check
  - 新增 HTTP 黑盒测试，覆盖 invalid driver、mount health、missing mount
- 落地 `resource` 一级模块闭环：
  - `resource_items` / `resource_versions` / `resource_bindings` / `resource_tags`
  - upload(create) / list / detail / update / delete / download-url
  - 课程/学期 binding、标签、可见性、版本元数据
  - 仅依赖 `storage` 暴露的稳定接口，不直连 S3 SDK
  - 新增 HTTP 黑盒测试，覆盖私有资源创建、下载保护、对象存储网络故障映射
- 完成 OpenAPI/共享契约收口：
  - 新增 `server/api/components/schemas/academics.yaml`
  - 新增 `server/api/components/schemas/resource.yaml`
  - 新增 `server/api/components/schemas/storage.yaml`
  - 新增 `server/api/paths/academics.yaml`
  - 新增 `server/api/paths/resource.yaml`
  - 新增 `server/api/paths/storage.yaml`
  - 重新生成 `server/internal/api/gen/server.gen.go`
  - 重新生成 `clients/shared/src/types/api.gen.ts`
- 完成统一 audit / outbox 基础设施收口：
  - migration：`000023_audit_events`
  - migration：`000024_unify_domain_event_outbox`
  - 新增 `server/internal/pkg/audit/repository.go`
  - 新增 `server/internal/pkg/outbox/repository.go`
  - 新增 `server/internal/pkg/outbox/worker.go`
  - `review` 管理操作日志统一进入 `audit_events`
  - `user external sync` 与 `review FGA sync` 统一委托共享 outbox repo/worker
  - 旧 `user_external_sync_outbox` / `review_fga_sync_outbox` 物理表已合并到 `domain_event_outbox`
  - 旧 `admin_operation_logs` 已迁入 `audit_events` 并从现行 schema 移除
- `resource` / `storage` / `academics` / `user admin` 等关键管理操作进入统一 `admin_operation` 审计类别
- 推进外部依赖治理：
  - 新增 `server/internal/pkg/objectstorage/errors.go`
  - 对象存储初始化、上传、删除、下载统一走 typed errors
  - `storage` 新增 `ErrDriverNotRegistered` / `ErrMountDisabled`
  - `resource` handler 对 storage/objectstorage 故障做更细粒度 HTTP 映射
- 完成 `user` 域存储边界统一：
  - 实名认证照片上传 / 预签名下载不再直连 `objectstorage.Store`
  - `user` 统一经 `storage.Service` 访问默认挂载点
  - 启动期校验从 bucket 直连检查切换为默认挂载点健康校验
  - `user` handler 已补充挂载禁用 / 驱动缺失 / 对象存储网络故障的 `503` 映射与测试
- 清理工程质量问题并回到绿色基线：
  - `go vet ./...` 通过
  - `golangci-lint run --timeout 5m` 通过
  - 覆盖率阈值脚本通过
  - 客户端 `type-check:all` 与 `@stuhelper/shared` 测试通过
- 本轮继续收口契约与前端适配层：
  - 修复 `academics.Source` JSON 契约，`/api/v1/admin/academics/sources` 现与 OpenAPI 一致，仅输出 `id/key/name/provider/enabled`，不再泄漏内部 `config`
  - 新增 `clients/admin/apps/web-ele/src/store/auth.test.ts`，修复 admin 端会话探测失败被误判为“未登录并跳登录”的问题；现在 `retryable_error` / `fatal_error` 会向上抛出，由路由守卫中止导航并保留真实错误语义
  - `clients/shared/src/api/result.ts` 统一兼容 `list` 与 `items` 两种分页载荷，避免新模块接入时被旧助手函数错误清空列表
  - `docs/product-specs/storage-driver-architecture.md` 已按代码现实修正：移除不存在的 `storage_object_refs` 与 `list` capability，并补充挂载点相对 `objectKey` 语义
- 本轮继续收口认证与安全失败语义：
  - `clients/admin/apps/web-ele/src/api/shared-client.ts` 不再在强制重登录程里静默 fallback 到 `/admin/`；当后端登录 URL 不可获取时，现在保留错误并向上抛出，避免重载循环掩盖真实故障
  - 新增 `clients/admin/apps/web-ele/src/api/shared-client.test.ts`，锁定“取不到登录 URL 时必须抛错、不能偷偷 fallback”的行为
  - `server/internal/pkg/token/blacklist.go` 收紧 Redis 故障降级语义：`IsBlacklisted` 现在只信任本地正向撤销缓存，负缓存绝不用于放行 token，避免跨实例撤销后在 Redis 故障窗口内被旧负缓存误放行
  - 新增 `server/internal/pkg/token/blacklist_test.go` 安全回归用例，覆盖“熔断打开 + 负缓存时必须拒绝”“熔断打开 + 正缓存时仍可拒绝”两条关键路径
  - `server/internal/pkg/token/session.go` 的 `RevokeAll` 不再吞掉逐个 session 撤销失败；现在会返回聚合错误，且只有全部撤销成功后才清理用户 session 索引，避免 `/auth/logout-all` 对部分失败误报成功
  - 新增 `server/internal/pkg/token/session_test.go` 回归用例，锁定“逐个撤销失败时必须返回错误且保留 session 索引”的语义
  - `clients/admin/apps/web-ele/src/api/core/auth.ts` 的显式登出改为走不带自动 refresh / 自动重登录的基础客户端；`clients/admin/apps/web-ele/src/store/auth.ts` 不再在服务端登出失败时直接清本地状态并重定向
  - 新增 `clients/admin/apps/web-ele/src/store/auth.test.ts` 回归用例，锁定“服务端登出失败时不能伪装成本地登出成功”的行为
  - `clients/web/src/router/index.ts` 现通过 `clients/web/src/router/auth-guard-decision.ts` 收紧守卫决策：`bootstrapSession()` 或过期 token 刷新遇到网络 / 5xx 且本地会话仍未明确失效时，取消受保护导航而不是跳转 `/login`
  - 新增 `clients/web/src/router/__tests__/auth-guard-decision.test.ts`，覆盖“bootstrap 未决 / refresh 失败但本地仍登录 / refresh 后本地已失效”三类守卫分支
  - `server/internal/pkg/middleware/auth.go` 新增 `RequireHealthyOptionalAuth()`，把 optional auth 的后端故障从“仅打诊断标记”收口为显式 `503`，避免请求明明携带认证态却被静默降级成匿名访问
  - `course` / `course/review` / `resource` 所有 optional-auth 读路由现统一串联 `middleware.RequireHealthyOptionalAuth()`，认证后端异常时会直接返回 `503`，不再匿名继续
  - 新增 `server/internal/pkg/middleware/auth_flow_test.go` 与 `server/internal/modules/resource/handler_integration_test.go` 回归用例，锁定 optional-auth 后端故障必须显式失败的行为
  - `clients/shared/src/api/auth.ts` 现已显式支持 native OIDC `refresh` / `logout` 传入 `X-Stuhelper-Session-ID`，共享契约不再把该 header 丢在调用边界之外
  - `clients/uniappx/src/api/native-session.ts` 新增统一 native token/session 持久化层；`exchange-native` 返回的 `sessionID` 现会持久化，`refresh` / `logout` 会自动附带 `X-Stuhelper-Session-ID`
  - `clients/uniappx/src/stores/auth.ts` 不再把“access token 已过期但 refresh token 仍可续期”的状态误判成未登录；native bootstrap 现在会继续走真实会话恢复链路
  - `server/internal/modules/auth/service.go` 与 `server/internal/modules/auth/handler_native_session.go` 已把 native OIDC `refresh` / `logout` 收紧为 fail-closed：缺失或无效 session header 时直接 `401`，不再生成未被 session store 追踪的新 token，也不再退化成只撤销 access token 的半成功路径
  - 新增 `clients/shared/src/api/__tests__/auth.test.ts`、`clients/uniappx/src/stores/auth.test.ts`，并扩展 `clients/uniappx/src/api/__tests__/shared-client.test.ts`、`server/internal/modules/auth/handler_oidc_flow_test.go`、`server/internal/modules/auth/handler_session_validation_test.go`、`server/internal/modules/auth/service_unit_test.go`，锁定上述 native session 语义

## remaining backlog

- 真实学校教务连接器：认证、抓取、字段语义、限流策略、失败重试规则仍需真实上下文。
- 第三方网盘驱动：`bhpan` / WebDAV / 私有网盘的真实配置、鉴权、能力矩阵仍需真实上下文。

## risks

- 当前 `course` 目录仍同时承载 catalog + review 语义，虽然文档边界已明确，但后续是否拆分仍需根据演进决定。
- `storage` 当前真实运行驱动仍只有 `s3` / `minio` family；openlist-like 体系只完成了接口边界和挂载模型。
- `academics` 当前真实 provider 仍是 fixture connector；连接器生命周期与 typed errors 已有骨架，但真实学校系统适配仍待外部上下文。

## pending external context

- 学校教务系统真实连接器的认证、抓取、字段语义与限流规则。
- 第三方网盘驱动的真实配置、鉴权与能力矩阵。

## touched files

- `README.md`
- `docs/PRODUCT.md`
- `docs/README.md`
- `docs/product-specs/index.md`
- `docs/product-specs/academics-data-integration.md`
- `docs/product-specs/resource-sharing.md`
- `docs/product-specs/storage-driver-architecture.md`
- `docs/architecture/0001-stuhelper-target-scope-and-module-boundaries.md`
- `docs/exec-plans/archived/stuhelper-master-plan.md`
- `docs/exec-plans/archived/todo.md`
- `server/api/components/parameters/common.yaml`
- `server/api/components/schemas/academics.yaml`
- `server/api/components/schemas/resource.yaml`
- `server/api/components/schemas/storage.yaml`
- `server/api/paths/auth.yaml`
- `server/api/paths/academics.yaml`
- `server/api/paths/resource.yaml`
- `server/api/paths/storage.yaml`
- `server/api/openapi.yaml`
- `server/api/openapi.bundled.yaml`
- `server/internal/api/gen/server.gen.go`
- `server/internal/app/modules.go`
- `server/internal/app/openapi_route_contract_test.go`
- `server/internal/app/runtime.go`
- `server/internal/modules/auth/*`
- `server/internal/modules/academics/*`
- `server/internal/modules/storage/*`
- `server/internal/modules/resource/*`
- `server/internal/modules/course/review/*`
- `server/internal/modules/user/*`
- `server/internal/pkg/audit/*`
- `server/internal/pkg/outbox/*`
- `server/internal/pkg/objectstorage/*`
- `server/internal/modules/storage/constants.go`
- `server/internal/pkg/cache/cache.go`
- `server/internal/pkg/metrics/metrics_test.go`
- `server/internal/pkg/middleware/auth.go`
- `server/internal/pkg/token/session.go`
- `server/internal/testutil/postgresfixture/postgresfixture.go`
- `server/internal/testutil/redisfixture/redisfixture.go`
- `server/migrations/000021_academics_module.up.sql`
- `server/migrations/000021_academics_module.down.sql`
- `server/migrations/000022_storage_resource_module.up.sql`
- `server/migrations/000022_storage_resource_module.down.sql`
- `server/migrations/000023_audit_events.up.sql`
- `server/migrations/000023_audit_events.down.sql`
- `server/migrations/000024_unify_domain_event_outbox.up.sql`
- `server/migrations/000024_unify_domain_event_outbox.down.sql`
- `clients/shared/src/types/api.gen.ts`
- `clients/shared/src/constants/capabilities.gen.ts`
- `clients/shared/src/api/__tests__/session-client.test.ts`
- `clients/shared/src/api/__tests__/auth.test.ts`
- `clients/uniappx/src/api/native-session.ts`
- `clients/uniappx/src/api/__tests__/shared-client.test.ts`
- `clients/uniappx/src/stores/auth.ts`
- `clients/uniappx/src/stores/auth.test.ts`

## tests run / results

- `cd server && make generate`
  - 通过
- `cd server && make lint-spec`
  - 通过
- `cd server && go vet ./...`
  - 通过
- `cd server && golangci-lint run --timeout 5m`
  - 通过，`0 issues`
- `cd server && bash scripts/check-coverage-threshold.sh`
  - 通过，所有阈值满足
- `cd server && go test ./internal/pkg/objectstorage ./internal/modules/storage ./internal/modules/resource ./internal/modules/course/review ./internal/modules/user ./internal/pkg/audit ./internal/pkg/outbox ./internal/app -count=1`
  - 通过
- `cd server && go test ./internal/modules/auth ./internal/modules/academics ./internal/modules/storage ./internal/modules/resource ./internal/modules/user ./internal/modules/course/review ./internal/app -count=1`
  - 通过
- `cd server && go test ./internal/pkg/audit ./internal/pkg/outbox ./internal/pkg/objectstorage -count=1`
  - 通过
- `cd server && go test ./internal/modules/resource ./internal/modules/storage ./internal/modules/academics -count=1`
  - 通过
- `cd server && go test ./internal/modules/user ./internal/modules/storage ./internal/pkg/outbox ./internal/pkg/audit ./internal/modules/course/review ./internal/app -count=1`
  - 通过
- `cd server && python3 -c "import subprocess,sys; sys.exit(subprocess.run(['go','test','./...','-count=1'], timeout=60).returncode)"`
  - 通过，60 秒硬超时内完成整仓后端测试
- `cd clients && npx --yes pnpm@10.32.1 run type-check:all`
  - 首轮因 `clients/shared/src/api/__tests__/session-client.test.ts` mock 缺少 `expiresIn` 失败，已修复并复跑通过
- `cd clients && pnpm --filter @stuhelper/shared test`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/modules/academics -run TestSourceJSONContract -count=1`
  - 通过
- `cd clients && pnpm --filter @stuhelper/shared test -- --run src/__tests__/result.test.ts`
  - 通过
- `cd clients && pnpm --filter @stuhelper/shared type-check`
  - 通过
- `cd clients/admin && pnpm exec vitest run apps/web-ele/src/store/auth.test.ts --config vitest.config.ts`
  - 通过
- `cd clients && pnpm type-check:admin`
  - 通过（当前 Node 版本为 `v25.9.0`，仅出现 engine warning，不影响 typecheck 结果）
- `cd clients/admin && pnpm exec vitest run apps/web-ele/src/api/shared-client.test.ts --config vitest.config.ts`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/pkg/token -run 'TestBlacklist_IsBlacklisted_(DeniesWhenCircuitOpenAndOnlyNegativeCacheExists|AllowsPositiveRevocationCacheWhenCircuitOpen)' -count=1`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/pkg/token -run TestSessionStore_RevokeAll -count=1`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/modules/auth -run 'Test(RevokeAllSessions_ContextCanceled|RevokeSessionAndRevokeAll|LogoutAll_RevokesAllSessions|LogoutAll_FailureBranch)' -count=1`
  - 通过
- `cd clients/admin && pnpm exec vitest run apps/web-ele/src/store/auth.test.ts apps/web-ele/src/api/shared-client.test.ts --config vitest.config.ts`
  - 通过
- `cd clients && pnpm type-check:admin`
  - 通过（当前 Node 版本为 `v25.9.0`，仅出现 engine warning，不影响 typecheck 结果）
- `cd clients/web && pnpm exec vitest run src/router/__tests__/auth-guard-decision.test.ts src/stores/__tests__/authBootstrap.test.ts src/stores/__tests__/authSessionReset.test.ts --config vitest.config.ts`
  - 通过
- `cd clients && pnpm --dir web type-check`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/pkg/middleware -run 'Test(OptionalAuthMiddleware_BackendFailureMarksDiagnostic|RequireHealthyOptionalAuth_RejectsBackendFailure)' -count=1`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/modules/resource -run 'Test(ResourceHandlers_CreatePrivateResourceAndProtectDownload|ResourceHandlers_MapObjectStorageNetworkFailureTo503|ResourceHandlers_OptionalAuthBackendFailureReturns503)' -count=1`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/modules/course ./internal/modules/course/review -run 'TestCourseRegisterRoutes_UsesOpenAPIPathParamNames|TestReviewRegisterRoutes_UsesOpenAPIPathParamNames|TestHandlerValidation|TestHandlerReadSuccess|TestHandlerAccessResolutionError|TestCourseHandler_SuccessBranches|TestCourseHandler_Validation' -count=1`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/modules/auth -run 'TestRotateSession_WithoutSessionIDRejectsRequestAndBlacklistsOldRefresh|TestRefreshToken_NativeOIDCRequiresTrackedSession|TestLogout_NativeOIDCRequiresTrackedSession' -count=1`
  - 通过
- `cd server && GOCACHE=/tmp/stuhelper-go-cache go test ./internal/modules/auth -count=1`
  - 通过
- `cd clients && pnpm --filter @stuhelper/shared test -- --run src/api/__tests__/auth.test.ts`
  - 通过
- `cd clients && pnpm --filter @stuhelper/shared type-check`
  - 通过
- `cd clients && pnpm --filter @stuhelper/uniappx exec vitest run src/api/__tests__/shared-client.test.ts src/stores/auth.test.ts --config vitest.config.ts`
  - 通过
- `cd clients && pnpm --filter @stuhelper/uniappx type-check`
  - 通过
- 受当前沙箱限制，`testcontainers` 依赖的 Go 集成测试仍无法运行：
  - `docker.sock` 无权限，`postgresfixture.Start(...)` 会失败
