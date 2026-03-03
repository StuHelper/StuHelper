# 后端开发指南

本文档介绍后端项目结构和开发模式，帮助你快速上手编写业务代码。

> 环境搭建请先完成 [快速开始](../tutorials/quick-start.md)。

## 项目结构

```
server/
├── cmd/stuhelper/       # 应用入口
├── api/                 # OpenAPI 3.0.3 规范（Spec-First 源文件）
│   ├── openapi.yaml     # 主入口
│   ├── paths/           # 路径定义（按领域拆分）
│   └── components/      # 公共组件（schemas/parameters/responses）
├── internal/
│   ├── api/gen/         # 自动生成的代码（禁止手动修改）
│   ├── modules/         # 业务模块（按领域划分）
│   │   ├── auth/        # 认证模块
│   │   └── course/      # 课程 + 评课模块
│   └── pkg/             # 公共包（中间件、工具函数等）
├── deployments/         # 环境变量（.env 已移至项目根目录）
└── scripts/             # 数据库初始化和种子数据
```

每个业务模块遵循三层架构：`Handler → Service → Repository`。

## 添加新 API 端点（Spec-First 流程）

项目采用 OpenAPI 3 Spec-First 模式。以"获取课程详情"为例：

### 1. 编写 OpenAPI 规范

在 `server/api/paths/` 中定义端点，在 `server/api/components/schemas/` 中定义数据模型：

```yaml
# api/paths/course.yaml
/api/v1/course/courses/{id}:
  get:
    operationId: getCourseDetail
    tags: [Course]
    parameters:
      - $ref: '../components/parameters/common.yaml#/PathID'
    responses:
      '200':
        description: 课程详情
        content:
          application/json:
            schema:
              allOf:
                - $ref: '../components/schemas/common.yaml#/SuccessResponse'
                - properties:
                    data:
                      $ref: '../components/schemas/course.yaml#/CourseDetail'
```

### 2. 生成代码

```bash
cd server
make generate   # lint → bundle → 生成 Go models/interface → 生成 TS 类型
```

生成的 `server.gen.go` 会包含 `GetCourseDetail` 的 handler 签名和请求/响应 models。

### 3. Repository — 数据访问

```go
// internal/modules/course/repository.go
func (r *Repository) GetByID(ctx context.Context, id int64) (*Course, error) {
    var course Course
    err := r.db.QueryRow(ctx, `SELECT id, name FROM courses WHERE id = $1`, id).
        Scan(&course.ID, &course.Name)
    if err != nil {
        return nil, fmt.Errorf("GetByID: %w", err)
    }
    return &course, nil
}
```

### 4. Service — 业务逻辑

```go
// internal/modules/course/service.go
func (s *Service) GetCourseDetail(ctx context.Context, id int64) (*Course, error) {
    return s.repo.GetByID(ctx, id)
}
```

### 5. Handler — HTTP 处理

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
        response.InternalError(c)
        return
    }
    response.Success(c, result)
}
```

### 6. 注册路由

```go
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    courses := r.Group("/courses")
    courses.GET("/:id", h.GetCourseDetail)
}
```

### 7. 验证

```bash
make lint-spec   # 确认规范无错误
make generate    # 重新生成，确认无漂移
make build       # 编译通过
```

开发环境访问 http://localhost:8080/docs/ 可在 Swagger UI 中查看和测试新端点。

## 常用命令

```bash
cd server
make run              # 运行
make test             # 测试
make lint             # 代码检查
make fmt              # 格式化
make build            # 构建二进制
make generate         # 重新生成 OpenAPI 相关代码
make lint-spec        # 验证 OpenAPI 规范
```

## 相关文档

- [分层架构](../architecture/layered-architecture.md) — 三层架构详解
- [API 概览](../reference/api-overview.md) — 接口规范
- [错误码](../reference/error-codes.md) — 统一错误码定义
- OpenAPI 规范: `server/api/openapi.yaml`
- Swagger UI: http://localhost:8080/docs/ （开发环境）
