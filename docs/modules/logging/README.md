# 日志系统模块

## 当前状态

🟢 已实现并在运行中使用

## 当前范围

日志系统已经覆盖这些场景：

- 请求日志
- 审计日志
- 敏感字段脱敏
- 管理后台日志查询

当前后台入口：

- `/admin/logs`

## 存储策略

- 当前主实现仍基于应用日志与结构化日志输出
- 管理后台可以查询操作日志
- 更重型的 Loki / Grafana / ClickHouse 方案仍保留为后续扩展方向

## 文档索引

| 文档                                         | 说明                 |
| -------------------------------------------- | -------------------- |
| [01-log-levels.md](01-log-levels.md)         | 日志级别定义         |
| [02-log-fields.md](02-log-fields.md)         | 日志字段规范         |
| [03-sensitive-data.md](03-sensitive-data.md) | 敏感数据处理         |
| [04-configuration.md](04-configuration.md)   | 日志配置             |
| [05-implementation.md](05-implementation.md) | Logger 核心实现      |
| [06-middleware.md](06-middleware.md)         | 中间件实现           |
| [07-usage-examples.md](07-usage-examples.md) | 使用示例             |
| [08-log-collection.md](08-log-collection.md) | 日志收集（扩展方案） |
| [09-audit-log.md](09-audit-log.md)           | 审计日志设计         |
