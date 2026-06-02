---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-05-22
---

# 生产拓扑与运维责任

## 部署架构

主站单机 Docker Compose 部署。StuHelper 应用、PostgreSQL、Redis、OpenFGA、对象存储与观测栈由仓库内 Compose 管理；公网入口由宝塔 Nginx 管理。公开登录认证入口和 OIDC issuer 是 `https://sso.stuhelper.com`（Casdoor）。StuHelper 账号中心、学生认证、QQ 绑定、授权应用和开发者应用当前由 `https://stuhelper.com` 主站承载；`https://join.stuhelper.com` 只承载入群验证业务闭环。公网入口清单只包含 `stuhelper.com`、`join.stuhelper.com` 和 `sso.stuhelper.com`。

> 生产前提：承载 `postgres_data` / `redis_data` / 对象存储数据目录的底层块设备必须开启静态加密（云盘 KMS/EBS/PD 或主机侧 LUKS）。仓库内的 Compose 只定义容器拓扑，不负责替代宿主机磁盘加密。

```
[Internet]
    │
    ▼
[Baota Nginx :443/:80]  ← 宿主机，负责 TLS 终止与路由
    ├── stuhelper.com /api/*        → 127.0.0.1:18080 → backend (Go, :8080)
    ├── stuhelper.com /admin/*      → 127.0.0.1:18001 → admin 前端 (Nginx, :8080)
    ├── stuhelper.com /             → 127.0.0.1:18000 → web 前端 (Nginx, :80)
    ├── stuhelper.com /verify 和 /verify/* → 404（不兼容旧入口）
    ├── join.stuhelper.com /verify/<token>?qq=<qq> → 127.0.0.1:18000 → web 前端
    ├── join.stuhelper.com /verify → 404
    ├── join.stuhelper.com /api/* 与 /health/* → 127.0.0.1:18080 → backend
    └── sso.stuhelper.com /.well-known/* /api/* /login/* → Casdoor
```

主站生产配置中 `IDENTITY_SERVER_ENABLED=false`，`IDENTITY_ISSUER=`，`WEB_VITE_IDENTITY_URL=`；不再发布 StuHelper 自研 identity issuer。`WEB_VITE_WEB_URL=https://stuhelper.com`，`WEB_VITE_SSO_URL=https://sso.stuhelper.com`，`CASDOOR_ISSUER=https://sso.stuhelper.com`，`CASDOOR_PUBLIC_AUTH_BASE_URL=https://sso.stuhelper.com`。`CASDOOR_REDIRECT_URI`、`CASDOOR_ADMIN_REDIRECT_URI` 与 `CASDOOR_UNIAPP_REDIRECT_URI` 固定回到 `https://stuhelper.com/api/v1/auth/callback`。`ADMISSION_PUBLIC_BASE_URL=https://join.stuhelper.com`。`CORS_ORIGINS` 必须包含 `https://stuhelper.com`、`https://join.stuhelper.com` 和 `https://sso.stuhelper.com`；`TOKEN_COOKIE_DOMAIN=.stuhelper.com`，让登录回调后签发的浏览器会话可用于主站和 admission 流程。

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
   ├── 调用 StuHelper API：QQ 绑定码消费、QQ 认证状态查询、admission session / pending action / freshman review
   ├── 持有本地 SQLite：群规则、消息账本、处罚记录、事件日志
   └── 持有独立服务令牌：不与主站浏览器令牌共享
```

运维责任边界：

- StuHelper 主站负责 API、身份系统、业务数据和 OpenAPI 契约。
- Koishi 负责机器人运行时、群管逻辑和本地群管域数据。
- NapCat 只负责 QQ 协议与 OneBot 适配。
- admission 后端记录的 `platform=qq` 是被验证账号的 subject platform；Koishi 生产 runtime 可以是 `onebot`，插件负责映射后调用后端，具体禁言 / 踢人 / 发消息仍由当前 OneBot bot 执行。

当前工作区默认约定：

- Koishi 本地控制台端口固定为 `5140`
- 群管本地数据库路径为 `bots/koishi/data/koishi.db`
- Koishi 与主站之间只通过 `/api/v1/bot/*` 服务令牌接口通信，不共享 PostgreSQL 或 Redis
- Koishi 容器必须通过 `.env`、Compose `env_file` 或等价机制注入 `STUHELPER_PLATFORM_BASE_URL`、`STUHELPER_PLATFORM_SERVICE_TOKEN`、`STUHELPER_FRESHMAN_MATERIAL_HOSTS` 和 Console 管理密码；真实 token 不写入仓库。

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
- `join.stuhelper.com` 与 `sso.stuhelper.com` 必须有公网 TLS。`join` 使用主站 Web/API 回环端口；`sso` 反代到 Casdoor。
- 主站宝塔 Nginx 根据域名和路径反代到本机回环端口，示例见 `infra/nginx/baota-stuhelper.conf` 和 `infra/nginx/baota-casdoor-sso.conf`。
- 保存或 reload 前，用 `infra/ops/nginx-public-ingress-preflight.sh` 审计实际配置；主站机器使用 `NGINX_PUBLIC_INGRESS_PROFILE=stuhelper`，SSO 机器使用 `NGINX_PUBLIC_INGRESS_PROFILE=sso`。
- 如果 SSO discovery 报 `.well-known` 404 或 SPA HTML，检查 `sso.stuhelper.com` 的宝塔 Nginx 是否把 `/.well-known/` 正确反代到 Casdoor upstream。
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
4. OIDC 连通性（`sso.stuhelper.com` discovery、JWKS、authorize/token/introspect/revoke/UserInfo 基础路由，按当前 Casdoor 配置验证）

当前生产门禁默认关闭 legacy `identity-public-smoke.sh`。入群验证公网门禁使用 `infra/ops/admission-public-smoke.sh`：`join.stuhelper.com/verify/<token>?qq=<qq>` 必须返回 Web SPA，`join.stuhelper.com/verify`、`stuhelper.com/verify` 和 `stuhelper.com/verify/<token>` 必须返回 404。SSO 入口由 `nginx-public-ingress-preflight.sh` 的 `sso` profile 和 Casdoor discovery/JWKS 检查覆盖。
