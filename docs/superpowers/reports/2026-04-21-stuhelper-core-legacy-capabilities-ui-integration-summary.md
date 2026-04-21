# StuHelper Core Legacy Capabilities UI Integration Summary

## Completed

- 在 `stuhelper-core` 中落地了新的页面域后端接口：
  - `dashboard`
  - `identity`
  - `review`
  - `config-governance`
- 新前端壳层已接入 grouphelper 风格顶栏导航，并保留：
  - `Dashboard`
  - `Config`
  - `Warns`
  - `Blacklist`
  - `Identity`
  - `Review`
  - `Roles`
  - `Logs`
  - `Chat`
  - `Subscriptions`
  - `Settings`
- `ReviewView` 已从“仅 review 可操作”升级为统一工作项动作面板：
  - `review`: `execute` / `reject`
  - `admission`: `approve` / `deny` / `defer`
  - `report`: `dismiss` / `escalate` / `create-review`
- `ConfigCenterView` 已接通治理保存动作：
  - 模板保存
  - 群绑定保存
  - 命令策略保存
- 前端页面逻辑已抽出纯 TS model：
  - `dashboard.ts`
  - `identity.ts`
  - `review.ts`
  - `config.ts`
- 启动链路中的 Koishi inject 告警已修复：
  - 不再出现 `console`
  - 不再出现 `database`
  - 不再出现 `stuhelperGroupCenter`

## Verification

已通过：

```bash
cd bots/koishi
node --import tsx --test \
  plugins/stuhelper-core/client/models/navigation.test.ts \
  plugins/stuhelper-core/client/models/dashboard.test.ts \
  plugins/stuhelper-core/client/models/identity.test.ts \
  plugins/stuhelper-core/client/models/review.test.ts \
  plugins/stuhelper-core/client/models/config.test.ts \
  plugins/stuhelper-core/src/core/services/dashboard-page.service.test.ts \
  plugins/stuhelper-core/src/core/services/identity-page.service.test.ts \
  plugins/stuhelper-core/src/core/services/review-page.service.test.ts \
  plugins/stuhelper-core/src/core/services/config-governance.service.test.ts \
  plugins/stuhelper-core/src/core/api/review-actions.test.ts \
  plugins/stuhelper-core/src/browser-entry.test.ts \
  plugins/stuhelper-core/src/runtime-contract.test.ts

./node_modules/.bin/tsc -p plugins/stuhelper-core/tsconfig.json --noEmit --skipLibCheck
./node_modules/.bin/yakumo build
node scripts/startup-smoke.mjs
```

## Notes

- `startup-smoke.mjs` 现在会显式拦截 inject 警告回归。
- `Review` 事件关联已兼容 `reportId` 与 legacy `reportID` 两种 payload 字段。
- 本轮未做 git commit；当前改动仍在工作区。
