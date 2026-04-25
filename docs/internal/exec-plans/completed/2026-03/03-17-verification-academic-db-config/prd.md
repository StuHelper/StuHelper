---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# 去除学籍库表硬编码

## Goal

让学籍匹配读取学校配置中的 `academic_db_table`，替代 `academic.buaa_students` 硬编码，并保证 SQL 安全和学校隔离。

## What I already know

- `school_configs` 已有 `academic_db_table` 字段
- 学籍查询仓储仍固定查询 `academic.buaa_students`
- LDAP 扩展学号路径依赖同一仓储，导致多校配置失效

## Assumptions (temporary)

- 表名必须使用严格白名单格式校验后拼接（参数占位符不能用于标识符）
- 默认回退表仍保留 `academic.buaa_students` 以兼容旧配置

## Open Questions

- 无阻塞问题，按配置化+安全拼接直接实现

## Requirements

- 新增学籍表名解析逻辑，优先学校配置，缺失时回退默认表
- 查询接口支持显式传入学籍表名
- 表名只允许 `schema.table` 且字符集受限，非法配置直接报错

## Acceptance Criteria

- [ ] 配置了 `academic_db_table` 的学校会命中对应表
- [ ] 未配置时回退默认表，行为不回归
- [ ] 非法表名不会进入 SQL 执行

## Definition of Done

- 新增独立测试文件覆盖表名解析和 SQL 安全校验
- 用户模块相关测试通过

## Technical Approach

在 `repository_academic.go` 新增受控标识符构造器；将学籍查询函数改为接收 `tableName` 参数；在 `service_profile.go` 读取学校配置并传入解析后的表名；在 `repository_config.go` 复用同一标准化函数，避免跨层歧义。

## Decision (ADR-lite)

Context: 学校可配置表名是既有模型，硬编码破坏多校扩展能力。  
Decision: 使用严格格式校验后动态 SQL 组装，禁止任意标识符注入。  
Consequences: 非法旧配置会显式失败，需在管理端修正配置值。

## Out of Scope

- 不在本任务引入跨库路由
- 不实现按学校自动建表

## Technical Notes

- 目标文件：`server/internal/modules/user/repository_academic.go`、`service_profile.go`、`repository_config.go`
- 测试文件：新增 `server/internal/modules/user/repository_academic_config_test.go`、`server/internal/modules/user/service_profile_academic_test.go`
