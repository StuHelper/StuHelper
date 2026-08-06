---
type: product-spec
audience: product, backend-dev, frontend-dev
status: current
authoritative-source: this file (index only) + listed specs
last-verified: 2026-08-05
---

# 产品规格索引

本目录只放**业务域规格**：`current` 文档描述当前代码库已实现的功能范围、业务规则和边界；
`draft` 文档描述已经进入评审、但尚未完成实现迁移的目标规格。草案不得替代 OpenAPI、migration、
源码和测试所表达的当前事实。

- 角色与产品形态 → [design/product-overview.md](../design/product-overview.md)
- 认证 / 授权 / 存储 / 安全 等**技术机制** → [design/](../design/)
- API 字段与 schema → `server/api/openapi.yaml`

## 业务域

| 域 | 规格 | 说明 |
|----|------|------|
| 课程与评课 | [course-review.md](course-review.md) | 课程实体、评课、回复、举报、收藏 |
| 教务展示 | [academics-data-integration.md](academics-data-integration.md) | 外部教务数据导入、标准化、我的课程、我的课表 |
| 资源共享 | [resource-sharing.md](resource-sharing.md) | 资源条目、版本、标签、绑定、下载 |
| 用户系统 | [user-system.md](user-system.md) | 账号聚合面、Casdoor 手机号边界与 QQ 绑定 |
| 学生认证与群聊准入 | [student-verification-and-group-admission.md](student-verification-and-group-admission.md) | 独立学生认证、学校适配、本地学籍快照、人工审核，以及 Koishi/QQ 作为消费方的解耦边界 |
| 通知 | [notification.md](notification.md) | 通知列表、未读数、SSE |
| 审计 | [audit-logging.md](audit-logging.md) | 审计日志与留痕 |

## 当前边界

- StuHelper 是**校园信息平台**，不是完整教务系统。
- 教务相关能力只做导入、标准化、展示与查询，不做写侧。
- 资源相关能力只做资源共享与可插拔存储，不做实验 / 作业附件平台。

更完整的目标范围和非目标见 [design/target-scope.md](../design/target-scope.md)。
