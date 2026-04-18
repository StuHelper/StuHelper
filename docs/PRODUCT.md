# 产品概览

StuHelper（航小伴）是面向校园的信息与社区平台。

## 产品形态

| 形态 | 面向对象 | 说明 |
|------|----------|------|
| 主站 | 学生 / 普通用户 | 课程、教师、评课、搜索、个人中心、通知 |
| 管理后台 | 学校管理员 / 平台管理员 | 审核、举报处理、学校配置、系统配置 |
| 统一登录 | 全部用户 | Zitadel OIDC + 手机号 OTP |

## 业务域

| 域 | 内容 |
|----|------|
| 课程实体 | 院系、课程分类、学期、课程详情、教师统计 |
| 评课社区 | 评课、回复、投票、举报、草稿、收藏、搜索 |
| 用户系统 | 实名认证、学生认证、手机号绑定、学校配置 |
| 通知中心 | 通知列表、未读数、已读、SSE 推送 |
| 管理运营 | 举报处理、评课审核、内容标记、教师维护、敏感词 |

## 用户角色

| 角色 | 典型场景 |
|------|----------|
| 游客 | 浏览课程、教师、公开评课预览 |
| 登录用户 | 查看更多内容，管理个人资料 |
| 已认证学生 | 查看完整评课、发布评课 |
| 学校管理员 / 志愿者 | 审核内容、处理举报、维护学校配置 |
| 平台管理员 | 平台级配置与运营 |

更细的 capability、组织 scope、资源级权限模型请看：
- `/Users/zxy/Code/StuHelper/docs/product-specs/rbac-authorization.md`
- `/Users/zxy/Code/StuHelper/docs/references/api-overview.md`

技术栈、部署与运行细节不在本页重复维护：
- 技术栈总览：`/Users/zxy/Code/StuHelper/AGENTS.md`
- 前端边界：`/Users/zxy/Code/StuHelper/docs/FRONTEND.md`
- API 路由索引：`/Users/zxy/Code/StuHelper/docs/references/api-overview.md`
