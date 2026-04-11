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

- 结构变更只改 `server/migrations/`
- `server/migrations/` 是唯一 schema 权威来源；不再维护仓库内 schema 快照
- 参数化查询，禁止拼接
- 动态排序使用白名单
- 分页优先 `COUNT(*) OVER()`

## 授权模型

1. Zitadel Token → 粗粒度角色
2. 角色静态展开 → capability（零 DB 查询）
3. 业务事实 + OpenFGA → 资源级判断

新路由需考虑：是否需要登录、需要哪些 capability、是否需要资源级校验。

## 基础设施依赖

| 组件 | 用途 |
|------|------|
| PostgreSQL | 业务数据 |
| Redis | 缓存、限流、token 黑名单 |
| Zitadel | OIDC / 角色同步 |
| OpenFGA | 资源关系授权 |
| Tencent SMS | 手机 OTP |
| OpenTelemetry | trace / metrics / logs |

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
