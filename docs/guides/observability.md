---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-07-31
---

# 可观测性运行手册

## 架构总览

StuHelper 采用 **方案 A** 的完整生产观测栈：

- **应用埋点**：Go 后端接入 OpenTelemetry（Gin / outbound HTTP / Redis / DB / OpenFGA）
- **Collector**：Grafana Alloy 负责接收 OTLP traces，并通过只读 Docker API 代理发现当前 Compose 项目、采集其容器日志后推送到 Loki
- **Metrics**：Prometheus 抓取应用、Grafana LGTM 组件、node-exporter、cAdvisor、postgres-exporter、redis-exporter、blackbox-exporter；生产发布脚本会启动 cAdvisor，因为基础设施 dashboard 依赖容器级 CPU / 内存指标
- **Logs**：Loki 聚合容器日志
- **Traces**：Tempo 存储链路数据
- **Dashboards**：Grafana 统一查看 metrics / logs / traces / alerts
- **Alerts**：Alertmanager 负责告警聚合与路由

PostgreSQL 指标采集使用独立 `stuhelper_metrics` 登录角色，密码由
`POSTGRES_EXPORTER_DB_PASSWORD` 提供。该角色只继承 PostgreSQL 预定义的
`pg_monitor`，连接数上限为 5，并且在仓库管理的 HBA 中只允许连接维护库
`postgres`；它不复用拥有全库读取能力的 `stuhelper_backup`。逻辑备份仍由
`stuhelper_backup` 连接 `stuhelper` 执行，两条权限链互不替代。

Redis 指标采集使用独立 `stuhelper_metrics` ACL 用户和
`REDIS_EXPORTER_PASSWORD`。该账号仅允许 redis_exporter 默认采集所需的
只读诊断命令，不具备应用 key 读写权限；应用账号也只允许运行代码实际
使用的 key、Lua 和通知 Pub/Sub 命令。两套密码不得复用。exporter 固定
连接本项目 Redis，并关闭多目标 `/scrape` 入口，避免把监控凭据转发到
调用方指定的目标。

## 目录与配置

源模板在 `infra/observability/`，Docker 容器挂载的是 `infra/generated/observability/` 下的生成版本。编辑模板后需运行 `render-observability.sh` 生成最终配置。

| 路径 | 作用 |
| --- | --- |
| `infra/observability/alloy/config.alloy` | OTLP 接收、按 Compose 项目隔离的 Docker 日志采集 |
| `infra/observability/prometheus/prometheus.yml.tmpl` | Prometheus 抓取配置模板 |
| `infra/observability/prometheus/rules/` | 告警规则 |
| `infra/observability/alertmanager/alertmanager.yml` | Alertmanager 路由 |
| `infra/observability/loki/loki.yaml` | Loki 配置 |
| `infra/observability/tempo/tempo.yaml` | Tempo 配置 |
| `infra/observability/grafana/provisioning/` | Grafana 数据源与 dashboard 自动导入 |
| `infra/ops/observability-smoke-check.sh` | 观测栈健康检查脚本 |

## 启动方式

### 仅观测栈

```bash
# 项目根目录下运行
make obs-up
```

### 连同生产应用一起启动

```bash
# 项目根目录下运行
make prod-deploy
```

## 默认入口

| 组件 | 默认地址 |
| --- | --- |
| Grafana | `http://127.0.0.1:3003` |
| Prometheus | `http://127.0.0.1:9090` |
| Alertmanager | `http://127.0.0.1:9093` |
| Loki | `http://127.0.0.1:3100` |
| Tempo | `http://127.0.0.1:3200` |
| Alloy UI / Metrics | `http://127.0.0.1:12345` |

## 核心环境变量

本地生产演练通常看这两份：

- `.env.prod.shared`
- `.env.prod.secrets.local`

远端部署机通常看这两份：

- `${DEPLOY_APP_DIR}/.env.prod.shared`
- `${DEPLOY_APP_DIR}/.env.prod.secrets`

另外还要确认远端部署控制面：

- `${DEPLOY_APP_DIR}/.deploy/remote.env`

至少要保证下面这些值已经配好：

- `METRICS_PASSWORD`
- `POSTGRES_EXPORTER_DB_PASSWORD`（独立数据库监控角色密码，不得与应用、备份或超级用户复用）
- `REDIS_EXPORTER_PASSWORD`（独立 Redis 监控账号密码，不得与 `REDIS_PASSWORD` 复用）
- `GRAFANA_ADMIN_PASSWORD`
- `ALERTMANAGER_WEBHOOK_URL`（推荐配置到值班系统 / ChatOps）
- `OTEL_ENABLED=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=http://alloy:4318`
- `OTEL_TRACE_SAMPLE_RATIO=0.2`（可按流量调整）
- `FRONTEND_METRICS_ALLOWED_ORIGINS=https://stuhelper.com`
- `ALLOY_DOCKER_LOG_MAX_AGE=1h`（首次启动或中断恢复时允许回补的日志窗口）
- `ALLOY_DOCKER_STREAM_TIMEOUT=24h`（Docker 日志长连接重连周期）

## Docker 日志安全边界

Alloy 不直接挂载 `/var/run/docker.sock`，也不挂载 `/var/lib/docker/containers`。`docker-socket-proxy` 是唯一持有宿主 Docker socket 的容器，并满足以下约束：

- 仅加入 `docker_api` 内部网络，不发布宿主端口；该网络只允许代理与 Alloy 加入。
- `POST=0`，只开放 Alloy 发现与读取日志需要的 `GET/HEAD` 容器、网络、事件、版本和 `_ping` 接口。
- Alloy 通过 `com.docker.compose.project=${STACK_NAME}` 标签只保留当前项目，不能把同机其他项目日志错误标成 StuHelper。
- Alloy 与 Loki 自身日志不回灌，避免错误日志形成递归放大。
- 首次启动或长时间中断后，超过 `ALLOY_DOCKER_LOG_MAX_AGE` 的积压日志不回放；实时日志和窗口内日志仍按持久化 position 续传。

代理拥有 Docker socket，因此仍属于高信任组件；不得把它接入 `frontend`、`backend`、`observability` 公共服务网络，也不得临时发布 `2375` 端口。镜像必须固定摘要并通过容器漏洞门禁。

## 本地与生产探针边界

`render-observability.sh dev|observability` 只渲染当前 Compose 内部目标，不访问公网 SSO。只有 `render-observability.sh prod` 才加入 `sso.stuhelper.com` OIDC discovery 可用性探针。需要做不触发外部请求的审计或本地验收时，必须使用 `dev` 或 `observability` 模式；不得直接复用生产生成文件。

严格 smoke 不只检查 exporter 的 HTTP `/metrics` 能否打开，还要求 Prometheus
中的 `up{job="postgres-exporter"}=1`、`pg_up=1`、
`up{job="redis-exporter"}=1`、`redis_up=1` 以及 node-exporter/cAdvisor
target 均为 1。这样可以区分“exporter 进程活着”与“exporter 确实能读取数据库”。

## 推荐告警接入

当前仓库保留了本地演练用的备用 receiver，正式上线时还是要把告警接到真实值班系统。推荐至少接其中一种：

1. Slack / 企业微信 / 飞书 webhook
2. PagerDuty / Opsgenie / OnCall 平台

建议把告警路由策略按 `severity=critical|warning` 分层，并单独为：

- API 5xx / 延迟
- DB / Redis
- OpenFGA / Casdoor / SMS / Oracle 学籍源等外部依赖
- `circuit_breaker_state{name=...}` 持续非零和 `external_requests_total{client="oracle_student_directory",status="error"}` 异常
- 证件审核 / SSE 队列积压

设置值班升级链路。

## 前端运行时错误

主站通过 `/api/v1/metrics/frontend-errors` 上报三类有限枚举：
`error`、`unhandledrejection` 和 `vue-error`。其中 `vue-error` 同时覆盖全局
`ErrorBoundary` 捕获的组件异常，以及边界外进入 Vue 全局 error handler 的异常。
ErrorBoundary 保持 fallback UI 所有权，阻止同一异常继续向上传播并重复计数。

后端当前只读取 `kind` 并增加 `frontend_errors_total{kind=...}`，不存储浏览器提交的
message、stack、组件 props 或用户输入。组件错误因此只上报 `{"kind":"vue-error"}`；
原始异常仅在开发环境控制台输出。未执行 `initObservability` 的页面（例如 E2E API stub，
以及没有 API base 的受限 join host）调用 reporter 时必须 no-op，错误上报自身也不得抛出
第二个应用错误。

## 发布后必做检查

```bash
# 项目根目录下运行
./infra/ops/smoke-check.sh
./infra/ops/observability-smoke-check.sh
```

Grafana 中至少确认：

- `StuHelper Overview`
- `StuHelper Application`
- `StuHelper Infrastructure`

三张 dashboard 在 5 分钟内都有数据。

## 故障排查顺序

1. 看 Prometheus `up` 指标，确认是采集失败还是业务失败
2. 看 Grafana Alerting / Alertmanager，确认告警范围与首次触发时间
3. 在 Grafana Explore 中用 `trace_id` / `request_id` 关联 Loki 与 Tempo
4. 优先判断：应用问题、数据库问题、Redis 问题、外部依赖问题、主机资源问题
5. 若观测栈自身异常，优先恢复 Alloy / Prometheus / Loki / Tempo 基础链路

## 运维铁律

- **不要把 Grafana / Prometheus 暴露到公网未鉴权地址**
- **生产至少保留 7 天 traces/logs，关键审计日志建议更久**
- **告警不接值班系统就不算真正上线**
- **每次发布后必须验证 dashboard 有新数据写入**
- **日志必须保留 `request_id` / `trace_id` / `span_id` 关联字段**
- **不得把宿主 Docker socket 或整个容器数据目录直接暴露给 Alloy**
