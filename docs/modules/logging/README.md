# 日志系统模块

当前项目没有单独的 logging 模块目录，日志能力分散在公共包和评课后台里。这份文档只记录已经落地的运行形态。

## 代码位置

| 位置 | 作用 |
| --- | --- |
| `server/internal/pkg/logger` | Zap 全局 logger、context 注入、敏感值脱敏 |
| `server/internal/pkg/middleware/logging.go` | 请求日志中间件 |
| `server/internal/modules/course/review/*log*` | 后台操作日志写入、查询、清理 |

## 当前能力

- 结构化日志输出，支持 console 和 JSON
- 基于请求上下文的字段透传
- 敏感字段脱敏
- 评课后台操作日志查询和清理

## 当前边界

- 应用运行日志走 Zap
- 后台可查询的审计数据是评课域 `admin_operation_logs`
- 更重型的日志采集和集中检索暂时不作为当前实现事实

## 相关入口

- 后台操作日志接口是 `/api/v1/course/review/admin/logs`
- 日志配置由 `internal/pkg/config` 装配
