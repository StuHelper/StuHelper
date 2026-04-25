---
type: internal
audience: maintainers, product
status: snapshot
authoritative-source: this file (a point-in-time assessment)
last-verified: 2026-04-19
---

# Release Readiness（阶段性评估）

> 本文档是**某个时间点的主观评分**，不是长期规范。它记录"到今天为止，各模块能不能进生产"的判断。过期后不要依赖这里的评级，直接去看代码、测试和 `docs/product-specs/` 的当前规格。

## 评分标准

| 等级 | 含义 |
|------|------|
| A | 代码、测试、自动化基本完整，可进入生产验收 |
| B | 核心流程可用，有少量待完善项 |
| C | 可运行，但仍在开发中 |
| D | 原型阶段 |

## 产品域

| 域 | 代码 | 测试 | 综合 | 备注 |
|----|------|------|------|------|
| 认证与会话 | A- | B | A- | OIDC / Cookie / CSRF / 手机号登录均已完成 |
| 课程与评课 | B+ | B | B+ | 核心流程完整，搜索和运营细节有优化空间 |
| 用户系统 | B+ | B- | B+ | 实名/学生认证核心流程完整，证件照存储在对象存储 |
| 授权 | B+ | B- | B+ | capability + OpenFGA 核心链路清晰 |
| 通知 | B- | C+ | B- | CRUD、SSE 与 Redis Pub/Sub 已统一收口到 `notification` 模块，当前主要剩体验层完善 |
| 观测与部署 | A- | B | A- | 一键启动、GitLab 发布、LGTM 栈、回滚均已接入 |

## 工程层

| 层 | 质量 | 备注 |
|----|------|------|
| OpenAPI 契约 | A | 契约驱动，生成链路完整 |
| 后端分层 | B+ | Handler / Service / Repository 边界清晰 |
| 前端工程 | B+ | Web / Admin / Shared 分层明确 |
| 测试体系 | B | 单测 + 类型检查 + Playwright 已接入，后端关键包覆盖率门禁已落地并通过 |
| 部署自动化 | A- | dev / obs / prod 一键化，GitLab + rollback + Ansible，production 改为手工审批发布 |
| 观测 | A- | Prometheus / Loki / Tempo / Grafana / Alertmanager 完整 |

## 主要差距

| 项 | 优先级 | 说明 |
|----|--------|------|
| 证件照后端中转上传 | 中 | 已切对象存储，可进一步优化为浏览器直传 |
| 真实教务连接器 / 第三方网盘驱动 | 中 | 真实学校系统与第三方网盘仍缺外部上下文，当前仅完成扩展骨架 |
| staging / production 实战不足 | 中 | 脚本齐全，需要完成完整联调 |
| 管理后台页面补齐 | 中 | API 已齐，页面体验可继续完善 |

## 结论

- 开发环境：开箱即用
- 本地生产演练：可完成验证
- 长期运营：需继续完成真实发布演练与外部连接器落地

---

*更新：2026-04-19，按当前代码库重新评分*
