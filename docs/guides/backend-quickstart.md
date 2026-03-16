# 后端开发指南

这份文档面向已经把服务跑起来的开发者，默认你要在现有后端上继续加功能或改契约。

> 环境准备先看 [快速开始](../tutorials/quick-start.md)。

## 当前目录结构

```text
server/
├── cmd/stuhelper/        # 应用入口
├── api/                  # OpenAPI 3 源文件
├── internal/
│   ├── api/gen/          # 生成代码，禁止手改
│   ├── modules/          # 业务模块
│   └── pkg/              # 公共能力
└── scripts/              # init.sql、seed.sql
```

业务模块统一遵循：

```text
Handler → Service → Repository
```

一个模块允许拆成多个 `handler_*.go`、`service_*.go`、`repository_*.go` 文件。当前项目已经按这个方式把用户系统、RBAC、评课读写、配置加载等大文件拆开。新代码也按这个规则走，单文件优先控制在 300 到 400 行附近。

## 改接口的标准顺序

### 1. 先改 OpenAPI

权威接口契约在 `server/api/openapi.yaml`。通常要一起改：

- `server/api/paths/*.yaml`
- `server/api/components/schemas/*.yaml`

### 2. 重新生成

```bash
cd server
make generate
```

这会更新：

- `server/internal/api/gen/`
- `clients/shared/src/types/api.gen.ts`

### 3. 再补实现

- Repository 只管 SQL 和数据映射
- Service 只管业务规则、事务和授权事实编排
- Handler 只管 HTTP 绑定、错误映射、响应包装

### 4. 跑完整验证

```bash
cd server
make fmt
make lint
make test
make build
```

如果动了 OpenAPI，还要补：

```bash
cd server
make lint-spec
make check-drift
```

## 开发时必须守住的几条线

- 不要手改 `server/internal/api/gen/` 和 `clients/shared/src/types/api.gen.ts`
- 不要在 handler 里写 SQL
- 不要直接 `c.JSON(...)`，统一用 `response.*`
- 不要把运行时配置写死在模块里，统一走 `internal/pkg/config`
- 不要把一个 handler、service 或 repository 累到几百行还不拆

## 相关文档

- [分层架构](../architecture/layered.md)
- [API 概览](../reference/api-overview.md)
- [数据库参考](../reference/database.md)
- [OpenAPI 开发指南](openapi-development-guide.md)
