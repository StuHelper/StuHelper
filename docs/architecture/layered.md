# Layered Architecture

The StuHelper backend follows a `Handler -> Service -> Repository` pattern. As modules grow, files are split by subdomain within the same layer.

## Layer Responsibilities

| Layer | Responsibilities |
| --- | --- |
| **Handler** | Parameter binding, service invocation, error mapping, `response.*` helpers, local cache read/write |
| **Service** | Business rules, transaction orchestration, access fact resolution, cross-repository coordination |
| **Repository** | SQL queries, result scanning, transactional data operations |

## File Splitting Strategy

Large modules in the repository are already split by subdomain:

| Module | Current Split |
| --- | --- |
| `user` | `handler.go`, `handler_self.go`, `handler_admin.go`, `service_identity.go`, `service_profile.go`, `service_admin.go` |
| `rbac` | `handler_roles.go`, `handler_users.go`, `handler_groups.go`, `service_roles.go`, `service_users.go`, `service_groups.go` |
| `course/review` | `review.go`, `review_read.go`, `review_reply.go`, `review_draft.go`, `service_review_write.go`, `service_report.go`, `service_admin_stats.go` |

## Directory Example

```text
review/
├── handler.go              # Main handler registration
├── admin.go                # Admin handler registration
├── admin_review.go         # Admin review operations
├── admin_export.go         # Admin export operations
├── review.go               # Public review handlers
├── review_read.go          # Review read handlers
├── review_reply.go         # Reply handlers
├── review_draft.go         # Draft handlers
├── service.go              # Main service struct
├── service_review_write.go # Review write operations
├── service_interaction.go  # Vote, favorite operations
├── service_report.go       # Report handling
├── service_admin_stats.go  # Admin statistics
├── repository.go           # Main repository struct
├── repository_review_query.go  # Review queries
├── repository_rating_stats.go  # Rating statistics
└── model.go                # Domain models
```

## Implementation Characteristics

- **SQL is centralized in repositories** - No SQL in handlers or services
- **Business composition is centralized in services** - No business logic in handlers
- **`response.*` helpers are used in handlers** - Consistent error and success responses
- **Generated code is in `server/internal/api/gen/`** - Never manually edited

## Dependency Injection

All dependencies are injected via constructors:

```go
// Service constructor
func NewService(
    repo *Repository,
    userRepo *user.Repository,
    logger *logger.Logger,
) *Service {
    return &Service{
        repo:     repo,
        userRepo: userRepo,
        logger:   logger,
    }
}

// Handler constructor
func NewHandler(
    svc *Service,
    authMW *middleware.AuthMiddleware,
) *Handler {
    return &Handler{
        svc:    svc,
        authMW: authMW,
    }
}
```

## Transaction Patterns

Services manage transactions:

```go
func (s *Service) CreateReview(ctx context.Context, req *CreateReviewRequest) error {
    return s.repo.WithTransaction(ctx, func(tx *gorm.DB) error {
        // All operations within transaction
        if err := s.repo.InsertReview(ctx, tx, review); err != nil {
            return err
        }
        if err := s.repo.UpdateCourseStats(ctx, tx, courseID); err != nil {
            return err
        }
        return nil
    })
}
```

## Error Handling

Errors flow up through layers:

```go
// Repository returns domain errors
func (r *Repository) GetReview(ctx context.Context, id string) (*Review, error) {
    var review Review
    if err := r.db.First(&review, "id = ?", id).Error; err != nil {
        if errors.Is(err, gorm.ErrRecordNotFound) {
            return nil, ErrReviewNotFound
        }
        return nil, fmt.Errorf("get review: %w", err)
    }
    return &review, nil
}

// Service adds context
func (s *Service) GetReview(ctx context.Context, id string) (*Review, error) {
    review, err := s.repo.GetReview(ctx, id)
    if err != nil {
        return nil, fmt.Errorf("service get review: %w", err)
    }
    return review, nil
}

// Handler maps to HTTP
func (h *Handler) GetReview(c *gin.Context) {
    id := c.Param("id")
    review, err := h.svc.GetReview(c.Request.Context(), id)
    if err != nil {
        response.Error(c, err)
        return
    }
    response.Success(c, review)
}
```

## Related Documentation

- [Backend Development Guide](../guides/backend-quickstart.md)
- [Error Handling](.trellis/spec/backend/error-handling.md)
- [Database Guidelines](.trellis/spec/backend/database-guidelines.md)
