# 使用示例

## 在 Handler 中使用

```go
func (h *Handler) CreateReview(c *gin.Context) {
    log := logger.ForGin(c)

    var req CreateReviewRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        log.Warn("invalid request body",
            zap.Error(err),
            zap.String("module", "course"),
        )
        c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request"})
        return
    }

    review, err := h.service.CreateReview(c.Request.Context(), &req)
    if err != nil {
        log.Error("failed to create review",
            zap.Error(err),
            zap.String("module", "course"),
            zap.String("action", "create"),
        )
        c.JSON(http.StatusInternalServerError, gin.H{"error": "failed"})
        return
    }

    log.Info("review created",
        zap.String("module", "course"),
        zap.String("review_id", review.ID),
    )
    c.JSON(http.StatusCreated, review)
}
```

## 在 Service 中使用

```go
func (s *Service) CreateReview(ctx context.Context, req *CreateReviewRequest) (*Review, error) {
    log := logger.FromContext(ctx)

    exists, err := s.repo.ExistsByUserAndCourse(ctx, req.UserID, req.CourseID)
    if err != nil {
        log.Error("failed to check existing review", zap.Error(err))
        return nil, err
    }

    if exists {
        log.Warn("duplicate review attempt",
            zap.String("user_id", req.UserID),
            zap.Int("course_id", req.CourseID),
        )
        return nil, ErrDuplicateReview
    }

    review, err := s.repo.Create(ctx, req)
    if err != nil {
        log.Error("failed to save review", zap.Error(err))
        return nil, err
    }

    return review, nil
}
```
