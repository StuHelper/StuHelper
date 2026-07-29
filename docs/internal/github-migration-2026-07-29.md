---
type: internal
audience: maintainers
status: snapshot
authoritative-source: GitLab develop + GitHub StuHelper/StuHelper migration evidence
last-verified: 2026-07-29
---

# GitHub 首次正式迁移证据

## 恢复锚点

- GitLab `develop` 最后一个迁移前提交：`ec728bfb0a0455f96fc3673f6d0041cb6e21bc8e`
- GitHub 仓库：`StuHelper/StuHelper`
- GitHub 迁移前 `main`：`4bb46c9613b534f17b4aa765b1b4ec98592c7c08`
- GitHub 迁移前 `develop`：`47d95478a397d20b18c293329db28d7690f63ccf`
- GitHub 迁移前 `refactor`：`acb17f8c160f583b0909489ad5ee49f81aa8f94e`

迁移前 GitHub 仓库没有 Pull Request、Issue 或 tag。三条旧分支均来自同一 StuHelper GitLab 历史的早期净化镜像，不包含独立的第三方历史。

## 迁移专属变更

- Go module 和仓库内 Go import canonical path 从旧 GitLab 地址改为 `github.com/StuHelper/StuHelper/server`。
- GitLab 保留完整原历史，禁止 force-push；GitHub 只接收隔离 bare mirror 中生成的净化历史。
- GitHub 导入仍保留 `main`、`develop`、`refactor` 三条业务分支。

## 历史净化范围

以下路径从 GitHub 的所有可达历史中删除：

- `infra/generated/`
- `server/certs/`
- `deploy/`
- `server/fga-setup`
- `.agents/`
- `.gitea/`
- `.project_rule/`
- `.superpowers/`
- `.trellis/`
- `docs/reviews/2026-06-10-full-codebase-review.md`
- `docs/reviews/2026-06-10-full-codebase-review.json`
- `docs/reviews/2026-06-10-full-codebase-review-schema.md`

历史作者和提交者姓名保留；非 GitHub noreply 邮箱在 GitHub 镜像中替换为不可逆的稳定匿名 noreply 地址，GitLab 恢复锚点继续保留原始作者元数据。

净化后在迁移头部恢复当前仍需要的空目录占位文件，并重建 `.gitleaksignore`：只允许使用完整的 `commit:path:rule:line` fingerprint 基线化人工确认的测试夹具、生成契约或静态字符表，不能忽略历史私钥、部署配置或整个路径。

## 验收门槛

- GitHub 三条分支之外不推送临时 rewrite ref 或备份 ref。
- 所有可达分支的完整历史 Gitleaks 扫描没有未基线化发现。
- 所有可达 blob 不再包含 PostgreSQL WAL、历史私钥、部署环境文件、内部工具缓存或 `server/fga-setup` 二进制。
- 当前发布树的 Gitleaks 文件系统扫描为零发现。
- Go lint、串行 package 测试、构建和生成漂移检查通过。
- GitHub Actions 使用只读默认权限、完整 SHA pin、fork PR 无 secrets，并以 `CI / Required` 作为分支门禁。

仓库根目录仍未授予统一开源许可证；这次操作是 public source 迁移，不代表整个 monorepo 已成为开源项目。
