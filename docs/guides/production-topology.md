---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-05-22
---

# 生产拓扑与运维责任

## 部署架构

主站单机 Docker Compose 部署。StuHelper 应用、PostgreSQL、Redis、OpenFGA、对象存储与观测栈由仓库内 Compose 管理；公网入口由宝塔 Nginx 管理。生产账号登录 SSO 已独立部署为 `https://sso.stuhelper.com`，不随主站 Compose 生命周期启动或停止；对一方 / 三方应用暴露的统一身份 issuer 是 `https://id.stuhelper.com`，由主站 web/frontend + backend 共同承载。

> 生产前提：承载 `postgres_data` / `redis_data` / 对象存储数据目录的底层块设备必须开启静态加密（云盘 KMS/EBS/PD 或主机侧 LUKS）。仓库内的 Compose 只定义容器拓扑，不负责替代宿主机磁盘加密。

```
[Internet]
    │
    ▼
[Baota Nginx :443/:80]  ← 宿主机，负责 TLS 终止与路由
    ├── stuhelper.com /api/*        → 127.0.0.1:18080 → backend (Go, :8080)
    ├── stuhelper.com /admin/*      → 127.0.0.1:18001 → admin 前端 (Nginx, :8080)
    ├── stuhelper.com /             → 127.0.0.1:18000 → web 前端 (Nginx, :80)
    ├── id.stuhelper.com /.well-known/* /oauth2/* /oidc/* → backend
    ├── id.stuhelper.com /login /consent /complete-profile /assets/* → web 前端
    └── id.stuhelper.com / → 302 到 https://stuhelper.com/developers/apps，且重定向响应禁用缓存

[sso.stuhelper.com]
    └── Baota Nginx /.well-known/* /api/* / → 127.0.0.1:8087 → 独立 Casdoor SSO 栈
```

主站生产配置中 `CASDOOR_ISSUER` 与 `WEB_VITE_SSO_URL` 固定指向 `https://sso.stuhelper.com`；`IDENTITY_ISSUER` 固定指向 `https://id.stuhelper.com`。`CORS_ORIGINS` 必须同时包含 `https://stuhelper.com` 和 `https://id.stuhelper.com`，`TOKEN_COOKIE_DOMAIN` 必须设置为 `.stuhelper.com`，让 Casdoor 回调到主站 API 后签发的浏览器会话可继续用于 `id.stuhelper.com` 的授权页。仓库内 `casdoor` compose service 只用于本地开发或显式本地 SSO 验证，生产发布脚本不得启动该服务。

## 外部机器人链路

Koishi 与 NapCat 当前不纳入主站 Docker Compose 拓扑，而是作为外部独立节点部署：

```text
[QQ 平台]
   │
   ▼
[NapCat / OneBot]
   │
   ▼
[Koishi]
   │
   ├── 调用 StuHelper API：QQ 绑定码消费、QQ 认证状态查询
   ├── 持有本地 SQLite：群规则、消息账本、处罚记录、事件日志
   └── 持有独立服务令牌：不与主站浏览器令牌共享
```

运维责任边界：

- StuHelper 主站负责 API、身份系统、业务数据和 OpenAPI 契约。
- Koishi 负责机器人运行时、群管逻辑和本地群管域数据。
- NapCat 只负责 QQ 协议与 OneBot 适配。

当前工作区默认约定：

- Koishi 本地控制台端口固定为 `5140`
- 群管本地数据库路径为 `bots/koishi/data/koishi.db`
- Koishi 与主站之间只通过 `/api/v1/bot/*` 服务令牌接口通信，不共享 PostgreSQL 或 Redis

## 公网入口

| 决策 | 内容 |
|------|------|
| Edge Proxy | 宝塔 Nginx（宿主机管理） |
| TLS 终止 | 宝塔 Nginx 证书管理，或外部 CDN/LB 终止后转发到宝塔 Nginx |
| 公网端口 | 443 (HTTPS)、80 (HTTP → 301 → HTTPS)，只由宝塔 Nginx 监听 |
| 容器宿主机端口 | `127.0.0.1:18080` backend、`127.0.0.1:18000` web、`127.0.0.1:18001` admin |

如果生产机已经由宝塔 Compose 管理全局 PostgreSQL，可用 `docker-compose.external-datastore.yml` 把 StuHelper 生产容器接入 `EXTERNAL_DATASTORE_NETWORK=baota_net`，并设置 `EXTERNAL_POSTGRES_ENABLED=true`。该模式不会启动 `stuhelper-prod-postgres`，但需要先在外部 PostgreSQL 中为 StuHelper / OpenFGA 创建独立数据库和独立账号，并把旧 StuHelper 专用 Postgres 中的 `stuhelper` / `openfga` 数据迁移到外部 Postgres；若外部 PostgreSQL 未启用 TLS，还必须显式设置 `EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true`。Redis 仍由 StuHelper Compose 以独立 TLS/ACL 实例运行，不复用全局 Redis，也不加入外部 datastore 网络。本机 `make prod-parity-smoke` 会运行 `prod-parity-datastore-smoke.sh`，用容器级连接检查证明共享 PostgreSQL 的库/账号隔离和 Redis 独立实例约束。

### 证书终止策略

**默认方案：宝塔 Nginx 终止 TLS**

- `stuhelper.com` 与 `www.stuhelper.com` 在宝塔面板中建站并配置证书。
- 主站宝塔 Nginx 根据路径反代到本机回环端口，示例见 `infra/nginx/baota-stuhelper.conf`。
- 外部 SSO 机器也需要应用 `infra/nginx/baota-casdoor-sso.conf` 或等价规则；`sso.stuhelper.com/.well-known/*` 必须反代到 Casdoor upstream，不能落到宝塔静态站点根目录。当前生产 SSO 现场端口是 `127.0.0.1:8087`；如果外部 SSO 机器实际使用其他端口，合并模板时必须同步替换 upstream，并在审计时设置 `NGINX_PUBLIC_INGRESS_CASDOOR_UPSTREAM=http://127.0.0.1:<port>`。
- 保存或 reload 前，用 `infra/ops/nginx-public-ingress-preflight.sh` 审计实际配置；主站机器使用 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper`，SSO 机器使用 `NGINX_PUBLIC_INGRESS_PROFILE=sso`。
- 如果公网 smoke 报 `SSL_ERROR_SYSCALL`、`.well-known` 404 或 SPA HTML，运行 `infra/ops/public-identity-ingress-diagnostic.sh` 生成脱敏诊断 evidence；该脚本会分别检查本机 resolver、`dns.google` 公共 DNS-over-HTTPS、SNI TLS、`id.stuhelper.com` discovery/JWKS 和 `sso.stuhelper.com` Casdoor discovery/JWKS。
- Docker Compose 中的业务端口只绑定 `127.0.0.1`，不直接暴露公网。

**备选方案：外部 LB/CDN 终止**

- 外部 Cloudflare/Nginx/ALB 终止 TLS 后转发到宝塔 Nginx。
- 需要保留 `X-Forwarded-Proto: https`、`X-Forwarded-Host` 和客户端 IP 链路。

## 备份责任

| 组件 | 责任方 | 执行方式 |
|------|--------|----------|
| PostgreSQL 逻辑备份 | 仓库内脚本 + systemd timer | `docker compose run --rm postgres pg_dump` |
| PostgreSQL base backup | 仓库内脚本 + systemd timer | `docker compose run --rm postgres pg_basebackup` |
| 备份同步到对象存储 | 仓库内脚本 + systemd timer | `infra/ops/sync-postgres-backups.sh` |
| 恢复演练 | 运维手工执行 | `infra/ops/restore-postgres.sh` |

## 观测责任

| 组件 | 责任方 | 文档 |
|------|--------|------|
| Prometheus + Alertmanager | docker-compose observability profile | `docs/guides/observability.md` |
| Grafana | docker-compose observability profile | 3 个预置 dashboard |
| 日志采集 (Alloy → Loki) | docker-compose observability profile | |
| 追踪 (Alloy → Tempo) | docker-compose observability profile | |

## 冒烟检查

部署后执行 `infra/ops/smoke-check.sh`，覆盖：

1. 基础设施就绪（API health、Web、Admin）
2. 公开业务端点（院系、课程、认证）
3. 观测链路（Grafana、指标端点）
4. OIDC 连通性（Casdoor well-known）

`https://sso.stuhelper.com/.well-known/openid-configuration` 必须返回 JSON object，且 `issuer=https://sso.stuhelper.com`；discovery 中的 `jwks_uri` 也必须能返回含 `keys` 数组的 JSON object。若 discovery 或 JWKS 返回 Casdoor SPA HTML / 404，通常说明宝塔 Nginx 把 `/.well-known/*` 当作静态文件处理，需在 SSO 机器上合并 `infra/nginx/baota-casdoor-sso.conf`，并运行 `NGINX_PUBLIC_INGRESS_PROFILE=sso ./infra/ops/nginx-public-ingress-preflight.sh`。如果同时存在主站或 `id.stuhelper.com` TLS 握手失败，先运行 `./infra/ops/public-identity-ingress-diagnostic.sh`，用 evidence 中的 `dns_non_public_address` / `public_dns_nxdomain` / `tls_handshake_failed` / `casdoor_well_known_served_by_spa` / `casdoor_jwks_not_proxied` 分类缩小排查面。该诊断脚本默认检查公网 StuHelper 域名，避免开发机 `.env` 中的 localhost 值掩盖公网入口状态；只有设置 `PUBLIC_IDENTITY_INGRESS_DIAGNOSTIC_USE_ENV_TARGETS=true` 时才使用 ENV_FILE 里的目标 URL。
