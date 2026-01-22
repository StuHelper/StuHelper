# 日志与可观测性模块概述

## 模块简介

StuHelper 企业级日志和分析系统，提供结构化日志记录、请求追踪、敏感信息脱敏和集中式日志收集能力。

## 设计目标

| 目标 | 说明 |
|------|------|
| **结构化日志** | 所有日志采用 JSON 格式，便于机器解析和查询 |
| **统一规范** | 定义标准字段、日志级别使用规范 |
| **高性能** | 使用 zap 零分配设计，对应用性能影响可忽略 |
| **可追踪** | 每个请求携带唯一 Request ID，支持全链路追踪 |
| **安全合规** | 敏感信息自动脱敏，审计日志持久化存储 |

## 技术选型

| 组件 | 选型 | 版本 | 用途 |
|------|------|------|------|
| 日志库 | go.uber.org/zap | v1.27+ | 结构化日志记录 |
| 日志轮转 | lumberjack.v2 | v2.2+ | 日志文件轮转 |
| ID 生成 | google/uuid | v1.6+ | Request ID 生成 |
| 日志收集 | Loki + Promtail | v2.9+ | 集中式日志存储 |
| 可视化 | Grafana | v10.2+ | 日志查询和仪表板 |

## 架构图

```
┌─────────────────────────────────────────────────────────────┐
│                    StuHelper 日志系统架构                      │
├─────────────────────────────────────────────────────────────┤
│                                                             │
│  Vue.js Frontend    Gin API Server    Background Workers    │
│         │                  │                  │             │
│         ▼                  ▼                  ▼             │
│  ┌─────────────────────────────────────────────────────┐   │
│  │              Log Middleware Layer                    │   │
│  │  Request Logger | Error Handler | Audit | OTel      │   │
│  └─────────────────────────────────────────────────────┘   │
│                          │                                  │
│                          ▼                                  │
│  ┌─────────────────────────────────────────────────────┐   │
│  │                 Zap Logger Core                      │   │
│  │  Level Control | Sampling | JSON Encoding | Writers │   │
│  └─────────────────────────────────────────────────────┘   │
│                          │                                  │
│         ┌────────────────┼────────────────┐                │
│         ▼                ▼                ▼                │
│     Stdout           Log Files        Promtail             │
│    (Docker)          (Rotate)         (Agent)              │
│                                           │                 │
│                                           ▼                 │
│                                         Loki                │
│                                           │                 │
│                                           ▼                 │
│                                        Grafana              │
└─────────────────────────────────────────────────────────────┘
```

## 文档索引

| 文档 | 说明 |
|------|------|
| [01_log_levels.md](01_log_levels.md) | 日志级别规范 |
| [02_log_fields.md](02_log_fields.md) | 日志字段规范 |
| [03_sensitive_data.md](03_sensitive_data.md) | 敏感信息脱敏 |
| [04_configuration.md](04_configuration.md) | 日志配置 |
| [05_implementation.md](05_implementation.md) | 核心实现代码 |
| [06_middleware.md](06_middleware.md) | 中间件实现 |
| [07_usage_examples.md](07_usage_examples.md) | 使用示例 |
| [08_log_collection.md](08_log_collection.md) | 日志收集（Loki + Grafana） |
| [09_audit_log.md](09_audit_log.md) | 审计日志设计 |

## 依赖项

```go
// go.mod 新增
go.uber.org/zap v1.27.0
gopkg.in/natefinch/lumberjack.v2 v2.2.1
github.com/google/uuid v1.6.0
```
