# API 概览

基础路径 `/api/v1`。字段和 schema 以 `server/api/openapi.yaml` 为准。

## 健康检查

| 路径 | 方法 | 说明 |
|------|------|------|
| `/health/live` | GET | 存活 |
| `/health/ready` | GET | 就绪 |

## 认证

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/auth/login` | GET | 登录跳转地址 |
| `/api/v1/auth/signup` | GET | 注册跳转地址 |
| `/api/v1/auth/callback` | GET | OIDC 回调 |
| `/api/v1/auth/exchange-native` | POST | 原生 App 令牌交换（OIDC code+state → access/refresh tokens） |
| `/api/v1/auth/phone/request-otp` | POST | 发送验证码 |
| `/api/v1/auth/phone/verify-otp` | POST | 验证码登录 |
| `/api/v1/auth/refresh` | POST | 续期 |
| `/api/v1/auth/me` | GET | 当前用户 |
| `/api/v1/auth/logout` | POST | 登出 |
| `/api/v1/auth/logout-all` | POST | 全设备登出 |

## 课程实体

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/departments` | GET | 院系列表 |
| `/api/v1/course/terms` | GET | 学期列表 |
| `/api/v1/course/categories` | GET | 课程分类 |
| `/api/v1/course/courses` | GET | 课程列表 |
| `/api/v1/course/courses/search` | GET | 搜索课程 |
| `/api/v1/course/courses/grouped` | GET | 按院系分组课程目录 |
| `/api/v1/course/courses/{courseID}` | GET | 课程详情 |
| `/api/v1/course/stats` | GET | 门户统计 |

## 评课（公开）

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/review/rating-dimensions` | GET | 评分维度 |
| `/api/v1/course/review/courses/{courseID}/rating-stats` | GET | 评分统计 |
| `/api/v1/course/review/courses/{courseID}/rating-trend` | GET | 评分趋势 |
| `/api/v1/course/review/courses/{courseID}/teachers` | GET | 课程教师 |
| `/api/v1/course/review/courses/{courseID}/reviews` | GET | 评课列表 |
| `/api/v1/course/review/reviews/latest` | GET | 最新评课 |
| `/api/v1/course/review/reviews/search` | GET | 搜索评课 |
| `/api/v1/course/review/reviews/batch` | GET | 批量读取 |
| `/api/v1/course/review/reviews/{reviewID}/replies` | GET | 回复列表 |
| `/api/v1/course/review/stats` | GET | 门户统计 |
| `/api/v1/course/review/rankings/hot` | GET | 热门排行 |
| `/api/v1/course/review/teachers` | GET | 公开教师列表 |
| `/api/v1/course/review/teachers/hot` | GET | 热门教师排行 |
| `/api/v1/course/review/teachers/{teacherID}/stats` | GET | 教师统计 |

## 评课（需认证）

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/review/reviews` | POST | 发布评课 |
| `/api/v1/course/review/reviews/{reviewID}` | PUT / DELETE | 编辑 / 删除 |
| `/api/v1/course/review/reviews/{reviewID}/votes` | POST | 投票 |
| `/api/v1/course/review/reviews/{reviewID}/reports` | POST | 举报 |
| `/api/v1/course/review/reviews/{reviewID}/replies` | POST | 回复 |
| `/api/v1/course/review/replies/{replyID}` | DELETE | 删除回复 |
| `/api/v1/course/review/content/check` | POST | 内容检查 |
| `/api/v1/course/review/courses/{courseID}/favorites` | GET / POST / DELETE | 收藏状态 / 收藏 / 取消收藏 |
| `/api/v1/course/review/drafts` | POST | 保存草稿 |
| `/api/v1/course/review/drafts/{courseID}` | GET / DELETE | 读取 / 删除草稿 |

## 用户中心

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/review/user/reviews` | GET | 我的评课 |
| `/api/v1/course/review/user/votes` | GET | 我的投票 |
| `/api/v1/course/review/user/favorites` | GET | 我的收藏 |
| `/api/v1/course/review/user/notifications` | GET | 通知列表 |
| `/api/v1/course/review/user/notifications/stream` | GET | SSE 实时推送 |
| `/api/v1/course/review/user/notifications/unread-count` | GET | 未读数 |
| `/api/v1/course/review/user/notifications/{notificationID}/read` | PUT | 标记已读 |
| `/api/v1/course/review/user/notifications/read-all` | PUT | 全部已读 |

## 评课后台

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/course/review/admin/reports` | GET | 举报列表 |
| `/api/v1/course/review/admin/reports/{reportID}` | PUT | 处理举报 |
| `/api/v1/course/review/admin/reviews` | GET | 评课列表 |
| `/api/v1/course/review/admin/reviews/{reviewID}` | PUT | 更新状态 |
| `/api/v1/course/review/admin/reviews/batch` | PATCH | 批量操作 |
| `/api/v1/course/review/admin/reviews/{reviewID}/edit` | POST | 编辑内容 |
| `/api/v1/course/review/admin/stats` | GET | 统计 |
| `/api/v1/course/review/admin/logs` | GET | 操作日志 |
| `/api/v1/course/review/admin/export` | GET | 导出 |
| `/api/v1/course/review/admin/teachers` | GET / POST | 教师管理 |
| `/api/v1/course/review/admin/teachers/{teacherID}` | PUT / DELETE | 单教师 |
| `/api/v1/course/review/admin/sensitive-words` | GET / POST | 敏感词 |
| `/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}` | PUT / DELETE | 单敏感词 |
| `/api/v1/course/review/admin/content-flags` | GET | 内容标记 |
| `/api/v1/course/review/admin/content-flags/{reviewID}/clear` | PUT | 清除标记 |

## 用户系统

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/user/identity` | GET / POST | 实名认证 |
| `/api/v1/user/identity/uploads` | POST | 上传证件照 |
| `/api/v1/user/profile` | GET | 学生认证档案 |
| `/api/v1/user/profile/verify` | POST | 发起学生认证 |
| `/api/v1/user/profile/bind-phone/otp` | POST | 请求绑定手机验证码 |
| `/api/v1/user/profile/bind-phone` | POST | 绑定手机号 |
| `/api/v1/user/profile/academic-info` | GET | 学籍信息 |
| `/api/v1/user/me` | GET | 用户聚合视图 |
| `/api/v1/user/schools` | GET | 学校列表 |

## 用户系统后台

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/admin/identities` | GET | 实名审核列表 |
| `/api/v1/admin/identities/{userID}` | PUT | 实名审核 |
| `/api/v1/admin/student-verifications` | GET | 学生审核列表 |
| `/api/v1/admin/student-verifications/{userID}` | PUT | 学生审核 |
| `/api/v1/admin/school-configs` | GET | 学校配置列表 |
| `/api/v1/admin/school-configs/{schoolID}` | PUT | 更新学校配置 |
| `/api/v1/admin/system-configs` | GET | 系统配置列表 |
| `/api/v1/admin/system-configs/{key}` | PUT | 更新系统配置 |

## 指标采集

| 路径 | 方法 | 说明 |
|------|------|------|
| `/api/v1/metrics/vitals` | POST | Web Vitals 指标上报 |
| `/api/v1/metrics/frontend-errors` | POST | 前端错误上报 |

## 认证说明

- Zitadel OIDC + HttpOnly Cookie + CSRF header
- 手机号登录仅授予 `user` 角色
- 后台入口由 capabilities 控制
