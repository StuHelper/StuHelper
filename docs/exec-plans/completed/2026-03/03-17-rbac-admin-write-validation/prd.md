# 修复 RBAC 管理写接口错误语义

## Goal

将 RBAC 管理写接口的可预期失败统一为稳定 4xx 业务错误，避免外键错误透传为 500。

## What I already know

- `SetUserRoles`、`SetGroupMembers`、`SetGroupPermissions` 基本直接写库
- 非法 ID 或外键冲突通常在 repository 层报数据库错误，被 handler 当作 500
- 角色权限写接口已有一套输入校验模式可复用

## Assumptions (temporary)

- 可预期的“资源不存在”应返回 404
- 非法选择集合应返回 400

## Open Questions

- 无阻塞问题，按现有错误码体系实现

## Requirements

- 写接口在 service 层先做存在性校验
- handler 层对业务错误做 4xx 映射
- repository 层保留事务与持久化职责

## Acceptance Criteria

- [ ] 无效 `roleID`/`userID`/`groupID`/`permissionID` 返回 4xx
- [ ] 数据库 FK 错误不再直接变成 500
- [ ] 现有成功路径不回归

## Definition of Done

- 新增独立测试文件覆盖错误映射
- RBAC 模块测试通过

## Technical Approach

在 service 层引入前置校验与领域错误；handler 识别这些错误并映射到 `BadRequest/NotFound`；repository 仅保留执行。

## Decision (ADR-lite)

Context: 管理端依赖稳定错误语义做交互提示，500 会导致不可恢复重试。  
Decision: 服务层做防御校验，传输层统一映射业务错误。  
Consequences: 调用方能得到可处理错误，日志噪音下降。

## Out of Scope

- 不改 RBAC 读接口
- 不重构权限模型

## Technical Notes

- 目标文件：`server/internal/modules/rbac/service_users.go`、`service_groups.go`、`handler_users.go`、`handler_groups.go`、`handler_helpers.go`
- 测试文件：新增 `server/internal/modules/rbac/service_write_validation_test.go`、`server/internal/modules/rbac/handler_write_validation_test.go`
