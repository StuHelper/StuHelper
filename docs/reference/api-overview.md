# API 概览

基础路径是 `/api/v1`。这里提供当前接口面的快速索引，详细字段以 `server/api/openapi.yaml` 为准。

## 健康检查

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/health/live` | GET | 存活检查 |
| `/health/ready` | GET | 就绪检查 |

## 认证

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/auth/login` | GET | 获取登录跳转地址 |
| `/api/v1/auth/signup` | GET | 获取注册跳转地址 |
| `/api/v1/auth/callback` | GET | OAuth 回调换取会话 |
| `/api/v1/auth/refresh` | POST | 刷新 access token |
| `/api/v1/auth/me` | GET | 获取当前用户、capabilities、后台可达性 |
| `/api/v1/auth/logout` | POST | 登出当前设备 |
| `/api/v1/auth/logout-all` | POST | 全设备登出 |

## 课程实体

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/course/departments` | GET | 院系列表 |
| `/api/v1/course/terms` | GET | 学期列表 |
| `/api/v1/course/categories` | GET | 课程分类 |
| `/api/v1/course/courses` | GET | 课程列表 |
| `/api/v1/course/courses/search` | GET | 搜索课程 |
| `/api/v1/course/courses/{courseID}` | GET | 课程详情 |
| `/api/v1/course/stats` | GET | 课程门户统计 |

## 评课社区公开接口

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/course/review/rating-dimensions` | GET | 评分维度配置 |
| `/api/v1/course/review/courses/{courseID}/rating-stats` | GET | 课程评分统计 |
| `/api/v1/course/review/courses/{courseID}/rating-trend` | GET | 课程评分趋势 |
| `/api/v1/course/review/courses/{courseID}/teachers` | GET | 课程教师列表 |
| `/api/v1/course/review/courses/{courseID}/reviews` | GET | 课程测评列表 |
| `/api/v1/course/review/reviews/latest` | GET | 最新测评 |
| `/api/v1/course/review/reviews/batch` | GET | 批量取课程测评 |
| `/api/v1/course/review/reviews/{reviewID}/replies` | GET | 回复列表 |
| `/api/v1/course/review/stats` | GET | 门户统计 |
| `/api/v1/course/review/rankings/hot` | GET | 热门课程排行 |
| `/api/v1/course/review/teachers/{teacherID}/stats` | GET | 教师评分统计 |

## 评课社区认证接口

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/course/review/reviews` | POST | 发布测评 |
| `/api/v1/course/review/reviews/{reviewID}` | PUT | 编辑测评 |
| `/api/v1/course/review/reviews/{reviewID}` | DELETE | 删除测评 |
| `/api/v1/course/review/reviews/{reviewID}/votes` | POST | 点赞或踩 |
| `/api/v1/course/review/reviews/{reviewID}/reports` | POST | 举报测评 |
| `/api/v1/course/review/reviews/{reviewID}/replies` | POST | 发表回复 |
| `/api/v1/course/review/replies/{replyID}` | DELETE | 删除回复 |
| `/api/v1/course/review/content/check` | POST | 内容检查 |
| `/api/v1/course/review/courses/{courseID}/favorites` | POST | 收藏课程 |
| `/api/v1/course/review/courses/{courseID}/favorites` | DELETE | 取消收藏 |
| `/api/v1/course/review/drafts` | POST | 保存草稿 |
| `/api/v1/course/review/drafts/{courseID}` | GET | 获取草稿 |
| `/api/v1/course/review/drafts/{courseID}` | DELETE | 删除草稿 |

## 评课用户中心

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/course/review/user/reviews` | GET | 我的测评 |
| `/api/v1/course/review/user/votes` | GET | 我的投票 |
| `/api/v1/course/review/user/favorites` | GET | 我的收藏 |
| `/api/v1/course/review/user/notifications` | GET | 通知列表 |
| `/api/v1/course/review/user/notifications/unread-count` | GET | 未读数 |
| `/api/v1/course/review/user/notifications/{notificationID}/read` | PUT | 标记已读 |
| `/api/v1/course/review/user/notifications/read-all` | PUT | 全部已读 |

## 评课后台

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/course/review/admin/reports` | GET | 举报列表 |
| `/api/v1/course/review/admin/reports/{reportID}` | PUT | 处理举报 |
| `/api/v1/course/review/admin/reviews` | GET | 后台测评列表 |
| `/api/v1/course/review/admin/reviews/{reviewID}` | PUT | 更新测评状态 |
| `/api/v1/course/review/admin/reviews/batch` | POST | 批量更新测评 |
| `/api/v1/course/review/admin/reviews/{reviewID}/edit` | POST | 后台编辑测评内容 |
| `/api/v1/course/review/admin/stats` | GET | 后台统计 |
| `/api/v1/course/review/admin/logs` | GET | 操作日志 |
| `/api/v1/course/review/admin/export` | GET | 导出测评 |
| `/api/v1/course/review/admin/teachers` | GET/POST | 教师管理 |
| `/api/v1/course/review/admin/teachers/{teacherID}` | PUT/DELETE | 单个教师管理 |
| `/api/v1/course/review/admin/sensitive-words` | GET/POST | 敏感词管理 |
| `/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}` | PUT/DELETE | 单个敏感词管理 |

## 用户系统

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/user/identity` | GET/POST | 实名认证状态与提交 |
| `/api/v1/user/profile` | GET | 学生认证档案 |
| `/api/v1/user/profile/verify` | POST | 发起学生认证 |
| `/api/v1/user/profile/bind-phone` | POST | 绑定手机号 |
| `/api/v1/user/profile/academic-info` | GET | 学籍信息 |
| `/api/v1/user/schools` | GET | 学校列表 |

## 用户系统后台

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/admin/identities` | GET | 实名认证审核列表 |
| `/api/v1/admin/identities/{userID}` | PUT | 实名认证审核 |
| `/api/v1/admin/student-verifications` | GET | 学生认证审核列表 |
| `/api/v1/admin/student-verifications/{userID}` | PUT | 学生认证审核 |
| `/api/v1/admin/school-configs` | GET | 学校配置列表 |
| `/api/v1/admin/school-configs/{schoolID}` | PUT | 更新学校配置 |
| `/api/v1/admin/system-configs` | GET | 系统配置列表 |
| `/api/v1/admin/system-configs/{key}` | PUT | 更新系统配置 |

## RBAC 后台

| 路径 | 方法 | 说明 |
| --- | --- | --- |
| `/api/v1/admin/roles` | GET/POST | 角色列表与创建 |
| `/api/v1/admin/roles/{roleID}` | PUT/DELETE | 单个角色维护 |
| `/api/v1/admin/roles/{roleID}/permissions` | GET/PUT | 角色权限查看与设置 |
| `/api/v1/admin/permissions` | GET | 权限列表 |
| `/api/v1/admin/users/{userID}/roles` | GET/PUT | 用户角色 |
| `/api/v1/admin/users/{userID}/permissions` | GET/PUT | 用户个人权限覆盖 |
| `/api/v1/admin/groups` | GET/POST | 用户组列表与创建 |
| `/api/v1/admin/groups/{groupID}` | PUT/DELETE | 单个用户组维护 |
| `/api/v1/admin/groups/{groupID}/members` | GET/PUT | 用户组成员 |
| `/api/v1/admin/groups/{groupID}/permissions` | PUT | 用户组权限 |

## 当前认证与权限语义

- 认证使用 Cookie 会话和 CSRF 头
- 后台接口读取应用 capability
- `isPlatformAdmin` 保持平台管理员字段语义
