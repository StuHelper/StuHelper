# StuHelper Core Legacy Capabilities UI Integration Summary

## 状态

- 基线提交：`8602c35`
- 文档更新时间：`2026-04-21 22:59:27 CST`
- 当前工作区：`dirty`
- 说明：本轮修复尚未提交，当前工作区包含已验证通过的未提交改动。

## 本轮已落地修复

- 控制台鉴权链路：
  - `koishi.yml` 已显式启用 `@koishijs/plugin-auth`
  - `registerConsoleAPI()` 已通过 `ctx.inject(['console', 'database', 'stuhelperGroupCenter', 'auth'], ...)` 绑定鉴权依赖
  - `runtime-contract.test.ts` 已校验 `auth` 配置与 API 注入依赖
- Review / Admission / Report 工作流：
  - 审核动作已透传真实 console 操作者身份
  - `review` 执行分支增加 claim/finalize/rollback，避免并发双执行
  - `admission` 的 `approve` / `deny` / `defer` 已使用条件更新避免重复处理
  - `report` 的 `dismiss` / `escalate` / `create-review` 已收口到事务包装路径
- 输入校验：
  - governance action 已切换到 `zod`
  - 模板、群绑定、命令策略都增加了 `trim`、长度上限、数值范围和空白归一化
- Identity 查询链路：
  - 已新增 `IdentityProfileLookup`
  - 支持并发上限、60 秒 TTL 缓存、结构化错误 `IdentityProfileLookupError`
  - `platform.fetch()` 默认带 `AbortSignal.timeout(8000)`
- 配置治理页：
  - 表单逻辑已抽出到 `use-config-governance.ts`
  - 三套提交状态已拆分
  - 已加入 dirty check、切换确认、`beforeunload` 提示
- 类型与事件声明：
  - `client/page-types.ts` 改为复用服务端 `page-types`
  - `augmentations.d.ts` 已补齐 `page/*`、`action/*`、`chat/*`、`cache/*`、`stats/*` 等 console 事件
  - 已移除 `page-api.ts`、`review-actions.ts`、`governance-actions.ts` 和核心 `index.ts` 中的 listener-name `as any`
- Service 清理：
  - `stuhelper-group-center.service.ts` 已去掉生产代码里的 `console.*`
  - 缓存预热改为可清理的 timer，并在 `stop()` 中显式 `clearTimeout`
  - 命令日志时间已改为 `Asia/Shanghai` 正确格式化，不再通过 `setHours(+8)` 伪造时区
  - `pushMessage()` / `logCommand()` 的 `any` 已替换为显式结构类型
- 测试补齐：
  - 新增/补强了 `review-actions.test.ts`
  - 新增 `governance-actions.test.ts`
  - 新增 `identity-profile-lookup.test.ts`
  - 新增 `client/models/config-editor.test.ts`

## 验证结果

已实际通过：

```bash
cd bots/koishi

node --import tsx --test \
  plugins/stuhelper-core/client/models/navigation.test.ts \
  plugins/stuhelper-core/client/models/dashboard.test.ts \
  plugins/stuhelper-core/client/models/identity.test.ts \
  plugins/stuhelper-core/client/models/review.test.ts \
  plugins/stuhelper-core/client/models/config.test.ts \
  plugins/stuhelper-core/client/models/config-editor.test.ts \
  plugins/stuhelper-core/src/core/services/dashboard-page.service.test.ts \
  plugins/stuhelper-core/src/core/services/identity-page.service.test.ts \
  plugins/stuhelper-core/src/core/services/review-page.service.test.ts \
  plugins/stuhelper-core/src/core/services/config-governance.service.test.ts \
  plugins/stuhelper-core/src/core/api/review-actions.test.ts \
  plugins/stuhelper-core/src/core/api/governance-actions.test.ts \
  plugins/stuhelper-core/src/core/api/identity-profile-lookup.test.ts \
  plugins/stuhelper-core/src/browser-entry.test.ts \
  plugins/stuhelper-core/src/runtime-contract.test.ts

./node_modules/.bin/tsc -p plugins/stuhelper-core/tsconfig.json --noEmit --skipLibCheck
./node_modules/.bin/yakumo build
node scripts/startup-smoke.mjs
```

核验要点：

- `34` 条核心单测全部通过
- `tsc --noEmit` 通过
- `yakumo build` 通过
- `startup-smoke.mjs` 通过，日志确认：
  - `StuHelper 群管中心插件已加载`
  - `WebSocket API registered`
  - `server listening at http://127.0.0.1:5140`
  - `console webui is available at http://127.0.0.1:5140`
  - 不再出现 `property console/database/stuhelperGroupCenter is not registered`
  - 不再出现我本轮引入的错误 auth 误报

## 备注

- 本文档反映的是当前真实状态，不再保留“工作区 clean”或“本轮未验证”的旧描述。
- 当前尚未提交 git commit，因此 `git status --short` 仍显示未提交改动，这是预期状态。
