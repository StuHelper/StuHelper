# 快速开始

本文档帮助新开发者快速搭建后端开发环境。

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
# 编辑 .env 配置数据库、Redis、Casdoor 参数
```

### 3. 启动服务

```bash
# 启动依赖
docker compose up -d postgres redis

# 运行后端
cd .. && go run cmd/stuhelper/main.go
```

访问 http://localhost:8080/health 验证。

## 常用命令

```bash
go test ./...           # 运行测试
go fmt ./...            # 格式化代码
go vet ./...            # 检查代码
```

## 相关文档

- 开发规范：`.project_rule/project_rules.md`
- 模块文档：`docs/modules/`
