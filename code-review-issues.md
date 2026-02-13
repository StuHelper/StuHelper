# StuHelper 代码审查问题清单

> 自动化多轮审查，每轮发现的问题合并整理。
> 审查日期：2026-02-13 | 已完成轮次：10/10

---

## 一、基础设施与配置

| # | 严重度 | 文件 | 问题描述 |
|---|--------|------|----------|
| I-1 | 🔴高 | `deployments/.env` | 开发环境包含真实 Casdoor 凭据和弱密码（`stuhelper_dev`） |
| I-2 | 🟡中 | `deployments/.env` | `sslmode=disable` 数据库连接未加密 |
| I-3 | 🟡中 | `deployments/.env` | `HMAC_SECRET` 长度刚好 32 字符，缺少安全余量 |
| I-4 | 🟡中 | `deployments/.env.example` | `JWT_SECRET` 为弱占位符，无生成指引 |
| I-5 | 🟡中 | `deployments/docker-compose.yml` | 应用服务缺少 health check 定义 |
| I-6 | 🟡中 | `.gitea/workflows/ci.yml` | CI 中 postgres/redis 测试服务无认证 |
| I-7 | 🔵低 | `clients/web/` | 前端缺少 `.env.example`，`VITE_*` 环境变量无文档 |
| I-8 | 🔵低 | `deployments/.env.example` | 限流参数未暴露为环境变量 |

## 二、安全

| # | 严重度 | 文件:行 | 问题描述 |
|---|--------|---------|----------|
| S-1 | 🔴高 | `router/index.ts:207-214` | Open Redirect：`redirect` 参数仅检查 `/` 前缀，未验证是否为合法应用路由 |
| S-2 | 🔴高 | `auth/handler.go:113` | `GetOAuthToken` 第二参数传 `appName` 而非 `state`，绕过 Casdoor 端 state 验证 |
| S-3 | 🔴高 | `auth/handler.go:199-224` | Logout 仅将 access token 加入黑名单，未处理 refresh token，攻击者可用旧 refresh token 获取新 access token |
| S-4 | 🔴高 | `auth/handler.go:320-325` | CSRF token 生成失败仅记录警告仍继续，客户端收到空 CSRF token 导致后续所有 POST 被拒 |
| S-5 | 🟡中 | `sso/state.go:32-48` | State 生成用 `Set` 而非 `SetNX`，极端并发下可能碰撞覆盖 |
| S-6 | 🟡中 | `token/blacklist.go:42-46` | Token 黑名单用纯 SHA256 无盐哈希 |
| S-7 | 🟡中 | `middleware/ratelimit.go:127-141` | 熔断日志记录完整 Redis 错误，泄露基础设施信息 |
| S-8 | 🟡中 | `auth/handler.go:339-347` | Refresh token cookie 路径限制为 `/api/v1/auth/refresh`，API 路由变更后 cookie 无法发送 |
| S-9 | 🟡中 | `auth/handler.go:293-307` | 刷新 token 后新 JWT 解析失败仅记录警告，token 未被追踪，`LogoutAll` 无法撤销 |
| S-10 | 🟡中 | `main.go:154-167` | CORS `AllowCredentials: true` 配合动态 origins，空列表时可能使用默认宽松配置 |
| S-11 | 🟡中 | `token/blacklist.go:56-76` | 黑名单 fail-closed 无降级策略，Redis 故障时服务完全不可用 |
| S-12 | 🔵低 | `middleware/auth.go:119-138` | Token 来源优先级（Cookie > Header）未文档化 |
| S-13 | 🔵低 | `response/response.go:25-37` | 已弃用错误码别名仍在使用 |
| S-14 | 🔵低 | `api/index.ts:85-89` | Refresh token 请求为 POST 但未携带 CSRF token header |
| S-15 | 🔵低 | `stores/auth.ts:70,91-92` | OAuth state 验证后未立即从 sessionStorage 清除，成功后才移除 |
| S-16 | 🔵低 | `AuthCallbackPage.vue:97` | 组织不匹配时 SSO 登出 URL 存储在 sessionStorage，可被篡改 |
| S-17 | 🔵低 | `stores/auth.ts:62-69` | 前端未验证后端返回的 OAuth URL 是否指向预期 Casdoor 端点 |

## 三、数据库

| # | 严重度 | 文件:行 | 问题描述 |
|---|--------|---------|----------|
| D-1 | 🔴高 | `init.sql` | `review_replies.review_id` 缺少索引，回复列表全表扫描 |
| D-2 | 🔴高 | `init.sql:102` | `reviews.term_id` 应为 `NOT NULL`（Go 端 `string` 非指针，NULL 导致 scan 失败） |
| D-3 | 🟡中 | `init.sql` | `review_votes(review_id, user_hash)` 缺少复合索引 |
| D-4 | 🟡中 | `init.sql` | `notifications(user_hash, is_read)` 缺少复合索引 |
| D-5 | 🟡中 | `init.sql:102` | `reviews.term_id` 缺少单独索引（频繁按学期过滤） |
| D-6 | 🟡中 | `init.sql:116` | `reviews.course_id` 外键缺少 `ON DELETE CASCADE`，删除课程后孤立评论 |
| D-7 | 🟡中 | `init.sql:130` | `reviews` 唯一约束为 `(user_hash, course_id)`，同一用户同课程不同学期无法多次评论 |
| D-8 | 🔵低 | `init.sql:261` | `review_replies.content` 缺少最小长度 CHECK |
| D-9 | 🔵低 | `init.sql:66` | `credits DECIMAL(3,1)` 范围偏小（最大 99.9），建议 `DECIMAL(4,1)` |

## 四、Go 后端

| # | 严重度 | 文件:行 | 问题描述 |
|---|--------|---------|----------|
| B-1 | 🔴高 | `middleware/ratelimit.go:97` | `generateUniqueID()` 使用 `panic()` 处理 crypto 错误 |
| B-2 | 🔴高 | `review/handler.go:39-43` | 限流阈值硬编码（5/30/10 次），违反项目规范 |
| B-3 | 🔴高 | `repository_rating.go:93-124` | `ListCourseTeachers` 使用 3 个关联子查询，N+1 性能问题 |
| B-4 | 🔴高 | `service_interaction.go:90-147` | `ProcessReport` 接受请求体中的 `ResolvedBy`，未从中间件上下文提取管理员身份 |
| B-5 | 🔴高 | `main.go:84-103` | 初始化失败时 Redis/PG 资源未清理，`token.NewService()` 失败导致连接泄漏 |
| B-6 | 🔴高 | `main.go:214-219` | Server goroutine 在 `ListenAndServe()` 成功时永不退出，`serverErr` channel 发送阻塞 |
| B-7 | 🔴高 | `sso/cache.go:57-61` | `UserCache.SetTTL()` 与 `Set()` 之间存在 `ttl` 字段竞态条件 |
| B-8 | 🟡中 | `repository_admin.go:359,405` | SQL 字符串拼接（`baseQuery+`） |
| B-9 | 🟡中 | `repository_admin.go:362,408` | `LIMIT 10000` 硬编码 |
| B-10 | 🟡中 | `repository_admin.go` 多处 | 错误未用 `fmt.Errorf("context: %w", err)` 包装 |
| B-11 | 🟡中 | `review/filter.go:76` | `ensureFresh` 刷新敏感词失败时静默忽略 |
| B-12 | 🟡中 | `wire/providers.go:49` | 使用 `log.Printf` 而非结构化 logger |
| B-13 | 🟡中 | `health/health.go:59,106,114` | 直接使用 `c.JSON()` 而非 `response.Success()` |
| B-14 | 🟡中 | `service_interaction.go:156-165` | `AddFavorite` TOCTOU：课程存在性检查与插入不在同一事务 |
| B-15 | 🟡中 | `service.go:352-411` | `UpdateReview` 验证在事务外 |
| B-16 | 🟡中 | `model.go:110` | `ReviewReport.ID` 类型 `string` 与数据库 `BIGSERIAL` 及 repository 签名 `int64` 冲突 |
| B-17 | 🟡中 | `repository_admin.go:389-436` | `ForEachReviewForExport` 回调出错时未检查 `rows.Err()` |
| B-18 | 🟡中 | `config/config.go` | `HMACSecret` 仅生产环境检查非空，未验证最小长度 32 |
| B-19 | 🟡中 | `config/config.go` | `QueryTimeout`/`MaxConns`/`MinConns` 无范围验证，`MinConns > MaxConns` 未检查 |
| B-20 | 🟡中 | `db/db.go:42-57` | `Query()` 无重试机制，临时网络错误直接失败 |
| B-21 | 🟡中 | `db/db.go:146-175` | `WithTx()` 事务超时倍数 `3x` 硬编码 |
| B-22 | 🟡中 | `cache/cache.go:17` | 所有缓存使用相同 `DefaultTTL=5min`，同时过期导致缓存雪崩 |
| B-23 | 🟡中 | `cache/cache.go` | 无缓存击穿防护（singleflight），高并发 miss 触发多次 DB 查询 |
| B-24 | 🟡中 | `cache/cache.go:200-203` | `BuildVersionedKey()` 每次调用查询 Redis 版本号，高并发下大量请求 |
| B-25 | 🟡中 | `main.go:232-238` | 优雅关闭未保证 PG/Redis 清理顺序，deferred cleanup 可能延迟 |
| B-26 | 🟡中 | `logging.go:137-139` | Panic 恢复使用 `gin.H` 而非标准 `response` 包，响应格式不一致 |
| B-27 | 🟡中 | `crypto/hmac.go:61` | 初始化检查失败时 panic 而非返回 error |
| B-28 | 🟡中 | `db/db.go:157-163` | `WithTx` panic 恢复后 re-panic，无法保证事务清理完成 |
| B-29 | 🟡中 | `circuitbreaker.go:82-103` | 状态转换持锁期间与 `RecordSuccess/Failure` 存在竞争 |
| B-30 | 🟡中 | `middleware/ratelimit.go:59-90` | `Allow` 方法 context 取消未正确传播，可能资源泄漏 |
| B-31 | 🟡中 | `main.go:71-76` | Logger 初始化在 config 之后，config 失败时错误输出到 stderr 而非结构化日志 |
| B-32 | 🟡中 | `repository_rating.go:30-119` | 6 处数据库查询错误直接返回未包装上下文 |
| B-33 | 🟡中 | `httputil/httputil.go:23,34` | `strconv.Atoi` 错误被 `_ =` 忽略，违反项目规范 |
| B-34 | 🟡中 | `review/review.go:413` | `strconv.Atoi(c.DefaultQuery("limit", "20"))` 错误被忽略 |
| B-35 | 🔵低 | `review/review.go:51` | 缓存 key 拼接未对 termID/tid 做清洗 |
| B-36 | 🔵低 | `repository_admin.go:27` | `status` 参数未做枚举白名单校验 |
| B-37 | 🔵低 | `review_interaction.go:123-132` | `SaveDraftRequest.Content` 无最小长度校验 |
| B-38 | 🔵低 | `service_interaction.go:103` | `ProcessReport` action 在进入事务后才校验 |
| B-39 | 🔵低 | `config.go:293-325` | `configParseErrors` 全局 slice 在并发 `Load()` 时存在竞态 |
| B-40 | 🔵低 | `health.go:68-69` | Readiness check context timeout 未被底层 Ping 调用有效执行 |
| B-41 | 🔵低 | `cache/cache.go:107-153` | `Invalidate` SCAN 限制 1000 key，超出部分静默跳过 |
| B-42 | 🔵低 | `db/db.go:42-57` | `Query()` 成功后若后续操作失败，rows 未关闭 |
| B-43 | 🔵低 | `security_headers.go:92-96` | `MaxBodySize` 中间件 body 大小检查失败后 wrapped body 未清理 |

## 五、Vue/TypeScript 前端

| # | 严重度 | 文件:行 | 问题描述 |
|---|--------|---------|----------|
| F-1 | 🔴高 | `api/index.ts:47-52` | Token 刷新竞态：失败后 `isRefreshing` 重置，可能无限循环 |
| F-2 | 🔴高 | `stores/auth.ts:141-153` | 登出时未重置其他 store（courseReview/user/notification），隐私泄露风险 |
| F-3 | 🔴高 | `TeacherProfilePage.vue:168-175` | ECharts 主题切换时未先 dispose 旧实例，每次切换泄漏一个 ECharts 实例 |
| F-4 | 🔴高 | `ReviewFeed.vue:129-132` | 排序变化时无 abort 机制，快速切换导致旧请求覆盖新结果 |
| F-5 | 🔴高 | `ReviewCard.vue:233-274` | 快速点赞/踩可并发发送多个投票请求，`isVoting` 仅防同类型 |
| F-6 | 🟡中 | `types/review.ts:16` | `termID` 标记为可选，但后端为必需 `string` |
| F-7 | 🟡中 | `FloatingModuleNav.vue:126-129` | 拖拽期间监听器在组件卸载时可能泄漏 |
| F-8 | 🟡中 | `ReviewDialog.vue:354` | `beforeunload` 监听器弹窗关闭时未清理 |
| F-9 | 🟡中 | `FloatingModuleNav.vue:67-70` | 导航标签硬编码中文，未使用 i18n |
| F-10 | 🟡中 | `composables/useCommandPalette.ts:37-43` | `handleKeydown` 每次创建新引用，`removeEventListener` 无效 |
| F-11 | 🟡中 | `stores/courseReview.ts:90-123` | 缓存无手动失效机制，发布评论后数据过期 |
| F-12 | 🟡中 | `api/course.ts:30,40` | 分页响应类型不一致（内联 vs `PaginatedResponse<T>`） |
| F-13 | 🟡中 | `i18n/locales/en-US/errors.ts` | 缺少 `orgMismatch`/`orgMismatchHint`/`ssoLogout` 三个翻译 key |
| F-14 | 🟡中 | `views/review/ReviewPage.vue:11` | `'课程列表'` 硬编码中文，未使用 i18n |
| F-15 | 🟡中 | `components/review/ReplyForm.vue:20` | CSS 类 `text-rating-3` 应为 `text-color-rating-3`（Tailwind v4 迁移遗漏） |
| F-16 | 🟡中 | `DepartmentSidebar.vue:139-159` | 快速点击同一院系可触发并发加载请求，无去重机制 |
| F-17 | 🟡中 | `FavoriteButton.vue:44-56` | 错误提示 `setTimeout` 未保存引用，组件卸载后仍触发 |
| F-18 | 🟡中 | `InlineSearch.vue:156-169` | blur 定时器在组件卸载时可能泄漏 |
| F-19 | 🟡中 | `ReviewDialog.vue:385-426` | 草稿恢复异步操作在快速开关弹窗时可并发执行 |
| F-20 | 🟡中 | `stores/user.ts:63-67` | `fetchMyFavorites` 覆盖 `favoriteIDs`，丢失并发 `toggleFavorite` 的乐观更新 |
| F-21 | 🟡中 | `api/index.ts:96` | Token 刷新失败 catch 块返回 false 但未记录错误信息 |
| F-22 | 🟡中 | `stores/user.ts:20-32` | `fetchPaginated` 无错误处理，API 失败时 `loadingRef` 永远为 true |
| F-23 | 🔵低 | `ReviewDialog.vue:340` | `countdownTimer` 可能在组件卸载后触发 |
| F-24 | 🔵低 | `ReviewDialog.vue:316` / `InlineSearch.vue:144` | API 错误静默吞掉 |
| F-25 | 🔵低 | `FloatingModuleNav.vue:20-21` | tooltip 缺少 `role="tooltip"` |
| F-26 | 🔵低 | `types/course.ts:46-56` | `FavoriteCourse` 缺少 `category` 字段 |
| F-27 | 🔵低 | `composables/useToast.ts` 等 | 多个 composable 缺少显式返回类型 |
| F-28 | 🔵低 | `TabBar.vue:8-9` | tab 按钮缺少 `aria-label` |
| F-29 | 🔵低 | `TeacherStatsCard.vue:5` | 装饰性图标缺少 `aria-hidden="true"` |
| F-30 | 🔵低 | `utils/auth.ts:87-91` | Token 过期预检硬编码 60s 阈值，短 TTL 场景下过于激进 |
| F-31 | 🔵低 | `DepartmentSidebar.vue:154-158` | 加载失败时缓存空数组，后续永远跳过重新加载 |
| F-32 | 🔵低 | `courseReview.ts:99-123` | 新请求进行中时旧请求 finally 将 `departmentsLoading` 置 false |

## 六、测试与 CI

| # | 严重度 | 文件 | 问题描述 |
|---|--------|------|----------|
| T-1 | 🔴高 | `clients/web/` | 前端完全缺失测试框架和测试代码（无 Vitest/Jest，无 `.test.ts`） |
| T-2 | 🔴高 | `server/internal/modules/course/review/` | 评课核心模块（3210 行 service+repository）零测试覆盖 |
| T-3 | 🔴高 | `server/internal/modules/auth/` | 认证模块仅 2 个基础 mock 测试，无 SSO/Token 刷新/登出流程测试 |
| T-4 | 🟡中 | `server/internal/pkg/middleware/` | auth/ratelimit/permission/security_headers 中间件无测试 |
| T-5 | 🟡中 | `clients/web/` | 无 ESLint 依赖和配置文件，项目规范要求强制但未实施 |
| T-6 | 🟡中 | `clients/web/` | 无 Prettier 配置文件（`.prettierrc`），项目规范要求强制 |
| T-7 | 🟡中 | `clients/web/package.json:11` | `lint` 脚本实际运行 `vue-tsc --noEmit`（类型检查），非真正 linting |
| T-8 | 🟡中 | `.golangci.yml:28-30` | `errcheck.check-blank: false` 允许 `_ =` 忽略错误，与项目规范矛盾 |
| T-9 | 🟡中 | `clients/web/.gitea/workflows/ci.yml` | 前端 CI 无测试步骤、无覆盖率收集、无 ESLint 检查 |
| T-10 | 🟡中 | `scripts/init.sql` | 数据库初始化脚本无语法验证和迁移测试 |
| T-11 | 🔵低 | `docs/tutorials/quick-start.md:12` | Go 版本写 `1.23+`，但 `go.mod` 指定 `1.24.0` |
| T-12 | 🔵低 | `docs/` | 前端开发指南、测试运行说明、构建部署文档缺失 |

## 七、API 设计与类型安全

| # | 严重度 | 文件:行 | 问题描述 |
|---|--------|---------|----------|
| A-1 | 🔴高 | `api/ranking.ts:7-15` vs `repository_rating.go` | `HotCourse` 前后端字段严重不匹配：前端 `id/name` vs 后端 `courseID/courseName`，前端多出 `code/departmentName/hotScore` |
| A-2 | 🟡中 | `types/course.ts:16-24` vs `model.go:47-57` | `RatingDimension.id` 前端为 `number`，后端 JSON tag 为 `string` |
| A-3 | 🟡中 | `api/course.ts:68-73` vs `repository_rating.go` | `RatingTrendItem` 前端 `reviewCount` vs 后端 `count`，字段名不匹配 |
| A-4 | 🟡中 | `review_interaction.go` vs `api/review.ts` | 成功响应格式不一致：部分返回 `{ message }` 部分返回 `{ success }` |
| A-5 | 🟡中 | `admin.go:78` vs `api/admin.ts:30` | Report ID 后端用 `ParseIDParam`（int64），前端传 `string` |
| A-6 | 🟡中 | `model.go:253-264` vs `api/content.ts` | `QualityCheckResult` 后端有完整类型，前端缺少对应类型定义 |
| A-7 | 🟡中 | `admin.go:248-273` vs `types/admin.ts:76` | 批量操作 ID 后端校验 UUID 格式，前端类型未约束 |
| A-8 | 🔵低 | `handler.go:98` | Draft 路由参数 `:courseId`（camelCase）违反项目命名规范（应为 `:courseID`） |
| A-9 | 🔵低 | `types/review.ts:25` vs `model.go:42` | `Review.status` 后端必需（无 omitempty），前端标记为可选 |
| A-10 | 🔵低 | `admin.go:366-367` vs `api/admin.ts:64` | CSV 导出无 Content-Type 校验，前端 `responseType: 'blob'` 假设固定格式 |

---

## 统计

| 类别 | 🔴高 | 🟡中 | 🔵低 | 合计 |
|------|------|------|------|------|
| 基础设施 | 1 | 5 | 2 | 8 |
| 安全 | 4 | 7 | 6 | 17 |
| 数据库 | 2 | 5 | 2 | 9 |
| Go 后端 | 7 | 27 | 9 | 43 |
| 前端 | 5 | 17 | 10 | 32 |
| 测试与 CI | 3 | 7 | 2 | 12 |
| API 设计 | 1 | 6 | 3 | 10 |
| **合计** | **23** | **74** | **34** | **131** |
