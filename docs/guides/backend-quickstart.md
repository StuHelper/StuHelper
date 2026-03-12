# 后端开发指南

这份文档帮你在现有后端上继续开发新接口和新模块。

> 环境搭建请先完成 [快速开始](../tutorials/quick-start.md)。

## 项目结构

```text
server/
├── cmd/stuhelper/       # 应用入口
├── api/                 # OpenAPI 3 规范（Spec-First 源文件）
│   ├── openapi.yaml     # 主入口
│   ├── paths/           # 路径定义（按领域拆分）
│   └── components/      # 公共组件（schemas / parameters / responses）
├── internal/
│   ├── api/gen/         # 自动生成的 Go 代码（禁止手改）
│   ├── modules/         # 业务模块
│   └── pkg/             # 公共包（中间件、日志、配置、响应等）
└── scripts/             # 初始化 SQL、种子数据
```

每个业务模块默认遵循三层结构：

```text
Handler → Service → Repository
```

详细约束见 [分层架构](../architecture/layered.md)。

## 新增一个接口时的推荐顺序

### 1. 先改 OpenAPI 3 规范

接口契约的权威来源是：

```text
server/api/openapi.yaml
```

通常你会同时修改：

- `server/api/paths/*.yaml`
- `server/api/components/schemas/*.yaml`

### 2. 生成代码

```bash
cd server
make generate
```

这一步会同时更新：

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

### 3. 补后端实现

推荐顺序：

1. Repository 处理数据库访问
2. Service 组织业务逻辑
3. Handler 解析参数并返回响应

### 4. 验证

```bash
cd server
make lint-spec
make test
make build
```

如果你同时动了前端接口使用，再跑：

```bash
cd ../clients
pnpm type-check
```

## 常用命令

```bash
cd server
make run
make test
make lint
make fmt
make build
make generate
make lint-spec
make check-drift
```

## 开发时最容易忽略的两件事

### 不要手改生成代码

这些目录都应视为生成产物：

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

### 不要跳过 OpenAPI

即使是临时接口，也应该先写规范。否则前端和后端很快会出现返回结构漂移。

## 相关文档

- [OpenAPI 3 开发指南](openapi-development-guide.md)
- [分层架构](../architecture/layered.md)
- [API 概览](../reference/api-overview.md)
- [错误码](../reference/error-codes.md)
