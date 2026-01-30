# 分层架构设计

本文档描述 StuHelper 后端服务的分层架构设计规范。

## 1. 架构概述

采用经典的三层架构模式，将代码按职责分离：

```
┌─────────────────────────────────────────┐
│           Handler 层 (HTTP)              │
│   - 请求解析、参数验证、响应格式化        │
│   - 缓存处理、错误码映射                  │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│           Service 层 (业务逻辑)          │
│   - 业务规则、数据验证、事务管理          │
│   - 调用 Repository 层                   │
└─────────────────────────────────────────┘
                    │
                    ▼
┌─────────────────────────────────────────┐
│         Repository 层 (数据访问)         │
│   - SQL 查询、数据库操作                 │
│   - 数据映射、结果扫描                   │
└─────────────────────────────────────────┘
```

## 2. 各层职责

### 2.1 Handler 层

**职责：**
- 处理 HTTP 请求和响应
- 解析请求参数（路径参数、查询参数、请求体）
- 调用 Service 层处理业务逻辑
- 格式化响应数据
- 处理缓存（读取/写入/失效）
- 将 Service 层错误映射为 HTTP 状态码

**禁止：**
- 直接编写 SQL 语句
- 包含业务逻辑
- 直接操作数据库

### 2.2 Service 层

**职责：**
- 实现业务逻辑和业务规则
- 数据验证（业务层面）
- 事务管理
- 调用 Repository 层进行数据操作
- 定义业务错误类型

**禁止：**
- 直接编写 SQL 语句
- 处理 HTTP 请求/响应
- 处理缓存

### 2.3 Repository 层

**职责：**
- 封装所有 SQL 查询
- 数据库 CRUD 操作
- 结果集扫描和数据映射
- 提供事务支持（接收 tx 参数）

**禁止：**
- 包含业务逻辑
- 处理 HTTP 请求/响应
- 处理缓存

## 3. 文件组织

每个模块应包含以下文件：

```
modules/
└── course/
    └── review/
        ├── handler.go      # Handler 层：HTTP 处理器
        ├── service.go      # Service 层：业务逻辑
        ├── repository.go   # Repository 层：数据访问
        ├── model.go        # 数据模型定义
        ├── cache.go        # 缓存相关方法
        └── utils.go        # 工具函数
```

## 4. 代码示例

### 4.1 Repository 层示例

```go
// repository.go
type Repository struct {
    db *db.DB
}

func NewRepository(database *db.DB) *Repository {
    return &Repository{db: database}
}

// 简单查询
func (r *Repository) GetByID(ctx context.Context, id int64) (*Model, error) {
    var m Model
    err := r.db.QueryRow(ctx, `SELECT id, name FROM table WHERE id = $1`, id).Scan(&m.ID, &m.Name)
    return &m, err
}

// 事务操作（接收 tx 参数）
func (r *Repository) Create(ctx context.Context, tx pgx.Tx, p CreateParams) error {
    _, err := tx.Exec(ctx, `INSERT INTO table (name) VALUES ($1)`, p.Name)
    return err
}
```

### 4.2 Service 层示例

```go
// service.go
type Service struct {
    db   *db.DB
    repo *Repository
    log  *zap.Logger
}

func NewService(database *db.DB, repo *Repository) *Service {
    return &Service{
        db:   database,
        repo: repo,
        log:  logger.L(),
    }
}

// 业务方法
func (s *Service) Create(ctx context.Context, params CreateParams) (*Result, error) {
    // 业务验证
    if params.Name == "" {
        return nil, ErrNameRequired
    }

    // 开启事务
    tx, err := s.db.Begin(ctx)
    if err != nil {
        return nil, err
    }
    defer tx.Rollback(ctx)

    // 调用 Repository 层
    err = s.repo.Create(ctx, tx, params)
    if err != nil {
        return nil, err
    }

    // 提交事务
    if err := tx.Commit(ctx); err != nil {
        return nil, err
    }

    return &Result{ID: params.ID}, nil
}
```

### 4.3 Handler 层示例

```go
// handler.go
type Handler struct {
    db      *db.DB
    cache   *redis.Client
    service *Service
}

func NewHandler(database *db.DB, cache *redis.Client) *Handler {
    repo := NewRepository(database)
    svc := NewService(database, repo)
    return &Handler{
        db:      database,
        cache:   cache,
        service: svc,
    }
}

// HTTP 处理方法
func (h *Handler) Create(c *gin.Context) {
    var req CreateRequest
    if err := c.ShouldBindJSON(&req); err != nil {
        c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
        return
    }

    // 调用 Service 层
    result, err := h.service.Create(c.Request.Context(), CreateParams{
        Name: req.Name,
    })
    if err != nil {
        // 错误映射
        switch {
        case errors.Is(err, ErrNameRequired):
            c.JSON(http.StatusBadRequest, gin.H{"error": "name is required"})
        default:
            c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
        }
        return
    }

    // 失效缓存
    _ = h.invalidateCache(c.Request.Context(), "prefix")

    c.JSON(http.StatusOK, gin.H{"id": result.ID})
}
```

## 5. 错误处理

### 5.1 Service 层错误定义

```go
// service.go
var (
    ErrNotFound       = errors.New("not found")
    ErrAlreadyExists  = errors.New("already exists")
    ErrInvalidInput   = errors.New("invalid input")
)
```

### 5.2 Handler 层错误映射

```go
// handler.go
switch {
case errors.Is(err, ErrNotFound):
    c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
case errors.Is(err, ErrAlreadyExists):
    c.JSON(http.StatusConflict, gin.H{"error": "resource already exists"})
case errors.Is(err, ErrInvalidInput):
    c.JSON(http.StatusBadRequest, gin.H{"error": "invalid input"})
default:
    c.JSON(http.StatusInternalServerError, gin.H{"error": "internal error"})
}
```

## 6. 最佳实践

1. **依赖注入**：通过构造函数注入依赖，便于测试和解耦
2. **接口隔离**：Service 层不应依赖具体的 HTTP 框架
3. **事务边界**：事务应在 Service 层管理，Repository 层接收 tx 参数
4. **错误传播**：使用 `errors.Is()` 进行错误类型判断
5. **日志记录**：在 Service 层记录业务日志，Handler 层记录请求日志
6. **缓存策略**：缓存逻辑放在 Handler 层，Service 层保持纯粹

## 7. 参考实现

完整的分层架构实现示例请参考：
- [review 模块](../../server/internal/modules/course/review/)
