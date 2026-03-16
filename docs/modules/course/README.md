# 评课社区模块

评课社区是当前最完整的业务域。它不只是一组测评接口，还包含课程索引、教师信息、用户交互、通知和后台内容治理。

## 代码范围

| 代码位置 | 职责 |
| --- | --- |
| `server/internal/modules/course` | 课程、院系、学期、分类等实体接口 |
| `server/internal/modules/course/review` | 评课读写、回复、收藏、通知、举报、后台审核 |
| `server/api/paths/course.yaml` | 课程实体契约 |
| `server/api/paths/review-*.yaml` | 评课子域契约 |

## 当前能力

| 子域 | 说明 |
| --- | --- |
| 课程实体 | 院系、学期、分类、课程搜索、课程详情 |
| 测评内容 | 发布、编辑、删除、评分维度、评分统计、教师统计 |
| 用户交互 | 点赞、踩、收藏、草稿、回复、我的测评、我的投票 |
| 站内通知 | 通知列表、未读数、已读、全部已读 |
| 内容治理 | 举报、敏感词、隐藏、恢复、批量审核、导出、操作日志 |

## 权限边界

- 公开列表接口允许匿名访问，但会按访问事实裁剪内容
- 发布测评需要登录、学生认证通过且实名认证通过
- 后台管理能力全部走应用内 RBAC capability，不再复用平台 `isAdmin`

## 推荐搭配文档

- 接口清单看 [../../reference/api-overview.md](../../reference/api-overview.md)
- 数据库结构看 [../../reference/database.md](../../reference/database.md)
- 安全细节看 [06-security.md](06-security.md)
- 评分维度设计看 [07-rating-dimensions.md](07-rating-dimensions.md)
