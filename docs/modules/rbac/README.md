# RBAC 权限控制模块

RBAC 模块管理应用级角色、权限、用户组和个人权限覆盖，计算用户最终生效的能力集合。

## 代码范围

| 代码位置 | 职责 |
| --- | --- |
| `server/internal/modules/rbac` | 角色、权限、用户组的 CRUD，能力计算，授权中间件 |
| `server/internal/pkg/capability` | 能力字符串常量和管理端入口能力集 |

## 数据模型

| 实体 | 数据库表 | 说明 |
| --- | --- | --- |
| 角色 | `roles` | 粗粒度业务角色（如"内容审核员"、"系统管理员"），包含 `is_system` 标记 |
| 权限 | `permissions` | 细粒度能力字符串（如 `admin:reviews:manage`），包含 `module`、`action`、`scope_school_ids`、`scope_roles` |
| 角色-权限关联 | `role_permissions` | 角色绑定的权限集合 |
| 用户-角色绑定 | `user_roles` | 用户分配的角色 |
| 用户组 | `user_groups` | 用户组定义 |
| 用户组成员 | `user_group_members` | 用户组成员关系 |
| 用户组-权限关联 | `user_group_permissions` | 用户组绑定的权限集合 |
| 个人权限覆盖 | `user_permissions` | 对个别用户的显式授权或拒绝（`granted` 字段） |

## 权限计算流程

```go
// 伪代码
func ComputeUserCapabilities(userID int64) []string {
    capabilities := []string{}

    // 1. 从用户角色获取权限
    roles := GetUserRoles(userID)
    for _, role := range roles {
        capabilities = append(capabilities, GetRolePermissions(role.ID)...)
    }

    // 2. 从用户组获取权限
    groups := GetUserGroups(userID)
    for _, group := range groups {
        capabilities = append(capabilities, GetGroupPermissions(group.ID)...)
    }

    // 3. 应用个人权限覆盖
    overrides := GetUserPermissionOverrides(userID)
    for _, override := range overrides {
        if override.Granted {
            capabilities = append(capabilities, override.PermissionName)
        } else {
            capabilities = remove(capabilities, override.PermissionName)
        }
    }

    return unique(sort(capabilities))
}
```

数据库层通过 `GetEffectivePermissions` 单次查询完成计算，返回的每条记录包含 `source`（`role`、`group` 或 `override`）标识权限来源。

## API 端点

所有管理端点挂载在 `/api/v1/admin` 下：

### 角色管理

| 端点 | 方法 | 所需能力 | 说明 |
| --- | --- | --- | --- |
| `/roles` | GET | `rbac:role:read` | 列出所有角色 |
| `/roles` | POST | `rbac:role:create` | 创建角色 |
| `/roles/:roleID` | PUT | `rbac:role:update` | 更新角色 |
| `/roles/:roleID` | DELETE | `rbac:role:delete` | 删除角色（系统角色禁止删除） |
| `/roles/:roleID/permissions` | GET | `rbac:role:read` | 查看角色绑定的权限 ID |
| `/roles/:roleID/permissions` | PUT | `rbac:role:update` | 设置角色权限（支持 `clearAll` 确认清空） |

### 权限查询

| 端点 | 方法 | 所需能力 | 说明 |
| --- | --- | --- | --- |
| `/permissions` | GET | `rbac:permission:read` | 列出权限（可按 `module` 过滤） |

### 用户权限管理

| 端点 | 方法 | 所需能力 | 说明 |
| --- | --- | --- | --- |
| `/users/:userID/roles` | GET | `rbac:user:read` | 查看用户角色 |
| `/users/:userID/roles` | PUT | `rbac:user:update` | 设置用户角色 |
| `/users/:userID/permissions` | GET | `rbac:user:read` | 查看用户最终生效权限 |
| `/users/:userID/permissions` | PUT | `rbac:user:update` | 设置用户个人权限覆盖 |

### 用户组管理

| 端点 | 方法 | 所需能力 | 说明 |
| --- | --- | --- | --- |
| `/groups` | GET | `rbac:group:read` | 列出所有用户组 |
| `/groups` | POST | `rbac:group:create` | 创建用户组 |
| `/groups/:groupID` | PUT | `rbac:group:update` | 更新用户组 |
| `/groups/:groupID` | DELETE | `rbac:group:delete` | 删除用户组 |
| `/groups/:groupID/members` | GET | `rbac:group:read` | 查看用户组成员 |
| `/groups/:groupID/members` | PUT | `rbac:group:update` | 设置用户组成员 |
| `/groups/:groupID/permissions` | PUT | `rbac:group:update` | 设置用户组权限 |

## 常用能力

| 能力字符串 | 用途 |
| --- | --- |
| `admin:dashboard:view` | 访问管理后台仪表盘 |
| `admin:reviews:manage` | 评课内容审核和管理 |
| `admin:reports:manage` | 举报处理 |
| `admin:teachers:manage` | 教师信息管理 |
| `admin:sensitive_words:manage` | 敏感词管理 |
| `admin:logs:view` | 操作日志查看 |
| `user:identity:read` | 查看实名认证列表 |
| `user:identity:review` | 审核实名认证 |
| `user:student:read` | 查看学生认证列表 |
| `user:student:review` | 审核学生认证 |
| `user:school:read` | 查看学校配置 |
| `user:school:update` | 更新学校配置 |
| `user:system:read` | 查看系统配置 |
| `user:system:update` | 更新系统配置 |
| `rbac:role:read` | 查看角色 |
| `rbac:role:create` | 创建角色 |
| `rbac:role:update` | 更新角色 |
| `rbac:role:delete` | 删除角色 |
| `rbac:permission:read` | 查看权限 |
| `rbac:user:read` | 查看用户权限信息 |
| `rbac:user:update` | 更新用户权限信息 |
| `rbac:group:read` | 查看用户组 |
| `rbac:group:create` | 创建用户组 |
| `rbac:group:update` | 更新用户组 |
| `rbac:group:delete` | 删除用户组 |

## 前端使用示例

```typescript
import { useAuth } from '@/composables/useAuth'

const { hasCapability } = useAuth()

// 在模板中
<el-button v-if="hasCapability('admin:reviews:manage')">
  审核评课
</el-button>

// 在脚本中
if (hasCapability('admin:reviews:manage')) {
  // 显示管理界面
}
```

## 后端使用示例

```go
// 路由注册时使用 RequirePermission 中间件
admin.GET("/reviews", rbac.RequirePermission(permissionService, "admin:reviews:manage"), handler.ListReviews)

// 路由组级别使用 RequireAnyPermission 纵深防御
adminGroup.Use(rbac.RequireAnyPermission(permissionService, capability.AdminEntryCapabilities...))
```

`RequirePermission` 中间件执行流程：
1. 解析 external_id 到内部用户 ID（结果缓存在 Gin context）
2. 加载用户最终生效权限列表（结果缓存在 Gin context）
3. 在生效权限中查找目标权限名称
4. 验证 scope 约束（`scope_school_ids` 白名单、`scope_roles` 角色要求）

## 消费模式

- `/auth/me` 调用 `GetUserCapabilities` 返回用户完整能力集
- 前端页面和管理后台根据能力集控制菜单和按钮可见性
- `isPlatformAdmin` 维持 Casdoor 平台管理员语义，与应用能力集独立

## 相关文档

- [授权模型](../policy/01-authorization-model.md)
- [授权决策流程](../policy/02-policy-evaluation.md)
