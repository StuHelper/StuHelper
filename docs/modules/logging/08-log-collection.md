# 日志收集（Loki + Grafana）

## Docker Compose 配置

```yaml
# deployments/docker-compose.logging.yml
version: '3.8'

services:
  loki:
    image: grafana/loki:2.9.0
    container_name: stuhelper-loki
    ports:
      - "3100:3100"
    volumes:
      - ./loki/config.yml:/etc/loki/config.yml
      - loki-data:/loki
    command: -config.file=/etc/loki/config.yml
    restart: unless-stopped

  promtail:
    image: grafana/promtail:2.9.0
    container_name: stuhelper-promtail
    volumes:
      - ./promtail/config.yml:/etc/promtail/config.yml
      - ../logs:/app/logs:ro
    command: -config.file=/etc/promtail/config.yml
    depends_on:
      - loki

  grafana:
    image: grafana/grafana:10.2.0
    container_name: stuhelper-grafana
    ports:
      - "3000:3000"
    volumes:
      - grafana-data:/var/lib/grafana
    environment:
      - GF_SECURITY_ADMIN_PASSWORD=admin
    depends_on:
      - loki

volumes:
  loki-data:
  grafana-data:
```

## Loki 配置

```yaml
# deployments/loki/config.yml
auth_enabled: false

server:
  http_listen_port: 3100

common:
  path_prefix: /loki
  storage:
    filesystem:
      chunks_directory: /loki/chunks
      rules_directory: /loki/rules
  replication_factor: 1
  ring:
    kvstore:
      store: inmemory

schema_config:
  configs:
    - from: 2020-10-24
      store: boltdb-shipper
      object_store: filesystem
      schema: v11
      index:
        prefix: index_
        period: 24h

limits_config:
  retention_period: 30d
```

## Promtail 配置

```yaml
# deployments/promtail/config.yml
server:
  http_listen_port: 9080
  grpc_listen_port: 0

positions:
  filename: /tmp/positions.yaml

clients:
  - url: http://loki:3100/loki/api/v1/push

scrape_configs:
  - job_name: stuhelper
    static_configs:
      - targets:
          - localhost
        labels:
          job: stuhelper
          __path__: /app/logs/*.log
    pipeline_stages:
      - json:
          expressions:
            level: level
            request_id: request_id
            module: module
      - labels:
          level:
          module:
```

## 启动命令

```bash
cd deployments
docker compose -f docker-compose.logging.yml up -d
```

访问 Grafana: http://localhost:3000 (admin/admin)
