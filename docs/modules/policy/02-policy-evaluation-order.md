# Authorization Decision Flow

Backend authorization decisions follow a fixed sequence. Both review and admin routes execute along this chain.

## Decision Sequence

```text
1. Session authentication
2. Local user sync
3. Capability computation
4. Admin capability check
5. Business access fact evaluation
6. Ownership / resource status check
7. Response content shaping
```

## Layer Entry Points

| Step | Code Entry Point |
| --- | --- |
| Session authentication | `internal/pkg/middleware/auth.go` |
| Local user sync | `internal/modules/auth/user_sync.go` |
| Capability computation | `internal/modules/rbac/service*.go` |
| Admin capability check | `internal/modules/rbac/middleware.go` |
| Business access fact evaluation | `internal/modules/course/review/access.go`, `internal/modules/user/service*.go` |
| Ownership check | `internal/modules/course/review/review*.go` |
| Response shaping | Handler layer and review access results |

## Example: Review Access Decision

```go
// 1. Session authentication (middleware)
func AuthMiddleware(c *gin.Context) {
    token := extractToken(c)
    if token == "" {
        response.Error(c, ErrTokenMissing)
        c.Abort()
        return
    }

    userID, err := validateToken(token)
    if err != nil {
        response.Error(c, err)
        c.Abort()
        return
    }

    c.Set("userID", userID)
    c.Next()
}

// 2-3. User sync and capability computation (in handler)
func (h *Handler) GetReview(c *gin.Context) {
    userID := c.GetString("userID")

    // Sync user if needed
    user, err := h.authService.SyncUser(c.Request.Context(), userID)
    if err != nil {
        response.Error(c, err)
        return
    }

    // 4. Skip admin capability check for public endpoints

    // 5. Evaluate access facts
    accessFacts := h.reviewService.GetAccessFacts(c.Request.Context(), userID)

    // 6. Get review with ownership and status checks
    review, err := h.reviewService.GetReview(c.Request.Context(), reviewID, accessFacts)
    if err != nil {
        response.Error(c, err)
        return
    }

    // 7. Shape response based on access facts
    if !accessFacts.StudentVerified {
        review.Content = truncateContent(review.Content)
    }

    response.Success(c, review)
}
```

## Example: Admin Action Decision

```go
// 1-3. Session auth, user sync, capability computation (middleware)

// 4. Admin capability check (middleware)
adminGroup := router.Group("/api/v1/admin")
adminGroup.Use(rbacMiddleware.RequireCapability("admin:dashboard:view"))

reviewsGroup := adminGroup.Group("/reviews")
reviewsGroup.Use(rbacMiddleware.RequireCapability("admin:reviews:manage"))

// 5-7. Business logic in handler
func (h *Handler) UpdateReviewStatus(c *gin.Context) {
    reviewID := c.Param("id")
    var req UpdateReviewStatusRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        response.Error(c, ErrBadRequest)
        return
    }

    // Service handles ownership and status checks
    if err := h.reviewService.UpdateReviewStatus(c.Request.Context(), reviewID, req.Status); err != nil {
        response.Error(c, err)
        return
    }

    response.Success(c, nil)
}
```

## Related Documentation

- [Authorization Model](01-hangxiaoban-authorization-model.md)
- [RBAC Module](../rbac/README.md)
- [Layered Architecture](../../architecture/layered.md)
