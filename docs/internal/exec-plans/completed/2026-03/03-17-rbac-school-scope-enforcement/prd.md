---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# 修复学校范围 RBAC 闭环

## Goal

修复学校范围权限在缺失学校上下文时被跳过的问题，保证 scope_school_ids 权限默认拒绝，并让中间件在不改业务 handler 的前提下，稳定从 path、query、body 提取 schoolID。

## What I already know

- 现有 `RequirePermission` 只从 query 读取 `schoolID`，路径参数和 body 都不会进入 scope 检查。
- 现有 `CheckPermissionScope` 逻辑在 `scope_school_ids` 非空但 `schoolID == nil` 时会放行，导致 fail-open。
- admin 路由存在 `:schoolID` 路径参数场景，也存在 `?schoolID=` 查询参数场景。
- 本任务限制只能修改 RBAC 中间件与权限服务相关文件，不改用户模块路由和 handler。

## Assumptions (temporary)

- 学校范围权限应采用 fail-closed：权限配置了学校白名单但请求没有学校上下文，必须拒绝。
- body 仅做 JSON 提取，不在中间件里引入业务级 schema 解析。
- 提取优先级按 path > query > body，避免路径资源语义被覆盖。

## Open Questions

- 无阻塞问题，当前输入足够实现。

## Requirements (evolving)

- `scope_school_ids` 存在时，缺少或空学校上下文必须拒绝。
- 中间件支持从 path/query/body 读取 schoolID，兼容键名 `schoolID`、`schoolId`、`school_id`。
- 中间件读取 body 后不能破坏后续 handler 继续读取请求体。
- 变更保持在 RBAC 模块文件，避免跨模块耦合修改。

## Acceptance Criteria (evolving)

- [ ] `CheckPermissionScope` 在学校范围权限且无 schoolID 时返回拒绝。
- [ ] `RequirePermission` 可从 path 参数提取 schoolID 并传入 scope 检查。
- [ ] `RequirePermission` 可从 query 参数提取 schoolID 并传入 scope 检查。
- [ ] `RequirePermission` 可从 JSON body 提取 schoolID 并传入 scope 检查。
- [ ] body 提取不会吞掉请求体，后续 handler 仍可读取。
- [ ] 新增独立测试文件通过。

## Definition of Done (team quality bar)

- 新增/更新测试通过
- 相关包 go test 通过
- 不引入静默放行路径
- 任务文档与实现一致

## Out of Scope (explicit)

- 不改 user 模块路由签名和 handler 入参
- 不改数据库权限数据结构
- 不做跨模块 school 上下文总线重构

## Technical Approach

在 `middleware.go` 引入学校上下文解析辅助函数，统一从 path/query/body 解析；在 `service_permissions.go` 引入学校 scope 判定辅助函数，并在 `CheckPermissionScope` 中使用 fail-closed 逻辑。

## Decision (ADR-lite)

**Context**: 现有实现存在 scope fail-open 安全缺口，同时请求上下文来源不完整。  
**Decision**: 采用中间件侧多来源解析 + service 侧 fail-closed 双保险。  
**Consequences**: 安全性提升，兼容现有路由；代价是中间件多一次轻量 JSON 读取，但通过恢复 request body 保持行为兼容。

## Technical Notes

- 代码位置：
  - `server/internal/modules/rbac/middleware.go`
  - `server/internal/modules/rbac/service_permissions.go`
- 相关测试：
  - 新增 `server/internal/modules/rbac/middleware_scope_test.go`
  - 新增 `server/internal/modules/rbac/service_scope_test.go`
