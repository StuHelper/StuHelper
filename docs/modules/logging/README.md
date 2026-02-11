# 日志系统模块

## 模块概述

日志系统负责日志采集、存储和分析告警。

## 日志级别

| 级别 | 用途 |
|------|------|
| DEBUG | 调试信息，仅开发环境 |
| INFO | 重要业务流程 |
| WARN | 警告，不影响主流程 |
| ERROR | 错误，需要关注 |

## 技术方案

- 初期：PostgreSQL UNLOGGED TABLE
- 未来：迁移至 ClickHouse

## 文档索引

| 文档 | 说明 |
|------|------|
| [01-log-levels.md](01-log-levels.md) | 日志级别定义 |
| [02-log-fields.md](02-log-fields.md) | 日志字段规范 |
| [03-sensitive-data.md](03-sensitive-data.md) | 敏感数据处理 |
| [04-configuration.md](04-configuration.md) | 日志配置 |
| [05-implementation.md](05-implementation.md) | Logger 核心实现 |
| [06-middleware.md](06-middleware.md) | 中间件实现 |
| [07-usage-examples.md](07-usage-examples.md) | 使用示例 |
| [08-log-collection.md](08-log-collection.md) | 日志收集（Loki + Grafana） |
| [09-audit-log.md](09-audit-log.md) | 审计日志设计 |
