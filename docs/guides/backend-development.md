---
type: guide
audience: backend-dev
status: current
authoritative-source: server/api/openapi.yaml + server/internal/
last-verified: 2026-07-31
---

# 后端开发规范

## 分层约束

```
Handler   → HTTP 绑定、响应、错误映射
Service   → 业务规则、事务编排、跨 Repository 协调
Repository → SQL、结果扫描、事务内操作
```

- SQL 只写在 Repository
- 业务判断只放在 Service
- HTTP 响应统一通过 `response.*` 返回，不直接 `c.JSON`
- 不手改 `server/internal/api/gen/`
- 配置从环境变量读取，不硬编码

## OpenAPI 生成代码边界

- `server/internal/api/gen/` 的 Go 生成代码主要用于：
  - embedded OpenAPI spec
  - 运行时请求契约校验（`internal/pkg/middleware/openapi_validation.go`）
  - drift gate / 生成链校验
- Handler 层允许继续使用局部 request DTO，只要：
  - 契约权威仍然是 `server/api/openapi.yaml`
  - 运行时已接入 OpenAPI request validation
  - 变更接口时始终先改 OpenAPI，再 `make generate`
- 不要求把 handler 强行改写成直接依赖 `gen.*` 类型；否则会把 HTTP 绑定细节、业务演进节奏和生成模型硬耦合在一起。

## 改接口流程

```bash
cd server

# 1. 改 OpenAPI
make lint-spec

# 2. 重新生成
make generate

# 3. 补充实现

# 4. 校验
make fmt && make lint && make test && make build
make check-drift
```

## 目录结构

```
server/
├── cmd/stuhelper/        # 入口
├── api/                  # OpenAPI 源文件
├── internal/
│   ├── api/gen/          # 生成代码（禁止手改）
│   ├── modules/          # 业务模块
│   └── pkg/              # 内部公共包
├── migrations/           # 数据库权威来源
└── scripts/              # schema 快照、seed
```

`modules/rbac/` 仅保留 `middleware.go`（capability 中间件），不再是完整 RBAC 模块。

## 数据库规则

- `server/migrations/` 中按版本顺序应用的完整 migration 集合是 schema 权威来源
- `000001_initial_schema.*.sql` 是不可变初始基线；任何结构或数据演进都新增递增编号的
  `.up.sql` / `.down.sql` 文件对，禁止修改已有编号向已迁移环境发布变更
- 迁移、回退、dirty state 和生产演练要求见
  [数据库迁移运行手册](database-migrations.md)
- 参数化查询，禁止拼接
- 动态排序使用白名单
- 分页优先 `COUNT(*) OVER()`

## 授权模型

1. Casdoor OIDC token → 只提供已验证身份、应用和 MFA proof；role claim 永不参与 allow/deny
2. PostgreSQL `authorization_grants` + DB 业务事实 → DB-derived access snapshot
3. snapshot role/scope 静态展开 → capability；撤权由 DB desired-state 立即围栏
4. OpenFGA serving projection → 资源级关系判断；可从 PostgreSQL 账本重建

新路由需考虑：是否需要登录、需要哪些 capability、是否需要资源级校验。

## 基础设施依赖

| 组件 | 用途 |
|------|------|
| PostgreSQL | 业务数据 |
| Redis | 缓存、限流、token 黑名单 |
| Casdoor | OIDC 认证、会话、token 与登录层 MFA |
| OpenFGA | 可从 PostgreSQL 重建的资源关系授权投影 |
| Tencent SMS | 手机 OTP（仅当 `SMS_ENABLED=true`） |
| OpenTelemetry | trace / metrics / logs |

### 版本化业务缓存的故障语义

- Redis 中不存在版本 key 时，版本化缓存可以使用初始版本 `v0`。
- Redis 连接、超时或请求取消导致版本无法确认时，必须把版本视为“未知”：缓存读取按 miss
  处理、缓存写入按 no-op 处理，并回源 PostgreSQL；不得把依赖故障伪装成 `v0`，也不得把
  这个结果写入本地版本缓存。
- `cache.Helper.BuildVersionedKey` 以空 key 表示版本未知；`GetRaw`、`GetAs` 和 `Set`
  对空 key 分别执行 miss、miss 和 no-op。业务 Handler 不应自行拼接版本化 key 或绕过这组语义。

## 日志和错误

- Zap 结构化日志，敏感信息必须脱敏
- 业务错误使用结构化错误码
- Handler 层做 `errors.Is` 分类

统一响应格式：

```json
{
  "success": false,
  "error": {
    "code": "A0010200",
    "message": "permission denied"
  }
}
```

## 日常命令

```bash
cd server
make fmt && make lint && make test && make build
make generate && make lint-spec && make check-drift
```

## 提交前检查

- [ ] OpenAPI 和实现一致
- [ ] SQL 都在 Repository
- [ ] 业务判断都在 Service
- [ ] 响应通过 `response.*` 返回
- [ ] 新接口有认证和 capability 边界
- [ ] 日志无敏感信息
- [ ] `make lint` / `make test` / `make build` 通过
