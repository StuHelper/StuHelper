---
type: adr
audience: maintainers, ops
status: current
authoritative-source: .github/workflows/ci.yml + .github/workflows/deploy.yml + infra/ops/verify-github-release.sh + infra/ops/remote-ci-release.sh
last-verified: 2026-08-03
---

# ADR-0011：暂缓 staging 时的直接生产晋级与短期 registry 凭据

**Date**: 2026-08-03
**Status**: accepted
**Deciders**: 项目 owner

## Context

ADR-0010 把同 SHA staging success 设为 production 的默认前置条件。项目 owner 现阶段决定暂缓建设
独立 staging，但仍要尽快完善自动部署生产。当前系统包含 PostgreSQL migration、Casdoor、OpenFGA、
对象存储、备份与单主机 Docker Compose；因此“跳过 staging”不能等价为合并后无人值守改写生产，
也不能绕过制品来源、生产审批、备份、预检或 smoke。

生产主机还需要从私有 GHCR 拉取不可变镜像。把个人 PAT 或长期 `read:packages` token 固化在主机、
仓库或命令行上会扩大泄露窗口；GitHub Actions 每次作业已经具备只在该次运行有效的
`github.token`，可作为更小生命周期的拉取凭据。

## Decision

1. Forward Deploy 使用显式的 promotion mode，不再用一个布尔“跳过 staging”同时表达常规策略和
   事故操作：

   - `staging`：只允许部署 staging；
   - `after-staging`：production 必须等待同一 SHA staging success；
   - `direct`：在 staging 暂缓期间采用的常规 production 路径；
   - `break-glass`：事故处置路径，语义和审计记录与常规 `direct` 分开。

2. 自动直接晋级只由 `main` 的可信 push 触发。只有 `CI / Required`、Go 与
   JavaScript/TypeScript CodeQL、三个镜像的发布和 provenance/digest 校验全部成功，提交仍是实时
   `main` head，并且仓库变量同时满足
   `PRODUCTION_AUTO_PROMOTION_ENABLED=true` 与 `PRODUCTION_PROMOTION_MODE=direct`，才创建
   production deployment。
3. `direct` 不要求 staging success，但必须填写至少 24 个字符的变更上下文。它不绕过：当前分支头、
   必需 checks、不可变 digest、provenance、受保护 production environment、`Xauryan` 人工审批、
   审批后的再次校验、远端预检、迁移前备份、业务 smoke、严格可观测性 smoke 和人工受控回滚。
4. `break-glass` 仅供手工事故操作，至少填写 24 个字符的事故上下文；不得把自动 promotion mode
   长期设置为 `break-glass`。常规直接发布必须明确使用 `direct`，避免把长期架构选择伪装成事故豁免。
5. production 继续采用单 reviewer 治理：唯一 reviewer 为 `Xauryan`，允许自批；不创建第二个占位
   管理员。自动化只负责在所有前置门禁通过后创建等待审批的 deployment，不自动代替业务发布判断。
6. GitHub Actions 通过 SSH 标准输入把该次 job 的短期 `github.token` 交给
   `remote-ci-release.sh`。远端只在权限为 `0700` 的临时 `DOCKER_CONFIG` 中登录 `ghcr.io`，完成
   deploy 或 rollback 后无论成功失败都删除该目录。token 不出现在 SSH 命令参数、远端配置、日志、
   release evidence 或长期 secret backend 中。远端配置必须使用
   `REGISTRY_AUTH_MODE=workflow-token`；`persistent-secret` 仅保留给明确管理的非 GitHub 兼容链路。
7. 根 Compose 文件显式使用 `name: ${STACK_NAME:-stuhelper}`。生产配置固定
   `STACK_NAME=stuhelper-prod`，使新的 `/opt/stuhelper` 发布控制面在经过资源映射核验后接管现有
   `stuhelper-prod` Compose project，而不是误建第二套 network/volume/container。
8. `PRODUCTION_AUTO_PROMOTION_ENABLED` 默认保持 `false`。只有专用 deploy 用户/SSH key、固定
   known_hosts、`/opt/stuhelper` 控制面、Vault 最小权限 periodic runtime token 与自动续期 timer、
   备份 timers、异机备份取回、Compose project 接管 dry-run、GHCR 拉取、真实预检和一次人工批准
   发布演练全部通过后才启用。部署 token 不得复用 Vault 初始化 root token；流水线代码完成不等于
   现网已具备自动部署能力。
9. staging 是延期而非永久删除。未来独立 staging 就绪后，把仓库变量切换为
   `PRODUCTION_PROMOTION_MODE=after-staging` 并启用 `STAGING_AUTO_DEPLOY_ENABLED`，恢复同制品逐环境
   晋级，无需重新设计构建、签名或回滚链路。

## Relationship to ADR-0010

本 ADR 部分取代 ADR-0010 的决策 3、6、10：production 不再永久强制 staging，也不把有意采用的
直接晋级称作 break-glass；启用顺序允许先完成 production。ADR-0010 的其余决策保持有效，尤其是
构建一次、不可变 digest、生产审批、当前可信 controller、审批后复验、禁止自动 schema 回滚和历史
镜像受控回滚。

## Consequences

### Positive

- 没有 staging 基础设施时仍可形成可审计、可批准、可回滚的生产交付闭环。
- 常规直接晋级和事故 break-glass 具有不同语义，报告、告警和复盘不会混淆。
- 生产主机不必保存个人 PAT 或长期 GHCR pull token；单次作业凭据在使用后自动消失。
- 未来补建 staging 只需切换 promotion mode，不改变已经验收的制品供应链。

### Negative

- production 成为首次真实环境验收点；配置、迁移和外部依赖缺陷的爆炸半径更大。
- 每次正常发布仍需 `Xauryan` 人工批准，不是完全无人值守 continuous deployment。
- GitHub Actions 到生产 SSH 仍是高权限交付边界；专用 deploy 账号加入 Docker group 时近似宿主 root。

### Risk controls

- 在启用自动 promotion 前完成一次只读资源映射核验和受控人工演练；不得直接在现有宝塔控制面上
  重建容器试错。
- 数据库变更坚持 expand/contract；smoke 失败时保留现场，不自动回滚 schema。
- 每次审批必须查看候选 SHA、镜像 digest、变更原因和预验证结果；等待期间 `main` 前移会让候选
  fail-closed。
- 将“建设隔离 staging”保留在交付路线图中；生产事故或发布失败应推动该事项优先级上升。
