# 应用内 RBAC 模块

RBAC 模块是航小伴自己的业务授权基础，不属于 Casdoor 身份平面。

## 代码范围

| 代码位置 | 作用 |
| --- | --- |
| `server/internal/modules/rbac` | 角色、权限、用户组、能力判断 |
| `server/internal/pkg/capability` | capability 常量和后台入口能力集合 |

## 当前模型

| 实体 | 说明 |
| --- | --- |
| Role | 粗粒度业务角色 |
| Permission | capability，对应具体后台能力 |
| User Role | 用户角色绑定 |
| User Group | 用户组及用户组权限 |
| User Permission Override | 对单个用户的显式覆盖 |

最终能力是角色权限、用户组权限和个人覆盖合并后的结果。

## 当前接口

后台接口统一挂在 `/api/v1/admin` 下：

- `/roles`
- `/roles/{roleID}`
- `/roles/{roleID}/permissions`
- `/permissions`
- `/users/{userID}/roles`
- `/users/{userID}/permissions`
- `/groups`
- `/groups/{groupID}`
- `/groups/{groupID}/members`
- `/groups/{groupID}/permissions`

## 当前规则

- 后台路由全部按 permission 判定
- `/auth/me` 返回的 `capabilities` 来自这里
- 平台 `isAdmin` 不会自动变成航小伴后台权限
