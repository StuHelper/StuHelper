# Backend Code Review Report

## Summary
- Total files reviewed: 9
- Issues found: 1
- Severity breakdown: 0 Critical / 0 High / 1 Medium / 0 Low

## Detailed Findings

### server/internal/modules/rbac/handler_test.go
#### Issue 1: Test helper `allAdminEffectivePermissions` 创建合成权限 ID
- **Severity**: Medium
- **Location**: Lines 154-163
- **Problem**: 测试 helper 生成合成的 `PermissionID` 值（1, 2, 3...），不匹配真实数据库 ID。虽然这对绕过 middleware 的 handler 测试有效，但如果 `CheckPermissionScope` 需要验证权限 ID 一致性，可能会掩盖 bug。
- **Recommendation**: 添加注释说明这些是测试隔离用的合成 ID，或考虑使用更明显的 ID 范围（如从 1000 开始）。
- **Code**:
```go
// Current
func allAdminEffectivePermissions() []EffectivePermission {
	caps := capability.AdminEntryCapabilities
	perms := make([]EffectivePermission, len(caps))
	for i, name := range caps {
		perms[i] = EffectivePermission{PermissionID: int64(i + 1), Name: name, Granted: true}
	}
	return perms
}

// Recommended (add comment)
// allAdminEffectivePermissions 返回所有 admin 能力对应的 EffectivePermission
// 列表，用于 handler 测试中绕过权限中间件。
// 注意：PermissionID 为测试用合成值，不对应真实数据库 ID。
func allAdminEffectivePermissions() []EffectivePermission {
	caps := capability.AdminEntryCapabilities
	perms := make([]EffectivePermission, len(caps))
	for i, name := range caps {
		perms[i] = EffectivePermission{PermissionID: int64(i + 1), Name: name, Granted: true}
	}
	return perms
}
```

## Positive Observations

### 架构与分层
- **优秀的 middleware 缓存模式**：`RequireAnyPermission` / `RequirePermission` 中间件链使用 Gin context 缓存（`ctxKeyInternalUserID`, `ctxKeyEffectivePerms`）避免冗余 DB 查询。这是教科书级的纵深防御 + 性能优化。
- **清晰的关注点分离**：`CheckPermission`（加载权限 + 验证）vs `CheckPermissionScope`（仅验证）为 middleware（缓存）vs 非 middleware（非缓存）上下文提供了正确的抽象。
- **合理的接口设计**：`PermissionService` 接口最小化且聚焦，易于在测试中 mock。

### 授权架构
- **正确的 admin 门控**：`main.go` 现在对 admin 组使用 `RequireAnyPermission(rbacService, capability.AdminEntryCapabilities...)`，然后在路由级使用 `RequirePermission` 进行细粒度检查。这与授权架构 spec 一致。
- **无 `isAdmin` 泄漏**：代码正确避免使用 Casdoor 的 `isAdmin` 标志作为业务授权来源。

### 错误处理
- **审核流程中显式字段清理**：`service_admin.go` 47-53 行在批准时显式设置 `rejectionReason = nil`，拒绝时设置 `verifyMethod = nil` / `verifiedAt = nil`。防止状态转换时遗留脏数据。
- **正确的错误包装**：所有 service 方法用上下文包装错误（如 `fmt.Errorf("ReviewIdentity get: %w", err)`）。

### 数据库
- **精确的 SQL 更新**：`UpdateIdentityReviewStatus` 使用目标 UPDATE，仅触及状态字段，避免敏感 PII 字段的读-改-写循环。
- **移除 `StatusUnverified` 过滤器**：repository 正确从 `ListIdentityReviewItems` 移除了 `StatusUnverified` case（78 行删除），handler 对其进行验证（测试 302-309 行）。防止"未提交"和"已提交但待审"的混淆。

### 测试
- **全面的 middleware 测试**：`middleware_test.go` 覆盖：
  - 缓存复用（99-139 行）
  - 权限拒绝（141-166 行）
  - 错误处理（73-93 行）
  - Scope 验证（398-436 行）
- **Handler 测试隔离**：RBAC handler 测试注入缓存键绕过 middleware，聚焦 handler 逻辑。User handler 测试使用 `allowAllPermissionService` 简化。两种方法都有效。
- **回归覆盖**：新增 `status=all` 清除过滤器测试（`user/handler_test.go` 265-300 行）和 `status=unverified` 拒绝测试（302-309 行）。

### 代码质量
- **无硬编码配置**：所有能力名称都是 `capability/capability.go` 中的常量。
- **结构化日志**：所有错误路径使用 `logger.FromGin(c).Error(...)` 和结构化字段。
- **响应 helper**：所有 handler 使用 `response.*` helper 而非临时 `c.JSON(...)`。

### 数据库 Schema
- **权限清理**：`init.sql` 移除了冗余的 admin 权限（`admin:users:manage`, `admin:roles:manage` 等），这些权限与细粒度 RBAC 权限重复。剩余权限聚焦且无重叠。

## Overall Assessment

这是一次高质量的授权重构，严格遵循项目架构指南。middleware 缓存模式执行得特别好，组级纵深防御（`RequireAnyPermission`）和路由级细粒度检查（`RequirePermission`）的分离是教科书级正确。

唯一问题是测试 helper 的文档缺口。代码已准备好生产。

## Verification Results
- Build: ✅ Passed
- Tests: ✅ Passed
