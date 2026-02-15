# 生产环境部署指南

本文档说明 StuHelper 后端在生产环境中的关键配置，重点覆盖数据库连接安全和基础设施配置。

> 开发环境搭建请参考 [快速开始](../tutorials/quick-start.md)。

## PostgreSQL 连接安全（SSL/TLS）

### sslmode 级别说明

| sslmode | 加密 | 证书校验 | 适用场景 |
|---------|------|----------|----------|
| `disable` | 否 | 否 | 仅限本地开发 |
| `require` | 是 | 否 | 加密传输但不验证服务端身份，存在 MITM 风险 |
| `verify-ca` | 是 | 验证 CA | 确认服务端证书由可信 CA 签发 |
| `verify-full` | 是 | 验证 CA + 主机名 | **生产环境推荐**，完整的身份验证 |

### 生产环境配置

生产环境**必须**使用 `verify-ca` 或 `verify-full`（推荐后者）。应用启动时会自动校验：

```go
// config.go 生产环境校验逻辑
if c.App.Env == "production" {
    if c.Database.SSLMode == "disable" || c.Database.SSLMode == "" {
        // 启动失败，强制要求配置 SSL
    }
}
```

### 环境变量

```bash
# 必填：SSL 模式
DB_SSL_MODE=verify-full

# 证书文件路径（verify-ca / verify-full 时必填）
DB_SSL_ROOT_CERT=/etc/ssl/certs/pg-ca.crt    # CA 根证书
DB_SSL_CERT=/etc/ssl/certs/pg-client.crt      # 客户端证书（双向 TLS 时）
DB_SSL_KEY=/etc/ssl/private/pg-client.key      # 客户端私钥（双向 TLS 时）
```

如果使用 `DATABASE_URL` 连接字符串，SSL 参数需包含在 URL 中：

```bash
DATABASE_URL=postgres://user:pass@db-host:5432/stuhelper?sslmode=verify-full&sslrootcert=/etc/ssl/certs/pg-ca.crt
```

### Docker Compose 生产配置示例

```yaml
# docker-compose.prod.yml
services:
  app:
    environment:
      DB_SSL_MODE: verify-full
      DB_SSL_ROOT_CERT: /etc/ssl/certs/pg-ca.crt
    volumes:
      - ./certs/pg-ca.crt:/etc/ssl/certs/pg-ca.crt:ro
```

### 云数据库服务

主流云服务商的托管 PostgreSQL 默认启用 SSL：

- **AWS RDS**: 下载 [RDS CA 证书](https://docs.aws.amazon.com/AmazonRDS/latest/UserGuide/UsingWithRDS.SSL.html)，设置 `DB_SSL_ROOT_CERT`
- **阿里云 RDS**: 从控制台下载 CA 证书，配置 `verify-ca`（部分实例不支持 `verify-full`）
- **自建 PostgreSQL**: 使用 `pg_hba.conf` 中 `hostssl` 规则强制 SSL 连接

## Redis 安全配置

生产环境 Redis 配置要点：

```bash
REDIS_PASSWORD=<strong-password>       # 必填
REDIS_TLS_ENABLED=true                  # 启用 TLS（如果 Redis 支持）
REDIS_TLS_CERT=/etc/ssl/certs/redis.crt
REDIS_TLS_KEY=/etc/ssl/private/redis.key
```

当前 `maxmemory-policy` 设置为 `volatile-lru`，仅淘汰设置了 TTL 的 key，保护 token 黑名单等无 TTL 数据。

## 相关文档

- [快速开始](../tutorials/quick-start.md) — 开发环境搭建
- [后端开发指南](./backend-quickstart.md) — 项目结构和开发模式
- [错误码参考](../reference/error-codes.md) — 统一错误码定义
