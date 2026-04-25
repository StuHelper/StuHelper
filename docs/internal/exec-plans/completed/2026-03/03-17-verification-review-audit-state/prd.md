---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# 修复学生认证复审状态与审计信息

## Goal

修复学生认证管理审核链路里三类缺口。驳回后理由丢失，已通过记录被驳回时保留旧 `verifiedAt`，以及前后端契约未完整表达审核信息。目标是在不扩散改动范围的前提下，让当前复审链路做到状态自洽、审核信息可回读、接口契约一致。

## What I already know

- 现有 `ReviewStudentVerification` 在通过时设置 `verifiedAt`，驳回时只改 `verificationStatus`，不会清理旧 `verifiedAt`。
- `user_profiles` 已有 `rejection_reason` 和 `reviewed_at` 列，但当前仓储和模型还没有把它们真正接进读写链路。
- 管理端学生认证列表响应里没有 `rejectionReason` 字段，前端也没有展示学生认证驳回理由。
- 实名认证（`user_identities`）已有 `rejection_reason`，学生认证与实名认证审核信息建模不一致。
- 本任务可改文件范围已经限定，且存在并行 worker，测试要避免改共享大测试文件。

## Assumptions (temporary)

- 本轮不新增新表，只把现有 `user_profiles.rejection_reason` 和 `user_profiles.reviewed_at` 正式接入。
- 本轮以状态自洽和契约完整为 MVP，统一独立审计表属于后续演进。

## Open Questions

- 无阻塞问题。当前输入足够收敛实现。

## Requirements

- 学生认证审核驳回时必须清空 `verifiedAt`，避免 rejected + verifiedAt 并存。
- 学生认证审核通过时必须清空旧驳回原因，保证状态语义单一。
- 学生认证审核要持久化审核元数据，至少包括驳回理由和审核时间。
- 管理端学生认证列表响应补齐审核元数据字段，前端可直接展示。
- 保持 `manualFormData` 只承载用户提交的业务表单字段，不夹带审核内部元数据。

## Acceptance Criteria

- [ ] 驳回流程会把 `verificationStatus` 设为 `rejected` 且 `verifiedAt` 置空。
- [ ] 通过流程会把 `verificationStatus` 设为 `verified` 且 `rejectionReason` 置空。
- [ ] 学生认证审核后重新查询，`rejectionReason` 与 `reviewedAt` 可回读。
- [ ] 管理端列表接口与 OpenAPI 契约包含 `rejectionReason` 与 `reviewedAt`。
- [ ] 管理端页面可展示驳回原因与审核时间。
- [ ] 新增独立测试文件覆盖关键审核状态切换和序列化行为。

## Definition of Done (team quality bar)

- 测试新增并通过（仅覆盖本任务相关模块）。
- Go 代码和前端代码通过格式与类型检查（在本任务改动范围内）。
- OpenAPI 文档与后端序列化字段一致。

## Out of Scope (explicit)

- 不在本轮引入新数据库表或迁移脚本。
- 不在本轮实现自动通过策略、审核人 ID 追踪、完整审计流水历史。
- 不在本轮重构实名认证审核链路。

## Research Notes

### What similar systems do

- 常见企业认证审核会区分「提交材料」与「审核元数据」，避免审核字段覆盖用户提交字段。
- 审核状态机通常保证互斥，`rejected` 不应保留 `verifiedAt`，`verified` 不应保留驳回理由。
- 审核结果列表会直接回传 `rejectionReason`，避免前端从杂合字段二次解析。

### Constraints from our repo/project

- `user_profiles` 已有审核字段，但当前代码没有把它们接成统一事实源。
- 本任务文件改动范围受限，不扩展到新表或迁移脚本。
- 并行开发场景下需要减少冲突，测试应独立新建。

### Feasible approaches here

**Approach A: 直接接入现有审核列（Recommended）**

- How it works:
  - `Profile`、repository、service、JSON 序列化、OpenAPI 和前端统一接入 `rejection_reason` 与 `reviewed_at`。
- Pros:
  - 利用现有 schema，语义最直接。
  - 不污染 `manual_form_data`。
  - 上下游契约清晰，后续扩展审计表也有明确边界。
- Cons:
  - 需要同步更新更多层的字段映射。

**Approach B: 只修状态，不持久化驳回理由**

- How it works:
  - 仅修复 `verifiedAt` 清理与状态切换。
- Pros:
  - 改动最小。
- Cons:
  - 信息仍丢失，不满足任务目标。

**Approach C: 立即引入独立审计表**

- How it works:
  - 新增审计表与 repository API。
- Pros:
  - 结构长期最优。
- Cons:
  - 超出本次改动边界，且会与并行任务冲突。

## Technical Approach

采用 Approach A。`Profile` 结构补充审核元数据字段，`ReviewStudentVerification` 在状态切换时统一维护 `verifiedAt`、`rejectionReason`、`reviewedAt`。仓储层直接读写 `user_profiles.rejection_reason` 与 `user_profiles.reviewed_at`。序列化层与 OpenAPI 同步增加学生认证审核返回字段。前端审核页面补充驳回原因与审核时间展示。

## Decision (ADR-lite)

**Context**
学生认证审核已经有数据库列，但代码链路没有把它们真正接起来，本轮需要把丢失信息和状态不一致问题一次修正。

**Decision**
直接接入 `user_profiles.rejection_reason` 与 `user_profiles.reviewed_at`，同时在服务层落地状态机一致性规则。

**Consequences**
本轮可快速闭环并保持对外契约清晰。后续若引入独立审计表，可在现有字段事实源之上继续扩展，不需要先清理临时存储结构。

## Technical Notes

- 主要改动文件：
  - `server/internal/modules/user/service_admin.go`
  - `server/internal/modules/user/repository_profile.go`
  - `server/internal/modules/user/models.go`
  - `server/internal/modules/user/handler_helpers.go`
  - `server/api/components/schemas/user-system.yaml`
  - `server/api/paths/admin-user-system.yaml`
  - `clients/admin/src/views/user-system/StudentVerificationReview.vue`
- 新增独立测试文件：
  - `server/internal/modules/user/service_admin_review_test.go`
  - `server/internal/modules/user/handler_admin_review_test.go`
