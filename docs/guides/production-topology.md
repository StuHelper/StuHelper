---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# 生产拓扑与运维责任

## 部署架构

单机 Docker Compose 部署，所有服务由仓库内 `docker-compose.yml` 定义。

> 生产前提：承载 `postgres_data` / `redis_data` / 对象存储数据目录的底层块设备必须开启静态加密（云盘 KMS/EBS/PD 或主机侧 LUKS）。仓库内的 Compose 只定义容器拓扑，不负责替代宿主机磁盘加密。

```
[Internet]
    │
    ▼
[Traefik Edge Proxy :443/:80]  ← 仓库内，负责 TLS 终止与路由
    ├── /api/*          → backend (Go, :8080)
    ├── /admin/*        → admin 前端 (Nginx, :80)
    └── /               → web 前端 (Nginx, :80)

[sso.stuhelper.com / CASDOOR_EXTERNALPORT]
    └── Casdoor SSO     → casdoor (:8000)
```

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
| Edge Proxy | Traefik（仓库内 docker-compose 管理） |
| TLS 终止 | Traefik ACME（Let's Encrypt），或外部 CDN/LB 终止后内部 HTTP |
| 公网端口 | 443 (HTTPS)、80 (HTTP → 301 → HTTPS) |

### 证书终止策略

**默认方案：Traefik ACME (Let's Encrypt)**

- `infra/traefik/traefik.yml` 中配置 `certificatesResolvers.letsencrypt`
- 证书自动续期，存储在 Docker volume `acme_data`（实际名称 `${STACK_NAME:-stuhelper}-acme-data`）
- 要求：域名 DNS 解析到部署机公网 IP，80 端口可达

**备选方案：外部 LB/CDN 终止**

- Traefik 仅监听 HTTP，由外部 Cloudflare/Nginx/ALB 终止 TLS
- `TRAEFIK_ENTRYPOINT_WEBSECURE_ENABLED=false`

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
