# 授权策略模块

授权策略模块描述应用的授权骨架，包括能力层、访问事实层和所有权层的统一决策流程。

## 代码范围

| 代码位置 | 职责 |
| --- | --- |
| `server/internal/modules/rbac` | 能力计算和管理端授权中间件 |
| `server/internal/modules/course/review/access.go` | 评课访问事实解析和内容裁剪 |
| `server/internal/pkg/capability` | 能力字符串常量和管理端入口能力集 |

## 文档索引

| 文档 | 内容 |
| --- | --- |
| [01-authorization-model.md](01-authorization-model.md) | 能力层、访问事实层、所有权层的授权模型 |
| [02-policy-evaluation.md](02-policy-evaluation.md) | 授权决策链的执行顺序和各步骤代码入口 |
