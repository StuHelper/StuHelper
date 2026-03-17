# 课程评论社区模块

课程评论社区是 StuHelper 的核心业务模块，由课程实体管理和评论子模块组成。课程实体层提供院系、学期、分类和课程的查询能力；评论子模块围绕评论的完整生命周期，覆盖内容发布、用户互动和管理审核。当前挂在评课路由下的是通知读取和已读接口，回复触发通知仍属于后续计划，长期应上收为应用级通知能力。

## 代码范围

| 代码路径                                                                | 职责                                                                    |
| ----------------------------------------------------------------------- | ----------------------------------------------------------------------- |
| `server/internal/modules/course/handler.go`                             | 课程实体 HTTP 路由注册，后台日志清理任务入口                            |
| `server/internal/modules/course/course.go`                              | 院系、学期、分类、课程列表、搜索、详情、统计 Handler                    |
| `server/internal/modules/course/service.go`                             | 课程实体业务逻辑层                                                      |
| `server/internal/modules/course/repository.go`                          | 课程实体数据访问层                                                      |
| `server/internal/modules/course/model.go`                               | 课程实体数据模型（Department / Course / Term / CourseCategory / Stats） |
| `server/internal/modules/course/review/handler.go`                      | 评论子模块路由注册，管理员操作日志辅助，缓存失效                        |
| `server/internal/modules/course/review/review.go`                       | 评论发布、更新、删除、投票、举报、内容检查 Handler                      |
| `server/internal/modules/course/review/review_read.go`                  | 评论列表、最新评论、批量评论、统计、热门排行、教师统计 Handler          |
| `server/internal/modules/course/review/review_interaction.go`           | 收藏、用户评论/投票/收藏列表 Handler                                    |
| `server/internal/modules/course/review/review_reply.go`                 | 回复创建、删除、列表 Handler                                            |
| `server/internal/modules/course/review/review_draft.go`                 | 草稿保存、加载、删除 Handler                                            |
| `server/internal/modules/course/review/review_notification.go`          | 当前挂在评课路由下的通知读取和已读状态 Handler                          |
| `server/internal/modules/course/review/rating.go`                       | 评分维度配置、课程评分统计（雷达图）Handler                             |
| `server/internal/modules/course/review/admin.go`                        | 举报列表/处理、管理员评论管理、批量操作、统计 Handler                   |
| `server/internal/modules/course/review/admin_review.go`                 | 管理员编辑评论内容 Handler                                              |
| `server/internal/modules/course/review/admin_export.go`                 | 操作日志查询、评论流式导出（NDJSON/CSV）Handler                         |
| `server/internal/modules/course/review/handler_teacher_admin.go`        | 教师管理 CRUD Handler                                                   |
| `server/internal/modules/course/review/handler_sensitive_word_admin.go` | 敏感词管理 CRUD Handler                                                 |
| `server/internal/modules/course/review/service.go`                      | 评论核心业务逻辑、评分校验、内容清洗、敏感词管理                        |
| `server/internal/modules/course/review/service_review_write.go`         | 评论发布、更新、删除、投票 Service                                      |
| `server/internal/modules/course/review/service_interaction.go`          | 收藏、草稿、回复和当前通知读取接口 Service                              |
| `server/internal/modules/course/review/service_report.go`               | 举报、举报处理 Service                                                  |
| `server/internal/modules/course/review/service_admin.go`                | 管理员评论管理、批量操作、操作日志、流式导出 Service                    |
| `server/internal/modules/course/review/service_admin_stats.go`          | 管理统计、评分趋势、热门排行、教师评分统计 Service                      |
| `server/internal/modules/course/review/access.go`                       | 评论访问规则（Access Facts），内容脱敏                                  |
| `server/internal/modules/course/review/filter.go`                       | 敏感词过滤器（block/warn 匹配）                                         |
| `server/internal/modules/course/review/consts.go`                       | 评论/举报状态常量，排序选项                                             |
| `server/internal/modules/course/review/model.go`                        | 评论子模块全部数据模型                                                  |
| `server/internal/modules/course/review/repository*.go`                  | 评论子模块数据访问层（按职责拆分为多个文件）                            |

## 功能概览

### 课程实体

- 院系列表查询，支持按分类过滤
- 学期列表查询，当前学期置顶排序
- 课程分类列表
- 课程列表查询，支持搜索、院系过滤、分类过滤和多种排序（名称/学分/评论数）
- 课程搜索（名称和编码模糊匹配）
- 课程详情
- 学习中心统计（课程数和院系数）

### 评论内容

- 评论发布（评分维度校验、敏感词检查、HTML 清洗、去重检查、成绩记录）
- 评论更新（所有权校验、事务内 FOR UPDATE 锁）
- 评论软删除（所有权校验）
- 评论草稿系统（按课程保存/加载/删除，每用户每课程一份）
- 评分维度配置查询
- 课程评分统计和雷达图数据
- 评分趋势（按学期）
- 教师评分统计和雷达图
- 课程授课教师统计卡片

### 用户互动

- 投票（like/dislike，同类型再次投票取消，不同类型切换，事务内计数更新）
- 回复（创建/删除，回复通知为后续计划）
- 课程收藏（添加/取消，分页查询）
- 用户中心（我的评论/我的投票/我的收藏，分页查询）

### 当前挂载的通知接口

- 通知列表
- 未读通知计数
- 单条标记已读
- 全部标记已读

这些接口现在由评课模块提供，但业务边界上更适合作为航小版应用级通知中心的一部分。

### 内容审核

- 举报创建（spam/inappropriate/harassment/false_info/other，同一用户同一评论只能举报一次）
- 管理员处理举报（reject/hide_review/delete_review）
- 评论管理（隐藏/恢复/删除，单条和批量操作，最大批量 100 条）
- 管理员直接编辑评论内容（保存原始内容和编辑原因）
- 敏感词管理 CRUD（当前只有 block/warn 两级）
- 教师管理 CRUD（含院系关联和评论统计）
- 评论数据导出（NDJSON/CSV 流式导出，CSV 含 BOM 头和公式注入防护）
- 操作审计日志（90 天保留，每日定时清理后台任务）
- 管理后台统计仪表盘（总评论/已发布/已隐藏/已删除/待处理举报/今日/本周评论数）
- 前端内容检查端点（当前只返回敏感词检查结果，需认证，防探测）

## 访问规则

| 身份                                         | 可执行的操作                                                                          |
| -------------------------------------------- | ------------------------------------------------------------------------------------- |
| 匿名用户                                     | 浏览课程实体、查看评分统计和排行，评论列表返回空标题和空内容                          |
| 已认证用户                                   | 匿名用户的全部操作 + 查看评论预览                                                     |
| 已通过学生认证且学校命中评课访问白名单的用户 | 已认证用户的全部操作 + 查看评论完整内容                                               |
| 同时通过学生认证和实名认证的用户             | 查看完整内容 + 发布/编辑/删除评论，投票/举报/回复，收藏/草稿，访问当前通知接口        |
| 管理员（持有对应 capability）                | 全部用户操作 + 查看完整内容 + 举报处理/评论管理/批量操作/数据导出/教师管理/敏感词管理 |

访问规则由 `access.go` 中的 `ReviewAccessFacts` 结构体统一决策，基于学生认证状态（`studentVerified`）、实名认证状态（`identityVerified`）和管理权限（`canManageReviews`）三项事实计算。

当前实现细节：

- 学校范围读取系统配置 `review_access_school_ids`；配置为空时，回退到已启用学校列表
- 预览标题长度读取 `review_preview_title_chars`
- 预览正文长度读取 `review_preview_content_chars`
- 预览正文展示比例读取 `review_preview_content_percent`

## API 端点

### 公开端点

```
GET  /api/v1/course/departments
GET  /api/v1/course/terms
GET  /api/v1/course/categories
GET  /api/v1/course/courses
GET  /api/v1/course/courses/search
GET  /api/v1/course/courses/{courseID}
GET  /api/v1/course/stats
GET  /api/v1/course/review/rating-dimensions
GET  /api/v1/course/review/courses/{courseID}/rating-stats
GET  /api/v1/course/review/courses/{courseID}/rating-trend
GET  /api/v1/course/review/courses/{courseID}/teachers
GET  /api/v1/course/review/courses/{courseID}/reviews
GET  /api/v1/course/review/reviews/latest
GET  /api/v1/course/review/reviews/batch
GET  /api/v1/course/review/reviews/{reviewID}/replies
GET  /api/v1/course/review/stats
GET  /api/v1/course/review/rankings/hot
GET  /api/v1/course/review/teachers/{teacherID}/stats
```

### 认证端点

```
POST   /api/v1/course/review/reviews
PUT    /api/v1/course/review/reviews/{reviewID}
DELETE /api/v1/course/review/reviews/{reviewID}
POST   /api/v1/course/review/reviews/{reviewID}/votes
POST   /api/v1/course/review/reviews/{reviewID}/reports
POST   /api/v1/course/review/reviews/{reviewID}/replies
DELETE /api/v1/course/review/replies/{replyID}
POST   /api/v1/course/review/content/check
POST   /api/v1/course/review/courses/{courseID}/favorites
DELETE /api/v1/course/review/courses/{courseID}/favorites
POST   /api/v1/course/review/drafts
GET    /api/v1/course/review/drafts/{courseID}
DELETE /api/v1/course/review/drafts/{courseID}
GET    /api/v1/course/review/user/reviews
GET    /api/v1/course/review/user/votes
GET    /api/v1/course/review/user/favorites
GET    /api/v1/course/review/user/notifications
GET    /api/v1/course/review/user/notifications/unread-count
PUT    /api/v1/course/review/user/notifications/{notificationID}/read
PUT    /api/v1/course/review/user/notifications/read-all
```

### 管理端点

```
GET    /api/v1/course/review/admin/reports
PUT    /api/v1/course/review/admin/reports/{reportID}
GET    /api/v1/course/review/admin/reviews
PUT    /api/v1/course/review/admin/reviews/{reviewID}
POST   /api/v1/course/review/admin/reviews/batch
POST   /api/v1/course/review/admin/reviews/{reviewID}/edit
GET    /api/v1/course/review/admin/stats
GET    /api/v1/course/review/admin/logs
GET    /api/v1/course/review/admin/export
GET    /api/v1/course/review/admin/teachers
POST   /api/v1/course/review/admin/teachers
PUT    /api/v1/course/review/admin/teachers/{teacherID}
DELETE /api/v1/course/review/admin/teachers/{teacherID}
GET    /api/v1/course/review/admin/sensitive-words
POST   /api/v1/course/review/admin/sensitive-words
PUT    /api/v1/course/review/admin/sensitive-words/{sensitiveWordID}
DELETE /api/v1/course/review/admin/sensitive-words/{sensitiveWordID}
```

## 数据库表

| 表名                   | 职责                                                                             |
| ---------------------- | -------------------------------------------------------------------------------- |
| `departments`          | 院系                                                                             |
| `courses`              | 课程                                                                             |
| `teachers`             | 教师                                                                             |
| `rating_dimensions`    | 评分维度配置，当前主要用于只读查询和统计聚合                                     |
| `reviews`              | 课程评论                                                                         |
| `review_votes`         | 投票（赞/踩）                                                                    |
| `review_reports`       | 用户举报                                                                         |
| `review_replies`       | 评论回复                                                                         |
| `course_favorites`     | 用户课程收藏                                                                     |
| `review_drafts`        | 评论草稿                                                                         |
| `notifications`        | 当前仍沿用旧通知表结构，并由评课模块挂载查询和已读接口，长期应归入应用级通知中心 |
| `admin_operation_logs` | 管理操作审计日志                                                                 |
| `sensitive_words`      | 敏感词库                                                                         |

## 文档索引

| 文档                                               | 内容                                                   |
| -------------------------------------------------- | ------------------------------------------------------ |
| [01-review-lifecycle.md](01-review-lifecycle.md)   | 评论生命周期：创建、更新、删除、草稿、状态流转         |
| [02-interaction.md](02-interaction.md)             | 互动系统：投票、回复、收藏、用户中心                   |
| [03-moderation.md](03-moderation.md)               | 内容审核：举报、评论管理、敏感词、教师管理、导出、审计 |
| [04-security.md](04-security.md)                   | 安全设计：匿名机制、内容安全、防刷、TOCTOU 防护        |
| [05-rating-dimensions.md](05-rating-dimensions.md) | 评分维度系统设计稿：目标能力、雷达图方案、历史兼容设想 |
