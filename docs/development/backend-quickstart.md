# 后端快速入门指南

本文档帮助新开发者快速搭建后端开发环境并开始开发。

## 环境要求

| 工具 | 版本 | 说明 |
|------|------|------|
| Go | 1.21+ | 后端开发语言 |
| PostgreSQL | 15+ | 主数据库 |
| Redis | 7+ | 缓存和会话管理 |
| Docker | 24+ | 容器化部署（可选） |

## 快速开始

### 1. 克隆项目

```bash
git clone https://gitea.stuhelper.com/StuHelper/StuHelper.git
cd StuHelper/server
```

### 2. 配置环境变量

```bash
cd deployments
cp .env.example .env
```

编辑 `.env` 文件，配置以下必要参数：

```bash
# 数据库配置
DATABASE_HOST=localhost
DATABASE_PORT=5432
DATABASE_USER=postgres
DATABASE_PASSWORD=your_password
DATABASE_NAME=stuhelper

# Redis 配置
REDIS_HOST=localhost
REDIS_PORT=6379

# Casdoor SSO 配置（联系管理员获取）
CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=your_client_id
CASDOOR_CLIENT_SECRET=your_client_secret
```

### 3. 启动依赖服务

**方式一：使用 Docker Compose（推荐）**

```bash
docker compose up -d postgres redis
```

**方式二：本地安装**

确保 PostgreSQL 和 Redis 已安装并运行。

### 4. 运行后端服务

```bash
cd ..  # 回到 server 目录
go run cmd/stuhelper/main.go
```

服务启动后访问 http://localhost:8080/health 验证。

## 项目结构

```
server/
├── cmd/stuhelper/       # 应用入口
├── internal/
│   ├── modules/         # 业务模块
│   │   ├── auth/        # 认证模块
│   │   └── course/      # 课程模块
│   └── pkg/             # 公共包
└── deployments/         # 部署配置
```

## 开发新功能

### 添加新 API 端点

1. **在对应模块创建 Handler 方法**

```go
// internal/modules/course/handler.go
func (h *Handler) GetCourseDetail(c *gin.Context) {
    // 1. 解析参数
    id, err := parseIDParam(c, "id")
    if err != nil {
        response.BadRequest(c, "invalid id")
        return
    }

    // 2. 调用 Service
    result, err := h.service.GetCourseDetail(c.Request.Context(), id)
    if err != nil {
        response.InternalError(c, "failed to get course")
        return
    }

    // 3. 返回响应
    response.Success(c, result)
}
```

2. **在 Service 层实现业务逻辑**

```go
// internal/modules/course/service.go
func (s *Service) GetCourseDetail(ctx context.Context, id int64) (*Course, error) {
    return s.repo.GetByID(ctx, id)
}
```

3. **在 Repository 层实现数据访问**

```go
// internal/modules/course/repository.go
func (r *Repository) GetByID(ctx context.Context, id int64) (*Course, error) {
    var course Course
    err := r.db.QueryRow(ctx, `SELECT id, name FROM courses WHERE id = $1`, id).
        Scan(&course.ID, &course.Name)
    return &course, err
}
```

4. **注册路由**

```go
// internal/modules/course/handler.go
func (h *Handler) RegisterRoutes(r *gin.RouterGroup) {
    courses := r.Group("/courses")
    courses.GET("/:id", h.GetCourseDetail)
}
```

## 常用命令

```bash
# 运行测试
go test ./...

# 格式化代码
go fmt ./...

# 检查代码
go vet ./...

# 构建二进制
go build -o bin/stuhelper cmd/stuhelper/main.go
```

## 相关文档

- [开发规范](guide.md) - 代码风格、Git 工作流
- [分层架构](../architecture/layered-architecture.md) - 三层架构详解
- [API 设计](../api/overview.md) - API 接口规范
- [错误码](../api/error-codes.md) - 统一错误码定义
