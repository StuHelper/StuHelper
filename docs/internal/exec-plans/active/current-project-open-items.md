---
type: internal
audience: maintainers
status: current
authoritative-source: this file
last-verified: 2026-05-18
---

# 当前项目待办

本文件是执行计划的唯一活跃入口。历史计划、已完成阶段和已废弃方案不再用未勾选
checkbox 表示当前待办。

## 已确认活跃任务

Open Platform v1 baseline 已落地；下一步应补齐生产运营面和用户可控面。完整目标见 [`docs/design/open-platform-v1.md`](../../../design/open-platform-v1.md)。

| 任务 | 范围 | 当前状态 | 完成标准 |
|------|------|----------|----------|
| Open Platform 用户授权管理 | `clients/web/src/modules/user/` + `server/internal/modules/openplatform/` | 当前已有 consent 授权写入，缺用户查看和撤销入口 | 用户可查看所有已授权应用，按 scope 或整应用撤销；撤销后 disclosure 立即返回 consent required |
| Open Platform 开发者门户 | `clients/web/src/modules/open-platform/` + `/api/v1/open-platform/apps` | 当前已有注册 API，缺应用列表、审核状态和密钥生命周期 UI | 开发者无需进入 Casdoor 控制台即可提交应用、查看状态、申请 redirect URI 变更和触发 secret rotation |
| Open Platform 管理审核 UI | `clients/admin/` + `/api/v1/admin/open-platform/*` | 当前已有审批 API，缺后台页面 | 管理员可审核 app / scope、暂停、吊销、查看敏感 scope 理由；全部动作落审计 |
| 第三方 token 最小化准入探针 | `server/internal/platform/casdoor/` + `infra/ops/` | 当前服务端生成第三方 OIDC URL 时只请求 `openid`，缺生产自动探针 | 每个第三方 app approved 前自动完成 OIDC code flow probe；token 出现业务 claim 时拒绝 approved |
| Disclosure 限流、指标和审计硬化 | `server/internal/modules/openplatform/` + metrics/audit | 当前有核心 consent/disclosure 流程，缺生产级可观测和限流闭环 | app、app+user、endpoint、consent 维度限流；Prometheus 标签低基数；审计失败 fail-closed |
| App secret 生命周期 | `server/internal/modules/openplatform/` + Web/Admin UI | 当前批准时返回一次 secret，缺轮换 / 吊销 UI | 支持 secret rotation、旧 secret 立即失效、泄漏处置记录与审计 |
| Open Platform 资源 API v1.1 | OpenFGA model + disclosure/resource APIs | v1 只做身份与认证事实披露 | app 到具体资源的 OpenFGA 关系清晰落地，且不与 scope consent 混用 |

## 待立项候选

下列内容来自历史计划的未完成描述，但没有被采纳为当前执行计划：

| 候选项 | 来源 | 当前状态 |
|--------|------|----------|
| Koishi 群管中心高阶运营工作流：更细粒度报表、历史版本、更复杂处置编排 | `docs/internal/exec-plans/archived/2026-04-19-koishi-moderation-center-implementation.md` | 待产品确认，不作为活跃任务 |

## 审查延后项

下列内容来自 2026-05-09 代码审查复核。它们已确认有改进价值，但需要独立
选型、schema 设计、部署策略或监控指标设计，不混入普通修复批次。

| 项 | 范围 | 当前状态 | 立项前置 |
|----|------|----------|----------|
| H8：uniappx refresh token 安全存储 | `clients/uniappx/src/api/native-session.ts` | refresh token 仍通过 `uni.setStorageSync` 写入普通本地存储；这是移动端持久凭据保护不足问题 | 选择并验证 iOS Keychain / Android EncryptedSharedPreferences / uni 原生安全存储插件；定义 native bridge 失败时的显式错误语义 |
| M12：outbox dead-letter 显式状态 | `server/internal/pkg/outbox/` + `server/migrations/` | 达到最大重试后仍写 `failed`，通过 100 年后的 `available_at` 作为事实死信标记 | 设计 `dead_letter` status migration、claim SQL、repository API、worker 终止逻辑和管理重放入口 |
| M13：audit 写入上下文 | `server/internal/pkg/audit/` | audit event 已保存 `trace_id` / `request_id`，但持久化用 `context.Background()`，DB span 不挂到请求 trace 树 | 设计 `LogContext(ctx, event)` 或等价 API，使用 `context.WithoutCancel(ctx)` 保留 trace baggage 且不被请求取消中断 |
| M15：生产内部 Postgres SSL | `docker-compose.yml` + 发布脚本 / runbook | 本地 compose 默认 `POSTGRES_INTERNAL_SSL_MODE=disable`，适合开发但不应作为生产默认假设 | 在生产 preflight / deploy checklist 中强制 `POSTGRES_INTERNAL_SSL_MODE=verify-ca`，开发环境显式允许 `disable` |
| L3：phone 日志脱敏 lint | `.gitlab-ci.yml` + 自定义 Semgrep 规则 | 当前 phone handler 手动 `maskPhone`，但 CI 未阻断未来直接记录原始手机号 | 建立项目自定义 Semgrep 规则目录，禁止 `zap.String("phone", raw)` 这类未脱敏日志 |
| L4：DB metrics table label | `server/internal/pkg/db/` + `server/internal/pkg/metrics/` | DB 指标有 table label，但当前调用统一传空串 | 先定指标 schema；推荐显式 `tableHint` / query metadata，不从 SQL 字符串猜表名 |
| L6：cache prefix metrics label | `server/internal/pkg/cache/` + `server/internal/pkg/metrics/` | cache hit/miss 只有 backend 维度，看不出业务 prefix | 先定低基数 prefix 枚举或 Helper namespace 方案，避免从 raw key 动态生成 Prometheus label |
| L12：MinIO read-only rootfs | `docker-compose.yml` | MinIO 没有启用 `read_only: true`，只有 `/data` volume | 真实启动验证 `read_only: true` + `tmpfs: ["/tmp"]`，按镜像实际写路径补 tmpfs 或 volume |

## 归档原则

- `docs/internal/exec-plans/active/` 只保留当前要推进的计划。
- 已完成计划进入 `docs/internal/exec-plans/completed/`。
- 被后续 ADR、设计或实现取代的计划进入 `docs/internal/exec-plans/archived/`
  或保留在 `docs/internal/design-snapshots/` 中并标记为历史快照。
- Runbook、QA checklist、发布检查表不是项目开发计划，不在本文件中跟踪。
