---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-07-31
---

# 课程与评课

> 状态：现行

## 覆盖范围

课程实体（院系、学期、分类、教师）、评课列表与搜索、发布与编辑、投票与回复、举报与审核、后台运营。

### 课程元数据完整性

课程名、学校和分类属于目录主干事实；课程代码、所属院系和学分可能因上游数据源不完整而暂缺。
API 对缺失代码/院系名省略可选字段，对必需出现的 `departmentID`、`credits` 明确返回 `null`。
客户端应展示“未分类/未提供”，不得把未知院系或学分渲染为 `0`；Repository 也不得用
`COALESCE(..., 0/'')` 伪造元数据。按学分排序时，未知学分排在已知值之后。

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
| `/api/v1/course/review/drafts` | GET | 获取当前用户的单槽草稿；不存在时返回 `data: null` |
| `/api/v1/course/review/drafts` | POST | 保存草稿 |
| `/api/v1/course/review/drafts` | DELETE | 删除当前用户的单槽草稿 |
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
| `/api/v1/course/review/admin/export` | GET | 导出（NDJSON/CSV；每次请求写成功或失败审计） |
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

软删除，同步更新计数。当前没有面向作者或管理员的受支持恢复入口，因此 Web“我的评课”
必须在发送 DELETE 前显示局部二次确认；首次点击只进入确认态，取消不得发请求，确认请求
进行中必须保持 single-flight。该交互约束不改变 API 的单次 DELETE 语义，也不等同于提供
undo 或 restore 能力。

## 互动

| 功能 | 规则 |
|------|------|
| 投票 | like / dislike，支持切换和取消 |
| 回复 | 楼中楼，经过净化和敏感词检查 |
| 收藏 | 按课程收藏 |
| 草稿 | 每用户一个服务端单槽，可选绑定课程；Web 支持自动保存，固定课程发布入口只能在课程匹配或未绑定时恢复，发布成功后不得删除其他课程的槽内容 |
| 举报 | 每用户每评课一次 |

## 评分展示

评课社区的公开展示面不显示精确评分数字，使用已发布的五级定性文案与表情语义。课程详情中的
维度条虽然通过长度和颜色可视化平均值，但每一行必须同时提供“维度名称 + 定性等级”的可访问
名称，并把纯视觉填充层从辅助技术树隐藏。无障碍修复不得借机把原始 `avgRating` 数值写入
`aria-label`、title 或隐藏文本。

## 内容审核

**敏感词**：`block` 拦截，`warn` 添加标记后台复核，`review` 设置状态为 `pending_review`，标记后台人工审核后发布。

**举报类型**：spam / inappropriate / harassment / false_info / other。后台可驳回、隐藏、删除。

**后台能力**：评课管理、举报处理、批量操作、内容编辑、教师管理、敏感词管理、操作日志、NDJSON/CSV 导出、内容标记。

### 敏感批量导出

评课导出最多处理 10,000 行，可按状态筛选；`json` 与 `ndjson` 都返回 NDJSON 流，CSV 与
NDJSON 只有出现 `# EXPORT_COMPLETE` 标记才代表响应完整。每次导出请求必须恰好写入一条
`category=admin_operation`、`event_type=data.export` 的审计事件，记录管理员、规范化后的格式
与状态、处理行数、行上限以及 success/failure；数据库流、序列化、写响应或完成标记失败时也
必须留下 failure 事件。审计持久化沿用不受请求取消影响的安全上下文，但 `row_count` 只表示
服务端已序列化/处理的行数，不等同于客户端已可靠保存的字节数。

## 访问控制

| 用户 | 权限 |
|------|------|
| 游客 | 课程/教师/评课预览；评课标题隐藏，正文只返回首个非空行，并受 `review_guest_preview_content_chars` 限制 |
| 已登录未认证 | 标题可见；正文只返回首个非空行，并同时受 `review_preview_content_chars` 与 `review_preview_content_percent` 限制 |
| 已认证学生 | 完整评课、发布评课 |
| 全局评课管理员 | 完整内容、后台管理 |
| 带学校/板块范围的评课管理员 | scoped grant 本身不提升公共列表正文权限；通过带资源边界的后台审核路由管理授权范围内内容 |

发布评课需要：已登录 + 实名认证通过 + 学生认证通过。

## 主要数据表

`departments`、`courses`、`teachers`、`terms`、`rating_dimensions`、`reviews`、`review_votes`、`review_reports`、`review_replies`、`course_favorites`、`review_drafts`、`notifications`、`audit_events`、`domain_event_outbox`、`sensitive_words`

## 代码入口

| 组件 | 位置 |
|------|------|
| 课程实体 | `server/internal/modules/course/` |
| 评课子系统 | `server/internal/modules/course/review/` |
| 净化 | `server/internal/pkg/sanitizer/` |
