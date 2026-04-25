---
type: design
audience: all
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# 产品概览

StuHelper（航小伴）是面向校园的**信息与社区平台**。

## 这是什么

- 一个让学生能看课、评课、讨论、分享资源的平台。
- 一个让学校管理员能审核、治理内容的后台。
- 一个与主平台协同的 QQ 机器人子系统，用于入群认证联动与群治理。
- 统一登录走 Zitadel OIDC；手机号 OTP 作为补充登录。

## 给谁用

| 角色 | 典型场景 |
|------|----------|
| 游客 | 浏览课程、教师、公开评课预览 |
| 登录用户 | 查看更多内容，管理个人资料 |
| 已认证学生 | 查看完整评课、发布评课 |
| 学校管理员 / 志愿者 | 审核内容、处理举报、维护学校配置 |
| 平台管理员 | 平台级配置与运营 |

## 不做什么

- 完整教务写侧（选课 / 退课 / 排课 / 调课 / 成绩）
- 实验 / 作业 / 提交 / 批改 / 评分系统
- 面向实验或作业的通用附件平台

## 继续阅读

- 业务域详细规格 → [product-specs/index.md](../product-specs/index.md)
- 主路线目标范围与模块边界 → [target-scope.md](target-scope.md)
- 认证 / 授权 / 会话机制 → [auth-and-session.md](auth-and-session.md)、[authorization-model.md](authorization-model.md)
