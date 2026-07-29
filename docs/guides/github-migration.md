---
type: guide
audience: maintainers
status: current
authoritative-source: .github/workflows/ + GitHub repository settings
last-verified: 2026-07-28
---

# GitHub 迁移与 Actions 治理

## 目标

- GitHub Organization：`StuHelper`
- 公开仓库：`StuHelper/StuHelper`
- 仓库形态：保留 monorepo
- 默认开发分支：迁移期先保留 `develop`
- 稳定发布分支：`main`
- 容器镜像：GitHub Container Registry（GHCR）
- GitLab：迁移验收完成前保留为只读恢复锚点

不要在 GitHub 新仓库中初始化 README、`.gitignore` 或 LICENSE。第一次推送必须来自已经净化并验证过的迁移镜像，避免产生无意义的 unrelated history。

## 为什么保留单仓

`server/api/openapi.yaml` 同时生成 Go 服务端契约和 `clients/shared` TypeScript 契约；Web、Admin、UniAppX、Koishi 与基础设施又共同依赖这些契约和能力常量。现在拆仓会引入跨仓 SDK 发布、版本兼容矩阵、联动 PR 和发布编排，而不会改善核心模块边界。

只有当 Koishi 同时满足以下条件时，才重新评估把 `bots/koishi` 拆为独立仓库：

1. 只依赖已发布且有语义化版本的 StuHelper SDK；
2. 有独立发布节奏和兼容性矩阵；
3. 根目录 Makefile、E2E、部署脚本不再依赖其源码路径；
4. 跨仓契约变更有自动化兼容测试。

## 公开前历史净化

迁移必须在隔离 bare mirror 中完成，禁止在日常开发工作区直接改写历史。净化范围包括：

- 历史私钥和配套生成证书；
- 曾提交的真实部署环境文件；
- `infra/generated/` 下的 PostgreSQL WAL、证书和运行时派生配置；
- 误提交的本地构建二进制；
- 已删除的内部工具缓存和内部安全审查导出。

净化后必须重新执行完整历史 Gitleaks 扫描。经人工确认的假阳性使用根目录 `.gitleaksignore` 中的 commit-scoped fingerprint 精确基线化；禁止按整个仓库或宽泛正则关闭规则。

历史删除不能使已经泄露的凭据失效。任何可能使用过的历史私钥、部署口令、Git 凭据、数据库连接信息或服务令牌，都必须在对应系统中轮换并验证旧值失效。

## GitHub Actions 工作流

| 工作流 | 触发 | 权限与职责 |
|--------|------|------------|
| `CI` | PR、`develop`/`main` push、手工 | PR 只读；按路径选择 Go、契约、前端、E2E、Koishi、Infra、Semgrep、完整历史 secret scan、PR 新增依赖审查，以及 22 个受管运行时镜像的 `HIGH` / `CRITICAL` / `UNKNOWN` 策略扫描 |
| `CodeQL` | PR、push、每周、手工 | Go 与 JavaScript/TypeScript 代码扫描 |
| `Publish images` | 受信任 push 的 `CI` 全部成功后 | 构建一次、扫描同一镜像、发布不可变 SHA tag，并为最终 digest 签发 provenance |
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

`production` 至少配置 required reviewers、禁止管理员绕过和仅允许 `main` 分支部署。`staging` 仅允许 `develop` 和 `main`。

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

迁移期对 `develop` 和 `main` 配置 ruleset：

- 禁止 force push 和删除；
- 合并必须经过 Pull Request；
- 至少 1 名审批者；
- 新提交后撤销旧审批；
- 必须解决所有 review conversation；
- 必须通过 `CI / Required`；
- `main` 额外要求 CodeQL 的 Go 与 JavaScript/TypeScript 检查；
- 要求分支在合并前更新；
- 高风险路径由 `.github/CODEOWNERS` 审批。

历史净化后的第一次导入是一次受控例外；导入完成并核对分支 SHA 后立即启用 ruleset。

### Code security

仓库创建并导入后立即完成：

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

`develop-latest` 和 `latest` 只用于人类识别。部署与回滚输入必须是完整 commit SHA；工作流先把对应 tag 解析为 manifest digest，再验证 `StuHelper/StuHelper/.github/workflows/publish-images.yml` 签发的 provenance、源 commit、源 branch 和 GitHub-hosted runner 身份，最终只向远端传递 `image@sha256:...`。首次推送后确认三个 package 都关联到 `StuHelper/StuHelper`，并根据公开部署策略设置 package visibility。

远端部署脚本当前仍执行 registry login。迁移 GHCR 时，需要在远端 secret backend 中配置独立、最小 `read:packages` 读取凭据，不能复用个人日常登录凭据。

## 迁移验收

1. 核对 GitHub 三个分支的 commit graph 和净化后 SHA；
2. 完整历史 Gitleaks 扫描为零未基线化发现；
3. `CI / Required` 与 CodeQL 全部通过；
4. 三个 GHCR 镜像均可按 digest 拉取，Trivy 和 provenance attestation 成功；
5. staging 手工部署、真实页面/E2E、服务健康和观测 smoke 全部通过；
6. production environment 审批与回滚演练通过；
7. GitLab 保留只读并记录迁移前最后 SHA；
8. `SECURITY.md` 的私密报告入口可用，Secret scanning、Push protection 和 Dependabot alerts 已启用；
9. 完成根目录许可证决策后再对外宣布“开源”；仅设为 public 不等于授予开源许可。

## 待确认的治理决策

仓库当前没有根目录 LICENSE。创建 public 仓库前必须由项目所有者明确选择：

- 保留所有权利（公开源码但不授予复用许可）；或
- 选择适用于整个 monorepo 的开源许可证，并处理 `clients/admin` 与 Koishi 现有许可证边界。

迁移自动化不得擅自替项目选择许可证。
