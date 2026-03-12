# 评课社区模块

> 访问方式：`https://stuhelper.com/review`、`https://stuhelper.com/courses`、`https://stuhelper.com/teachers/:id`

## 模块定位

评课社区是 StuHelper 当前最完整的业务域，覆盖课程检索、匿名测评、教师主页、用户中心、通知与管理后台。

## 当前功能状态

| 功能                   | 状态      |
| ---------------------- | --------- |
| Casdoor SSO 登录       | 🟢 已完成 |
| 课程 / 院系 / 教师查询 | 🟢 已完成 |
| 动态评分维度           | 🟢 已完成 |
| 发布、编辑、删除测评   | 🟢 已完成 |
| 回复、收藏、投票       | 🟢 已完成 |
| 草稿保存与恢复         | 🟢 已完成 |
| 站内通知               | 🟢 已完成 |
| 举报审核与敏感词管理   | 🟢 已完成 |
| 管理后台               | 🟢 已完成 |

## 前端路由概览

| 路径                        | 说明         |
| --------------------------- | ------------ |
| `/review`                   | 评课社区首页 |
| `/courses`                  | 课程列表     |
| `/courses/:id`              | 课程概览     |
| `/courses/:id/reviews`      | 课程测评列表 |
| `/courses/:id/reviews/post` | 发布测评页   |
| `/teachers/:id`             | 教师主页     |

## 文档索引

| 文档                                               | 说明         |
| -------------------------------------------------- | ------------ |
| [01-data-model.md](01-data-model.md)               | 数据模型设计 |
| [02-api.md](02-api.md)                             | API 接口设计 |
| [03-components.md](03-components.md)               | 前端组件设计 |
| [04-routes.md](04-routes.md)                       | 前端路由设计 |
| [05-ui-spec.md](05-ui-spec.md)                     | UI 设计规范  |
| [06-security.md](06-security.md)                   | 安全设计     |
| [07-rating-dimensions.md](07-rating-dimensions.md) | 评分维度配置 |
