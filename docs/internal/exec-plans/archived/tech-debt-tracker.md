---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# 技术债追踪

> 状态：已归档。2026-04-19 复核时已无活跃技术债，本文件仅保留历史记录。

> 集中记录已知技术债。每项标注优先级和预计解决时机。

## 活跃技术债

| ID | 事项 | 优先级 | 预计解决时机 |
|----|------|--------|-------------|
| — | 当前无活跃技术债 | — | — |

## 已解决

| ID | 事项 | 解决方式 | 解决时间 |
|----|------|----------|----------|
| TD-08 | notification 模块双轨残留 | 通知 CRUD、SSE、Redis Pub/Sub 与路由注册已统一收口到 `server/internal/modules/notification/`；仅保留 `/api/v1/course/review/user/notifications/*` 作为对外命名空间，不再由 `review` handler 旁路提供通知读写路径 | 2026-04-19 |
| TD-07 | `clients/admin/apps/web-tdesign/` 和 `backend-mock/` 未清理变体 | 本次清理中删除 | 2026-04-16 |
| TD-06 | `clients/web/src/modules/course/courseCatalogLoader.ts` 死代码 | 本次清理中删除 | 2026-04-16 |
| TD-05 | 实名自动匹配的多校化 | 删除默认学籍表，强制要求学校上下文；学籍查询方法仅接受指定表名 | 2026-03-31 |
| TD-04 | 认证方式与审批策略解耦 | `school_configs` 新增 `approval_policy` 字段（auto/manual），认证结果根据策略决定状态 | 2026-03-31 |
| TD-03 | 应用级通知中心 | 独立 `notification` 模块，`user_id` 归属键，SSE 实时推送 + Redis Pub/Sub | 2026-03-31 |
| TD-02 | 评课内容审核流水线 | `reviews` 新增 `content_flag` 字段，`warn` 持久化 + 管理员复核清除链路 | 2026-03-31 |
| TD-01 | RBAC 跨请求权限缓存 | 角色改由 OIDC Token 下发，应用侧静态展开 capabilities，中间件不再按请求查库 | 2026-03 IAM 核心链路完成 |
| — | RBAC 学校范围鉴权 | 改为默认拒绝 | 2026-03 feature/fix-rbac-and-verification-issues |
| — | RBAC 管理写接口缺业务校验 | 补齐校验、错误码和 4xx 归类 | 2026-03 同上 |
| — | 学生认证未接学校配置 | 改为学校配置驱动，fail-closed | 2026-03 同上 |
| — | 学校统一身份认证配置假数据 | 改成真实配置 | 2026-03 同上 |
| — | 学生认证复审缺审核记录 | 补齐 `rejection_reason` 和 `reviewed_at` | 2026-03 同上 |
