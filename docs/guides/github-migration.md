---
type: guide
audience: maintainers
status: current
authoritative-source: .github/workflows/ + GitHub repository settings
last-verified: 2026-08-01
---

# GitHub 仓库与 Actions 治理

## 当前仓库形态

- GitHub Organization：`StuHelper`
- 公开仓库：`StuHelper/StuHelper`
- 仓库形态：保留 monorepo
- GitHub 默认分支：`main`
- 日常集成分支：`develop`
- 稳定发布分支：`main`
- 容器镜像：GitHub Container Registry（GHCR）

仓库内只保留 GitHub Actions 工作流；GitHub 是代码评审、质量门禁、镜像发布、部署和回滚的唯一仓库级自动化控制面。
项目文档以仓库内 `docs/` 为权威来源，GitHub Wiki 已关闭，避免形成无法随代码评审和版本化的第二套事实源。

## 为什么保留单仓

`server/api/openapi.yaml` 同时生成 Go 服务端契约和 `clients/shared` TypeScript 契约；Web、Admin、UniAppX、Koishi 与基础设施又共同依赖这些契约和能力常量。现在拆仓会引入跨仓 SDK 发布、版本兼容矩阵、联动 PR 和发布编排，而不会改善核心模块边界。

只有当 Koishi 同时满足以下条件时，才重新评估把 `bots/koishi` 拆为独立仓库：

1. 只依赖已发布且有语义化版本的 StuHelper SDK；
2. 有独立发布节奏和兼容性矩阵；
3. 根目录 Makefile、E2E、部署脚本不再依赖其源码路径；
4. 跨仓契约变更有自动化兼容测试。

## 历史与秘密基线

所有可达提交都必须持续满足完整历史 Gitleaks 门禁，不得包含：

- 历史私钥和配套生成证书；
- 曾提交的真实部署环境文件；
- `infra/generated/` 下的 PostgreSQL WAL、证书和运行时派生配置；
- 误提交的本地构建二进制；
- 已删除的内部工具缓存和内部安全审查导出。

经人工确认的假阳性使用根目录 `.gitleaksignore` 中的 commit-scoped fingerprint 精确基线化；禁止按整个仓库、整个规则或宽泛正则关闭检测。CI 必须以 `fetch-depth: 0` 检出并扫描所有可达提交。

任何泄露过的凭据都必须在对应系统中失效；删除 Git 内容不能替代凭据失效和审计。

## GitHub Actions 工作流

| 工作流 | 触发 | 权限与职责 |
|--------|------|------------|
| `CI` | PR、`develop`/`main` push、手工 | PR 只读；按路径选择 Go、契约、前端、E2E、Koishi、Infra、Semgrep、完整历史 secret scan、PR 新增依赖审查，以及 22 个受管运行时镜像的 `HIGH` / `CRITICAL` / `UNKNOWN` 策略扫描 |
| `CodeQL` | PR、push、每周、手工 | Go 与 JavaScript/TypeScript 代码扫描 |
| `Publish images` | 受信任 push 的 `CI` 全部成功后 | 同一 commit 只构建一次、扫描同一镜像、发布不可变 SHA tag；另一受信任分支只能复用已验证 digest，并为最终 digest 签发 provenance |
| `Deploy` | 手工 | 验证指定 40 位 commit SHA、发布工作流身份、源分支、源提交和镜像 digest 后部署 |
| `Rollback` | 手工 | 使用同一 environment 锁回滚到经过相同 provenance 校验的 40 位 commit SHA |

所有外部 Action 必须固定到完整 commit SHA，并在注释中保留对应主版本。公开 fork 的 PR 不得使用 `pull_request_target` 检出或运行不受信任代码。

依赖更新由 `.github/dependabot.yml` 每周检查 GitHub Actions、Go modules、三个 JavaScript workspace 和 Docker 基础镜像。Dependabot PR 仍必须经过相同的 review、测试和安全门禁，不允许自动绕过 ruleset。

## 仓库设置

### Actions

在 `Settings → Actions → General` 中：

1. Workflow permissions 设为 **Read repository contents and packages permissions**；
2. 禁止 Actions 直接创建或批准 Pull Request；
3. 只允许 GitHub 官方 Action和仓库中已固定 SHA 的允许列表；
4. 保留 fork PR 的 secrets 禁用状态。

`Publish images` 在 job 级显式申请 `packages: write`、`attestations: write`、`id-token: write` 和 `artifact-metadata: write`。最后一项用于 `actions/attest` 将已发布 digest 登记到组织 Linked Artifacts；Deploy 与 Rollback 仅申请 `packages: read` 和 `attestations: read`；其他工作流不继承这些权限。

### 免费额度与预算护栏

公开仓库使用标准 GitHub-hosted runner 的计算时间免费，但 larger runner、Actions artifact、cache 和 package 存储有各自规则，不能把“公开仓库 Actions 免费”理解为所有关联资源无限免费。

- 工作流只使用标准 `ubuntu-latest`，不启用 larger runner；
- 测试 artifact 只在失败诊断或审计需要时上传，普通测试当前保留 7 天，运行时镜像 JSON 扫描 evidence 保留 14 天；
- Actions cache 保持仓库默认 10 GB 上限，禁止为提速擅自扩大付费上限；
- 三个 GHCR package 在验收后设为 public；Container registry 当前公开存储和带宽政策需定期复核；
- 组织 Billing 中为 Actions、Packages 和 cache 设置预算告警；若目标是严格零成本，预算上限设为 0，并接受超限后缓存只读或任务被阻止。

### Environments

创建 `staging` 和 `production` 两个 environment。两个 environment 使用相同 secret 名称、不同值：

- `DEPLOY_HOST`
- `DEPLOY_PORT`
- `DEPLOY_USER`
- `DEPLOY_APP_DIR`
- `DEPLOY_SSH_KEY`
- `DEPLOY_SSH_KNOWN_HOSTS`

环境变量：

- `PUBLIC_URL`

`production` 只允许 `main` 分支并要求 environment reviewer；`staging` 仅允许 `develop` 和 `main`。生产真正启用前必须至少有两名具备相应仓库权限的维护者，并启用“发起部署者不能批准自己的部署”。只有一个合格 reviewer 时直接启用该选项会造成无法部署，不能把单人自批当作双人复核。

SSH known_hosts 必须预先固定真实 host public key，且条目必须与 `DEPLOY_HOST` 和 `DEPLOY_PORT` 对应；工作流不允许运行时 `ssh-keyscan` 或 TOFU。部署主机使用 DNS 名称或规范 IPv4 地址，部署用户使用规范 Linux 账号名，应用目录使用不含软解析片段的绝对路径。

### Repository variables

以下变量都是公开构建参数，不得放密钥：

- `WEB_VITE_API_URL`
- `WEB_VITE_SSO_URL`
- `WEB_VITE_WEB_URL`
- `WEB_VITE_API_TIMEOUT_MS`
- `WEB_VITE_QQ_BOT_ENTRY`
- `WEB_VITE_QQ_BIND_COMMAND`
- `ADMIN_PUBLIC_URL`
- `ADMIN_VITE_API_URL`
- `ADMIN_VITE_BASE`

### Branch protection

对 `develop` 和 `main` 配置 ruleset：

- 禁止 force push 和删除；
- 合并必须经过 Pull Request；
- 至少 1 名审批者；
- 新提交后撤销旧审批；
- 必须解决所有 review conversation；
- 必须通过 `CI / Required`；
- 必须通过 CodeQL 的 Go 与 JavaScript/TypeScript 检查；
- 要求分支在合并前更新；
- 高风险路径由 `.github/CODEOWNERS` 审批。

仓库采用一项明确、最小的单维护者例外：ruleset bypass list 只包含 GitHub 用户
`Xauryan`（user ID `268165484`），且 `bypass_mode` 必须保持为 `pull_request`。这意味着
`Xauryan` 仍必须通过 Pull Request 留下变更、检查和合并审计轨迹，但可以在所有必需检查通过、
review conversation 已解决后，选择绕过“另一名审批者 / CODEOWNER / 最后推送者之外审批”门禁。
该例外不得扩大为组织管理员、仓库角色或其他用户，也不得改成长期 `always` 或 `exempt`；
因此它不允许直接推送、force push 或无审计绕过。

GitHub 不允许 PR 作者提交 `APPROVED` 自审记录。上述行为在 GitHub 中表现为由 `Xauryan`
使用 ruleset bypass 合并，而不是伪造的“作者批准”。其他身份仍完整适用 1 个独立 approval、
CODEOWNERS 和最后推送者之外审批要求。若未来增加第二名常任维护者，应重新评估并优先移除
这个单维护者例外。

### Code security

仓库必须保持：

1. 确认公开仓库的 Secret scanning 已运行，并逐条处理任何 alert；
2. 启用仓库级 Push protection；绕过必须留下理由并由指定人员复核；
3. 启用 Private vulnerability reporting，确保 `SECURITY.md` 的私密报告链接可用；
4. 启用 Dependabot alerts、security updates 和 dependency graph；
5. 将 CodeQL 结果纳入 `main` 的合并门禁，禁止用删除工作流或降级权限的方式规避。

GitHub 原生检测不能替代仓库内的完整历史 Gitleaks 门禁：两者覆盖模式、时机和审计证据不同，应同时保留。

## GHCR

镜像名称：

- `ghcr.io/stuhelper/backend:<full-commit-sha>`
- `ghcr.io/stuhelper/frontend:<full-commit-sha>`
- `ghcr.io/stuhelper/admin:<full-commit-sha>`

`develop-latest` 和 `latest` 只用于人类识别。部署与回滚输入必须是完整 commit SHA；同一 commit 的发布任务以 commit SHA 串行，首次构建使用 commit 时间固定镜像与文件元数据。若 SHA tag 已存在，工作流必须先验证仓库、签发工作流、源 commit、受信任源 branch 和 GitHub-hosted runner 身份，随后直接复用原 digest，禁止覆盖该 tag。部署工作流把 tag 解析为 manifest digest，重复执行相同 provenance 校验，最终只向远端传递 `image@sha256:...`。首次推送后确认三个 package 都关联到 `StuHelper/StuHelper`，并根据公开部署策略设置 package visibility。

远端部署脚本当前仍执行 registry login。迁移 GHCR 时，需要在远端 secret backend 中配置独立、最小 `read:packages` 读取凭据，不能复用个人日常登录凭据。

## 仓库治理验收

1. 核对 GitHub `develop` / `main` 的 commit graph 和受保护分支设置；
2. 完整历史 Gitleaks 扫描为零未基线化发现；
3. `CI / Required` 与 CodeQL 全部通过；
4. 三个 GHCR 镜像均可按 digest 拉取，Trivy 和 provenance attestation 成功；
5. staging 手工部署、真实页面/E2E、服务健康和观测 smoke 全部通过；
6. production environment 审批与回滚演练通过；
7. `SECURITY.md` 的私密报告入口可用，Secret scanning、Push protection 和 Dependabot alerts 已启用；
8. 根目录许可证状态与项目对外表述一致；仅设为 public 不等于授予开源许可。

## 当前就绪状态

以下状态于 2026-08-01 通过 GitHub API 和受保护工作流重新核验：

| 项目 | 状态 | 当前事实 |
|------|------|----------|
| 仓库与分支 | 已验证 | `StuHelper/StuHelper` 为 public，默认分支为 `main`；人类长期分支只有 `main` / `develop`，二者受同一 ruleset 保护并保持相同提交。Dependabot 为开放依赖更新 PR 创建的临时 head branch 不属于长期分支，关闭或合并对应 PR 后删除 |
| 合并门禁 | 已验证 | 默认要求 PR、1 个 approval、CODEOWNERS、撤销旧审批、最后推送者之外的批准、解决 review thread、线性历史和 squash merge；Required、Go CodeQL、JavaScript/TypeScript CodeQL 为必需检查。唯一例外是 `Xauryan` 的 `pull_request`-only ruleset bypass；它仍要求 PR 和审计轨迹，不产生作者自审记录，也不允许直接推送 |
| Actions 供应链 | 已验证 | Actions 已启用 selected-actions 策略，外部 action 固定完整 commit SHA，默认 `GITHUB_TOKEN` 只读且不能批准 PR |
| Code security | 已验证 | Secret scanning、push protection、Dependabot alerts/security updates 和 private vulnerability reporting 已启用；当前 CodeQL、Dependabot、secret-scanning alert 均为 0 |
| Environments | 部分验证 | `staging` 与 `production` 分支策略存在；production 当前只有一名 reviewer 且允许自批，不构成双人复核 |
| 部署凭据 | 未就绪 | 两个 environment 的 secrets 和 variables 都为空，仓库级 Actions secrets/variables 也为空 |
| GHCR | 部分验证 | `backend`、`frontend`、`admin` container package 已由受保护 `main` / `develop` 工作流发布 full-SHA immutable tag、branch alias 和 provenance；当前 visibility 为 private。实际部署前仍须验证目标主机能以最小 `read:packages` 凭据按 digest 拉取 |
| 真实部署与回滚 | 未验证 | environment secrets/variables 仍未配置，production 尚无第二名独立 reviewer，private GHCR 镜像的目标主机拉取链路也未验收；尚未执行 staging/production 部署或回滚演练 |

因此，代码、仓库治理和镜像发布控制面已经建立；真实 staging、production 和 rollback 必须在
environment 配置、GHCR 拉取凭据与独立生产审批条件补齐后单独验收，不能用本地 smoke、
镜像发布成功或 workflow 静态检查替代。

## 许可证状态

项目所有者已决定用根目录 `LICENSE` 将 StuHelper monorepo 按 GNU Affero General Public
License v3.0 only（SPDX：`AGPL-3.0-only`）授权。OpenAPI 和 StuHelper 自有包元数据必须与根许可证保持一致。

`clients/admin/` 中源自 Vben 的代码继续保留其目录内的 MIT 许可证和版权声明；第三方组件
仍分别受其自身许可证约束。根许可证不得被解释为移除、替代或缩减这些既有通知与义务。
