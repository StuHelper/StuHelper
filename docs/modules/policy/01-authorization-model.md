# 授权模型

授权模型由三层组成：能力层、访问事实层和所有权层。

## 能力层

管理能力来自 `roles`、`permissions`、`role_permissions`、`user_roles`、`user_group_*` 和 `user_permissions` 的组合计算。

### 常用能力

| 能力字符串 | 用途 |
| --- | --- |
| `admin:dashboard:view` | 管理后台仪表盘访问 |
| `admin:reviews:manage` | 评课内容审核和管理 |
| `admin:reports:manage` | 举报处理 |
| `admin:teachers:manage` | 教师信息管理 |
| `admin:sensitive_words:manage` | 敏感词管理 |
| `admin:logs:view` | 操作日志查看 |
| `user:identity:read` / `review` | 实名认证查看和审核 |
| `user:student:read` / `review` | 学生认证查看和审核 |
| `user:school:read` / `update` | 学校配置查看和更新 |
| `user:system:read` / `update` | 系统配置查看和更新 |
| `rbac:role:*` | 角色管理 |
| `rbac:permission:read` | 权限查看 |
| `rbac:user:*` | 用户权限管理 |
| `rbac:group:*` | 用户组管理 |

能力计算流程：角色权限 + 用户组权限合并，然后应用个人权限覆盖（`granted=true` 授予，`granted=false` 拒绝）。结果去重排序后返回。

## 访问事实层

评课和用户系统模块读取以下业务事实来决定内容可见性和操作资格：

| 事实 | 数据来源 | 用途 |
| --- | --- | --- |
| `studentVerified` | `user_profiles.verification_status == 'verified'` 且 `school_id` 匹配 | 评课完整内容可见性 |
| `identityVerified` | `user_identities.verified == true` | 发布评课的资格条件 |
| `schoolID` | `user_profiles.school_id` | 学校范围过滤 |
| `canManageReviews` | 能力检查 `admin:reviews:manage` | 隐藏内容可见性和管理视图 |

访问事实在 `review/access.go` 的 `resolveReviewAccessFacts` 中解析，组装为 `ReviewAccessFacts` 结构体。

## 所有权层

评论编辑、评论删除、回复删除等操作在服务层事务内检查内容所有权和资源状态。所有权通过 `user_hash` 匹配判断。

## 授权结果

| 场景 | 规则 |
| --- | --- |
| 匿名浏览评课 | 评论标题和内容置空（仅显示评分和元数据） |
| 已认证但未通过学生认证 | 评论标题和内容截断预览（标题 24 字符，内容 120 字符） |
| 已通过学生认证 | 显示评论完整内容 |
| 发布评课 | 需要学生认证通过且实名认证通过 |
| 管理操作 | 由能力字符串控制入口和可用操作；管理员可查看隐藏内容 |

## 相关文档

- [授权决策流程](02-policy-evaluation.md)
- [RBAC 权限控制](../rbac/README.md)
