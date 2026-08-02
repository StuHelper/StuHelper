---
type: internal
audience: maintainers
status: snapshot
authoritative-source: GitHub API and protected workflow observations on 2026-08-01
last-verified: 2026-08-01
---

# GitHub 交付就绪快照（2026-08-01）

本文件只保存 2026-08-01 的时点证据，不是 GitHub 当前状态真源。分支、ruleset、environment、
secrets、packages 和安全开关在使用前必须重新查询。

| 项目 | 当时状态 | 2026-08-01 观察结果 |
|------|----------|---------------------|
| 仓库与分支 | 已验证 | `StuHelper/StuHelper` 为 public，默认分支为 `main`；人类长期分支只有 `main` / `develop`，二者受同一 ruleset 保护并保持相同提交。Dependabot 临时 head branch 不属于长期分支 |
| 合并门禁 | 已验证 | 默认要求 PR、1 个 approval、CODEOWNERS、撤销旧审批、最后推送者之外批准、解决 review thread、线性历史和 squash merge；Required 与两种 CodeQL 为必需检查；`Xauryan` 具有 `pull_request`-only bypass |
| Actions 供应链 | 已验证 | Actions 使用 selected-actions 策略，外部 action 固定完整 commit SHA，默认 `GITHUB_TOKEN` 只读且不能批准 PR |
| Code security | 已验证 | Secret scanning、push protection、Dependabot alerts/security updates 和 private vulnerability reporting 已启用；当时 CodeQL、Dependabot、secret-scanning alert 均为 0 |
| Environments | 部分验证 | `staging` 与 `production` 分支策略存在；production 当时只有一名 reviewer 且允许自批，不构成双人复核 |
| 部署凭据 | 未就绪 | 两个 environment 的 secrets 和 variables、仓库级 Actions secrets/variables 当时均为空 |
| GHCR | 部分验证 | 三个 container package 已由受保护分支发布 full-SHA immutable tag、branch alias 和 provenance；当时 visibility 为 private，目标主机最小读取凭据尚未验收 |
| 真实部署与回滚 | 未验证 | 当时未完成 staging/production 部署或回滚演练 |

该快照只能证明当时的代码治理和镜像发布控制面状态，不能替代后续真实部署、回滚或当前仓库
设置核验。
