# 航小伴授权模型

这份文档描述当前已经落地的授权主干，以及后续还能往哪里演进。

## 当前运行时模型

### 1. 身份平面和业务授权分离

- Casdoor 负责证明用户是谁
- 航小伴后端负责决定用户在应用里能做什么
- `isPlatformAdmin` 不再充当航小伴后台总开关

### 2. 应用内 RBAC 负责后台能力

当前后台权限已经统一收口到 capability，例如：

| 能力 | 用途 |
| --- | --- |
| `admin:dashboard:view` | 后台首页 |
| `admin:reviews:manage` | 评课审核、隐藏、恢复、编辑 |
| `admin:reports:manage` | 举报处理 |
| `admin:teachers:manage` | 教师管理 |
| `admin:sensitive_words:manage` | 敏感词管理 |
| `admin:logs:view` | 操作日志 |
| `user:*` | 用户系统后台 |
| `rbac:*` | 角色、权限、用户组管理 |

这些能力由本地 `roles`、`permissions`、`role_permissions`、`user_roles`、`user_group_*`、`user_permissions` 共同决定。

### 3. 访问事实负责内容可见性和发布资格

评课模块当前不是只看角色，还会解析一组访问事实：

| 事实 | 来源 | 用途 |
| --- | --- | --- |
| `studentVerified` | 用户档案 | 决定是否可看完整内容 |
| `identityVerified` | 实名认证状态 | 决定是否可发布 |
| `schoolID` | 用户档案 | 当前学校范围判断 |
| `canManageReviews` | 应用 capability | 后台查看隐藏内容、越权审核 |

当前评课规则是：

- 未登录只能看脱敏内容
- 已登录但未完成学生认证只能看截断内容
- 学生认证通过可以看完整内容
- 学生认证通过且实名认证通过才允许发布

### 4. owner 规则仍然由业务数据决定

例如测评编辑、删除、回复删除，当前依然基于内容拥有者判断，而不是给每条内容下发角色。

## 长期演进方向

如果后续出现高基数的课程管理员、分类管理员、教师默认编辑权之类的需求，推荐继续往下面演进：

- capability RBAC 继续负责粗粒度后台权限
- 资源关系负责高基数委派
- 访问事实继续负责学校、认证状态、身份类型

这时再引入 OpenFGA / SpiceDB 这类关系引擎才是合理时机。

## 当前反模式

- 不能再用 `isAdmin` 兜底航小伴后台
- 不能把课程级业务权限塞回 Casdoor
- 不能只靠前端做内容裁剪
