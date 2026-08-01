# 参与 StuHelper 开发

感谢你愿意改进 StuHelper。当前仓库采用 monorepo，后端、浏览器客户端、管理后台、跨端实验项目、Koishi 插件和部署契约在同一提交中演进。

## 开始之前

1. 阅读 [快速开始](docs/QUICKSTART.md) 和根目录 [AGENTS.md](AGENTS.md)；
2. 对较大的功能、数据模型或兼容性变更，先创建 Issue 说明问题、边界、迁移与回滚方案；
3. 安全漏洞不得提交到公开 Issue。请使用仓库 Security 页面中的私密漏洞报告入口；
4. 不要提交真实凭据、生产数据、生成证书、数据库 WAL、运行时环境文件或本地构建产物。

除提交内容自身明确、合法地保留其他兼容许可证外，向本仓库提交贡献即表示你有权按
[GNU Affero General Public License v3.0 only](LICENSE)（SPDX：`AGPL-3.0-only`）提供该贡献。
不得移除第三方代码已有的版权、许可证或通知。

## 开发约束

- API 契约以 `server/api/openapi.yaml` 为权威来源。修改接口时先改 OpenAPI，再运行 `make generate`；
- 不要手改 `server/internal/api/gen/` 或 `clients/shared/src/types/api.gen.ts`；
- 后端遵循 Handler → Service → Repository，SQL 只进入 Repository；
- 前端统一通过 `clients/shared` 使用 API 和生成类型；
- IAM、认证、session、outbox、审计留存和后台任务变更必须遵守
  [IAM 实现护栏](docs/design/iam-implementation-guardrails.md)；
- 配置必须来自环境变量或受支持的 secret backend，不得在源码中硬编码。

## 提交前验证

根据改动范围运行相关检查；跨域或基础设施变更应运行完整门禁。

```bash
# 后端
cd server
make fmt && make lint && make test && make security && make build && make check-drift

# 浏览器客户端和管理后台
cd clients
pnpm install --frozen-lockfile
pnpm type-check:all && pnpm lint:all && pnpm test:all
pnpm build:web && pnpm build:admin && pnpm build:uni:h5

# Koishi
cd bots/koishi
corepack yarn install --immutable
corepack yarn build && corepack yarn test:ui

# 仓库与基础设施契约
make check-docs
bash scripts/check-uniappx-shadow-files.sh
bash infra/ops/tests/run-infra-contracts.sh
```

后端数据库集成测试默认将 Go 包并行度限制为 1，以保证 Testcontainers 清理器和包级共享数据库的生命周期确定。可通过 `TEST_PACKAGE_PARALLELISM` 显式覆盖，但提高并行度前必须提供独立测试数据库隔离。

## Pull Request

- PR 保持单一目的，并说明用户可见影响、风险、验证证据和回滚方式；
- 使用与现有历史一致的 Conventional Commits 风格，例如 `feat(course): ...`、`fix(auth): ...`；
- 如果生成代码发生变化，同时提交权威契约和生成结果；
- 如果数据库 schema 发生变化，提交迁移、回滚/兼容说明和相关测试；
- 不得通过降低门禁、宽泛忽略扫描结果或向 fork PR 暴露 secret 来让 CI 变绿；
- 合并前解决所有 review conversation，并取得 CODEOWNERS 要求的审批。
