---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-05-22
---

# 生产拓扑与运维责任

## 部署架构

主站单机 Docker Compose 部署。StuHelper 应用、PostgreSQL、Redis、OpenFGA、对象存储与观测栈由仓库内 Compose 管理；公网入口由宝塔 Nginx 管理。对浏览器和一方 / 三方应用暴露的统一身份入口是 `https://id.stuhelper.com`，由主站 web/frontend + backend 共同承载；Casdoor 只作为后端上游登录源，不再作为默认公网站点入口。

> 生产前提：承载 `postgres_data` / `redis_data` / 对象存储数据目录的底层块设备必须开启静态加密（云盘 KMS/EBS/PD 或主机侧 LUKS）。仓库内的 Compose 只定义容器拓扑，不负责替代宿主机磁盘加密。

```
[Internet]
    │
    ▼
[Baota Nginx :443/:80]  ← 宿主机，负责 TLS 终止与路由
    ├── stuhelper.com /api/*        → 127.0.0.1:18080 → backend (Go, :8080)
    ├── stuhelper.com /admin/*      → 127.0.0.1:18001 → admin 前端 (Nginx, :8080)
    ├── stuhelper.com /             → 127.0.0.1:18000 → web 前端 (Nginx, :80)
    ├── stuhelper.com /identity /account/profile /account/security /connect /login /auth/callback /consent /complete-profile /developers/* /user/authorized-apps /user/*-verification /user/*-binding /user/academic-info
    │       → 302 https://id.stuhelper.com$request_uri
    ├── id.stuhelper.com /.well-known/* /oauth2/* /oidc/* → backend
    ├── id.stuhelper.com /api/v1/*  → backend
    ├── id.stuhelper.com /login/oauth/* /signup/oauth/* /api/* /static/* /img/* /buttons/* /flag-icons/* /web/* /mfa/* /account /signup /forget
    │       → 127.0.0.1:8087 → Casdoor upstream
    ├── id.stuhelper.com /identity /account/profile /account/security /connect /login /auth/callback /consent /complete-profile /developers/* /user/authorized-apps /user/*-verification /user/*-binding /user/academic-info /assets/*
    │       → web 前端
    ├── id.stuhelper.com / → 302 到 /identity，且重定向响应禁用缓存
    └── id.stuhelper.com 其他主站路径 → 302 https://stuhelper.com$request_uri
```

主站生产配置中 `IDENTITY_ISSUER`、`WEB_VITE_SSO_URL`、`WEB_VITE_IDENTITY_URL` 与 `CASDOOR_PUBLIC_AUTH_BASE_URL` 固定指向 `https://id.stuhelper.com`，`WEB_VITE_WEB_URL` 固定指向 `https://stuhelper.com`，用于从主站跳到 `id` 登录后再回到原主站页面；`CASDOOR_ISSUER` 保留为后端识别 Casdoor issuer 的上游配置，不代表浏览器应直接访问 `sso.stuhelper.com`。`CASDOOR_REDIRECT_URI`、`CASDOOR_ADMIN_REDIRECT_URI` 与 `CASDOOR_UNIAPP_REDIRECT_URI` 固定回到 `https://id.stuhelper.com/api/v1/auth/callback`。`CORS_ORIGINS` 必须同时包含 `https://stuhelper.com` 和 `https://id.stuhelper.com`，`TOKEN_COOKIE_DOMAIN` 必须设置为 `.stuhelper.com`，让回调后签发的浏览器会话可同时用于主站和 `id.stuhelper.com` 的身份页。仓库内 `casdoor` compose service 只用于本地开发或显式本地 SSO 验证，生产发布脚本不得启动该服务。

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

当前生产拓扑不使用 Traefik。Traefik 与 Nginx 技术上可以共存，但不能同时拥有公网 `80/443` 或分别承担同一批域名的入口职责；否则 TLS 终止、`X-Forwarded-*`、OIDC discovery、JWKS、授权页回调和路径路由会出现双层漂移。StuHelper 在宝塔单机环境中固定选择宝塔 Nginx 作为唯一公网入口，Compose 只把应用服务暴露到宿主机回环端口。

如果生产机已经由宝塔 Compose 管理全局 PostgreSQL，可用 `docker-compose.external-datastore.yml` 把 StuHelper 生产容器接入 `EXTERNAL_DATASTORE_NETWORK=baota_net`，并设置 `EXTERNAL_POSTGRES_ENABLED=true`。该模式不会启动 `stuhelper-prod-postgres`，但需要先在外部 PostgreSQL 中为 StuHelper / OpenFGA 创建独立数据库和独立账号，并把旧 StuHelper 专用 Postgres 中的 `stuhelper` / `openfga` 数据迁移到外部 Postgres；若外部 PostgreSQL 未启用 TLS，还必须显式设置 `EXTERNAL_POSTGRES_ALLOW_PLAINTEXT=true`。Redis 仍由 StuHelper Compose 以独立 TLS/ACL 实例运行，不复用全局 Redis，也不加入外部 datastore 网络。本机 `make prod-parity-smoke` 会运行 `prod-parity-datastore-smoke.sh`，用容器级连接检查证明共享 PostgreSQL 的库/账号隔离和 Redis 独立实例约束；随后 `prod-parity-smoke-data.sh` 会写入本机专用最小课程 / 教师 / 评课数据和入群认证会话、刷新评分统计和教师物化视图，使浏览器 smoke 能验证生产镜像在真实后端数据下的课程详情、课程评课详情、评课聚合、教师详情和入群认证链接渲染。

### 证书终止策略

**默认方案：宝塔 Nginx 终止 TLS**

- `stuhelper.com` 与 `www.stuhelper.com` 在宝塔面板中建站并配置证书。
- 主站宝塔 Nginx 根据路径反代到本机回环端口，示例见 `infra/nginx/baota-stuhelper.conf`。
- Casdoor upstream 由 `id.stuhelper.com` 的 `/login/oauth/*`、`/signup/oauth/*`、`/api/*` 和静态资源路径反代到本机 `127.0.0.1:8087`，不要要求用户浏览器直接访问 `sso.stuhelper.com`。
- 保存或 reload 前，用 `infra/ops/nginx-public-ingress-preflight.sh` 审计实际配置；主站机器使用 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper`。历史兼容的 `NGINX_PUBLIC_INGRESS_PROFILE=sso` 只用于显式保留独立 `sso.stuhelper.com` 公网入口的环境，不是默认发布门禁。
- 如果公网 smoke 报 `SSL_ERROR_SYSCALL`、`.well-known` 404 或 SPA HTML，运行 `infra/ops/public-identity-ingress-diagnostic.sh` 生成脱敏诊断 evidence；默认重点检查 `stuhelper.com` 与 `id.stuhelper.com`，只有显式传入 Casdoor upstream 目标或打开 upstream 检查时才把 `sso` 作为公网诊断对象。
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
4. OIDC 连通性（`id.stuhelper.com` discovery、JWKS、authorize/token/introspect/revoke/UserInfo 基础路由）

默认生产门禁不再要求 `sso.stuhelper.com` 是公网可达站点。`infra/ops/identity-public-smoke.sh` 默认验证 `stuhelper.com` health 与 `id.stuhelper.com` OIDC/OAuth/UserInfo/logout 路由；只有设置 `IDENTITY_PUBLIC_SMOKE_CASDOOR_UPSTREAM_ENABLED=true` 时，才额外验证 `CASDOOR_ISSUER` 的 discovery/JWKS。`remote-preflight.sh` 同理默认只检查 Web 与 Identity 的公网 DNS/TLS；只有设置 `PUBLIC_INGRESS_CASDOOR_UPSTREAM_PREFLIGHT_ENABLED=true` 时，才把 Casdoor upstream 纳入公网门禁。
