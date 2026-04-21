# Koishi Console UI 重构完成总结

日期：2026-04-21
范围：`bots/koishi/plugins/stuhelper-console`

## 本次完成内容

1. 统一了复核动作、复核状态、身份认证状态和时间格式化的单点格式器，驾驶舱、队列、审计、详情抽屉不再各自翻译一套文案。
2. 按首页设计要求重建驾驶舱结构，补齐状态带、系统状态、最近事件、最近变更，并修正深链跳转目标。
3. 将页面首屏未就绪态从整页空状态改为骨架屏，避免把加载态和空数据态混为一谈。
4. 拆分原本过大的 `use-console-page.ts`，将导航、抽屉、通知、审计和提交动作分离为独立 composable。
5. 修复 URL 单向同步问题，新增 `popstate` 恢复，保证后退/前进可以恢复控制台上下文。
6. 修复详情抽屉持有旧对象引用的问题，改为仅存 `kind/id` 并从最新数据快照实时取值。
7. 修复复核抽屉执行后自动聚焦候选集的问题，优先使用当前可见或冻结的候选列表，不再错误跳到全队列。
8. 为举报记录补齐 URL 选中、高亮和详情上下文；举报抽屉补充群号、频道、平台等信息。
9. 为关键词规则保存链路引入专用输入类型 `StuhelperKeywordRuleInput`，清除客户端伪造 `createdAt/updatedAt` 的协议污染。
10. 为控制台 listener 增加 transport 边界输入解析，非法输入会被明确拒绝。
11. 将命令策略候选 ID 上移到共享运行时常量，补齐真实已用命令集合，消除控制台策略下拉框缺项。
12. 将抽屉、空状态、标签之外的剩余基础控件继续收敛到 Element Plus：队列表格改为 `ElTable`，主要操作按钮改为 `ElButton`，输入控件改为 `ElInput/ElSelect/ElInputNumber/ElCheckbox`，控制台视觉基座不再同时维护两套基础控件体系。
13. 修正 `save-keyword-rule` 的 console 事件签名，改为真正的输入类型而不是输出类型。

## 补充测试

1. 新增 `client/source-architecture.test.ts`
   约束关键页面、表格、表单和按钮必须走 Element Plus 基础控件，防止回退到自研基础控件。
2. 新增 `src/console-contract.test.ts`
   约束 `save-keyword-rule` listener 事件签名必须使用 `StuhelperKeywordRuleInput`。

## 最新验证结果

1. `corepack yarn test:unit`
   结果：`54 pass / 0 fail`
2. `corepack yarn build`
   结果：通过
   说明：仅出现 Vite CJS Node API deprecation warning，不影响构建成功。
3. `corepack yarn test:startup`
   结果：通过
   关键日志：`server listening at http://127.0.0.1:5140`

## 结论

这轮修复覆盖了此前确认需要修复的功能问题、契约问题、结构问题和主要 UI 基础控件一致性问题。当前代码已经通过单测、构建和启动烟测验证，可作为本轮 Koishi 控制台重构的完成基线。
