# StuHelper 全审计合并报告（2026-04-16，重建版）

**来源文档**：
1. `doc1` `/Users/zxy/Code/StuHelper/docs/exec-plans/active/full-codebase-audit-2026-04-16.md`
2. `doc2` 历史标签：`2026-04-16-full-parallel-agents-audit.md`
3. `doc3` 历史标签：`2026-04-16-client-front-audit-round2.md`
4. `doc4` 历史标签：`2026-04-16-full-parallel-maintainability-audit-round3.md`
5. `doc5` 历史标签：`2026-04-16-backend-quality-audit-round4.md`
6. `doc6` 历史标签：`2026-04-16-fallback-duplication-audit-round4.md`
7. `doc7` 历史标签：`2026-04-16-api-boundary-doc-structure-audit-round4.md`
8. `doc8` 历史标签：`2026-04-16-all-agents-audit-consolidated-round4.md`（只是 `doc2-7` 的原始拼接件，不计入独立命中）

> 说明：`docs/reviews/*.md` 原始并行审计稿已在 2026-04-17 完成复核并删除；未完成项统一保留在本文件，已完成/已核销项统一迁移到 `docs/exec-plans/completed/2026-04/2026-04-17-audit-closed-items.md`。`doc2-doc8` 仅作为历史来源标签保留，具体原文可通过 Git 历史追溯。

**合并规则**：命中次数按独立源文档数计算；严重级别直接继承源标注并取最高级；所有来源统一写成 `docN 原始ID@行号`。经 claude + codex 双轮复核后，部分条目的级别、问题描述或修复方向已基于代码事实修订。

## 汇总矩阵

| 领域 | CRITICAL | HIGH | MEDIUM | LOW | 合计 |
|------|----------|------|--------|-----|------|
| 一、认证 / 会话安全 | 0 | 0 | 0 | 0 | 0 |
| 二、授权 / 权限模型 | 0 | 0 | 0 | 0 | 0 |
| 三、OpenAPI 契约与代码漂移 | 0 | 0 | 0 | 0 | 0 |
| 四、前端架构与边界 | 0 | 0 | 0 | 0 | 0 |
| 五、前端代码质量 / DRY | 0 | 0 | 0 | 0 | 0 |
| 六、后端架构与分层 | 0 | 0 | 0 | 0 | 0 |
| 七、后端代码质量 / DRY | 0 | 0 | 0 | 0 | 0 |
| 八、死代码与清理 | 0 | 0 | 0 | 0 | 0 |
| 九、测试覆盖与质量 | 0 | 0 | 1 | 0 | 1 |
| 十、基础设施 | 0 | 0 | 0 | 0 | 0 |
| 十一、文档一致性 | 0 | 0 | 0 | 0 | 0 |
| 十二、可观测性与错误处理 | 0 | 0 | 0 | 0 | 0 |
| **Total** | **0** | **0** | **1** | **0** | **1** |

> 注：已完成/已核销条目已迁移到 `docs/exec-plans/completed/2026-04/2026-04-17-audit-closed-items.md`；历史复核与迁移计划已移出 active。


---

## 一、认证 / 会话安全

当前无活动项。

## 二、授权 / 权限模型

当前无活跃高优先级问题；组织 scope 问题已迁移至 completed。

## 三、OpenAPI 契约与代码漂移

当前无活动项（`server/internal/api/gen/` 的职责已明确为 embedded spec + request validation + drift gate；handler 保留局部 DTO 作为有意的分层边界）。

## 四、前端架构与边界

## 五、前端代码质量 / DRY

## 六、后端架构与分层

当前无活动项（`server/internal/pkg/response/mapped_error.go`、`server/internal/modules/user/http_errors.go`、`server/internal/modules/course/review/http_errors.go` 已作为集中域错误映射层落地，user/review handler 不再各自手写一套 `switch errors.Is(...)` 到 HTTP/code/message 的重复逻辑）。


## 七、后端代码质量 / DRY

当前无活动项。

## 八、死代码与清理

当前无活动项（`clients/web/src/vendor/` 不存在，`clients/web/storybook-static/` 已清理且根 `.gitignore` 已忽略该目录）。

## 九、测试覆盖与质量

### [MEDIUM] 关键覆盖率仍未全部达到 80% 目标
**命中次数**: 1（来源: doc1 GO-H6@81）
**关键文件**: `internal/modules/course/review`（73.7%）, `modules/auth`（75.2%）, `pkg/middleware`（77.4%）, `course`（81.7%）, `pkg/oidc`（83.7%）, `pkg/fga`（83.3%）
**问题**: 覆盖率已从“大面积裸奔”收敛到“仅剩少数关键包未过线”：`course`、`pkg/oidc`、`pkg/fga` 已稳定达到 80%+，`pkg/middleware` 逼近目标线，剩余短板主要集中在 `review` 与 `auth` 的业务主链路和异常分支。
**修复方向**: 继续按收益优先补齐：1) review 管理/举报/FGA sync/缓存失效等剩余分支；2) auth refresh / revoke / OIDC callback 剩余失败分支；3) 把新增用例纳入稳定 helper，避免后续功能迭代再把关键包覆盖率拉回 70% 以下。

## 十、基础设施

当前无活动项。

## 十一、文档一致性

当前无活动项（`docs/README.md`、`docs/PRODUCT.md`、`docs/product-specs/index.md`、`docs/FRONTEND.md`、`docs/design-docs/frontend-architecture.md` 已收敛为导航/边界说明；人工 API 清单仅保留在 `docs/references/api-overview.md`）。

## 十二、可观测性与错误处理

当前无活动项（前端源码中的裸 `catch {}` 已清零，`clients/scripts/check-no-empty-catch.sh` 与 `clients/eslint.config.mjs` 已作为回归门禁）。
