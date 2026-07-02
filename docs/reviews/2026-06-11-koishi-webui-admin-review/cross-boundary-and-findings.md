# 跨面对接边界分析 + 既有 Review Findings 提取

---

## Part A — 既有 Review Findings 提取（来源：docs/reviews/2026-06-10-full-codebase-review.md）

范围：bots/koishi（插件与 WebUI）、clients/admin、koishi↔server 对接、admission/blacklist/member 相关 server 模块。共提取 **52 条**（HIGH 8 · MEDIUM 27 · LOW 17）。

### HIGH — 已对抗验证确认（5 条）

| # | 标题 | 定位 | 一句话描述 |
|---|---|---|---|
| **F001** | freshman 临时凭证过期后 user_profiles 投影永久保持 verified（P0，与 F007 同根因） | `server/internal/modules/admission/service_expiry.go:109` | 凭证过期处理只 revoke 凭证不回写 user_profiles，过期新生永久享有已认证学生权限（评课访问、入群自动通过） |
| **F007** | 新生凭证过期后 join decision 的 profile 回退分支架空过期机制（P0） | `server/internal/modules/admission/repository_join_decision.go:37` | GetVerifiedAdmissionUserByQQ 的 UNION 第二分支以 verification_status='verified' 为准且无过期条件，凭证分支失效后 profile 分支永久命中 |
| **F004** | moderation 消息台账无限增长 + 每条群消息全表拉取内存排序（P1） | `bots/koishi/packages/moderation-core/src/store.ts:83` | saveMessage 永久存全文无 TTL，listRecentMessages 每条消息 O(N) 全表载入排序，活跃群数月后必然性能崩塌 |
| **F005** | 群审批量禁言误用 split limit，空格分隔的成员 ID 被静默丢弃（P1） | `bots/koishi/plugins/stuhelper-admin/src/commands.ts:320` | `source.split(/\s+/, 2)` 截断而非合并剩余部分，'120 10011 10012' 只禁言第一人且无提示 |
| **F006** | guild-member-request/added 事件处理器未捕获异常（P1） | `bots/koishi/plugins/stuhelper-group-guard/src/events.ts:21` | cordis 忽略监听器返回的 Promise，后端 5xx/超时时入群请求既不批准也不拒绝、无重试、仅成为 unhandledRejection |

### HIGH — 未验证（验证代理被网关终止，建议人工复核后按 P1 处理，3 条）

| # | 标题 | 定位 | 一句话描述 |
|---|---|---|---|
| **F019** | 管理命令私聊/无 guildId 时绕过命令策略，可跨群读全部待复核队列 | `bots/koishi/plugins/stuhelper-admin/src/command-access.ts:29` | `if (!session \|\| !guildId) return` 放行 + store 对空 guildId 查全表，authority≥3 用户私聊即可越权读取所有群复核数据 |
| **F020** | 新生申请未校验目标学校与会话所属学校一致，可跨校签发凭证 | `server/internal/modules/admission/service_freshman.go:37` | 全程不校验 policy.SchoolID == input.SchoolID，A 校管理群操作员可为 B 校签发学生凭证 |
| **F023** | typed-nil 接口使 bot 凭证校验器 nil 守卫失效，触发可远程引发的 panic | `server/internal/app/modules.go:332` | BOT_SERVICE_TOKEN 为空时返回 (*Verifier)(nil) 装入接口非 nil，任何带 Bearer 头的 /api/v1/bot/** 请求在非生产环境触发 nil 解引用 panic |

### MEDIUM（27 条）

**Server · admission/blacklist/member 域（9 条）**

| # | 标题 | 定位 | 描述 |
|---|---|---|---|
| F058 | admission_bot_action_outbox 无清理机制 | `admission/repository_bot_action_outbox.go:34` | 每个未完成会话约产生 96 条 remind 行，succeeded/stale/dead_letter 行无任何 DELETE/归档，表无限膨胀 |
| F059 | 过期 worker 无并发声明语义 | `admission/repository_expiry.go:17` | 无 FOR UPDATE SKIP LOCKED，多实例重复处理 freshman 过期；blacklist 过期遇 NotFound 整批中断 |
| F060 | 入群审批决策不检查 member blacklist | `admission/service_queries.go:150` | ResolveJoinRequestDecision 不查 member_blacklist_entries，join_request_review 策略下黑名单已验证用户被自动批准入群 |
| F061 | 凭证学校与会话学校不一致时返回内存伪造的 verified 状态 | `admission/service_school_sso.go:160` | 跨校验证时 DB 会话仍 linked 但返回对象被强改 verified，用户到期被踢出且无提示 |
| F062 | auth_url 列持久化明文 join token，token_hash 防护失效 | `admission/service_session.go:1047` | 同表同行存 HMAC 哈希与含明文 token 的 auth_url，DB 泄露即可把任意 QQ 绑定到攻击者账号 |
| F033 | 投影 upsert 覆盖 school_id 但保留旧 active_student_id | `admission/repository_verified_profile.go:53` | 跨校重新认证导致 A 校学号错挂 B 校，且可能触发唯一索引冲突使认证事务失败 |
| F072 | admission 路径 5 个操作的 OpenAPI security 声明与实现相反 | `server/api/paths/admission.yaml:260` | mobile-camera-handoffs 三端点漏标 `security: []`，school-sso login/callback 错标 `security: []`，契约测试只比对路径+方法捕获不到 |
| F038 | seed.sql 含无条件 DELETE 与真实 QQ 群策略覆盖 | `server/scripts/seed.sql:392` | 误指生产库会清空评分统计表并用开发参数覆盖真实群 178037297 的入群策略 |
| F032 | migrate-verify 对目标库做破坏性 down+up（涉及 admission 迁移） | `server/Makefile:160` | 000015 down 丢弃 join_handling_strategy 与自定义拒绝文案、000013 down 静默关闭用户 SMS MFA |

**Koishi 机器人（10 条）**

| # | 标题 | 定位 | 描述 |
|---|---|---|---|
| F046 | incrementWarning 读-改-写无原子性 | `moderation-core/src/store.ts:97` | 并发命中丢失告警计数，延迟自动禁言升级 |
| F047 | warningThresholdExpression 保存时不校验可解析性 | `packages/shared/src/guard/behavior-settings.ts:231` | 非法表达式使每次命中处理抛错，自动禁言静默失效 |
| F048 | 关键词正则启发式拦不住重叠交替型 ReDoS | `packages/shared/src/keyword-pattern.ts:3` | `(a\|a)+$` 类规则可阻塞整个事件循环；存量非法规则会让该群消息处理整体抛错 |
| F049 | 运行时设置查询无缓存，每条消息打 DB | `plugins/stuhelper-binding/src/index.ts:31` | binding 中间件 + group-guard 事件每条消息各一次 DB 读，叠加关键词规则全表查询 |
| F050 | console guild scope 无角色时 fail-open 为全局权限 | `plugins/stuhelper-core/src/core/api/console-guild-scope.ts:44` | 无角色或角色 guildIds 为空均返回 'all'，移除受限角色反而扩大可见范围 |
| F051 | regenerate/已验证释放缺少 skip 已有的 unmute 失败容忍 | `plugins/stuhelper-group-guard/src/admission-console-api.ts:340` | unmute 在 markBackendSynced 之前且无 try/catch，失败后本地永不更新新 session |
| F052 | freshman 材料转发队列毒丸阻塞 + 部分失败重复转发 | `plugins/stuhelper-group-guard/src/member-guard.ts:412` | 单 item 失败中断整个循环且永远排队头；部分群成功时下轮重复刷屏 |
| F053 | canExecuteCommand 无策略记录时默认允许 | `packages/moderation-core/src/access.ts:10` | stuhelper-admin 无默认策略回退，与 group-guard 的 fail-closed 不一致 |
| F054 | stuhelper-admin 命令策略缺 guildId 时 fail-open | `plugins/stuhelper-admin/src/command-access.ts:29` | 与 F019 同位置，仅靠 Koishi authority 兜底，遗漏 authority 选项即成越权路径 |
| F055 | QQ 绑定码消费缺少 bot 侧限速与尝试上限 | `plugins/stuhelper-binding/src/index.ts:31` | 任意私聊消息直接调用 consumeQQBindingCode，400→「无效验证码」提供清晰爆破 oracle |

**clients/admin（7 条）**

| # | 标题 | 定位 | 描述 |
|---|---|---|---|
| F025 | 学生认证列表对非数字 schoolId 静默丢弃过滤条件 | `web-ele/src/api/admin/user-system.ts:42` | NaN 时省略 schoolID，受限管理员的范围过滤静默失效退化为全量查询 |
| F026 | step-up 完全被动依赖后端，跳转失败破坏返回契约 | `web-ele/src/api/shared-client.ts:168` | 后端误配 development 即无任何二次验证；redirectToStepUp 异常突破 ApiCallResult 契约外抛 |
| F027 | step-up 错误文案仅匹配 412，与传输层 412/428 双状态不一致 | `web-ele/src/api/shared-result.ts:36` | 状态码集合与错误码 A0010205 在两个文件各自硬编码，已漂移 |
| F028 | 教师表单 departmentID 必填校验被空字符串绕过 | `web-ele/src/views/content/teachers/index.vue:135` | `v-model.number` 清空后保留 ''，绕过 `=== null` 校验，向后端发非法值 |
| F029 | 批量创建入群策略无部分失败处理 | `web-ele/src/views/users/admission-policy/index.vue:214` | 中途失败时已创建群号会被重复提交，错误不指明失败群号 |
| F030 | admission-policy 客户端默认值在保存时写回后端 | `web-ele/src/views/users/admission-policy/index.vue:100` | 前端硬编码中文默认文案随全量保存静默覆盖后端存量数据 |
| F031 | 新生审核「通过」按钮无确认步骤 | `web-ele/src/views/users/freshman-verification/index.vue:317` | 误点一次即通过；「带天数通过」天数为 0 时静默退化为普通通过 |

**横切（1 条）**

| # | 标题 | 定位 | 描述 |
|---|---|---|---|
| F045 | SAST 完全未覆盖 bots/koishi（与 uniappx） | `tools/semgrep/stuhelper-security.yml:2` + `.gitlab-ci.yml:224` | 处理 QQ 绑定/新生材料转发的高敏感代码无任何 SAST 扫描 |

### LOW（17 条）

**Server · admission（4 条）**：F130 policy 数值校验仅靠 DB CHECK 兜底返回 500（`handler_admin_queries.go:35`）；F131 camera handoff SSE 每连接 1s×3 查询、bot action stream 每 2s 写事务（`handler_user.go:208`）；F132 批量声明动作单行错误丢弃整批并消耗重试（`service_bot_actions.go:50`）；F112 000007 down 重建的 admission 会话唯一索引谓词与基线漂移（`migrations/000007_*.down.sql:3`）。

**Koishi（6 条）**：F119 运行期新增 bot 不会获得 action stream（`admission-action-stream.ts:35`）；F120 提醒私聊投递失败完全静默（`admission-reminder-delivery.ts:125`）；F121 群审 AI apiKey 明文存 sqlite（`packages/shared/src/guard/ai-settings.ts:93`）；F122 绑定日志记录 qqID↔userID PII 映射（`stuhelper-binding/src/index.ts:51`）；F159 koishi e2e 共享 worker 级 page 状态耦合（`e2e/fixtures/auth.ts:32`）；G8 .yarn/patches 依赖补丁未纳入审查范围。

**clients/admin（7 条）**：F103 batchUpdateReviews reason 被丢弃且无调用方（`api/admin/content.ts:40`）；F104 resolveOIDCRedirectURL 对绝对 URL 原样放行（`api/core/auth.ts:98`）；F105 路由守卫裸 console.warn + 引导失败白屏（`router/guard.ts:127`）；F106 views/content/logs 死代码（`views/content/logs/index.vue:1`）；F107 敏感词创建未 trim、空 category 绕过后端默认值（`views/content/sensitive-words/index.vue:127`）；F108 oxlint 2 个 error 级非空断言阻塞 lint（`admission-policy/index.vue:120`）；F109 admission-policy/freshman-verification/member-blacklist 等新页面硬编码中文绕过 i18n（`admission-policy/index.vue:50`）。

---

## Part B — 跨面对接边界分析

### 链路 1：koishi 插件 ↔ Go server

**契约现状**

- **客户端**：`bots/koishi/packages/shared/src/platform/index.ts`（496 行）手写 fetch 客户端 + `freshman-client.ts`。22 个方法覆盖 `/api/v1/bot/**` 全部 24 条端点（路径常量在 index.ts:35-47），路径集合与 `server/api/openapi.yaml:398-446` **逐条对得上**，无幽灵端点。
- **认证**：单一静态 Bearer service token（index.ts:469-473 `withAuthHeaders`），支持 `${{ env.X }}` 占位符与 `STUHELPER_PLATFORM_BASE_URL/SERVICE_TOKEN` 环境变量回退（index.ts:428-445），启动时 fail-fast（assertPlatformConfig，index.ts:414-426）。服务端为 `serviceaccount.Verifier`（HMAC），按 scope 分组鉴权（`server/internal/platform/serviceaccount/constants.go:8-15`，凭证名 `koishi-runtime` 见 `server/internal/pkg/botcredential/botcredential.go:6`）。OpenAPI 侧 bot 路径统一声明 `serviceTokenAuth`，与实现一致。**F023 的 typed-nil panic 正处于这条认证接缝**。
- **错误处理**：koishi 侧自定义 `PlatformAPIError(message, status, code)`（index.ts:66-75），从 `{error:{code,message}}` 信封解析（buildPlatformError，index.ts:482-487）——与服务端响应信封语义一致，但 `APIEnvelope` 在 index.ts:60-64 **本地重新定义**，未与 clients/shared 的 `ApiEnvelope`（`clients/shared/src/api/result.ts`）共享。
- **超时**：统一 8s `AbortSignal.timeout`（index.ts:50, 475-480），SSE 流除外。

**脱节/重复证据（手写类型 vs 契约）**

koishi 类型 100% 手写：`bots/koishi/packages/shared/src/types/index.ts`（470 行）+ `member-blacklist.ts`（122 行），共 592 行，与 `server/api/components/schemas/{admission,member-blacklist,user-system}.yaml` 重复维护。已发现的具体漂移：

| 字段 | OpenAPI（事实源） | koishi 手写类型 | 证据 |
|---|---|---|---|
| `AdmissionSession.userID` | `type: string` | `number \| string \| null` | `admission.yaml:52-53` vs `types/index.ts:315` |
| `AdmissionSession` 字段集 | 含 `botSelfID`、`tokenConsumedAt`、`verifiedAt`、`cancelledAt` | 四个字段全部缺失 | `admission.yaml:26-89` vs `types/index.ts:309-330` |
| `FreshmanApplication.userID` | `type: string`（required） | `number \| string` | `admission.yaml:241-242` vs `types/index.ts:436` |
| `AdmissionJoinRequestDecision.userID` | — | `number \| string \| null` 防御性联合 | `types/index.ts:401` |

这类 `number | string | null` 防御性联合正是「不信任自己手写类型」的典型症状。而**生成类型其实已经存在**：`clients/shared/src/types/api.gen.ts:2726-2899` 由 `openapi-typescript ../server/api/openapi.bundled.yaml` 生成（`clients/package.json:24`），完整包含全部 bot 端点——koishi 没有复用它的原因是 workspace 隔离：clients 是 pnpm workspace，bots/koishi 是独立 yarn berry workspace，`bots/koishi/node_modules/@stuhelper/` 下只有 `koishi-shared`、`koishi-moderation-core` 两个内部包的 symlink，无 `@stuhelper/shared`。

### 链路 2：clients/admin ↔ Go server

**契约现状：这是三条链路中最健康的一条。**

- 类型 100% 来自生成契约：admin API 模块（`web-ele/src/api/admin/admission.ts:1-17`、`user-system.ts:1-6` 等）从 `@stuhelper/shared/api` 导入 `createAdmissionApi`/`createUserAdminApi` 工厂和类型；工厂内部用 `components['schemas'][...]` 与 `operations[...]`（`clients/shared/src/api/admission.ts:1-35`）直接绑定 api.gen.ts。
- 传输层复用 shared 的 `createSessionApiClient`（`web-ele/src/api/shared-client.ts:14, 253-258`），在其上叠加 admin 特有的 step-up/MFA 重定向处理。
- 漂移防护：`check:api-drift`（`clients/package.json:25`）保证 api.gen.ts 与 openapi.bundled.yaml 同步。

**残余问题**（均已收录于 Part A）：F027 step-up 错误码 `A0010205` 与状态码集合在 `shared-client.ts:33-37` 与 `shared-result.ts:36` 各自硬编码已漂移；F103 `batchUpdateReviews` 签名声明了契约中不存在的 `reason` 参数；F072 的 security 声明漂移会误导基于 spec 的客户端生成。

### 链路 3：server → koishi 反向通道

**server 从不主动连接 koishi，不存在 webhook/回调。** 反向投递由三层机制组成，全部建立在 bot 主动发起的连接上：

1. **持久化队列**：`admission_bot_action_outbox`（`server/internal/modules/admission/repository_bot_action_outbox.go`），FOR UPDATE SKIP LOCKED 声明 + 30s dispatch 超时重派 + attempt 上限 dead_letter + 幂等 action_key。
2. **SSE 下行流**：bot GET `/api/v1/bot/admission/actions/stream`，服务端 `handleStreamBotAdmissionActions`（`handler_bot_queries.go:110-150`）以 2s ticker claim + `event: action` 推送 + 15s keepalive；契约在 `server/api/paths/bot-admission.yaml:345-375` 有明文描述（含 ACK 义务）。koishi 侧用 fetch + **手写 SSE 帧解析器**消费（`platform/index.ts:330-382`），ACK 经 POST `/actions/{id}/events`。
3. **兜底轮询**：`/sessions/pending` + `/actions/claim`（fallbackScan）。

**该通道的已知缺陷**（Part A 已列）：F119（新增 bot 不建流）、F131（无动作也产生 claim 写事务）、F132（单行坏数据拖累整批进死信）、F006（事件路径异常未捕获）。另外手写 SSE 解析器没有任何契约测试锁定 `event: action`/`keepalive`/`error` 三种事件语义，服务端改事件名两侧都不会报错。

### 共享类型复用结论

| 消费方 | 是否复用 @stuhelper/shared 生成类型 | 重复定义位置 |
|---|---|---|
| clients/web | ✅（api.gen.ts + API 工厂） | — |
| clients/admin | ✅（同上，经 createSessionApiClient） | 仅错误码常量重复（shared-client.ts:33-37 vs shared-result.ts:36） |
| bots/koishi | ❌ **完全不复用** | `bots/koishi/packages/shared/src/types/index.ts`（470 行）+ `member-blacklist.ts`（122 行）整体重复 openapi schemas；信封类型在 `platform/index.ts:60-64` 第三次定义 |

### 统一方案建议

1. **koishi 类型改为生成**（最高价值）：在 bots/koishi workspace 增加 `api:generate` 脚本，用 openapi-typescript 从 `server/api/openapi.bundled.yaml` 生成到 `packages/shared/src/types/api.gen.ts`，手写类型改为 `type AdmissionSession = components['schemas']['AdmissionSession']` 别名后逐步删除；CI 增加与 clients 对等的 `check:api-drift` 守卫。由于两个 workspace 包管理器不同（pnpm vs yarn berry），按 workspace 各自从同一 bundled spec 生成 + drift 检查，比强行跨 workspace 共享包更稳。生成后 `number|string|null` 防御性联合与缺失字段问题自然消除。
2. **security 声明契约测试**（采纳 F072 建议）：在 `openapi_route_contract_test.go` 增加断言「spec 中 `security: []` 的操作对应路由不得挂 authMW，反之亦然」，同时覆盖 bot 路径的 serviceTokenAuth 与实现的 verifier 中间件一致性——并顺手修 F023 的 typed-nil 装配。
3. **SSE 下行流契约锁定**：为 koishi 的 `dispatchAdmissionActionStreamEvent` 与服务端 `handleStreamBotAdmissionActions` 各补一条事件名/payload 结构的契约测试（事件名常量可放进生成物或共享常量文件），避免双侧手写协议静默漂移。
4. **错误码常量单一来源**：把 `A0010205` 等 step-up/MFA/CSRF 错误码收敛到 `@stuhelper/shared` 的 error-codes 模块，admin 的 shared-client.ts 与 shared-result.ts 统一引用（修 F027）。
5. **SAST 覆盖补齐**（F045）：将 `bots/koishi` 加入 frontend_sast 扫描路径，作为对接面统一治理的一部分。
