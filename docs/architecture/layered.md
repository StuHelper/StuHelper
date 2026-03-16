# 分层架构

StuHelper 后端统一按 `Handler → Service → Repository` 分层，但不要求每个模块只用一个 `handler.go`、`service.go`、`repository.go` 文件。

## 分层职责

### Handler

- 绑定参数
- 调用 service
- 映射错误
- 返回 `response.*`
- 做缓存读写和失效

### Service

- 组织业务规则
- 编排事务
- 解析授权事实
- 调用 repository

### Repository

- 持有 SQL
- 负责扫描和映射
- 提供事务内方法

## 当前文件拆分约定

当一个模块继续长大时，直接按子域拆文件，不要把所有逻辑塞回一个巨型文件。当前项目已经在这些地方这么做：

- `user/handler.go` + `handler_self.go` + `handler_admin.go`
- `rbac/handler.go` + `handler_roles.go` + `handler_users.go` + `handler_groups.go`
- `review/review.go` + `review_read.go`
- `review/service.go` + `service_review_write.go` + `service_report.go` + `service_admin_stats.go`
- `review/admin.go` + `admin_review.go` + `admin_export.go`
- `config/config.go` + `env.go` + `validation.go` + `security.go`

推荐规则：

- 继续保持三层，不要跨层
- 同层允许多个文件
- 单文件优先控制在 300 到 400 行附近
- 明显跨越两个子域时就拆，不要等到 700 行再收拾

## 当前禁止项

- handler 里写 SQL
- service 里直接依赖 gin
- repository 里做 HTTP 响应
- 直接 `c.JSON(...)`
- 手改生成代码

## 一个典型目录

```text
review/
├── handler.go
├── review.go
├── review_read.go
├── review_reply.go
├── service.go
├── service_review_write.go
├── service_interaction.go
├── service_report.go
├── repository.go
├── repository_review_query.go
├── repository_rating_stats.go
└── model.go
```

重点不是文件名漂不漂亮，而是职责清楚、文件别重新长成一坨。
