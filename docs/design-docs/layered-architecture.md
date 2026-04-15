# 后端分层架构

> 状态：现行

## 分层

```
Handler → Service → Repository
```

| 层 | 职责 |
|----|------|
| Handler | 参数绑定、调用 Service、错误映射、`response.*` 响应 |
| Service | 业务规则、事务编排、访问事实解析、跨 Repository 协调 |
| Repository | SQL 查询、结果扫描、事务内操作 |

## 文件拆分

模块变大时按子领域拆文件：

| 模块 | 拆分示例 |
|------|----------|
| `user` | `handler.go` / `handler_self.go` / `handler_admin.go` / `external_sync.go`（含 FGA 逻辑）/ `service_identity.go` / `service_profile.go` / `service_admin.go` / `repository_identity.go` / `repository_profile.go` |
| `course/review` | `handler.go` / `admin.go` / `review.go` / `review_read.go` / `review_reply.go` / `review_draft.go` / `service_review_write.go` / `service_report.go` / `service_admin.go` / `repository_review_query.go` / `repository_rating.go` |
| `rbac` | `middleware.go`（仅 capability 中间件） |

## 依赖注入

通过构造函数注入，不用全局变量：

- Repository 接受数据库连接
- Service 接受 Repository 和外部依赖
- Handler 接受 Service

## 事务

Service 层管理事务边界。Repository 提供事务内读写方法，Handler 不开事务。

## 错误流动

```
Repository → 领域错误或包装后的基础设施错误
Service    → 补业务上下文
Handler    → 映射为 HTTP 状态码和错误码
```

Repository 不决定 HTTP 语义，Handler 不猜数据库错误。

## 约束

- SQL 集中在 Repository
- 业务编排集中在 Service
- HTTP 响应统一通过 `response.*` 返回
- 生成代码 `server/internal/api/gen/` 禁止手改
- 配置统一通过 `internal/pkg/config` 读取
