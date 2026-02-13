# 后端开发指南

本文档介绍后端项目结构和开发模式，帮助你快速上手编写业务代码。

> 环境搭建请先完成 [快速开始](../tutorials/quick-start.md)。

## 项目结构

```
server/
├── cmd/stuhelper/       # 应用入口
├── internal/
│   ├── modules/         # 业务模块（按领域划分）
│   │   ├── auth/        # 认证模块
│   │   ├── course/      # 课程模块
│   │   └── review/      # 评课模块
│   └── pkg/             # 公共包（中间件、工具函数等）
├── deployments/         # Docker Compose、.env 配置
└── scripts/             # 数据库初始化和种子数据
```

每个业务模块遵循三层架构：`Handler → Service → Repository`。

## 添加新 API 端点

以"获取课程详情"为例，展示完整的开发流程。

### 1. Repository — 数据访问

```go
// internal/modules/course/repository.go
func (r *Repository) GetByID(ctx context.Context, id int64) (*Course, error) {
    var course Course
    err := r.db.QueryRow(ctx, `SELECT id, name FROM courses WHERE id = $1`, id).
        Scan(&course.ID, &course.Name)
    return &course, err
}
```

### 2. Service — 业务逻辑

```go
// internal/modules/course/service.go
func (s *Service) GetCourseDetail(ctx context.Context, id int64) (*Course, error) {
    return s.repo.GetByID(ctx, id)
}
```

### 3. Handler — HTTP 处理

```go
// internal/modules/course/handler.go
func (h *Handler) GetCourseDetail(c *gin.Context) {
    id, err := parseIDParam(c, "id")
    if err != nil {
        response.BadRequest(c, "invalid id")
        return
    }
    result, err := h.service.GetCourseDetail(c.Request.Context(), id)
    if err != nil {
        response.InternalError(c, "failed to get course")
        return
    }
    response.Success(c, result)
}
```

### 4. 注册路由

```go
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    courses := r.Group("/courses")
    courses.GET("/:id", h.GetCourseDetail)
}
```

## 常用命令

```bash
cd server
make run       # 运行
make test      # 测试
make lint      # 代码检查
make fmt       # 格式化
make build     # 构建二进制
```

## 相关文档

- [分层架构](../architecture/layered-architecture.md) — 三层架构详解
- [API 概览](../reference/api-overview.md) — 接口规范
- [错误码](../reference/error-codes.md) — 统一错误码定义
