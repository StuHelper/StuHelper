---
type: adr
audience: maintainers, backend-dev, frontend-dev, ops
status: current
authoritative-source: .github/workflows/ci.yml + .github/workflows/publish-images.yml + .github/workflows/deploy.yml + .github/workflows/rollback.yml + infra/ops/verify-github-release.sh
last-verified: 2026-08-02
---

# ADR-0010: 构建一次、同制品晋级与受控生产发布

**Date**: 2026-08-02
**Status**: accepted
**Deciders**: 项目 owner

## Context

StuHelper 是同时包含 Go 后端、Web、Admin、UniAppX、Koishi 和部署控制面的 monorepo。现有 GitHub
Actions 已经具备按路径测试、CodeQL、完整历史 secret scan、候选镜像 Trivy 扫描、不可变 SHA
tag、provenance、SSH host key 固定、生产备份/迁移/smoke 和人工回滚，但交付编排仍有几个明确缺口：

- Deploy 在进入受保护 environment 后才验证制品，审批者看不到完整的预验证结果；
- Forward Deploy 可接受任意历史 SHA，并会执行该历史版本中的部署脚本；
- staging 和 production 没有形成“同一主分支制品先验证、再晋级”的强制链路；
- 发布制品只有 provenance，没有随镜像保存可审计 SBOM；
- Rollback 会把历史源码和历史运维脚本一起恢复，扩大了旧控制代码接触生产 secret 的范围；
- 只有路径选择 CI，没有独立的周期性全量回归。

StuHelper 当前由单一常任维护者 `Xauryan` 管理。项目 owner 已明确接受 production environment
由该用户单人审批并允许自批，不采用为了流程形式而创建第二个管理员账号的方案。该治理选择不应
削弱制品验证、环境隔离、不可变审计和回滚安全。

## Decision

1. Pull Request 是代码评审和变更验证入口；`develop` / `main` 的可信 push 只有在 `CI / Required`
   成功后才发布镜像。CodeQL 的 Go 与 JavaScript/TypeScript 结果继续作为受保护分支门禁。
2. 每个可信 commit 的 backend、frontend、admin 镜像只构建一次。候选镜像先在本地完成
   `HIGH` / `CRITICAL` 漏洞门禁，再推送不可变完整 SHA tag；发布后为 digest 签发 provenance 和
   CycloneDX SBOM attestation。`latest` / `develop-latest` 只用于人类识别，永不作为部署输入。
3. `main` 的不可变 digest 集合是环境晋级单位。启用 `STAGING_AUTO_DEPLOY_ENABLED=true` 后，
   `main` CI 和三个镜像发布全部成功才自动部署 staging。再启用
   `PRODUCTION_AUTO_PROMOTION_ENABLED=true` 后，staging 成功会自动创建同 SHA production
   deployment 并等待 environment 审批。production 只能部署当前 `main` head，且默认必须先存在
   同一 SHA 的最新成功 staging deployment。
4. production 保留 GitHub protected environment 人工审批。当前唯一 reviewer 为 `Xauryan`，
   `prevent_self_review=false`；审批发生在分支、必需 checks、provenance 和 digest 全部验证之后。
5. Forward Deploy 不再接受人工输入 commit SHA。手工运行者只选择当前 workflow ref、目标环境和
   变更原因；目标 SHA 固定为 `github.sha`，必须同时等于 workflow controller SHA 和该分支的实时
   head。旧版本只能通过独立 Rollback 工作流选择。
6. production 若没有同 SHA staging success，默认 fail-closed。只有显式选择
   `skip_staging_gate=true`、填写至少 24 个字符的事故上下文并通过 production 审批，才能走可审计
   break-glass；该开关不适用于 staging，也不绕过 checks、provenance、digest 或分支规则。
7. 部署前验证 job 不绑定 environment，也不读取部署 secret。只有验证完成后的 deploy job 才进入
   environment 审批并获得环境级 SSH secrets；审批后、任何 SSH 前再次校验实时 branch head、checks
   和 staging gate，防止等待期间候选过期。可信分支 push 的发布 run 不允许被后续 push 半途取消，
   过期 run 由二次校验失败关闭。上传 bundle 使用唯一 run ID 文件名、固定 host key、专用私钥、
   传输后 SHA-256 校验和严格 SSH 超时。
8. Rollback 始终使用当前可信 `main` / `develop` controller 和当前运维脚本，只把经过 provenance
   验证的历史应用镜像 digest 作为回滚目标。禁止让历史 release 的 workflow 或运维控制脚本重新
   获得 environment secrets。
9. 部署失败不自动回退数据库 schema，也不无条件自动切换旧镜像。生产 migration 必须遵循
   expand/contract；失败后由操作者依据备份、迁移兼容性和 smoke 结果启动有原因、有审批的
   Rollback。
10. CI 每周在默认分支执行一次不受路径选择影响的全量回归；每个 job 设置明确 timeout。两个自动
    promotion 开关默认关闭，只有独立 staging 主机、两个环境的 secrets、运行时配置和真实 smoke
    都就绪后才按 staging、production 顺序启用。

## Security boundaries

- GitHub Actions 只保存到目标机的 SSH 传输凭据；应用、数据库、Casdoor、OpenFGA、对象存储和
  registry pull secrets 继续由目标机的非 file secret backend 管理。
- GitHub-hosted runner 不能访问维护者工作站的 localhost Vault；本机 Vault 不是 GitHub
  Environment Secrets 的替代品。
- SSH deploy 用户不得复用维护者日常 root 登录身份。Docker group 本身近似宿主 root 权限，目标机
  应优先采用受控部署账号、受限 sudo/gateway 或后续短期 SSH 证书，而不是把个人 root key 上传到
  GitHub。
- staging 必须是可以承受失败的隔离目标；与 production 共用数据库、secret backend、对象存储
  bucket 或同一 Compose project 不能算有效 staging 验收。
- Web/Admin 的 `VITE_*` 公开配置当前在镜像构建期固化，因此 staging 必须验证与 production
  相同的 public URL/SSO contract。未来若 staging 必须使用不同 SSO issuer 或浏览器 origin，应先实现
  受 CSP 和 schema 校验约束的运行时公开配置，再继续复用同一 digest；不得按环境悄悄重建镜像。
- provenance 和 SBOM 证明来源与依赖清单，不证明业务正确性；真实 SSO、授权、迁移、页面、QQ、
  备份恢复和可观测性仍以目标环境 smoke/evidence 为准。

## Alternatives Considered

### 合并 `main` 后无人值守直发生产

- **Pros**: 交付速度最快，流程最少。
- **Cons**: 单维护者仓库中的误合并会立即影响生产，且数据库迁移和外部系统缺少人工变更窗口判断。
- **Why not**: StuHelper 包含有状态迁移、Casdoor/OpenFGA、QQ 与生产 evidence；风险不适合完全取消
  environment gate。

### 每个环境重新构建镜像

- **Pros**: 可以把环境 URL 直接烘焙进静态前端。
- **Cons**: staging 测试的不是 production 最终字节，供应链证据和回滚矩阵成倍增加。
- **Why not**: 当前 staging 必须采用与 production-compatible artifact 相同的 public URL contract；
  secret 和后端运行配置由目标环境注入。需要不同浏览器 origin/SSO issuer 时，应先实现显式的运行时
  公开配置机制，不能通过重新构建改变已验收制品。

### 自动失败回滚

- **Pros**: 健康检查失败时恢复速度快。
- **Cons**: 在 migration 已成功、旧应用不兼容新 schema 时可能扩大事故；外部系统副作用也不能靠镜像
  回滚撤销。
- **Why not**: 当前 Compose/单主机场景应先 fail-closed 并保留证据，再由受控 Rollback 判断。

### Kubernetes / Argo CD 重构

- **Pros**: 原生 GitOps、渐进式发布和多副本编排能力更强。
- **Cons**: 当前单体 monorepo 与单主机规模没有足够收益，反而新增集群、控制面和运维复杂度。
- **Why not**: 先把现有 Docker Compose 交付链路做成不可变、可验证、可审计；达到明确多节点需求后再
  以 ADR 评估编排平台迁移。

## Consequences

### Positive

- 审批者只会看到已经通过 checks、分支和制品来源验证的候选发布。
- staging 与 production 可以消费同一组不可变 digest，消除环境重建漂移。
- 历史 workflow/运维脚本不再因 forward deploy 或 rollback 获得生产 secret。
- 每个镜像同时具备漏洞门禁、provenance、SBOM 和短期 Actions artifact 证据。
- 单维护者例外被限制在人工批准决策，不扩展为直接推送、跳过 CI 或跳过制品验证。

### Negative

- production 默认依赖 staging 就绪；没有独立 staging 基础设施时发布会被明确阻断。
- 周期性全量回归和 SBOM 生成会增加 Actions 时间与 artifact 存储。
- 只允许部署当前分支 head；需要恢复旧版本时必须使用 Rollback，不能借 Deploy 绕过回滚审计。
- 当前 SSH 传输仍是长期需要演进为短期凭据或受控部署 gateway 的边界。

### Risks

- staging 配置若错误地连接 production 数据，会制造伪验收或真实副作用。启用自动 staging 前必须
  核对数据库、对象存储、OAuth client、域名和通知通道隔离。
- 当前控制脚本与历史镜像可能不兼容。Rollback 必须保持 expand/contract，并在 staging 定期演练
  最近一个已发布版本。
- break-glass 可能被日常化。任何使用都必须有事故原因、production approval 和后续复盘；不能把
  仓库变量长期改为跳过 staging。
