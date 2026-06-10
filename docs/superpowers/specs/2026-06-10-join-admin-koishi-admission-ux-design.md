---
type: superpowers-design
status: approved-design
approval: user-approved方案A
approved-at: 2026-06-10
scope:
  - clients/web JOIN admission UI
  - clients/admin admission policy and session management
  - bots/koishi admission runtime WebUI
  - admission cross-system E2E coverage
  - behavior-preserving code quality review
---

# JOIN / Admin / Koishi 入群认证体验与测试重构设计

## 目标

本设计固化已批准的方案 A：分阶段产品化重构 JOIN、StuHelper Admin 和 Koishi WebUI 的入群认证体验，并补齐端到端测试与质量护栏。

本次目标有四个同等重要的部分：

1. 全面审查并优化 JOIN 页面 UI，使用 `ui-ux-pro-max` 和 `frontend-design` 的设计约束重构 `/verify/<code>` 与新增 `/start` 入口。
2. 全面审查 StuHelper Koishi 全流程对接，用 `brainstorming` 形成 Admin 与 Koishi WebUI 的功能设计、布局设计和权威边界。
3. 补强 E2E，覆盖关键业务需求、跨系统同步和操作失败/部分成功路径。
4. 审查相关代码质量，在不改变原始业务逻辑和业务效果的前提下拆分过大页面、修复测试漂移、增强错误表达与可维护性。

## 当前事实

JOIN 当前只有带 token 的认证入口：

- Web 路由：`clients/web/src/router/index.ts` 中只有 `/verify/:code` 与 `/admission/freshman/camera/:token`。
- JOIN 域隔离：`clients/web/src/router/join-domain.ts` 只放行 `/verify/<code>` 和移动拍照路径。
- 设计文档：`docs/design/join-self-service-entry.md` 已定义 `join.stuhelper.com/start`，但尚未实现。
- 现有 `/verify/<code>` 页面集中在 `clients/web/src/modules/admission/views/AdmissionPage.vue`，状态完整但页面较大，视觉偏临时工具页，部分按钮触控高度低于 44px。

Admin 当前承担策略权威源：

- 策略页：`clients/admin/apps/web-ele/src/views/users/admission-policy/index.vue`。
- 会话页：`clients/admin/apps/web-ele/src/views/users/admission-sessions/index.vue`。
- 策略页已包含目标认证群、入群处理策略、材料转发、审核通知群、失败/拉黑等字段，但全部平铺在长表单里，容易混淆“目标认证群”和“材料审核通知群”。

Koishi 当前承担执行态：

- WebUI：`bots/koishi/plugins/stuhelper-core/client/components/AdmissionView.vue`。
- Console API：`bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts`。
- 后端策略同步：`bots/koishi/plugins/stuhelper-group-guard/src/guard-policy-bootstrap.ts`。
- 同步逻辑将后端 admission policy target 转为 Koishi guard binding；`join_request_review` 会同步为非 post-join guard 的停用执行绑定。
- Koishi WebUI 已能显示运行开关、目标群绑定、模板和受限成员队列，但“Admin 是权威源、Koishi 是执行缓存”的界面表达仍需强化。

测试当前状态：

- Koishi admission 关键单测通过，覆盖 WebUI skip、解除禁言失败容错、policy target 同步和提醒渠道。
- Web `/verify/<code>` Playwright E2E 覆盖较多，包括登录回跳、token 错误态、老生邮箱 OTP、新生手机拍照接力和材料提交。
- `/start` 入口没有实现，也没有 E2E。
- Admin admission 主要是组件和源码契约测试，缺少跨系统 E2E。
- 当前已知 Web 质量缺口：
  - `src/stores/__tests__/authAuthorizeFlow.test.ts` 的 mock 缺 `rememberAdmissionAuthReturn`，属于测试漂移。
  - `src/modules/review/__tests__/ratingDisplayPolicy.test.ts` 发现教师评分页仍展示精确评分数字，属于非 admission 但应纳入代码质量整改的现有失败。

## 非目标

本次第一版 `/start` 不做“无 token 自动通过当前入群会话”。

理由：

- `/verify/<code>` 的 token 证明浏览器流程对应某个 QQ 入群会话。
- `/start` 没有 admission token，不能安全声明当前用户正在处理哪个入群会话。
- 如果要在 QQ 绑定后按 QQ 号归并 active admission session，并自动释放禁言，需要新增后端服务层事务、冲突规则、审计事件和 Koishi 出队动作。这会改变 admission session 状态机，必须单独设计。

因此 `/start` 第一版只完成账号级事实：

- 用户已登录。
- 学生认证已通过或进入既有审核流程。
- QQ 已通过机器人绑定到当前账号。

页面不得展示“当前群认证已通过”“已自动解禁”“已处理当前入群申请”等 admission session 结果。

## 产品边界

### `/verify/<code>`

`/verify/<code>` 是“已产生入群会话后的认证闭环”。

职责：

- 校验 admission token。
- 引导登录或注册后回到同一 JOIN URL。
- 将当前 StuHelper 账号与 token 对应的 QQ 入群会话绑定。
- 校验 QQ 是否匹配。
- 进入老生认证或新生材料提交。
- 展示 pending review、projection pending、approved、invalid、expired、account mismatch、QQ mismatch 等 session 结果。
- 通过后允许表达“群内禁言会由机器人自动解除”。

不改变的业务逻辑：

- token 消费和已消费 token 的恢复策略保持现有语义。
- QQ 绑定确认仍要求用户手动输入本次入群 QQ。
- 老生认证和新生认证接口保持现有语义。
- 失效、无效、账号不匹配和 QQ 不匹配的恢复方式保持现有语义。

### `/start`

`/start` 是“无验证码入群前准备 / 自助补齐账号条件”。

职责：

- 在 `join.stuhelper.com` 内提供登录、学生认证和 QQ 绑定。
- 不接收、不消费 admission token。
- 不信任 query 里的 QQ、群号、学校或来源参数。
- 不进入或变更 admission session 状态。
- 不依赖主站 AppShell 或账号中心导航。
- 登录和注册按钮固定使用当前 canonical URL 作为 redirect。

推荐路由：

```text
https://join.stuhelper.com/start
```

对应本地：

```text
http://join.localhost:3000/start
```

主站路径 `https://stuhelper.com/start` 必须返回 404，避免把 `/start` 变成主站业务页。

## JOIN UI 设计

### 视觉方向

JOIN 是“认证服务窗口”，不是营销落地页。视觉应具备：

- 正式、可信、清晰、移动优先。
- 第一屏直接是可操作流程，不做 hero 宣传页。
- 避免主站完整导航，降低用户在 QQ 内置浏览器或手机浏览器里的认知负担。
- 使用紧凑而明确的步骤结构，减少大段说明。
- 每个主操作按钮触控高度不低于 44px。
- 错误信息必须包含原因和恢复路径。
- 不使用单一深蓝/灰色的一-note 调色，应在克制基础上加入明确的状态色：成功、警告、危险、处理中。
- 不使用装饰性渐变球、卡片套卡片或营销式分屏 hero。

### `/verify/<code>` 结构

页面分为四层：

1. 顶部上下文：StuHelper、入群身份认证、目标 QQ、当前状态摘要。
2. 步骤条：打开链接、登录账号、确认 QQ、完成学生认证、等待处理/完成。
3. 当前动作区：根据状态展示登录、确认绑定、老生认证、新生认证、等待审核或错误恢复。
4. 帮助区：管理员重发命令、复制按钮、错误恢复提示。

当前大组件 `AdmissionPage.vue` 保持状态机主控，但抽出至少三类纯展示/表单子组件：

- `AdmissionShell`：JOIN 页面外壳、标题、状态摘要、主要内容容器。
- `AdmissionStatePanel`：loading、needsLogin、ready、approved、invalid、expired、mismatch 等通用状态面板。
- `AdmissionReissueHint`：管理员重发命令与复制按钮。

抽取后不改变状态来源、API 调用顺序和 session 语义。

### `/start` 结构

新增 `JoinStartPage.vue`，内部自管登录，而不是使用 `requiresAuth` 全局守卫。

状态模型：

| 状态 | 展示 |
| --- | --- |
| `loading` | 检查登录、学生认证、QQ 绑定状态 |
| `anonymous` | 登录 / 注册按钮，说明会回到当前页面 |
| `studentRequired` | 嵌入式学生认证面板 |
| `qqRequired` | 嵌入式 QQ 绑定面板 |
| `complete` | 学生认证与 QQ 绑定均完成 |
| `error` | 加载失败，提供重试 |

`/start` 默认顺序：

1. 登录。
2. 学生认证。
3. QQ 绑定。
4. 完成。

理由是 QQ 绑定成功提示可以明确告诉用户“账号条件已完成；后续加入受控群时可被识别”。

### 可复用组件边界

现有 `StudentVerificationPage.vue` 和 `QQBindingPage.vue` 是页面级组件，带主站返回按钮和账号中心语境，不能直接嵌入 `/start`。

需要抽取：

- `StudentVerificationPanel.vue`
  - 输入：`embedded`、`showBack`、`redirectAfterVerified` 或等价控制。
  - 继续使用 `useVerificationStore`、学校列表、邮箱 OTP 和人工认证逻辑。
  - 在 embedded 模式下不显示主站返回按钮，不跳转主站账号中心。

- `QQBindingPanel.vue`
  - 输入：`embedded`、`showBack` 或等价控制。
  - 继续使用 `useVerificationStore.createQQBindingCode()`、`fetchQQBinding()` 和轮询。
  - 保留机器人入口缺失提示、绑定命令复制、状态刷新。
  - 在 embedded 模式下不显示主站返回按钮。

现有主站页面继续复用 panel 并保留原页面外壳。

## Admin 设计

Admin 是 admission policy 的权威源。页面必须强化“修改这里才会影响 Koishi 目标群同步”的事实。

### 入群认证策略页

将长表单改为分区信息架构：

1. 策略头部
   - 平台、目标认证群号、启用状态、同步影响摘要。
   - 明确“目标认证群”是 Koishi post-join guard 的执行目标。

2. 入群处理
   - `guardEnabled`
   - `joinHandlingStrategy`
   - `unverifiedJoinRejectReason`
   - 策略文案解释：
     - `post_join_guard`：入群后禁言并要求认证。
     - `join_request_review`：申请时审核，不启用 post-join guard binding。

3. 时间与提醒
   - `initialMuteDurationSeconds`
   - `linkWaitSeconds`
   - `submissionWaitSeconds`
   - `manualReviewTimeoutSeconds`
   - `reminderIntervalSeconds`

4. 新生材料与审核通知
   - `freshmanChannelEnabled`
   - `freshmanChannelClosesAt`
   - `freshmanDefaultExpiresAt`
   - `managementGuildIDs`
   - `forwardRawMaterialToQQ`
   - 明确 `managementGuildIDs` 是材料审核通知群，不是目标认证群。

5. 失败与拉黑
   - `failedJoinLimit`
   - `blacklistDurationSeconds`
   - `maxExtensionDays`
   - `maxMaterialBytes`

6. 保存前影响摘要
   - 保存按钮附近展示将影响的目标群、启用状态、入群处理策略、审核通知群数量。
   - 保存成功后提示“Koishi 会在下次同步后显示执行态”。

不新增 SQL 直接编辑入口。管理员手动可视化编辑应通过现有 Admin API 和后端服务层完成。

### 入群认证会话页

保持现有筛选、复制链接、复制重发命令、重发、重建、取消能力。

增强方向：

- 状态列使用操作语义而不是裸状态。
- 操作失败显示具体恢复路径。
- `lastBotError` 和 `authURL` 的诊断信息更明显。
- 与 Koishi WebUI 的现场队列保持概念一致：Admin 是后端 session，Koishi 是本地 guard record。

## Koishi WebUI 设计

Koishi WebUI 是执行态和现场处置台，不是策略权威源。

### 页面结构

Admission 运行页分为五个区域：

1. 运行健康
   - 后端 API base URL。
   - service token 是否配置。
   - Bot 平台、selfId、状态。
   - Action Stream 状态。
   - 兜底扫描状态。
   - 最后生成/同步时间。

2. 运行开关
   - Action Stream。
   - 兜底扫描。
   - 公开命令。
   - 群审命令。
   - 准入命令。
   - 消息风控。
   - 材料转发。
   - 群内提醒。
   - 私聊提醒。

3. 同步目标群
   - 只读展示 Admin policy 同步来的 binding。
   - 展示 enabled / disabled、停用原因、更新时间。
   - 明确提示“目标认证群请在 Admin 后台入群认证策略中修改”。

4. 模板
   - 仍可跳转配置治理工作区编辑 Koishi 本地模板。
   - 说明模板只影响 Koishi 执行动作，不改变后端 admission policy。

5. 受限成员队列
   - 显示 active guard record。
   - 展示 QQ、群、verification state、backend pending、deadline、admission session、last error。
   - 操作：查询、重发、重建、跳过、清次数、解拉黑。

### 运行开关约束

提醒通道有两个独立开关：

- 群内提醒。
- 私聊提醒。

产品约束：至少开启一个。

如果用户尝试同时关闭两个，WebUI 应阻止保存并提示原因。后端/Koishi runtime settings 层也应有等价保护，避免绕过 UI 造成“有人入群但完全不会收到提醒”的配置。

私聊提醒语义：

- 已是好友时发送好友私聊。
- 非好友时使用 QQ 临时会话。
- OneBot/NapCat 不支持或失败时，记录可读错误，不阻塞其他已启用提醒渠道成功。

### 动作反馈

WebUI 成员动作要区分三类结果：

- 成功：动作完成，队列刷新。
- 部分成功：例如跳过已标记本地释放，但 QQ 解除禁言失败。此时不能显示为整体失败，应显示“已跳过，自动解除禁言失败”，并保留错误详情。
- 失败：后端 session、Koishi 本地记录或 QQ 操作均未达到预期。

当前 `skip` 对 `set_group_ban duration=0 retcode 1200` 已有容错，UI 需保持并更清楚表达这类部分成功。

## 数据流

### `/verify/<code>`

```text
QQ 入群事件
  -> Koishi 创建/同步 admission session
  -> 后端生成 authURL: join.stuhelper.com/verify/<code>
  -> 用户打开 JOIN URL
  -> Web 校验 token
  -> 用户登录/注册
  -> Web link admission session
  -> 老生认证或新生材料
  -> 后端推进 session 状态
  -> Koishi action stream / fallback scan 执行解除禁言或后续动作
```

### `/start`

```text
用户打开 join.stuhelper.com/start
  -> Web bootstrap 登录态
  -> 未登录则 SSO 登录/注册并回到 /start
  -> fetch verification status
  -> 未学生认证则提交学生认证
  -> 未 QQ 绑定则生成绑定码
  -> 用户私聊机器人绑定 code
  -> Web 轮询 QQ binding
  -> 完成账号级准备
```

`/start` 不创建 admission session，不消费 admission token，不触发 Koishi 解禁。

### Admin -> Backend -> Koishi

```text
Admin 保存 admission policy
  -> 后端 service/repository 持久化 policy
  -> Koishi 定期/启动时调用 /api/v1/bot/admission/policies/targets
  -> syncGuardPolicyFromAdmissionTargets 写入 Koishi guard binding cache
  -> Koishi WebUI 只读展示 binding
```

权威边界：

- 目标群增删、启停、入群处理策略：Admin / 后端是权威源。
- Koishi guard binding：执行缓存，只读展示同步结果。
- Koishi runtime switches：Koishi 本地运行态权威源。
- Admission session：后端是 session 权威源。
- Guard member active queue：Koishi 本地执行态权威源。

## API 与基础设施变更范围

### 前端路由

新增：

```ts
{
  path: "/start",
  name: "join-self-service-start",
  component: lazyLoad(() => import("@/modules/admission/views/JoinStartPage.vue")),
  meta: { title: "入群准备", layout: "none" },
}
```

`/start` 不使用 `requiresAuth`，页面内部调用 auth store 登录/注册。

### JOIN 域隔离

`clients/web/src/router/join-domain.ts` 需要区分：

- admission token path：`/verify/<code>`、移动拍照。
- self-service path：`/start`。

约束：

- `join.stuhelper.com/start` 放行。
- `join.stuhelper.com/start/` 放行或规范化。
- `join.stuhelper.com/` 仍 404。
- `join.stuhelper.com/courses` 仍 404。
- `join.stuhelper.com/user/student-verification` 仍 404。
- `stuhelper.com/start` 404。
- `stuhelper.com/verify/*` 继续 404。

### Nginx / ingress

需要同步：

- `infra/nginx/baota-stuhelper.conf`
- `infra/nginx/prod-parity-local-ingress.conf`
- `clients/web/nginx.conf`
- `infra/ops/install-local-prod-parity-ingress.sh`
- `infra/ops/nginx-public-ingress-preflight.sh`
- `infra/ops/admission-public-smoke.sh`
- `infra/ops/tests/*` 中相关 contract

JOIN 域新增：

```nginx
location = /start {
    proxy_pass http://127.0.0.1:18000;
}

location = /start/ {
    proxy_pass http://127.0.0.1:18000;
}
```

其余主站业务路径仍返回 404。

### 后端

第一版 `/start` 不新增后端 API。

复用：

- `GET /api/v1/auth/login`
- `GET /api/v1/auth/signup`
- `GET /api/v1/auth/me`
- `GET /api/v1/user/profile`
- `POST /api/v1/user/profile/verify`
- `GET /api/v1/user/schools`
- `GET /api/v1/user/qq-binding`
- `POST /api/v1/user/qq-binding/code`
- Koishi 既有 QQ binding consume API

如果 UI 发现现有 API 无法表达某个必要状态，应先更新 OpenAPI，再按生成链路改实现。

## E2E 与测试矩阵

### Web JOIN 单元测试

新增或更新：

- `join-domain.test.ts`
  - join host 放行 `/start`。
  - main host 阻断 `/start`。
  - join host 继续阻断主站业务路径。
- `JoinStartPage` 单测
  - anonymous 显示登录/注册。
  - 登录按钮使用当前 canonical URL。
  - studentRequired 渲染 embedded student panel。
  - qqRequired 渲染 embedded QQ binding panel。
  - complete 显示学生认证和 QQ 绑定完成。
  - error 状态可重试。
- `StudentVerificationPanel` 单测
  - embedded 模式不显示返回按钮。
  - 主站页面模式保留返回按钮。
  - 邮箱 OTP / manual / pending / verified 语义保持。
- `QQBindingPanel` 单测
  - embedded 模式不显示返回按钮。
  - 生成绑定码后开始轮询。
  - 轮询到绑定后停止轮询。
  - 机器人入口缺失提示保留。

### Web JOIN Playwright

新增：

- `join.localhost/start` 未登录显示登录/注册。
- `/start` 登录后回到同一 URL。
- 未学生认证用户可完成学生认证并停留在 JOIN 域。
- 未 QQ 绑定用户可生成绑定码，模拟绑定后页面轮询到 complete。
- 已学生认证且已 QQ 绑定直接 complete。
- `join.localhost/`、`join.localhost/courses`、`join.localhost/user/student-verification` 仍 404。
- main host `/start` 仍 404。

更新：

- `/verify/<code>` 视觉重构后继续通过现有 Playwright admission 测试。
- 增加 mobile viewport 对 `/verify` 和 `/start` 的布局断言：无水平滚动，关键按钮可见且不重叠。

### Admin 测试

新增或更新：

- 策略页分区存在性测试。
- 目标认证群和材料审核通知群的标签、说明和字段分离测试。
- 保存请求仍发送 `managementGuildIDs`、`joinHandlingStrategy`、`guardEnabled` 等现有字段。
- 保存前影响摘要能正确描述启用/停用和策略。
- 新增目标群仍支持从已有策略复制。
- 会话页操作仍支持 copy auth URL、copy reissue command、resend、regenerate、cancel。

### Koishi 单元测试

新增或更新：

- runtime settings 禁止同时关闭 `reminderGroupEnabled` 和 `reminderDirectEnabled`。
- WebUI model 暴露提醒开关约束所需状态。
- action result 区分部分成功和失败。
- policy target sync 中 `join_request_review` 继续映射为 disabled post-join guard binding。
- stale binding note 保持可读。

### Koishi Playwright

新增或更新：

- admission runtime 顶部显示 Admin 权威源和 Koishi 执行缓存说明。
- 同步目标群只读，无保存绑定按钮。
- 同步绑定显示 enabled/disabled 和 note。
- 关闭最后一个提醒通道时显示错误，不保存。
- skip 部分成功提示可见，并刷新队列。
- 运行健康区显示 service token 配置、Bot、Action Stream、兜底扫描。

### 跨系统契约 / E2E

新增或增强：

- Admin policy target API 返回目标群状态后，Koishi `syncGuardPolicyFromAdmissionTargets` 映射结果符合预期。
- Admin 保存 `743762161 guardEnabled=false` 后，Koishi binding 为 disabled。
- Admin 保存 `178037297 guardEnabled=true` 后，Koishi binding 为 enabled。
- `join_request_review` 不启用 post-join guard binding。
- public smoke 覆盖 `join.stuhelper.com/start` 可加载，且主站 `/start` 404。
- prod-parity browser smoke 覆盖 `/start` 页面非空、登录按钮 redirect 正确。

## 代码质量计划

本次质量整改只围绕已审查范围和当前已知失败展开。

允许的改进：

- 拆分过大的 Vue 页面组件。
- 抽出可复用 panel 和状态展示组件。
- 用明确类型替代重复字符串判断。
- 为 admission action result 增强可读错误映射。
- 修复测试 mock 漂移。
- 修复现有评分展示策略测试失败，但必须遵守“不改变原始业务逻辑和效果”的约束：如果产品策略确认不显示精确评分，则改 UI 展示；如果策略测试过期，则更新测试前必须先确认权威产品文档。本设计默认当前测试是有效业务策略。

不允许的改动：

- 手改生成代码。
- 绕过 OpenAPI 增加 API 字段。
- 在 Koishi 直接写后端策略数据库。
- 把 Admin 和 Koishi 各自保存一份目标群权威数据。
- 让 `/start` 隐式变更 admission session。
- 为了让测试变绿而移除业务断言。

## 实施顺序

1. JOIN 基础入口
   - 更新路由和 JOIN 域隔离。
   - 抽出 `StudentVerificationPanel` 和 `QQBindingPanel`。
   - 新增 `JoinStartPage`。
   - 补 `/start` 单测和 Playwright。

2. `/verify` UI 重构
   - 抽出 shell、state panel、reissue hint。
   - 保持状态机和 API 调用顺序。
   - 补移动端和错误态视觉测试。

3. Admin policy 信息架构
   - 分区重构策略页。
   - 增加保存影响摘要。
   - 保持现有 API payload。
   - 补组件和契约测试。

4. Koishi WebUI admission 重构
   - 强化运行健康、同步来源、只读 binding 和部分成功反馈。
   - 增加提醒开关至少一个启用的约束。
   - 补单测和 Console Playwright。

5. 跨系统和运维 smoke
   - 更新 Nginx、prod-parity、public smoke。
   - 补 Admin -> Backend -> Koishi 同步契约。
   - 复跑 Web/Admin/Koishi 关键 E2E。

6. 质量整改
   - 修复 auth mock 漂移。
   - 处理评分展示策略测试失败。
   - 复跑相关单测和 E2E。

## 验收标准

完成后需要同时满足：

- `join.stuhelper.com/start` 可用，`stuhelper.com/start` 404。
- `/start` 可完成登录回跳、学生认证、QQ 绑定和 complete 状态。
- `/verify/<code>` 原业务流程不回退。
- JOIN 页面移动端和桌面端无明显重叠、无水平滚动、关键按钮满足触控尺寸。
- Admin 策略页能清楚区分目标认证群和材料审核通知群。
- Koishi WebUI 清楚表达 Admin 权威源与 Koishi 执行缓存。
- Koishi 目标群同步与 Admin policy target 一致。
- 提醒渠道不能全部关闭。
- WebUI skip 的部分成功可读。
- 新增和既有 E2E 能覆盖上述业务路径。
- 当前已知 Web 单测失败得到处理。
- 没有手改生成代码。
- 没有改变 admission token、session、QQ 绑定、学生认证的业务语义。

## 风险与回滚

风险：

- `/start` 新增 JOIN 域入口会影响 ingress 门禁，需要同步更新生产、prod-parity 和 smoke。
- 抽出 panel 可能影响主站学生认证和 QQ 绑定页面。
- Admin 页面重构可能导致保存 payload 漏字段。
- Koishi runtime settings 增加“至少一个提醒通道”约束会改变当前允许全部关闭的测试预期，但符合产品要求。

缓解：

- 先补单测，再改 UI。
- panel 抽取后主站页面和 `/start` 都跑测试。
- Admin 保存 payload 用单测固定。
- Koishi runtime settings 约束在 store/API 层和 UI 层都测试。
- ingress 改动必须跑 contract 和 smoke。

回滚：

- `/start` 路由和 ingress 可独立回滚，不影响 `/verify/<code>`。
- Admin 页面重构保持 API payload，不需要后端数据回滚。
- Koishi WebUI 页面调整不改变已有 SQLite schema；提醒约束如需回滚，可恢复允许全部关闭，但必须同时接受产品风险。

## 后续扩展

如果后续要让“已入群但丢失验证码的人”在 `/start` 完成认证和 QQ 绑定后自动处理已有入群会话，需要单独设计：

- 后端按 QQ 查找 active admission sessions。
- 多 session 冲突规则。
- 用户身份和 QQ 绑定一致性校验。
- session 状态机变更。
- 审计事件。
- Koishi action stream 出队和幂等。
- 失败恢复和管理员可见诊断。

该增强不属于本设计第一版实施范围。
