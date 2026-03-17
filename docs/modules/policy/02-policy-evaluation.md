# 授权决策流程

后端授权决策沿固定序列执行。评课路由和管理路由都在此决策链上运行。

## 决策链序列

```text
1. 会话认证
2. 本地用户同步
3. 能力计算
4. 管理能力校验
5. 业务访问事实解析
6. 所有权 / 资源状态检查
7. 响应内容裁剪
```

## 各步骤代码入口

| 步骤             | 代码入口                                                                                                                     |
| ---------------- | ---------------------------------------------------------------------------------------------------------------------------- |
| 会话认证         | `internal/pkg/middleware/auth.go` — `AuthMiddleware` 从 Cookie 或 Header 提取 token，校验黑名单，验证 JWT（iss/aud/alg/exp） |
| 本地用户同步     | `internal/modules/auth/user_sync.go` — `UpsertUser` 在 `buildUserInfo` 中调用                                                |
| 能力计算         | `internal/modules/rbac/service_permissions.go` — `GetUserCapabilities` 和 `GetEffectivePermissions`                          |
| 管理能力校验     | `internal/modules/rbac/middleware.go` — `RequirePermission` 和 `RequireAnyPermission`，含 scope 验证                         |
| 业务访问事实解析 | `internal/modules/course/review/access.go` — `resolveReviewAccessFacts`；`internal/modules/user/service*.go`                 |
| 所有权检查       | `internal/modules/course/review/review*.go` — 服务层事务内通过 `user_hash` 匹配                                              |
| 响应内容裁剪     | Handler 层和 `stripReviewsForResponse` — 根据访问事实裁剪内容                                                                |

## 评论访问决策示例

```go
// 伪代码
func HandleListReviews(c *gin.Context) {
    // 1. 可选认证中间件解析 token（匿名可访问）
    externalID := middleware.GetUserID(c)

    // 5. 解析访问事实
    facts := resolveReviewAccessFacts(ctx, externalID)
    // facts.Authenticated — 是否已认证
    // facts.StudentVerified — 学生认证是否通过
    // facts.IdentityVerified — 实名认证是否通过
    // facts.CanManageReviews — 是否持有 admin:reviews:manage
    // facts.CanViewFull — CanManageReviews || StudentVerified
    // facts.CanPostReview — StudentVerified && IdentityVerified
    // facts.PreviewTitleRunes / PreviewContentRunes / PreviewContentPct
    //   来自后台评课访问策略

    // 6. 查询评论列表
    reviews := queryReviews(courseID, page, pageSize)

    // 7. 根据访问事实裁剪内容
    for review in reviews {
        if review.Status == "hidden" && !facts.CanManageReviews {
            review.Title = ""
            review.Content = ""
        } else if !facts.Authenticated {
            review.Title = ""
            review.Content = ""
        } else if !facts.CanViewFull {
            review.Title = previewText(review.Title, facts.PreviewTitleRunes, 100)
            review.Content = previewText(review.Content, facts.PreviewContentRunes, facts.PreviewContentPct)
        }
    }

    return reviews
}
```

## 管理操作决策示例

```go
// 伪代码
func HandleAdminHideReview(c *gin.Context) {
    // 1. AuthMiddleware — 校验 token
    // 2. UpsertUser — 同步本地用户（在 /auth/me 中完成）
    // 3-4. RequireAnyPermission + RequirePermission
    //   a. 解析 external_id -> internal_user_id（缓存到 Gin context）
    //   b. 加载 effective permissions（缓存到 Gin context）
    //   c. 查找 "admin:reviews:manage" 并验证 scope
    //      - scope_school_ids: 检查请求的 schoolID 是否在白名单内
    //      - scope_roles: 检查用户是否持有要求的角色

    // 5-7. 业务逻辑
    reviewID := c.Param("id")
    err := reviewService.HideReview(ctx, reviewID, adminUsername)
    // 服务层内部检查 review 状态，记录操作日志
}
```

## 相关文档

- [授权模型](01-authorization-model.md)
- [RBAC 权限控制](../rbac/README.md)
