# 分层架构

StuHelper 后端当前遵循 `Handler -> Service -> Repository` 分层模式。模块变大后，会在同一层内按子领域拆文件，而不是把所有逻辑继续堆在单文件里。

## 各层职责

| 层         | 职责                                                                  |
| ---------- | --------------------------------------------------------------------- |
| Handler    | 参数绑定、调用 Service、错误映射、`response.*` 响应包装、本地缓存读写 |
| Service    | 业务规则、事务编排、访问事实解析、跨仓储协调                          |
| Repository | SQL 查询、结果扫描、事务内数据操作                                    |

## 文件拆分策略

仓库里的大模块已经按子领域拆开：

| 模块            | 当前拆分                                                                                                                                      |
| --------------- | --------------------------------------------------------------------------------------------------------------------------------------------- |
| `user`          | `handler.go`、`handler_self.go`、`handler_admin.go`、`service_identity.go`、`service_profile.go`、`service_admin.go`                          |
| `rbac`          | `handler_roles.go`、`handler_users.go`、`handler_groups.go`、`service_roles.go`、`service_users.go`、`service_groups.go`                      |
| `course/review` | `review.go`、`review_read.go`、`review_reply.go`、`review_draft.go`、`service_review_write.go`、`service_report.go`、`service_admin_stats.go` |

## 目录示例

```text
review/
├── handler.go                  # 主路由注册
├── admin.go                    # 后台路由和处理器
├── admin_review.go             # 后台评论操作
├── admin_export.go             # 后台导出
├── review.go                   # 公开评论 Handler
├── review_read.go              # 评论读取 Handler
├── review_reply.go             # 回复 Handler
├── review_draft.go             # 草稿 Handler
├── service.go                  # 主 Service 结构
├── service_review_write.go     # 评论写入
├── service_interaction.go      # 投票、收藏、回复
├── service_report.go           # 举报处理
├── service_admin_stats.go      # 后台统计
├── repository.go               # 主 Repository 结构
├── repository_review_query.go  # 评论查询
├── repository_rating_stats.go  # 评分统计
└── model.go                    # 领域模型
```

## 当前实现特征

- SQL 集中在 Repository，不写进 Handler 和 Service
- 业务编排集中在 Service，不把业务判断散进 Handler
- HTTP 响应统一走 `response.*`
- 生成代码位于 `server/internal/api/gen/`，禁止手改

## 依赖注入

依赖统一通过构造函数注入，不在运行时临时拼装。

常见模式是：

- Repository 接受数据库连接
- Service 接受 Repository 和外部依赖
- Handler 接受 Service、中间件或权限服务

这样做的目的不是形式统一，而是让测试替换依赖更容易，也避免模块在运行时偷偷耦合。

## 事务模式

事务由 Service 管理。只要一个业务动作会同时写多张表，或要保证先查后写的一致性，就应该把事务边界放在 Service 层。

Repository 负责提供事务内的具体读写方法，不在 Handler 里开事务。

## 错误处理

错误沿层向上流动：

- Repository 返回领域错误或包装后的基础设施错误
- Service 在需要时补业务上下文
- Handler 负责把已知业务错误映射为稳定的 HTTP 状态码和错误码

不要让 Repository 直接决定 HTTP 语义，也不要让 Handler 去猜数据库错误。

## 相关文档

- [后端开发指南](../guides/backend-quickstart.md)
- [数据库参考](../reference/database.md)
- [错误码](../reference/error-codes.md)
