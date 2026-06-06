---
type: guide
audience: ops
status: current
authoritative-source: this file
last-verified: 2026-05-09
---

# 可观测性运行手册

## 架构总览

StuHelper 采用 **方案 A** 的完整生产观测栈：

- **应用埋点**：Go 后端接入 OpenTelemetry（Gin / outbound HTTP / Redis / DB / OpenFGA）
- **Collector**：Grafana Alloy 负责接收 OTLP traces，并采集 Docker 容器日志推送到 Loki
- **Metrics**：Prometheus 抓取应用、Grafana LGTM 组件、node-exporter、cAdvisor、postgres-exporter、redis-exporter、blackbox-exporter；生产发布脚本会启动 cAdvisor，因为基础设施 dashboard 依赖容器级 CPU / 内存指标
- **Logs**：Loki 聚合容器日志
- **Traces**：Tempo 存储链路数据
- **Dashboards**：Grafana 统一查看 metrics / logs / traces / alerts
- **Alerts**：Alertmanager 负责告警聚合与路由

## 目录与配置

源模板在 `infra/observability/`，Docker 容器挂载的是 `infra/generated/observability/` 下的生成版本。编辑模板后需运行 `render-observability.sh` 生成最终配置。

| 路径 | 作用 |
| --- | --- |
| `infra/observability/alloy/config.alloy` | OTLP 接收与日志采集（模板） |
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
- `GRAFANA_ADMIN_PASSWORD`
- `ALERTMANAGER_WEBHOOK_URL`（推荐配置到值班系统 / ChatOps）
- `OTEL_ENABLED=true`
- `OTEL_EXPORTER_OTLP_ENDPOINT=http://alloy:4318`
- `OTEL_TRACE_SAMPLE_RATIO=0.2`（可按流量调整）
- `FRONTEND_METRICS_ALLOWED_ORIGINS=https://stuhelper.com`

## 推荐告警接入

当前仓库保留了本地演练用的备用 receiver，正式上线时还是要把告警接到真实值班系统。推荐至少接其中一种：

1. Slack / 企业微信 / 飞书 webhook
2. PagerDuty / Opsgenie / OnCall 平台

建议把告警路由策略按 `severity=critical|warning` 分层，并单独为：

- API 5xx / 延迟
- DB / Redis
- OpenFGA / Casdoor / SMS 外部依赖
- 证件审核 / SSE 队列积压

设置值班升级链路。

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
