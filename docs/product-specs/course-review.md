# 课程与评课

> 状态：现行

## 覆盖范围

课程实体（院系、学期、分类、教师）、评课列表与搜索、发布与编辑、投票与回复、举报与审核、后台运营。

## 浏览与搜索

| 端点 | 说明 |
|------|------|
| `/api/v1/course/courses` | 课程列表 |
| `/api/v1/course/courses/search` | 课程搜索 |
| `/api/v1/course/courses/{courseID}` | 课程详情 |
| `/api/v1/course/review/courses/{courseID}/reviews` | 课程评课列表 |
| `/api/v1/course/review/reviews/latest` | 最新评课 |
| `/api/v1/course/review/reviews/search` | 评课搜索 |
| `/api/v1/course/review/teachers/{teacherID}/stats` | 教师统计 |

搜索在服务端执行。

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

**敏感词**：`block` 拦截，`warn` 添加标记后台复核。

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

`departments`、`courses`、`teachers`、`terms`、`rating_dimensions`、`reviews`、`review_votes`、`review_reports`、`review_replies`、`course_favorites`、`review_drafts`、`notifications`、`admin_operation_logs`、`sensitive_words`

## 代码入口

| 组件 | 位置 |
|------|------|
| 课程实体 | `server/internal/modules/course/` |
| 评课子系统 | `server/internal/modules/course/review/` |
| 净化 | `server/internal/pkg/sanitizer/` |
