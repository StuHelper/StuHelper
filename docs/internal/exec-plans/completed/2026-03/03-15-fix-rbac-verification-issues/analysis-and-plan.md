---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# 问题原因分析与修复方案

## 问题概览

| 问题编号 | 严重程度 | 问题描述 | 影响范围 |
|---------|---------|---------|---------|
| 问题1 | **高** | RBAC 权限中间件未实际应用到 admin 路由 | 后端安全 |
| 问题2 | **中** | 实名认证列表状态查询逻辑不完整 | 后端业务逻辑 |
| 问题3 | **低** | 前端实名认证成功文案判断错误 | 前端显示 |
| 问题4 | **低** | Admin 路由引用了 logs 页面 | 前端路由 |

---

## 问题1：RBAC 权限中间件未实际应用（高优先级）

### 原因分析

**现状**：
- 项目已实现完整的 RBAC 权限判断逻辑：
  - `rbac/middleware.go:13` 有 `RequirePermission(...)` 中间件
  - `rbac/service.go:225` 会真正查询用户的角色、组、个人覆盖、school scope、role scope
  - 数据库能配置角色、权限、用户组、override
  - 后台页面能管理这些数据

**问题**：
- `main.go:290` 挂载 admin 路由时，只使用了 `authMW + RequireAdmin`
- 没有把 `RequirePermission(...)` 挂到任何具体接口上
- 运行时访问 `/api/v1/admin/...` 时根本不看权限配置

**根本原因**：
这套 RBAC 现在是"权限配置管理系统"，不是"真实生效的权限系统"。功能落空。

### 修复方案

#### 方案选择

根据 `.trellis/spec/guides/rbac-role-permission-flow.md` 的规范，需要在每个 admin 路由上应用细粒度权限检查。

#### 需要修改的文件

1. **`server/internal/modules/rbac/handler.go`**
   - 修改 `RegisterAdminRoutes` 方法签名，接收 `PermissionService` 参数
   - 为每个路由添加 `RequirePermission` 中间件
   - 权限命名规范：`rbac:role:read`, `rbac:role:write`, `rbac:group:read`, `rbac:group:write`, `rbac:permission:read`

2. **`server/internal/modules/user/handler.go`**
   - 修改 `RegisterAdminRoutes` 方法签名，接收 `rbac.PermissionService` 参数
   - 为每个路由添加 `RequirePermission` 中间件
   - 权限命名规范：`user:identity:read`, `user:identity:write`, `user:profile:read`, `user:profile:write`

3. **`server/cmd/stuhelper/main.go`**
   - 修改 admin 路由注册逻辑，传递 `rbacService` 给各模块的 `RegisterAdminRoutes`

#### 权限定义清单

需要在数据库中定义以下权限（通过 seed 脚本或 migration）：

**RBAC 模块**：
- `rbac:role:read` - 查看角色列表和详情
- `rbac:role:write` - 创建、更新、删除角色
- `rbac:group:read` - 查看用户组列表和详情
- `rbac:group:write` - 创建、更新、删除用户组
- `rbac:permission:read` - 查看权限列表

**User 模块**：
- `user:identity:read` - 查看实名认证列表
- `user:identity:write` - 审核实名认证
- `user:profile:read` - 查看学生认证列表
- `user:profile:write` - 审核学生认证

#### 实施步骤

1. 先修改 `rbac/handler.go` 和 `user/handler.go` 的方法签名
2. 为每个路由添加对应的权限中间件
3. 修改 `main.go` 传递 service 参数
4. 创建权限定义的 seed 脚本或 migration
5. 运行测试验证权限检查生效

---

## 问题2：实名认证列表状态查询逻辑（中优先级）

### 原因分析

**文档定义**：
- `admin-user-system.yaml` 定义状态为 `pending / verified / rejected / all`

**数据层实现**：
- `user/repository.go:175` 的 `ListIdentitiesByStatus` 方法
- 数据库只有 `verified` 布尔值，没有显式 `status` 字段
- 区分 pending 和 rejected 依赖 `rejection_reason` 是否为 NULL

**当前查询逻辑**（`repository.go:94-103`）：
```go
switch status {
case StatusPending:
    qb.WriteString(` AND verified = false AND rejection_reason IS NULL`)
case StatusRejected:
    qb.WriteString(` AND verified = false AND rejection_reason IS NOT NULL`)
case StatusVerified:
    qb.WriteString(` AND verified = true`)
case StatusUnverified:
    qb.WriteString(` AND verified = false`)
}
```

**问题判断**：
经过代码审查，**当前逻辑已经正确实现**：
- `pending` = `verified=false AND rejection_reason IS NULL`
- `rejected` = `verified=false AND rejection_reason IS NOT NULL`
- `verified` = `verified=true`
- `unverified` = `verified=false`（包含 pending + rejected）

### 修复方案

**结论：此问题实际上已经正确实现，无需修改。**

但需要验证：
1. API 文档中的状态定义是否与代码一致
2. 前端调用时传递的状态参数是否正确
3. 是否有其他地方的查询逻辑不一致

#### 验证步骤

1. 检查 `admin-user-system.yaml` 中的状态枚举定义
2. 检查前端 admin 页面调用 API 时传递的状态参数
3. 确认 `user_profiles` 表的 `ListProfilesByStatus` 方法是否也正确实现

---

## 问题3：前端实名认证成功文案判断（低优先级）

### 原因分析

**用户报告**：
- `IdentityVerificationPage.vue:37` 判断 `identity.verifyMethod === 'auto'`
- 后端实际返回 `academic_db_match`、`manual`、`tencent_cloud`
- "自动通过"的成功文案永远不会出现

**代码审查结果**（`IdentityVerificationPage.vue:45-46`）：
```vue
<p v-if="identity.verifyMethod === 'academic_db_match' || identity.verifyMethod === 'tencent_cloud'">
  您的身份已通过自动认证系统验证
</p>
```

**数据库 schema**（`init.sql:429-439`）：
```sql
verify_method VARCHAR(50) CHECK (verify_method IN ('academic_db_match', 'tencent_cloud', 'manual'))
```

### 修复方案

**结论：当前代码已经正确，用户报告的问题可能基于旧版本代码。**

当前逻辑：
- `academic_db_match` 或 `tencent_cloud` → 显示"自动认证"文案
- `manual` → 显示"人工审核"文案

#### 可选优化

为了更健壮，可以改为：
```vue
<p v-if="identity.verifyMethod !== 'manual'">
  您的身份已通过自动认证系统验证
</p>
```

这样即使未来添加新的自动认证方式，也不需要修改前端代码。

---

## 问题4：Admin 路由引用 logs 页面（低优先级）

### 原因分析

**用户报告**：
- `router/index.ts:51` 引用了 `admin-logs` 路由
- 该路由指向 `@/views/logs/index.vue`
- 文件可能不存在

**代码审查结果**：
```typescript
// clients/admin/src/router/index.ts:51-55
{
  path: 'logs',
  name: 'admin-logs',
  component: () => import('@/views/logs/index.vue'),
  meta: { title: '操作日志', icon: Document },
}
```

**文件检查**：
- `clients/admin/src/views/logs/index.vue` **文件存在**

### 修复方案

**结论：此问题不存在，文件已经存在。**

用户可能基于旧版本代码或本地工作区状态做出判断。

#### 验证步骤

1. 确认 `clients/admin/src/views/logs/index.vue` 文件内容是否完整
2. 确认该页面是否能正常渲染
3. 如果页面是空白占位，考虑添加基本内容

---

## 修复优先级与实施顺序

### 第一阶段：高优先级修复（必须完成）

1. **问题1：应用 RBAC 权限中间件**
   - 修改 `rbac/handler.go` 和 `user/handler.go`
   - 修改 `main.go`
   - 创建权限定义 seed 脚本
   - 测试验证

### 第二阶段：验证与优化（可选）

2. **问题2：验证状态查询逻辑**
   - 检查 API 文档定义
   - 检查前端调用参数
   - 确认无误后关闭

3. **问题3：优化前端文案判断**
   - 改为 `verifyMethod !== 'manual'` 的更健壮写法
   - 可选，不影响功能

4. **问题4：验证 logs 页面**
   - 确认文件存在且内容完整
   - 确认无误后关闭

---

## 风险评估

### 问题1 修复风险

**风险**：
- 修改路由注册逻辑可能影响现有功能
- 权限定义不完整可能导致管理员无法访问某些功能
- 测试覆盖不足可能遗漏边界情况

**缓解措施**：
- 先在开发环境测试
- 确保超级管理员角色拥有所有权限
- 为每个权限编写单元测试
- 手动测试所有 admin 路由

### 问题2-4 风险

**风险**：低，主要是验证性工作

---

## 测试计划

### 单元测试

1. **RBAC 权限中间件测试**
   - 测试有权限时允许访问
   - 测试无权限时返回 403
   - 测试未登录时返回 401

2. **状态查询逻辑测试**
   - 测试 pending 状态查询
   - 测试 rejected 状态查询
   - 测试 verified 状态查询

### 集成测试

1. **端到端权限检查**
   - 创建测试角色和权限
   - 分配权限给角色
   - 用该角色登录并访问 admin 路由
   - 验证权限检查生效

2. **前端功能测试**
   - 测试实名认证页面文案显示
   - 测试 admin logs 页面访问

---

## 预期结果

### 问题1 修复后

- 所有 admin 路由都应用细粒度权限检查
- 管理员只能访问其角色拥有权限的接口
- 权限配置在数据库中真实生效

### 问题2-4 验证后

- 确认状态查询逻辑正确
- 确认前端文案显示正确
- 确认 logs 页面存在且可访问

---

## 后续工作

1. **权限管理文档**
   - 编写权限定义文档
   - 说明每个权限的作用范围
   - 提供权限分配最佳实践

2. **权限测试覆盖**
   - 为所有 admin 路由添加权限测试
   - 确保测试覆盖率达到 80% 以上

3. **前端权限控制**
   - 考虑在前端也添加权限检查
   - 隐藏用户无权访问的菜单项
   - 提升用户体验
