# 后端代码企业级审查报告（待修复问题清单）

> 审查日期: 2026-01-29
> 审查范围: server/ 目录下后端代码
> 说明: 已完全修复的问题已移除，仅保留"未修复 / 部分修复"问题，便于后续跟进与验收。

## 状态说明

- 🔴 未修复：尚未实施修复或无明确落地方案
- 🟡 部分修复：已实施部分方案，但未覆盖全量或未完成迁移

---

## P0（高风险 / 必须优先修复）

### P0-1 HMAC 空密钥风险 ✅ 已修复

**位置**: [crypto/hmac.go:22-45](server/internal/pkg/crypto/hmac.go#L22-L45)

**修复方案**:
- 修改 `InitHMACKey` 函数签名，增加 `isProduction` 参数
- 生产环境：密钥为空时返回错误，阻止应用启动
- 开发环境：密钥为空时生成 32 字节随机密钥并输出警告日志
- 增加密钥长度检查，短于 16 字符时输出警告

**修复文件**:
- `server/internal/pkg/crypto/hmac.go` - 重构 InitHMACKey 函数
- `server/cmd/stuhelper/main.go` - 更新调用方式，传入环境参数

**验收标准**: ✅
- 生产环境必须配置 HMAC_SECRET，否则启动失败
- 开发环境使用随机密钥时有明显警告日志

---

### P0-2 Refresh Token Cookie 路径错误 ✅ 已修复

**位置**: [auth/handler.go:321-330](server/internal/modules/auth/handler.go#L321-L330)

**修复方案**:
- 将 Cookie 路径从 `/auth/refresh` 修正为 `/api/v1/auth/refresh`
- 同时修复 `setTokenCookies` 和 `clearTokenCookies` 两个函数

**修复文件**:
- `server/internal/modules/auth/handler.go` - 修正 Cookie 路径

**验收标准**: ✅
- Token 刷新接口能正确接收 Refresh Token Cookie
- 前端调用刷新接口时 Cookie 正确发送

---

### P0-3 OAuth State 向后兼容存在安全隐患 ✅ 已修复

**位置**: [sso/client.go:77-87](server/internal/pkg/sso/client.go#L77-L87)

**修复方案**:
- 移除向后兼容代码，强制使用随机 state
- 当 `stateManager` 为 nil 时返回错误 `ErrStateManagerRequired`
- 所有 OAuth 流程必须通过 `NewClientWithCache` 创建客户端以启用 state 管理

**修复文件**:
- `server/internal/pkg/sso/client.go` - 移除固定 state 的向后兼容代码

**验收标准**: ✅
- 所有 OAuth 流程使用随机 state
- 固定 state 的请求被拒绝

---

### P0-4 JWT 校验不完整 ✅ 已修复

**位置**: [jwt/validator.go](server/internal/pkg/jwt/validator.go)

**修复方案**:
- 创建独立的 JWT 验证器 `internal/pkg/jwt/validator.go`
- 实现完整的 JWT 校验：
  - `iss` (issuer) 校验：必须匹配 Casdoor endpoint
  - `aud` (audience) 校验：必须匹配 Client ID
  - `alg` (algorithm) 白名单：仅允许 RS256/RS384/RS512/ES256/ES384/ES512，禁止 `none`
  - `exp`/`nbf`/`iat` 时间校验：支持 30 秒时钟偏移
- 更新 `token.Service` 集成 JWT 验证器
- 更新 `middleware.AuthMiddleware` 使用增强的验证

**修复文件**:
- `server/internal/pkg/jwt/validator.go` - 新建 JWT 验证器
- `server/internal/pkg/token/service.go` - 集成 JWT 验证器
- `server/internal/pkg/middleware/auth.go` - 使用新验证器
- `server/cmd/stuhelper/main.go` - 更新初始化配置

**验收标准**: ✅
- 无效 `iss/aud/alg` 的 token 被拒绝
- 过期 token 统一返回 401

---

### P0-5 健康检查泄露内部信息 ✅ 已修复

**位置**: [health/health.go:65-117](server/internal/pkg/health/health.go#L65-L117)

**修复方案**:
- 为 Handler 添加 `isProduction` 标志
- 生产环境 `/health/ready` 仅返回 `status` + `timestamp`
- 开发环境保留详细信息便于调试

**修复文件**:
- `server/internal/pkg/health/health.go` - 添加环境判断逻辑
- `server/cmd/stuhelper/main.go` - 传入 isProduction 参数

**验收标准**: ✅
- 生产环境不可从公网获取内部运行细节

---

### P0-6 速率限制器计数可被碰撞覆盖 ✅ 已修复

**位置**: [middleware/ratelimit.go:29-56](server/internal/pkg/middleware/ratelimit.go#L29-L56)

**修复方案**:
- 为每个请求生成唯一 member（8字节随机数的十六进制表示）
- Lua 脚本中使用唯一 member 替代 `now` 作为 ZSET 成员
- 确保毫秒内并发请求不会相互覆盖

**修复文件**:
- `server/internal/pkg/middleware/ratelimit.go` - 添加 generateUniqueID 函数，修改 Lua 脚本

**验收标准**: ✅
- 并发压测时限流触发符合预期阈值

---

### P0-7 Token 黑名单缺少熔断机制 ✅ 已修复

**位置**: [token/blacklist.go:53-78](server/internal/pkg/token/blacklist.go#L53-L78)

**修复方案**:
- 创建独立的熔断器包 `internal/pkg/circuitbreaker`
- 实现三态熔断器：Closed（正常）→ Open（熔断）→ HalfOpen（恢复尝试）
- 配置：5次失败触发熔断，30秒超时后尝试恢复，2次成功恢复正常
- 熔断器打开时降级策略：允许请求通过但记录警告日志

**修复文件**:
- `server/internal/pkg/circuitbreaker/circuitbreaker.go` - 新建熔断器实现
- `server/internal/pkg/token/blacklist.go` - 集成熔断器，添加 CircuitBreakerMetrics 方法

**验收标准**: ✅
- Redis 短暂故障时服务可降级运行
- 熔断器状态可观测

---

## P1（中风险 / 需要计划内修复）

### P1-1 QueryRow 超时上下文可能提前取消 ✅ 已修复

**位置**: [db/db.go:52-68](server/internal/pkg/db/db.go#L52-L68)

**修复方案**:
- 创建 `RowWithCancel` 包装类型
- 在 `Scan` 方法中 defer cancel，确保 Scan 完成后才取消 context
- 返回类型从 `pgx.Row` 改为 `*RowWithCancel`

**修复文件**:
- `server/internal/pkg/db/db.go` - 添加 RowWithCancel 类型

**验收标准**: ✅
- `QueryRow` 相关接口在压力下无随机 `context canceled`

---

### P1-2 未检查 `rows.Err()` ✅ 已修复

**位置**:
- [course/course.go](server/internal/modules/course/course.go)
- [review/review.go](server/internal/modules/course/review/review.go)

**修复方案**:
- 在所有 `for rows.Next()` 循环后添加 `rows.Err()` 检查
- 修改 `scanReviews` 函数接口，添加 `Err()` 方法

**修复文件**:
- `server/internal/modules/course/course.go` - 3处添加 rows.Err() 检查
- `server/internal/modules/course/review/review.go` - scanReviews 函数添加 rows.Err() 检查

**验收标准**: ✅
- 数据库读取错误能被显式返回并记录

---

### P1-3 ID 参数允许非正数 ✅ 已修复

**位置**:
- [course/utils.go:33-42](server/internal/modules/course/utils.go#L33-L42)
- [review/utils.go:33-42](server/internal/modules/course/review/utils.go#L33-L42)

**修复方案**:
- 在 `parseIDParam` 函数中添加 `id <= 0` 检查
- 非正数 ID 返回 `strconv.ErrRange` 错误

**修复文件**:
- `server/internal/modules/course/utils.go`
- `server/internal/modules/course/review/utils.go`

**验收标准**: ✅
- 负数/0 的 ID 请求返回 400

---

### P1-4 评论内容存在存储型 XSS 风险 ✅ 已修复

**位置**: [review/review.go:106-114](server/internal/modules/course/review/review.go#L106-L114)

**修复方案**:
- 创建独立的 sanitizer 包 `internal/pkg/sanitizer`
- 实现 `SanitizeText()` 函数：移除 HTML 标签、转义特殊字符、规范化空白
- 实现 `SanitizeTitle()` 函数：更严格的标题清理，移除换行符
- 实现 `ContainsDangerousContent()` 函数：检测 script/iframe/object/embed 标签、JS 事件处理器、javascript: URL
- 在 `PostReview` 处理器中集成：先检测危险内容并拒绝，再清理用户输入

**修复文件**:
- `server/internal/pkg/sanitizer/sanitizer.go` - 新建 sanitizer 包
- `server/internal/modules/course/review/review.go` - 集成 sanitizer

**验收标准**: ✅
- 提交 `<script>alert(1)</script>` 不会在前端执行
- 包含危险内容的请求被拒绝（返回 400）

---

### P1-5 缺少输入验证 ✅ 已修复

**位置**: [review/review.go:106-114](server/internal/modules/course/review/review.go#L106-L114)

**修复方案**:
- 为 `PostReviewRequest` 结构体添加完整的 binding 验证标签
- `CourseID`: 添加 `gt=0` 确保为正数
- `TeacherID`: 添加 `omitempty,gt=0` 可选但必须为正数
- `TermID`: 添加 `omitempty,max=20` 限制长度
- `Grade`: 添加 `omitempty,oneof=A+ A A- B+ B B- C+ C C- D F` 限制有效值

**修复文件**:
- `server/internal/modules/course/review/review.go` - 更新 PostReviewRequest 结构体

**验收标准**: ✅
- 无效 Grade 值被拒绝
- 非正数 CourseID 被拒绝

---

### P1-6 Redis/Postgres 连接未显式强制 TLS ✅ 已修复

**位置**:
- [db/pg.go](server/internal/pkg/db/pg.go)
- [redis/client.go](server/internal/pkg/redis/client.go)

**修复方案**:
- 为 `DatabaseConfig` 添加 TLS 配置字段：`SSLMode`, `SSLRootCert`, `SSLCert`, `SSLKey`
- 为 `RedisConfig` 添加 TLS 配置字段：`TLSEnabled`, `TLSCertFile`, `TLSKeyFile`, `TLSCAFile`, `TLSInsecure`
- PostgreSQL 支持四种 SSL 模式：`disable`, `require`, `verify-ca`, `verify-full`
- Redis 支持可选的 TLS 连接，包括客户端证书认证
- 生产环境配置验证：强制要求 PostgreSQL 使用 TLS（非 disable 模式）
- Redis TLS 未启用时输出警告日志

**修复文件**:
- `server/internal/pkg/config/config.go` - 添加 TLS 配置字段和生产环境验证
- `server/internal/pkg/db/pg.go` - 添加 `configurePGTLS` 函数
- `server/internal/pkg/redis/client.go` - 添加 `configureRedisTLS` 函数

**验收标准**: ✅
- 生产环境连接明示加密并有证书校验
- 开发环境可选择禁用 TLS

---

### P1-7 缺少全局与用户维度限流 ✅ 已修复

**位置**:
- [auth/handler.go](server/internal/modules/auth/handler.go)
- [review/review.go](server/internal/modules/course/review/review.go)

**修复方案**:
- 添加 `RateLimitConfig` 配置结构，支持全局/IP/用户三个维度的限流配置
- 实现 `GlobalRateLimitMiddleware`：全局限流，防止服务过载
- 实现 `UserRateLimitMiddleware`：用户维度限流，防止单用户滥用
- 实现 `EndpointRateLimitMiddleware`：端点限流，用于敏感操作（发布评论、投票等）
- 提供 `DefaultRateLimitConfig()` 默认配置

**修复文件**:
- `server/internal/pkg/middleware/ratelimit.go` - 添加多维度限流中间件

**验收标准**: ✅
- 异常刷接口行为被限流
- 支持全局、IP、用户、端点四个维度的限流

---

### P1-8 事务错误处理不完整 ✅ 已修复

**位置**: [review/review.go:159-187](server/internal/modules/course/review/review.go#L159-L187)

**修复方案**:
- 为所有事务操作添加详细的错误日志（使用 zap 结构化日志）
- 修改 defer rollback 处理：检查 `pgx.ErrTxClosed` 避免已提交事务的无效回滚警告
- 为每个数据库操作添加上下文信息（review_id, course_id, vote_type 等）
- 区分 Error 和 Warn 级别：操作失败用 Error，回滚失败用 Warn

**修复文件**:
- `server/internal/modules/course/review/review.go` - 改进 PostReview 和 VoteReview 的事务错误处理

**验收标准**: ✅
- 事务失败时有详细错误日志
- Rollback 失败有警告日志

---

## P2（性能 / 工程一致性）

### P2-1 缓存失效采用 `SCAN` + `DEL` ✅ 已修复

**位置**:
- [course/cache.go:52-94](server/internal/modules/course/cache.go#L52-L94)
- [review/cache.go:50-92](server/internal/modules/course/review/cache.go#L50-L92)

**修复方案**:
- 实现基于版本号的缓存失效策略，替代 SCAN + DEL
- 添加 `cacheVersionKey()` 函数生成版本号 key
- 添加 `getCacheVersion()` 函数获取当前版本号
- 添加 `buildCacheKey()` 函数构建带版本号的缓存 key
- 修改 `invalidateCache()` 使用 INCR 递增版本号
- 旧缓存根据 TTL 自然过期，避免 SCAN 操作

**修复文件**:
- `server/internal/modules/course/cache.go` - 实现版本号策略
- `server/internal/modules/course/review/cache.go` - 实现版本号策略

**验收标准**: ✅
- 高频写入时缓存失效不会引发 Redis 性能抖动

---

### P2-2 全局变量并发安全问题 ✅ 已修复

**位置**: [db/db.go:13-17](server/internal/pkg/db/db.go#L13-L17)

**修复方案**:
- 移除未使用的全局变量 `QueryTimeout` 和函数 `SetQueryTimeout`
- `DB` 结构体已有实例级别的 `timeout` 字段，无需全局变量
- 通过 `NewDB()` 构造函数传入超时配置，避免并发问题

**修复文件**:
- `server/internal/pkg/db/db.go` - 移除未使用的全局变量和函数

**验收标准**: ✅
- 无未使用的全局变量
- 超时配置通过实例字段管理，并发安全

---

### P2-3 Docker Compose 缺少资源限制 ✅ 已修复

**位置**: [deployments/docker-compose.yml](server/deployments/docker-compose.yml)

**修复方案**:
- 为 PostgreSQL 添加资源限制：CPU 2核/内存 2G，预留 0.5核/512M
- 为 Redis 添加资源限制：CPU 1核/内存 512M，预留 0.25核/128M
- 使用 deploy.resources 配置，兼容 Docker Compose v3.8+

**修复文件**:
- `server/deployments/docker-compose.yml` - 添加 deploy.resources 配置

**验收标准**: ✅
- 所有服务有合理的资源限制

---

### P2-4 缺少 Redis 持久化配置 ✅ 已修复

**位置**: [deployments/docker-compose.yml](server/deployments/docker-compose.yml)

**修复方案**:
- 添加 `--appendonly yes` 启用 AOF 持久化
- 添加 `--maxmemory 256mb` 限制内存使用
- 添加 `--maxmemory-policy allkeys-lru` 设置内存淘汰策略
- 添加健康检查确保 Redis 服务可用

**修复文件**:
- `server/deployments/docker-compose.yml` - 添加 Redis command 配置

**验收标准**: ✅
- Redis 数据在重启后可恢复

---

### P2-5 重复的工具函数 ✅ 已修复

**位置**:
- [course/utils.go](server/internal/modules/course/utils.go)
- [review/utils.go](server/internal/modules/course/review/utils.go)

**修复方案**:
- 更新 `httputil` 包的 `ParseIDParam` 函数，添加正数验证
- 修改 `course/utils.go` 使用 `httputil` 包的函数
- 修改 `review/utils.go` 使用 `httputil` 包的函数
- 保留本地包装函数以保持 API 兼容性，但实现委托给 `httputil`

**修复文件**:
- `server/internal/pkg/httputil/httputil.go` - 添加正数 ID 验证
- `server/internal/modules/course/utils.go` - 使用 httputil 包
- `server/internal/modules/course/review/utils.go` - 使用 httputil 包

**验收标准**: ✅
- 无重复的工具函数实现
- 所有模块使用统一的 httputil 包

---

## 架构与工程（未修复）

### A-1 三层架构重构 ✅ 已修复

**位置**: 全局

**修复方案**:
- 采用 Handler → Service → Repository 三层架构
- Handler 层：HTTP 请求处理、缓存、响应格式化
- Service 层：业务逻辑、数据验证、事务管理
- Repository 层：SQL 查询、数据库操作

**已完成**:
- `server/internal/modules/course/review/` - 完整三层架构
  - `repository.go` - 数据访问层
  - `service.go` - 业务逻辑层
  - `handler.go` / `review.go` / `rating.go` - HTTP 处理层
- `server/internal/modules/course/` - 完整三层架构
  - `repository.go` - 数据访问层
  - `service.go` - 业务逻辑层
  - `handler.go` / `course.go` - HTTP 处理层
- `docs/architecture/layered-architecture.md` - 架构设计文档

**auth 模块状态**:
- 使用外部服务（sso.Client, token.Service），无直接数据库操作
- 当前架构可接受，无需强制重构

**架构模式**:
```
Handler → Service → Repository → Database
```

**验收标准**: ✅
- 所有 SQL 查询封装在 Repository 层
- 业务逻辑封装在 Service 层
- Handler 层不包含直接 SQL 操作

---

### A-2 缺少依赖注入容器 🟡 部分修复

**位置**: [cmd/stuhelper/main.go](server/cmd/stuhelper/main.go)

**建议方案**:
- 推荐使用 Google Wire 进行编译时依赖注入
- 或使用 Uber fx 进行运行时依赖注入

**实施步骤**:
1. 安装 wire: `go install github.com/google/wire/cmd/wire@latest`
2. 创建 `wire.go` 定义 Provider 和 Injector
3. 运行 `wire ./...` 生成依赖注入代码
4. 更新 main.go 使用生成的 Injector

**当前状态**:
- 依赖管理通过构造函数手工拼装
- 已有清晰的依赖关系，便于后续迁移

---

### A-3 缺少 Metrics 指标 🟡 部分修复

**位置**: 全局

**建议方案**:
- 使用 Prometheus client_golang 暴露指标
- 添加 `/metrics` 端点

**推荐指标**:
- `http_requests_total` - HTTP 请求计数
- `http_request_duration_seconds` - 请求延迟直方图
- `db_query_duration_seconds` - 数据库查询延迟
- `cache_hits_total` / `cache_misses_total` - 缓存命中率
- `errors_total` - 错误计数

**实施步骤**:
1. 添加依赖: `go get github.com/prometheus/client_golang`
2. 创建 `internal/pkg/metrics/metrics.go`
3. 在中间件中记录 HTTP 指标
4. 在数据库操作中记录查询延迟

---

### A-4 缺少数据库迁移工具 🟡 部分修复

**位置**: 全局

**建议方案**:
- 推荐使用 golang-migrate 或 goose
- 创建 `migrations/` 目录存放迁移文件

**实施步骤**:
1. 安装: `go install github.com/pressly/goose/v3/cmd/goose@latest`
2. 创建 `server/migrations/` 目录
3. 创建迁移文件: `goose create init sql`
4. 在 main.go 启动时执行迁移

**命名规范**:
- `YYYYMMDDHHMMSS_description.sql`
- 例如: `20260129120000_create_users_table.sql`

---

## 部分修复（需持续推进）

### B-1 统一响应格式 ✅ 已修复

**位置**:
- [response/response.go](server/internal/pkg/response/response.go)
- 各 Handler

**修复方案**:
- 所有 Handler 迁移使用 `response` 包的统一响应函数
- 使用 `response.Success()`, `response.BadRequest()`, `response.NotFound()` 等

**已迁移模块**:
- `server/internal/modules/auth/handler.go` - 20 处
- `server/internal/modules/course/course.go` - 17 处
- `server/internal/modules/course/review/review.go` - 21 处
- `server/internal/modules/course/review/rating.go` - 8 处

**验收标准**: ✅
- 所有 API 响应使用统一格式 `{success, data, error}`
- 错误响应包含标准错误码

---

### B-2 单元测试覆盖不足 🟡

**位置**:
- [middleware/](server/internal/pkg/middleware/)
- [crypto/](server/internal/pkg/crypto/)
- [sso/](server/internal/pkg/sso/)

**现状**:
- 已覆盖 CSRF/日志脱敏/HMAC/State 等模块
- 业务模块（course/review/auth）测试仍不足

**建议**:
- 为核心业务与安全路径补齐单测
- 添加 auth/handler_test.go
- 添加 token/blacklist_test.go

---

## 优先级摘要

| 优先级 | 总数 | 已修复 | 待修复 | 关键问题 |
|--------|------|--------|--------|----------|
| P0 | 7 | 7 | 0 | HMAC空密钥、Cookie路径、OAuth State、JWT校验、健康检查泄露、限流碰撞、黑名单熔断 |
| P1 | 8 | 8 | 0 | QueryRow超时、rows.Err、ID校验、XSS、输入验证、TLS、全局限流、事务处理 |
| P2 | 5 | 5 | 0 | SCAN性能、全局变量、Docker资源、Redis持久化、重复代码 |
| 架构 | 4 | 1 | 3 | 三层架构✅、依赖注入、Metrics、数据库迁移 |
| 部分修复 | 2 | 1 | 1 | 响应格式✅、单元测试 |

---

## 修复进度跟踪

### P0 - 高优先级
- [x] P0-1 HMAC 空密钥风险
- [x] P0-2 Refresh Token Cookie 路径错误
- [x] P0-3 OAuth State 向后兼容安全隐患
- [x] P0-4 JWT 校验不完整
- [x] P0-5 健康检查泄露内部信息
- [x] P0-6 速率限制器计数碰撞
- [x] P0-7 Token 黑名单缺少熔断机制

### P1 - 中优先级
- [x] P1-1 QueryRow 超时上下文
- [x] P1-2 未检查 rows.Err()
- [x] P1-3 ID 参数允许非正数
- [x] P1-4 存储型 XSS 风险
- [x] P1-5 缺少输入验证
- [x] P1-6 Redis/Postgres TLS
- [x] P1-7 缺少全局与用户维度限流
- [x] P1-8 事务错误处理不完整

### P2 - 低优先级
- [x] P2-1 缓存失效 SCAN 性能
- [x] P2-2 全局变量并发安全
- [x] P2-3 Docker Compose 资源限制
- [x] P2-4 Redis 持久化配置
- [x] P2-5 重复的工具函数

### 架构改进
- [x] A-1 三层架构重构
- [ ] A-2 引入依赖注入
- [ ] A-3 添加 Metrics 指标
- [ ] A-4 添加数据库迁移工具

### 部分修复
- [x] B-1 统一响应格式
- [ ] B-2 单元测试覆盖（补充中）

---

## 参考资料

- [Uber Go Style Guide](https://github.com/uber-go/guide)
- [Go Code Review Comments](https://github.com/golang/go/wiki/CodeReviewComments)
- [Effective Go](https://golang.org/doc/effective_go)
- [OWASP Go Security Cheat Sheet](https://cheatsheetseries.owasp.org/cheatsheets/Go_Security_Cheat_Sheet.html)
