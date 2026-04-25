---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-04-19
---

# 课程与评课

> 状态：现行

## 覆盖范围

课程实体（院系、学期、分类、教师）、评课列表与搜索、发布与编辑、投票与回复、举报与审核、后台运营。

## 端点

### 公开（无需认证）

| 端点 | 说明 |
|------|------|
| `/api/v1/course/courses` | 课程列表 |
| `/api/v1/course/courses/search` | 课程搜索 |
| `/api/v1/course/courses/{courseID}` | 课程详情 |
| `/api/v1/course/review/rating-dimensions` | 评分维度配置 |
| `/api/v1/course/review/courses/{courseID}/rating-stats` | 课程评分统计 |
| `/api/v1/course/review/courses/{courseID}/rating-trend` | 课程评分趋势 |
| `/api/v1/course/review/courses/{courseID}/teachers` | 课程教师列表 |
| `/api/v1/course/review/courses/{courseID}/reviews` | 课程评课列表（optionalAuth） |
| `/api/v1/course/review/reviews/latest` | 最新评课（optionalAuth） |
| `/api/v1/course/review/reviews/search` | 评课搜索（optionalAuth） |
| `/api/v1/course/review/reviews/batch` | 批量课程评课（optionalAuth） |
| `/api/v1/course/review/stats` | 评课统计 |
| `/api/v1/course/review/rankings/hot` | 热门课程排行 |
| `/api/v1/course/review/teachers` | 教师公开列表 |
| `/api/v1/course/review/teachers/hot` | 热门教师列表 |
| `/api/v1/course/review/teachers/{teacherID}/stats` | 教师评分统计 |
| `/api/v1/course/review/reviews/{reviewID}/replies` | 回复列表 |

### 需要认证

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/review/reviews` | POST | 发布评课 |
| `/api/v1/course/review/reviews/{reviewID}` | PUT | 更新评课 |
| `/api/v1/course/review/reviews/{reviewID}` | DELETE | 删除评课 |
| `/api/v1/course/review/reviews/{reviewID}/votes` | POST | 投票（like/dislike） |
| `/api/v1/course/review/reviews/{reviewID}/reports` | POST | 举报评课 |
| `/api/v1/course/review/reviews/{reviewID}/replies` | POST | 创建回复 |
| `/api/v1/course/review/replies/{replyID}` | DELETE | 删除回复 |
| `/api/v1/course/review/courses/{courseID}/favorites` | GET | 收藏状态 |
| `/api/v1/course/review/courses/{courseID}/favorites` | POST | 添加收藏 |
| `/api/v1/course/review/courses/{courseID}/favorites` | DELETE | 取消收藏 |
| `/api/v1/course/review/drafts` | POST | 保存草稿 |
| `/api/v1/course/review/drafts/{courseID}` | GET | 获取草稿 |
| `/api/v1/course/review/drafts/{courseID}` | DELETE | 删除草稿 |
| `/api/v1/course/review/content/check` | POST | 内容敏感词检查 |

### 用户中心（需要认证）

| 端点 | 说明 |
|------|------|
| `/api/v1/course/review/user/reviews` | 我的评课 |
| `/api/v1/course/review/user/votes` | 我的投票 |
| `/api/v1/course/review/user/favorites` | 我的收藏 |
| `/api/v1/course/review/user/notifications` | 通知列表 |
| `/api/v1/course/review/user/notifications/unread-count` | 未读数 |
| `/api/v1/course/review/user/notifications/{notificationID}/read` | 标记已读 |
| `/api/v1/course/review/user/notifications/read-all` | 全部已读 |

### 后台管理（需要 admin capability）

| 端点 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/review/admin/reports` | GET | 举报列表 |
| `/api/v1/course/review/admin/reports/{reportID}` | PUT | 处理举报 |
| `/api/v1/course/review/admin/reviews` | GET | 评课管理列表 |
| `/api/v1/course/review/admin/reviews/{reviewID}` | PUT | 更新评课状态（隐藏/恢复/删除） |
| `/api/v1/course/review/admin/reviews/{reviewID}/edit` | POST | 编辑评课内容 |
| `/api/v1/course/review/admin/reviews/batch` | PATCH | 批量操作 |
| `/api/v1/course/review/admin/stats` | GET | 后台统计 |
| `/api/v1/course/review/admin/logs` | GET | 操作日志 |
| `/api/v1/course/review/admin/export` | GET | 导出（NDJSON/CSV） |
| `/api/v1/course/review/admin/teachers` | GET | 教师管理列表 |
| `/api/v1/course/review/admin/teachers` | POST | 创建教师 |
| `/api/v1/course/review/admin/teachers/{teacherID}` | PUT | 更新教师 |
| `/api/v1/course/review/admin/teachers/{teacherID}` | DELETE | 删除教师 |
| `/api/v1/course/review/admin/sensitive-words` | GET | 敏感词列表 |
| `/api/v1/course/review/admin/sensitive-words` | POST | 创建敏感词 |
| `/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}` | PUT | 更新敏感词 |
| `/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}` | DELETE | 删除敏感词 |
| `/api/v1/course/review/admin/content-flags` | GET | 内容标记列表 |
| `/api/v1/course/review/admin/content-flags/{reviewID}/clear` | PUT | 清除内容标记 |

## 评课生命周期

### 创建

访问控制 → 输入校验 → 文本净化 → 敏感词检查 → 事务内（检查课程/教师/去重 → 写 review 与统计）

### 更新

仅作者可编辑，关键路径 `FOR UPDATE`，状态需允许编辑。

### 删除

软删除，同步更新计数。

## 互动

| 功能 | 规则 |
|------|------|
| 投票 | like / dislike，支持切换和取消 |
| 回复 | 楼中楼，经过净化和敏感词检查 |
| 收藏 | 按课程收藏 |
| 草稿 | 每用户每课程一份，支持自动保存 |
| 举报 | 每用户每评课一次 |

## 内容审核

**敏感词**：`block` 拦截，`warn` 添加标记后台复核，`review` 设置状态为 `pending_review`，标记后台人工审核后发布。

**举报类型**：spam / inappropriate / harassment / false_info / other。后台可驳回、隐藏、删除。

**后台能力**：评课管理、举报处理、批量操作、内容编辑、教师管理、敏感词管理、操作日志、NDJSON/CSV 导出、内容标记。

## 访问控制

| 用户 | 权限 |
|------|------|
| 游客 | 课程/教师/评课预览 |
| 已登录未认证 | 比游客多，评课正文有限制 |
| 已认证学生 | 完整评课、发布评课 |
| 管理员 | 完整内容、后台管理 |

发布评课需要：已登录 + 实名认证通过 + 学生认证通过。

## 主要数据表

`departments`、`courses`、`teachers`、`terms`、`rating_dimensions`、`reviews`、`review_votes`、`review_reports`、`review_replies`、`course_favorites`、`review_drafts`、`notifications`、`audit_events`、`domain_event_outbox`、`sensitive_words`

## 代码入口

| 组件 | 位置 |
|------|------|
| 课程实体 | `server/internal/modules/course/` |
| 评课子系统 | `server/internal/modules/course/review/` |
| 净化 | `server/internal/pkg/sanitizer/` |
