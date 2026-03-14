# StuHelper

北航校园信息平台 - 课程评价、资料共享、校园服务一站式解决方案

[![License](https://img.shields.io/badge/license-MIT-blue.svg)](LICENSE)
[![Go Version](https://img.shields.io/badge/go-1.23+-00ADD8.svg)](https://go.dev/)
[![Vue Version](https://img.shields.io/badge/vue-3.5+-4FC08D.svg)](https://vuejs.org/)

## 项目简介

StuHelper 是一个面向北航校园的信息聚合平台，提供课程评价、资料共享、空教室查询、校园卡管理等服务。采用前后端分离架构，支持 SSO 单点登录。

**核心功能**：
- 📚 课程评价与资料共享
- 🪑 空教室查询与预约
- 💳 校园卡余额查询与提醒
- 🚌 校车时刻表与实时查询
- 🔔 订阅提醒与消息推送

## 技术栈

### 后端
- **语言**: Go 1.23+
- **框架**: Gin
- **数据库**: PostgreSQL 16+
- **缓存**: Redis 7+
- **认证**: Casdoor SSO
- **部署**: Docker + Docker Compose

### 前端
- **框架**: Vue 3.5+
- **语言**: TypeScript 5+
- **构建**: Vite 6+
- **UI**: Element Plus
- **状态管理**: Pinia
- **国际化**: vue-i18n

## 快速开始

### 前置要求

- Go 1.24+
- Node.js 24+
- PostgreSQL 16+
- Redis 7+
- pnpm 10+

### 安装

```bash
# 克隆仓库
git clone https://git.stuhelper.com/stuhelper/StuHelper.git
cd StuHelper

# 安装后端依赖
cd server
go mod download

# 安装前端依赖
cd ../clients
pnpm install
```

### 配置

```bash
# 复制配置文件模板
cp server/configs/config.example.yaml server/configs/config.yaml

# 编辑配置文件，填入数据库、Redis、Casdoor 等配置
vim server/configs/config.yaml
```

### 运行

**开发环境**：

```bash
# 启动后端（支持热重载）
cd server
air

# 启动前端开发服务器
cd clients
pnpm dev:web
```

**生产环境**：

```bash
# 使用 Docker Compose
docker-compose up -d
```

访问 `http://localhost:3000` 查看前端应用。

## 项目结构

```
StuHelper/
├── server/                 # 后端服务
│   ├── cmd/               # 应用入口
│   ├── internal/          # 内部代码
│   │   ├── modules/       # 业务模块
│   │   │   ├── auth/      # 认证模块
│   │   │   └── course/    # 课程模块
│   │   └── pkg/           # 公共包
│   ├── configs/           # 配置文件
│   └── migrations/        # 数据库迁移
├── clients/               # 前端 Monorepo
│   ├── web/               # Web 主站
│   ├── shared/            # 跨端共享 API / 类型层
│   └── uniappx/           # uni-app x 实验性脚手架
├── docs/                  # 项目文档
│   ├── guides/            # 开发指南
│   ├── modules/           # 模块文档
│   └── reference/         # API 参考
├── .trellis/              # AI workflow, spec, workspace, tasks
│   ├── spec/              # 项目规范入口
│   ├── workspace/         # 开发日志与项目归档
│   └── tasks/             # Trellis 任务目录
```

## 开发指南

### 后端开发

```bash
cd server

# 运行测试
go test ./...

# 代码检查
go vet ./...

# 格式化代码
go fmt ./...

# 生成 Wire 依赖注入代码
wire ./internal/wire
```

### 前端开发

```bash
cd clients

# Web 类型检查
pnpm --filter @stuhelper/web type-check

# Web 代码检查
pnpm --filter @stuhelper/web lint

# Web 构建生产版本
pnpm --filter @stuhelper/web build
```

### 数据库迁移

```bash
cd server

# 创建新迁移
migrate create -ext sql -dir migrations -seq migration_name

# 执行迁移
migrate -path migrations -database "postgresql://user:pass@localhost:5432/stuhelper?sslmode=disable" up

# 回滚迁移
migrate -path migrations -database "postgresql://user:pass@localhost:5432/stuhelper?sslmode=disable" down 1
```

## 文档

- [快速开始指南](docs/guides/README.md)
- [后端开发指南](docs/guides/backend-quickstart.md)
- [认证模块文档](docs/modules/auth/README.md)
- [课程模块文档](docs/modules/course/README.md)
- [API 参考](docs/reference/api-overview.md)
- [错误码参考](docs/reference/error-codes.md)

## 贡献指南

欢迎贡献代码、文档或提出建议！

1. Fork 本仓库
2. 创建特性分支 (`git checkout -b feature/amazing-feature`)
3. 提交更改 (`git commit -m 'feat: add amazing feature'`)
4. 推送到分支 (`git push origin feature/amazing-feature`)
5. 创建 Pull Request

**提交规范**：遵循 [Conventional Commits](https://www.conventionalcommits.org/)

**代码规范**：
- Go: 遵循 [Effective Go](https://go.dev/doc/effective_go)
- TypeScript: 遵循 [Vue 3 风格指南](https://vuejs.org/style-guide/)

详见 [项目规范入口](.trellis/spec/guides/index.md) 与 [开发工作流](.trellis/workflow.md)

## 许可证

本项目采用 MIT 许可证 - 详见 [LICENSE](LICENSE) 文件

## 联系方式

- 项目地址: https://git.stuhelper.com/stuhelper/StuHelper
- 问题反馈: https://git.stuhelper.com/stuhelper/StuHelper/-/issues

---

**注意**: 本项目仅供北航校内使用，需要校内 SSO 认证。
