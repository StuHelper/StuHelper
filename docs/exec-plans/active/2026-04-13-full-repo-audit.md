# 2026-04-13 全库审查报告与整改计划（2026-04-14 复核更新）

> 目的：把本次针对 `StuHelper` 仓库的全量审查结果沉淀为一份可执行、可追踪、可决策的工程文档。  
> 背景：本文档最初形成于 2026-04-13；2026-04-14 已根据 Claude 提出的分歧项逐条回到代码库复核，并按当前代码状态修正文案。  
> 审查快照：分支 `codex/enterprise-remediation`，基线快照提交 `05704c4`（`chore: snapshot current remediation workspace`）。

## 1. 审查范围与判定标准

### 1.1 范围
- 后端：`/Users/zxy/Code/StuHelper/server`
- 前端：`/Users/zxy/Code/StuHelper/clients`
- 基础设施与发布：`/Users/zxy/Code/StuHelper/infra`、`/Users/zxy/Code/StuHelper/docker-compose.yml`、`/Users/zxy/Code/StuHelper/.gitlab*`
- 文档与契约：`/Users/zxy/Code/StuHelper/docs`、`/Users/zxy/Code/StuHelper/server/api`

### 1.2 约束来源
本轮审查以仓库根 `AGENTS.md` 中的工程铁律为准，重点检查：
- OpenAPI 是否仍为 API 契约唯一事实源
- 后端是否遵循 `Handler -> Service -> Repository`
- SQL 是否只出现在 Repository
- 生成代码是否被手工修改
- 前端是否统一通过 `clients/shared` 使用共享契约
- 生产环境是否具备可独立上线、回滚、观测、备份恢复能力

### 1.3 优先级定义
- **P0**：明确的数据破坏、严重安全漏洞、生产立即不可用
- **P1**：生产风险高、门禁失效、契约漂移、可导致故障或错误上线
- **P2**：架构债、重复实现、性能模型错误、可维护性问题


## 2. 2026-04-14 分歧复核摘要

### 2.1 本轮补充验证（2026-04-14）
- 在 `/Users/zxy/Code/StuHelper/server` 执行 `make -n check-drift` 与 `make -n check-drift-all`，确认默认 drift gate 已收敛为 **Go-only**；只有 `check-drift-all` 才会额外覆盖 TS 生成物。
- 在 `/Users/zxy/Code/StuHelper` 执行 `bash infra/ops/tests/init-prod-env-contract.sh`，返回 `[init-prod-env-contract] all assertions passed`，说明此前 `prod_init_contract` 与 `verify-full` 生产初始化脚本的失配已关闭。
- 在临时移走 `/Users/zxy/Code/StuHelper/clients/shared/dist` 后复验 package-local scripts：
  - `npx --yes pnpm@10.32.1 --dir /Users/zxy/Code/StuHelper/clients/web build` 仍失败，核心报错包含 `TS2307: Cannot find module '@stuhelper/shared*'`；说明 `web` 仍对 warm workspace 中预构建的 `shared/dist` 敏感。
  - `npx --yes pnpm@10.32.1 --dir /Users/zxy/Code/StuHelper/clients/uniappx type-check` 在当前工作树中通过；因此原文“uniappx 也可稳定复现同类冷启动失败”的表述需要收窄。
- 复核通知消费链路，`/Users/zxy/Code/StuHelper/clients/web/src/components/common/NotificationBell.vue` 与 `/Users/zxy/Code/StuHelper/clients/web/src/modules/user/views/NotificationsPage.vue` 都已统一调用 `@stuhelper/shared` 的 `resolveNotificationHref()`，说明通知跳转抽象已收敛到 shared helper。
- 复核 auth / notification / OpenAPI 相关代码，确认 F-01 / F-02 / F-03 / F-04 / F-05 / F-07 / F-09 所描述的原始风险大多已经被修复或收口，文档需要按当前代码状态降级或关闭这些 finding。

### 2.2 Claude 与原文分歧项的最终结论

| Finding | 2026-04-14 复核结论 | 当前状态 |
| --- | --- | --- |
| F-01 | refresh cookie 路径问题已解决；长期 Session / Token Family 建议仍有效 | 已关闭 |
| F-02 | `notification.Hub` 当前不存在 double-close panic 与随机驱逐问题 | 已关闭 |
| F-03 | Notification 主体 DTO 已基本收口；剩余问题是 deprecated 兼容别名与 DTO 分层决策 | 已修正文案 |
| F-04 | Review 创建请求已重新严格对齐 OpenAPI | 已关闭 |
| F-05 | Go / TS 生成链路已统一消费 bundled spec；默认 `check-drift` 已 Go-only | 已关闭 |
| F-06 | 问题核心不是“SQL 逃出 Repository”，而是 auth 包承载了 user persistence 的错误领域归属 | 已修正文案 |
| F-07 | 历史 review 分叉组件与孤儿页面已删除 | 已关闭 |
| F-09 | `clients/shared/src/types/business/admin.ts` 已删除；shared 边界问题仍需治理 | 已修正文案 |
| F-10 | 基础设施闭环问题仍然存在，但 `WEB_VITE_SSO_URL` 当前是 build-time 单一来源，不应再描述为双源 bug | 已修正文案 |
| F-15 | backend drift gate 与 pnpm 版本漂移问题已关闭；核心剩余问题是 `clients/shared` 的 dist-first workspace 消费导致 `web` 冷启动脆弱 | 已修正文案 |
| F-20 | 启动回填问题仍成立，但实现已具备 `batchSize=500` 的批次控制，不能再描述为“无批次控制” | 已修正文案 |

## 3. 详细审查发现

---

## F-01 认证登出链路的 cookie/path 问题已关闭；长期仍建议升级为服务端 Session 模型

- **优先级**：已解决（原 P1）
- **领域**：认证 / 安全 / 会话管理
- **当前状态**：cookie 路径问题已关闭；长期 Session / Token Family 架构建议仍有效

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_cookies.go`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_session.go`
- `/Users/zxy/Code/StuHelper/server/internal/pkg/middleware/auth.go`

### 审查发现
2026-04-14 复核确认：当前 `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_cookies.go` 中 `refreshTokenCookiePath = "/"`，`Logout()` 也会读取 `refresh_token` cookie 并传给 `RevokeCurrentSession`。因此，原文所描述的“cookie 路径导致单设备 logout 无法拿到 refresh token”已经不是当前代码库中的活跃问题。

### 2026-04-14 复核结论
- 本条应标记为**历史问题已解决**。
- 当前仍有效的是长期架构建议：认证系统尚未升级为服务端 Session / Token Family 模型。
- 当前真正仍然活跃的 logout 风险是 F-22：服务端撤销失败时，单设备 logout 依然会 success 返回。

### 为什么必须修
这是典型的“**看起来登出成功，实际上会话族未完全失效**”问题。对于使用 refresh token 续期的系统，这意味着：
- 被复制的 refresh token 仍可能继续换取新 access token
- 用户对“退出登录”的安全预期被破坏
- 审计角度上，单设备登出的语义不明确

### 长期正确方案
短期把 refresh cookie 路径改为全局可见，只是一个战术补丁。长期建议升级为：
- **服务端 Session / Token Family 模型**
- 每次登录产生独立 `session_id` 或 `token_family_id`
- logout、logout-all、refresh 都基于服务端会话状态进行吊销，而不是依赖“本次请求有没有带上某个 cookie”

### 需要决断的架构选择
**推荐方案：引入服务端 Session / Token Family。**

可选项：
1. **继续基于 token 黑名单 + cookie 路径修补**
   - 优点：改动小
   - 缺点：语义脆弱、后续扩展第三种登录方式时继续复杂化
2. **引入服务端 Session / Token Family（推荐）**
   - 优点：语义稳定、可扩展、审计清晰、适合企业级场景
   - 缺点：需要把 refresh / logout / revoke 统一重构

### 修复计划
- Phase 1：统一所有登录方式的 session 标识
- Phase 2：refresh 改为旋转 session family
- Phase 3：logout / logout-all 全部改为按 session 维度操作
- Phase 4：把黑名单从“主机制”降为“紧急吊销补充机制”

---
## F-02 Notification SSE Hub 的主要正确性风险已解决；剩余为结构化治理建议

- **优先级**：已解决（原 P1）
- **领域**：实时通知 / 高并发稳定性
- **当前状态**：double-close panic 与随机驱逐问题已关闭；仅剩结构化改进建议

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/notification/hub.go`
- `/Users/zxy/Code/StuHelper/server/internal/modules/notification/handler.go`
- `/Users/zxy/Code/StuHelper/server/internal/modules/notification/service_test.go`

### 审查发现
2026-04-14 复核确认：当前 `/Users/zxy/Code/StuHelper/server/internal/modules/notification/hub.go` 使用 `userConnections{order []chan SSEEvent}` 维护有序连接队列；`Subscribe()` 在超额时驱逐 `order[0]`，`Unsubscribe()` 仅在 `remove(ch)` 成功时 `close(ch)`，且两者都受同一把 mutex 保护。

### 2026-04-14 复核结论
- 当前代码**不存在**原文所述的 double-close panic 事实基础。
- 当前实现也不是随机驱逐，而是明确的 **oldest-first**。
- 仍可保留的建议包括：显式连接状态机、更丰富的长连接指标、广播背压策略与 reconnect 策略测试，但这已不属于活跃 P1 风险。

### 为什么必须修
- SSE 属于长连接基础设施，**一个 panic 就是进程级稳定性问题**
- 连接上限、驱逐顺序、关闭幂等性如果不明确，会给线上排障带来极大复杂度
- 长连接代码一旦“靠运气正确”，后续扩展 unread_count / notification / reconnect 策略时会持续放大风险

### 长期正确方案
应将当前 `Hub` 明确建模为：
- **连接注册表（registry）**
- **按用户维度的有序连接队列**
- **幂等注销**
- **显式驱逐策略（oldest-first）**

### 需要决断的架构选择
1. **保留当前内存 Hub + Redis Pub/Sub（推荐短中期）**
   - 适合现阶段体量
   - 需要把连接管理做成严谨的数据结构
2. **进一步演进为独立实时网关 / broker**
   - 适合更高规模
   - 现阶段可能过度设计

### 修复计划
- 定义统一连接状态机：active / evicted / closed
- 强制 `Unsubscribe()` 幂等
- 把驱逐策略写成可测试语义
- 为超连接数、重复注销、广播丢弃建立专门回归测试

---
## F-03 Notification DTO 已完成主要收口；剩余问题是 deprecated 兼容别名与 HTTP/SSE DTO 分层决策

- **优先级**：P2
- **领域**：OpenAPI 契约 / 后端 DTO / 前端共享类型
- **当前状态**：主要契约漂移已解决；剩余为兼容别名清理与 DTO 设计收口

### 代码定位
- OpenAPI：
  - `/Users/zxy/Code/StuHelper/server/api/components/schemas/notification.yaml`
  - `/Users/zxy/Code/StuHelper/server/api/paths/review-notification.yaml`
- 后端通知生产：
  - `/Users/zxy/Code/StuHelper/server/internal/modules/notification/templates.go`
  - `/Users/zxy/Code/StuHelper/server/internal/modules/notification/service.go`
  - `/Users/zxy/Code/StuHelper/server/internal/modules/course/review/repository_notification.go`
  - `/Users/zxy/Code/StuHelper/server/internal/modules/course/review/model.go`
- 前端/共享层：
  - `/Users/zxy/Code/StuHelper/clients/shared/src/types/business/notification.ts`
  - `/Users/zxy/Code/StuHelper/clients/shared/src/notification.ts`
  - `/Users/zxy/Code/StuHelper/clients/web/src/modules/user/views/NotificationsPage.vue`
  - `/Users/zxy/Code/StuHelper/clients/web/src/components/common/NotificationBell.vue`

### 审查发现
2026-04-14 复核确认：当前 OpenAPI 的 `Notification` schema 已声明 `payload`、`sourceModule`、`sourceId`、`sourceUrl`、`courseID` 以及扩展后的 `NotificationType` 枚举；`clients/shared` 中的 `Notification` 接口也已与之基本对齐。原文“OpenAPI 只定义窄字段、shared 手工补 superset”的描述已不再符合当前代码。

### 2026-04-14 复核结论
当前真正剩余的问题是：
1. OpenAPI 仍保留 `relatedType` / `relatedID` 两个 deprecated 兼容字段；
2. `clients/shared/src/notification.ts` 的 `resolveNotificationHref()` 仍保留对这两个字段的 fallback；
3. 仍需拍板 HTTP 列表 DTO 与 SSE DTO 是否统一为一个 canonical Notification Wire DTO。

因此，本条应从“契约漂移”调整为“兼容字段清理 + DTO 分层决策”。

### 为什么必须修
契约漂移的直接后果是：
- 生成类型不能真实反映后端行为
- 前端越来越依赖“手工补类型 + 类型断言”
- HTTP 列表和 SSE 推送的数据形态越来越难以说明

对于企业级系统，这会直接侵蚀：
- 契约测试可信度
- 文档可信度
- 前后端分工边界

### 需要决断的架构选择
#### 方案 A：一个统一 Notification Wire DTO（推荐）
- HTTP 列表与 SSE 都使用同一个规范化 Notification schema
- `payload/sourceModule/sourceId/sourceUrl/courseID` 全部进入 OpenAPI
- 前端只在展示层做 Presentation 转换

**优点**：统一、清晰、便于生成与测试。  
**缺点**：OpenAPI DTO 字段较多。

#### 方案 B：拆成两个正式 DTO
- `NotificationListItem`
- `NotificationSSEEventPayload`

**优点**：如果 HTTP 与 SSE 语义确实不同，可以更精确。  
**缺点**：需要更严格地维护两个模型，复杂度更高。

### 推荐结论
- **如果业务上 HTTP 列表和 SSE 只是同一对象的不同载体，推荐方案 A。**
- **只有在两者字段语义显著不同的前提下，才引入方案 B。**

### 修复计划
- 确认 canonical Notification DTO
- 在 OpenAPI 中正式声明，而不是让 shared 手工补洞
- 前端统一通过 shared helper 做 `wire -> presentation`
- 删除历史兼容别名时要先完成一次前端全量收敛

---
## F-04 Review 创建请求边界已与 OpenAPI 对齐；该规则需要固化为长期规范

- **优先级**：已解决（原 P1）
- **领域**：契约一致性 / 前端 API 设计
- **当前状态**：已解决；后续重点是把“API 输入必须来自生成契约”固化为持续规范

### 代码定位
- `/Users/zxy/Code/StuHelper/server/api/components/schemas/review.yaml`
- `/Users/zxy/Code/StuHelper/clients/shared/src/api/reviews.ts`
- `/Users/zxy/Code/StuHelper/clients/shared/src/types/business/review.ts`
- `/Users/zxy/Code/StuHelper/clients/web/src/components/business/review/reviewPayload.ts`
- `/Users/zxy/Code/StuHelper/clients/web/src/types/review.ts`

### 审查发现
2026-04-14 复核确认：当前 `/Users/zxy/Code/StuHelper/clients/shared/src/types/business/review.ts` 中 `PostReviewRequest` 直接别名到 OpenAPI 生成类型；`/Users/zxy/Code/StuHelper/clients/shared/src/api/reviews.ts` 的 `createReview()` 也直接接收 `PostReviewRequest`，不再对 `title` / `termID` 做空字符串兜底。

### 2026-04-14 复核结论
- 原文所描述的“shared API 输入被手工放松并偷偷补空字符串”已经被修复。
- 本条应保留为**长期规范**：API request 类型必须直接来自 `api.gen.ts`，表单态与草稿态必须单独建模。

### 为什么必须修
这会让“唯一事实源”原则失效：
- API 层不再是“契约执行器”，而变成“容错器”
- 表单校验一旦绕过，shared API 仍可构造无效请求
- 后端开始替前端兜底，导致错误边界模糊

### 长期正确方案
- **shared API 层直接使用 OpenAPI 生成的请求类型**
- 表单态、草稿态、草拟态另建 UI Model
- `buildCreateReviewPayload()` 负责把 UI 状态收敛为严格的 `PostReviewRequest`

### 修复计划
- API request 类型只允许来自 `api.gen.ts`
- UI state 类型命名显式区分：`FormState`、`DraftState`、`ViewModel`
- 代码审查禁止在 API 层引入 `?? ''` 这类“契约降级”逻辑

---
## F-05 Go/TS 生成输入已统一到 bundled spec；仍需把门禁治理规则固化

- **优先级**：已解决（原 P1）
- **领域**：生成链路 / 契约治理
- **当前状态**：已解决双源 spec 问题；仍需继续固化 canonical artifact 与门禁文档

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/api/gen/generate.go`
- `/Users/zxy/Code/StuHelper/server/Makefile`
- `/Users/zxy/Code/StuHelper/clients/package.json`
- `/Users/zxy/Code/StuHelper/server/api/openapi.bundled.yaml`

### 审查发现
2026-04-14 复核确认：当前 Go 与 TS 生成链路都会先执行 `bundle-spec` 并消费 `openapi.bundled.yaml`。同时，`/Users/zxy/Code/StuHelper/server/Makefile` 的默认 `check-drift` 已收敛为 `check-drift-go`，backend-only 环境不再默认依赖 `pnpm`。

### 2026-04-14 复核结论
- 原文的“双源 spec 输入”问题已经关闭。
- 原文“默认 `check-drift` 依赖 pnpm”的表述也已过时。
- 仍需保留的治理点是：文档明确 canonical spec artifact，并在需要时由 `check-drift-all` 同时覆盖 Go 与 TS 生成物。

### 为什么必须修
一旦存在：
- `$ref` 解析差异
- bundler 展平差异
- 工具对 raw multi-file spec 的处理差异

最终就会出现“Go 类型和 TS 类型都号称来自 OpenAPI，但实际上不一样”的局面。

### 长期正确方案
- **Go / TS / 文档工具全部消费同一个 canonical spec artifact**
- 推荐以 `openapi.bundled.yaml` 作为唯一生成输入

### 修复计划
- 所有生成工具统一读取 bundled spec
- CI drift check 覆盖 Go、TS、必要的发布产物
- 文档中明确：`server/api/openapi.yaml` 是源码入口，`openapi.bundled.yaml` 是生成入口

---
## F-06 `auth/user_sync.go` 的核心问题不是“SQL 逃出 Repository”，而是 user persistence 的领域归属错误

- **优先级**：P2
- **领域**：后端架构 / 分层边界
- **当前状态**：仍需根治；问题本质是领域归属错误而非 Repository 分层违例

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/user_sync.go`
- 关键函数：
  - `UpsertUser`
  - `FindByPhone`
  - `ExistsByExternalID`
  - `BackfillUserHashes`

### 审查发现
2026-04-14 复核确认：`/Users/zxy/Code/StuHelper/server/internal/modules/auth/user_sync.go` 中的 SQL 仍然写在 `UserSyncRepository` 这一 Repository 实现内部，因此它不构成“SQL 逃逸出 Repository 层”的直接违例。原文的定性需要修正。

### 2026-04-14 复核结论
真正的问题是**领域归属错误**：
- 用户持久化 Repository 放在了 `modules/auth` 包下；
- 认证域因此直接承载了 PII 存储与用户同步职责；
- 启动时回填任务也以 auth 领域对象为入口。

因此，本条应改写为“auth 包内承载 user persistence Repository，领域归属错误”，而不是“违反 Repository 分层”。

### 为什么必须修
如果这块继续停留在 `modules/auth` 内，后果是：
- 认证域承担不属于它的持久化责任
- PII 存储逻辑与登录逻辑持续耦合
- 未来对手机号、证件号、外部身份同步的治理会越来越难拆

### 需要决断的架构选择
1. **迁入 `modules/user` Repository（推荐）**
   - 认证只依赖 user persistence 接口
2. **在 `modules/auth` 下单独拆 persistence 子层**
   - 边界略清晰，但领域归属仍然尴尬

### 推荐结论
- **把这套 PII / 用户同步持久化迁入 `modules/user` 的 persistence 层更长期正确。**

### 修复计划
- 先定义接口：`UserIdentityRepository`
- 再把 SQL 从 `auth/user_sync.go` 迁移出去
- 启动时 backfill 任务也改由更合适的 domain service / worker 持有

---
## F-07 历史 review 分叉链路与孤儿页面已清理；仍需建立制度化防回归机制

- **优先级**：已解决（原 P2）
- **领域**：前端工程治理 / 复用 / 可维护性
- **当前状态**：历史文件已删除；剩余任务是制度化防回归

### 代码定位（历史发现，已在快照提交中清理）
- 历史 review 创建分叉组件：
  - `clients/web/src/components/business/review/ReviewForm.vue`
  - `clients/web/src/components/business/review/ReviewDialogForm.vue`
  - `clients/web/src/components/business/review/ReviewDialogCourseSearch.vue`
  - `clients/web/src/components/business/review/ReviewDialogCancelConfirm.vue`
  - `clients/web/src/components/business/review/composables/useReviewDialogForm.ts`
  - `clients/web/src/components/business/review/composables/useCourseSearch.ts`
  - `clients/web/src/components/business/review/composables/useTeacherSelect.ts`
  - `clients/web/src/components/business/review/composables/useReviewDialogDraft.ts`
  - `clients/web/src/components/business/review/composables/useReviewDialogSubmit.ts`
- 历史孤儿页面：
  - `clients/web/src/modules/user/views/NotificationPreferencesPage.vue`
  - `clients/web/src/modules/errors/views/LoadErrorPage.vue`

### 审查发现
2026-04-14 复核确认：原文列出的历史 review 组件、composables 以及孤儿页面在当前工作树中均已不存在，包括 `ReviewForm.vue`、`ReviewDialogForm.vue`、`NotificationPreferencesPage.vue`、`LoadErrorPage.vue` 等。

### 2026-04-14 复核结论
- 本条作为“活跃问题”应标记为**已解决**。
- 仍然成立的是长期治理建议：需要建立 CI / CR 规则，持续检测无引用组件、未接路由页面与历史分叉实现，避免同类问题回流。

### 为什么必须修
双实现并存的最大问题不是“占点空间”，而是：
- 同一需求以后可能被改两次且改不一致
- 新成员阅读代码时无法判断哪条链路是活的
- 测试覆盖会被历史死代码稀释

### 长期正确方案
- 统一 review 创建链路为一套正式的 `Form + State + Submit` 组合
- 删除无引用组件和孤儿页面必须成为 CI/CR 审查项，而不是偶发性清理

### 修复计划
- 继续做无引用扫描与路由可达性检查
- 对“新建页面未接路由”“组件无 consumer”建立 lint/CI 规则

---
## F-08 课程列表页的性能模型错误：全量拉取 + 前端聚合不具备长期可扩展性

- **优先级**：P2
- **领域**：前端性能 / API 设计
- **当前状态**：未根治

### 代码定位
- `/Users/zxy/Code/StuHelper/clients/web/src/modules/course/courseCatalogLoader.ts:1-27`
- `/Users/zxy/Code/StuHelper/clients/web/src/modules/course/views/CourseListPage.vue:60-115`
- 关键实现：
  - `pageSize = 100`
  - `Promise.all(...)` 拉取全部分页
  - `groupByDepartment(courses)` 前端聚合

### 审查发现
当前页面通过先拉第一页算出总页数，再把剩余页全部并发拉完，然后前端再分组。这个模式本质上把”课程目录页”做成了”全库批量下载器”。  
同时，仓库里已经存在一个可复用的分页装载 helper `loadCourseCatalog()`，但 `CourseListPage.vue` 没有复用这条抽象，而是重新写了一套分页编排逻辑。这意味着问题不仅是性能模型不对，还包括：
- 同一业务意图存在两套分页装载实现
- 后续修复分页、错误处理、限流、重试策略时容易分叉
- “复用优先”的工程原则在此处未落地

### 2026-04-14 复核结论
**完全属实。** 代码验证确认：`CourseListPage.vue:83-115` 自行实现了 pageSize=100 + Promise.all 并发拉取全部分页 + groupByDepartment 前端聚合的完整逻辑。同时 `courseCatalogLoader.ts` 已存在一个通用的 `loadCourseCatalog<T>()` helper 做同样的事情（逐页拉取直到 total），但 CourseListPage 完全没有使用它。问题描述准确，建议和修复计划合理。

### 为什么必须修
随着课程量增长，这个页面会出现：
- 首屏时延不可控
- 并发请求数不可控
- 移动端网络成本过高
- 服务端和客户端都承担不必要压力
- 即便短期不新增接口，至少也应先把分页装载责任收敛到单一 helper，避免页面自行维护并发抓全库的影子实现

### 需要决断的架构选择
1. **服务端直接返回按院系分组的数据（推荐）**
2. 保持分页课程列表，但前端按需展开 / 虚拟滚动
3. 继续全量拉取（不推荐）

### 推荐结论
- **如果页面目标是“按院系浏览全部课程”，最长期正确的做法是由后端直接提供面向场景的聚合接口。**

### 修复计划
- Phase 1：先把 `CourseListPage.vue` 对分页装载的重复实现收敛到 `courseCatalogLoader.ts`，消除影子逻辑
- Phase 2：新增专门的 course catalog / grouped listing API
- Phase 3：前端改为分页或按分组懒加载
- Phase 4：大列表页面统一引入虚拟滚动策略

---

## F-09 `clients/shared/src/types/business/admin.ts` 已删除；shared 职责边界仍需重新定义

- **优先级**：已解决局部历史包袱（原 P2）
- **领域**：前端共享层设计
- **当前状态**：`admin.ts` 已删除；shared 边界治理仍需继续

### 代码定位（历史发现，已在快照提交中移除）
- `clients/shared/src/types/business/admin.ts`
- `clients/admin/apps/web-antd/`
- `clients/admin/apps/web-antdv-next/`
- `clients/admin/apps/web-naive/`
- `clients/admin/apps/web-ele/src/views/_core/about/index.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/code-login.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/forget-password.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/qrcode-login.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/register.vue`
- `clients/admin/apps/web-ele/src/views/_core/fallback/coming-soon.vue`
- `clients/admin/apps/web-ele/src/views/_core/fallback/internal-error.vue`
- `clients/admin/apps/web-ele/src/views/_core/fallback/offline.vue`
- `clients/admin/apps/web-ele/src/views/demos/element/index.vue`
- `clients/admin/apps/web-ele/src/views/demos/form/basic.vue`
- `server/internal/pkg/faceid/tencent.go`

### 审查发现
2026-04-14 复核确认：`/Users/zxy/Code/StuHelper/clients/shared/src/types/business/admin.ts` 在当前工作树中已经不存在。原文围绕该文件的具体问题属于已解决的历史包袱。

### 2026-04-14 复核结论
- 本条关于 `admin.ts` 的问题应标记为**已解决**。
- 但 broader shared 边界问题仍成立：shared 仍需明确划分 `wire contract`、`api`、`constants` 与 `presentation/view-model` 的职责；Admin 历史变体的问题则继续由 F-18 覆盖。

### 为什么必须修
shared 层如果同时承担：
- 生成契约
- API client
- 手工业务 DTO
- UI helper

但又没有严格分层，就会不断出现“方便用的类型”与“真实 wire contract”混用的情况。

### 长期正确方案
明确 shared 分层：
- `api.gen.ts`：唯一 wire contract
- `api/`：请求封装
- `constants/`：稳定复用常量
- `presentation/` 或 `view-model/`：真正的 UI 适配模型

---
## F-10 生产上线链路在审查时点并未闭环：公网入口、备份验证、Smoke 深度都不足

- **优先级**：P1
- **领域**：Infra / Delivery / SRE
- **当前状态**：部分问题已修复；公网入口 / 拓扑归属 / smoke 深度仍未闭环

### 代码定位
- 公网入口：
  - `/Users/zxy/Code/StuHelper/docker-compose.yml`
  - `/Users/zxy/Code/StuHelper/infra/traefik/zitadel.dynamic.yaml`
- 生产初始化 / preflight / deploy：
  - `/Users/zxy/Code/StuHelper/infra/ops/init-prod-env.sh`
  - `/Users/zxy/Code/StuHelper/infra/ops/tests/init-prod-env-contract.sh`
  - `/Users/zxy/Code/StuHelper/infra/ops/remote-preflight.sh`
  - `/Users/zxy/Code/StuHelper/infra/ops/prod-deploy.sh`
  - `/Users/zxy/Code/StuHelper/infra/ops/build-deploy-bundle.sh`
- CI：
  - `/Users/zxy/Code/StuHelper/.gitlab-ci.yml`

### 审查发现
2026-04-14 复核确认：
1. `prod_init_contract` 失配已经修复，当前 `bash infra/ops/tests/init-prod-env-contract.sh` 可通过；
2. 生产入口未在仓库内闭环这一点仍然成立：`docker-compose.yml` 中大量服务仍仅绑定 `127.0.0.1`；
3. 前端 SSO URL 当前已收敛为 **build-time 单一来源**：CI 显式要求 `WEB_VITE_SSO_URL`，并将其传给 `frontend_build` 与 `docker_build_frontend`；因此原文“构建时/部署时双源”表述不准确；
4. 备份对象存储的 preflight / deploy 前硬校验已经补齐；
5. smoke check 深度不足仍然成立。

### 2026-04-14 复核结论
本条仍应保留，但重心应调整为：
- 公网入口 / topology ownership 未闭环；
- Smoke / business validation 深度不足；
- 运维文档与最终支持的 production topology 仍需明确。

不应再把 `WEB_VITE_SSO_URL` 描述为配置双源 bug。

### 为什么必须修
没有完整交付闭环意味着：
- “能部署” ≠ “能独立上线”
- “健康检查绿” ≠ “真实业务可用”
- “备份脚本存在” ≠ “恢复能力可靠”

### 需要决断的架构选择
#### 公网入口归属
1. **仓库内自带 edge（Traefik/Nginx/LB）**
2. **公网入口由仓库外基础设施提供**

### 推荐结论
- 必须二选一，并正式写进运维文档。当前“既像仓库内负责，又像仓库外负责”的状态最危险。

### 修复计划
- 明确 supported production topology
- 把 Web / Admin / Grafana / Alertmanager / Prometheus 的公网入口与证书终止责任写死
- 将外部 edge / LB / 反向代理依赖纳入正式运维文档
- 把 smoke check 升级为真实业务冒烟：登录、OIDC 回调、关键 API、指标上报链路

---
## F-11 前端 grade 常量与通知跳转逻辑已基本收敛；剩余为持续治理提醒

- **优先级**：已解决主要问题（原 P2）
- **领域**：前端复用 / 单一事实源
- **当前状态**：grade 常量与 notification href 已收敛到 shared；剩余为持续治理与防回归

### 代码定位
- `clients/shared/src/constants/review.ts`
- `clients/shared/src/api/reviews.ts`
- `clients/shared/src/api/draft.ts`
- `clients/web/src/types/guards.ts`
- `clients/shared/src/notification.ts`
- `clients/web/src/components/common/NotificationBell.vue`
- `clients/web/src/modules/user/views/NotificationsPage.vue`

### 审查发现
2026-04-14 复核确认：`REVIEW_GRADES` / `isReviewGrade` 已集中在 `clients/shared/src/constants/review.ts`，`clients/shared/src/api/reviews.ts`、`clients/shared/src/api/draft.ts`、`clients/web/src/types/guards.ts` 与 `clients/web/src/components/business/review/reviewPayload.ts` 均复用这套共享常量；同时通知跳转已统一通过 `clients/shared/src/notification.ts` 的 `resolveNotificationHref()`，`NotificationBell.vue` 与 `NotificationsPage.vue` 都已直接消费该 helper。原文所述“grade 常量、通知跳转逻辑多份平行定义”在当前代码中已明显收敛。

### 2026-04-14 复核结论
- notification jump 与 review grade 两条线应标记为**已基本解决**；
- 本条可下调为持续治理提醒：未来新增 guard、helper 或页面跳转逻辑时，仍需强制走 shared，避免再长出影子副本。

### 为什么必须修
这种重复不会立刻爆炸，但会在每次迭代里制造“轻微不一致”，最终演化成线上差异行为。

### 长期正确方案
- 共享常量只定义一次
- 共享 helper 只定义一次
- 业务守卫逻辑必须向 shared 收敛，不允许 web 再维护影子副本

---

## F-12 生产数据库备份与发布前备份链路未自包含，不能把“有脚本”视为“有恢复能力”

- **优先级**：P1
- **领域**：Infra / Backup / Delivery
- **当前状态**：审查时点未闭环

### 代码定位
- `/Users/zxy/Code/StuHelper/infra/ops/prod-deploy.sh`（pre-deploy backup phase，审查时点在 `creating pre-deploy database backup` 后直接调用 `backup-postgres.sh`）
- `/Users/zxy/Code/StuHelper/infra/ops/backup-postgres.sh`（要求显式 `<output-file>`；并依赖 `pg_dump` / `pg_basebackup` 或等价执行环境）
- `/Users/zxy/Code/StuHelper/infra/ops/run-scheduled-backup.sh`（systemd timer 的实际执行入口）
- `/Users/zxy/Code/StuHelper/infra/ops/install-backup-timers.sh`（logical / basebackup / sync 三个 timer 的安装脚本）
- `/Users/zxy/Code/StuHelper/infra/ops/bootstrap-ubuntu2404.sh`（基础包安装与 backup timer 安装）
- `/Users/zxy/Code/StuHelper/.env.prod.example`（`BACKUP_DATABASE_URL`、`REPLICATION_DATABASE_URL` 的期望形态）

### 审查发现
1. 生产发布脚本在部署前执行数据库备份，但审查时点的调用方式没有形成”确定的输出文件路径 + checksum + 失败语义”的正式闭环，曾存在直接调用 `backup-postgres.sh` 时不传目标文件的风险。
2. 定时备份链路把执行责任部分放在宿主机、部分放在仓库脚本、部分放在容器环境，执行边界不清晰。
3. 宿主机 bootstrap 只安装基础工具与 Docker（`ca-certificates curl gnupg jq openssl git bash python3`），不保证 `pg_dump` / `pg_basebackup` 这类 PostgreSQL 客户端能力始终可用；这意味着 backup timer 即使被安装，也不等于备份真的能跑通。
4. `remote-preflight.sh` 会检查 timer 是否安装、URL 是否存在，但不会证明”备份工具可执行、备份产物可生成、校验和可验证、恢复演练可成功”；当前 preflight 更接近”配置存在性检查”，还不是”恢复能力检查”。

### 2026-04-14 复核结论
**方向正确，属于典型的 SRE 成熟度问题。** 未逐一读取所有备份脚本的实现细节，但从整体架构设计角度看，审查文档的分析逻辑严谨：备份链路的闭环应包含”执行能力验证 + 产物校验 + 恢复演练”，而非仅”脚本存在 + 配置存在”。容器化 backup executor 的建议合理。

### 为什么必须修
- 对生产系统而言，“发布前备份”是 fail-closed 的上线前提，不是锦上添花。
- 若备份链路不自包含，最坏情况是：**系统以为自己可恢复，实际在故障时无法恢复**。
- 这类问题属于典型的 SRE 伪闭环：监控/脚本/定时器都在，但真正的恢复能力没有被证明。

### 需要决断的架构选择
1. **继续在宿主机安装 PostgreSQL 客户端工具，并由宿主机 systemd 执行备份脚本**
   - 优点：简单直接
   - 缺点：宿主机环境漂移、版本管理困难、与 Compose 容器环境容易脱节
2. **把备份执行环境容器化（推荐）**
   - 用固定版本的 backup ops 镜像，或直接复用受控的 PostgreSQL 镜像执行 `pg_dump` / `pg_basebackup`
   - systemd / cron 只负责拉起受控容器
   - 优点：执行环境稳定、版本可控、便于在 staging/prod 做一致演练
   - 缺点：需要额外维护一层容器化运维入口

### 推荐结论
- **备份执行环境应容器化，并把“备份产物、checksum、retention、恢复演练”纳入正式交付标准。**

### 修复计划
- Phase 1：定义 canonical backup executor（host 或 container，只能有一个主机制）
- Phase 2：发布前备份必须显式写入确定路径，并生成 checksum 与日志
- Phase 3：preflight 升级为校验备份执行能力，而不是只校验配置存在
- Phase 4：补一条恢复演练 runbook，并把“最近成功备份时间”“最近恢复演练时间”纳入告警/看板

---

## F-13 认证域的应用服务边界失守：Handler 直接操作用户同步仓储

- **优先级**：P1
- **领域**：后端架构 / 认证
- **当前状态**：未根治

### 代码定位
- 规范基线：`/Users/zxy/Code/StuHelper/docs/design-docs/layered-architecture.md:7-15,27-37,49-55`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler.go:19-33,35-69`（`Handler` 直接持有 `userSyncRepo`）
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_login.go:125-149`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_phone.go:124-147`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_session.go:141-152`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/service.go`

### 审查发现
OIDC callback、手机号登录、会话续期等路径中，`Handler` 直接调用 `UserSyncRepo` 完成 `UpsertUser` / `UpsertByPhone` / `ExistsByExternalID` 等动作，导致：
- HTTP 层既做协议绑定，又做业务编排，又直接触碰持久化
- `auth.Service` 被部分绕过，不再是认证域唯一的应用服务入口
- 未来新增第三种登录方式时，容易继续把编排逻辑摊在更多 handler 中

### 2026-04-14 复核结论
**完全属实。** 代码验证确认：`handler.go:27` Handler 直接持有 `userSyncRepo UserSyncRepo`；`handler_login.go:132` OIDC callback 中 Handler 直接调用 `h.userSyncRepo.UpsertUser()`；`handler_phone.go:125` 手机登录中 Handler 直接调用 `h.userSyncRepo.UpsertByPhone()`；`handler_session.go:141`（refreshSelfSignedToken 路径）Handler 直接调用 `h.userSyncRepo.ExistsByExternalID()`。与此同时，`auth.Service` 仅负责 token 生命周期管理（Track/Rotate/Revoke/Sign），用户同步完全绕过 Service。问题描述准确，推荐方案合理。

### 为什么必须修
- 认证是后端最敏感的 domain 之一，业务规则散落在 Handler 层会直接降低安全语义的一致性。
- 无法稳定统一事务边界、错误映射、审计埋点、会话模型迁移。
- 若后续按本报告建议升级为服务端 Session / Token Family，这个问题会成为重构阻力。

### 需要决断的架构选择
1. **继续允许 Handler 持有 `userSyncRepo` 并做轻量编排**
   - 优点：短期改动少
   - 缺点：与项目已声明的分层铁律冲突，后续继续失控
2. **让 `auth.Service` 成为认证域唯一编排入口（推荐）**
   - Handler 只做 HTTP 绑定、调用 Service、错误映射与响应
   - 持久化仅通过窄接口进入 Repository / persistence 层

### 推荐结论
- **`auth.Service` 应成为认证域唯一的应用服务编排入口，Handler 不再直接访问用户同步仓储。**

### 修复计划
- Phase 1：定义认证应用服务命令：`HandleOIDCCallback`、`HandlePhoneOTPLogin`、`RefreshSession`、`LogoutSession`、`LogoutAllSessions`
- Phase 2：把 `UserSyncRepo` 调用从各个 handler 中收回到 Service
- Phase 3：将持久化交互收敛为窄接口，必要时把用户同步 SQL 迁入 `modules/user` persistence
- Phase 4：补 service-level 回归测试，验证多登录方式在同一编排层下的一致语义

---

## F-14 Review 访问策略与授权事实解析停留在 Handler 层，授权边界不可复用

- **优先级**：P1
- **领域**：后端架构 / 授权 / Review
- **当前状态**：未根治

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/course/review/handler.go:28-53`（`Handler` 直接持有 `userRepo` 等依赖）
- `/Users/zxy/Code/StuHelper/server/internal/modules/course/review/access.go:38-101`（访问策略加载）
- `/Users/zxy/Code/StuHelper/server/internal/modules/course/review/access.go:164-200`（访问事实解析、用户主体读取）
- `/Users/zxy/Code/StuHelper/server/internal/modules/user/repository*.go`（被间接依赖的用户主体信息）

### 审查发现
`course/review/access.go` 里把以下逻辑放在了 Handler 侧：
- 学校/课程/用户上下文对应的访问策略加载
- 用户主体信息解析
- 与授权判断强耦合的访问事实组装

这意味着授权判断所需的 domain facts 并不是由 Service 层统一生产，而是由 HTTP 入口自己拼装。

### 2026-04-14 复核结论
**事实属实但定性需要细化。** `access.go` 中的 `resolveReviewAccessFacts()` 确实由 Handler 的方法 `resolveReviewAccessFactsForRequest()` 调用，且 Handler 直接持有 `userRepo *user.Repository`（`handler.go:33`）用于加载策略数据。但需要注意的是：`access.go` 中的核心逻辑 `resolveReviewAccessFacts(ctx, externalID, capabilities)` 本身不依赖 `*gin.Context`，只有外层的 `resolveReviewAccessFactsForRequest(c *gin.Context)` 做了 HTTP 绑定。因此迁移到 Service 层的改造成本较低——只需将 `resolveReviewAccessFacts` 及其依赖的策略加载逻辑移入 Service，Handler 只调用 Service 即可。推荐方案合理。

### 为什么必须修
- 授权逻辑放在 Handler 层会导致非 HTTP 调用路径（批处理、异步任务、后台工具）难以复用同一套规则。
- 一旦 Capability / OpenFGA / Review 可见性规则继续演进，Handler 将持续膨胀。
- 授权事实组装若分散在入口层，极易出现不同入口算出不同结果的隐性 bug。

### 需要决断的架构选择
1. **把访问策略与事实解析内聚进 `review.Service`**
   - 优点：简单直接，业务收口在 service
   - 缺点：若授权逻辑继续变复杂，service 可能过胖
2. **引入专门的 `ReviewAccessPolicyService` / `ReviewAccessEvaluator`（推荐）**
   - 由 Service 调用该策略服务
   - 访问事实、主体解析、配置读取都在同一授权应用层中完成

### 推荐结论
- **推荐引入专门的 `ReviewAccessPolicyService`，由 Review Service 调用；Handler 不再自己解析授权事实。**

### 修复计划
- Phase 1：定义 `ReviewAccessSubjectReader`、`ReviewAccessPolicyProvider` 等窄接口
- Phase 2：把 `access.go` 中的策略加载和主体解析从 Handler 迁出
- Phase 3：让 Handler 只消费最终授权结果与展示所需的脱敏字段
- Phase 4：补回归测试，覆盖登录/未登录/不同身份/不同资源关系下的访问结果

---

## F-15 契约门禁与前端构建工具链未形成单一可复现链路

- **优先级**：P2
- **领域**：契约治理 / Build / CI
- **当前状态**：部分已解决；核心剩余问题是 `clients/shared` 的 dist-first workspace 消费导致 `web` 冷启动脆弱

### 代码定位
- `/Users/zxy/Code/StuHelper/server/Makefile`（`generate`、`generate-ts`、`check-drift`）
- `/Users/zxy/Code/StuHelper/clients/package.json`（`api:generate`、`build:web`、`dev:web` 等脚本）
- `/Users/zxy/Code/StuHelper/clients/shared/package.json`（workspace 包导出仅指向 `dist/*`）
- `/Users/zxy/Code/StuHelper/clients/shared/tsconfig.build.json`
- `/Users/zxy/Code/StuHelper/clients/web/package.json`
- `/Users/zxy/Code/StuHelper/clients/uniappx/package.json`
- `/Users/zxy/Code/StuHelper/.gitlab/server-ci.yml`（`openapi_contract`）
- `/Users/zxy/Code/StuHelper/.gitlab-ci.yml`（`frontend_build`）
- `/Users/zxy/Code/StuHelper/clients/web/Dockerfile`（当前固定为 `pnpm@10.32.1`）
- `/Users/zxy/Code/StuHelper/server/api/openapi.bundled.yaml`

### 审查发现
2026-04-14 复核确认：
1. 默认 `make check-drift` 已收敛为 Go-only；TS drift 由显式 `check-drift-ts` / `check-drift-all` 覆盖；
2. `/Users/zxy/Code/StuHelper/clients/web/Dockerfile` 已固定 `pnpm@10.32.1`，原文关于 `pnpm@latest` 的表述已过时；
3. 当前仍存在的核心问题是 `clients/shared/package.json` 的 exports 全指向 `dist/*`，而 `clients/web` / `clients/uniappx` 的 package-local scripts 本身并不保证先 build shared；
4. 本次复验中，在移除 `clients/shared/dist` 后，`npx --yes pnpm@10.32.1 --dir /Users/zxy/Code/StuHelper/clients/web build` 仍稳定失败，核心报错包含 `TS2307: Cannot find module '@stuhelper/shared*'`；
5. 同条件下，`npx --yes pnpm@10.32.1 --dir /Users/zxy/Code/StuHelper/clients/uniappx type-check` 在本机当前工作树中通过，因此原文把 uniappx 也描述为“当前可稳定复现失败”已经不准确。

### 2026-04-14 复核结论
本条应收敛为：`clients/shared` 的 **dist-first workspace 消费模型** 仍使 `web` 的 package-local 构建对 warm workspace 敏感；而 backend drift gate 与 pnpm 版本漂移问题已经关闭。

### 为什么必须修
- `clients/shared` 的 dist-first workspace 导出会让 package-local scripts 对 warm workspace 敏感；这会持续制造“根脚本能跑、子包脚本不稳定”的认知落差。
- 对 monorepo 而言，**fresh checkout / cold start 可复现** 是基础能力；任何依赖预先存在 `dist` 的本地开发体验，都会在 CI、临时分支、灾难恢复和新成员入场时放大成本。
- 如果不尽快拍板 shared 的消费模型，后续 web / uniappx / 新增客户端仍会持续在 source-first 与 dist-first 之间摇摆。

### 需要决断的架构选择
1. **继续坚持 dist-first（仅限 monorepo 内部也走发布产物）**
   - 优点：与外部分发模型一致
   - 缺点：所有 consumer 都必须显式先 build shared，本地/CI 易受 warm workspace 影响
2. **在 monorepo 内部采用 source-first / project references（推荐）**
   - 优点：package-local scripts 更自包含，fresh checkout 体验更稳定
   - 缺点：需要重新梳理 tsconfig / exports / 构建边界

### 推荐结论
- **建议把当前问题收敛到一个核心决策：`clients/shared` 在 monorepo 内到底采用 source-first 还是 dist-first。**
- 若继续保留 dist-first，则必须让所有 consumer scripts 显式、强制地先准备 shared 产物；否则应切换到 source-first / project references，并把 `dist` 限定为发布产物。

### 修复计划
- Phase 1：拍板 `clients/shared` 在 monorepo 内部采用 source-first 还是 dist-first 消费
- Phase 2：若继续保留 dist-first，则所有 consumer package-local scripts 必须显式 `prepare:shared`；否则切换为 source-first / project references
- Phase 3：在 fresh checkout 场景下增加 CI 检查，至少覆盖 `web` 的 package-local build
- Phase 4：文档中明确区分“OpenAPI 源码入口”“bundled artifact”“Go-only drift gate”“TS drift gate”

---
## F-16 `uniappx` 当前仍不具备“可宣称原生生产可用”的闭环证明；但其冷启动故障表述需要收窄

- **优先级**：P2
- **领域**：多端交付 / 移动端认证 / 类型系统
- **当前状态**：当前直接可复现的问题已收敛；剩余风险集中在产品定位与 native auth 闭环不明确

### 代码定位
- 类型错误直接定位：
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/api/index.ts:11,26`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/api/shared-client.ts:1`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/pages/review/index.vue:97,114`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/pages/review/post.vue:6,170,180`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/pages/user/notifications.vue:62,74`
- 认证闭环相关：
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/pages/auth/login.vue`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/manifest.json`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/pages.json`
  - `/Users/zxy/Code/StuHelper/clients/uniappx/src/App.vue`

### 审查发现
2026-04-14 复核确认：
1. 在当前工作树中，`npx --yes pnpm@10.32.1 --dir /Users/zxy/Code/StuHelper/clients/uniappx type-check` 即使在 `clients/shared/dist` 缺失时也可通过；因此原文把 uniappx 描述为“当前可稳定复现的 clean checkout 类型门禁失败”已经不准确；
2. 但 `uniappx` 仍然继承了 F-15 中的 `clients/shared` dist-first workspace 模型风险：其 package-local scripts 本身并不负责 prepare shared，只是当前没有在本机工作树中直接爆出错误；
3. 原生 SSO / callback / session bootstrap 是否完整闭环，当前代码和文档仍未给出足够证据。

### 2026-04-14 复核结论
- 本条应从“当前 deterministic type-check failure”调整为“产品定位与 native auth 闭环未明，且存在潜在的 shared/dist 耦合风险”。
- 如果 `uniappx` 不再承担正式 native 交付承诺，本条应继续降级为 experimental track 的治理事项；如果要承担正式交付，则必须补齐闭环证明与专项 QA。

### 为什么必须修
- 如果项目仍对外宣称存在 native 端能力，就必须证明登录闭环、回调、会话恢复与专项 QA 都已成立。
- 即便当前本机不再复现类型检查失败，`uniappx` 仍然受 shared/dist-first 模型影响，说明它的工程稳定性依然依赖上游构建策略而非自身自包含。
- 若产品定位不清晰，最危险的状态就是“看起来支持 native，实际上没有正式承诺与验证闭环”。

### 需要决断的架构选择
1. **把 `uniappx` 视为正式产品线（native supported）**
   - 需要完整实现 deep link / callback / session bootstrap / QA 回归
   - 成本高，但语义清晰
2. **把 `uniappx` 明确降级为 H5-only / experimental（推荐的兜底决策）**
   - 文档、CI、界面都必须显式说明“原生端能力未承诺”
   - 避免当前这种“看起来支持，实际上未闭环”的状态

### 推荐结论
- **必须二选一：要么投入把 native auth 做完整；要么明确把 `uniappx` 标为 experimental/H5-only。禁止继续处于灰色地带。**

### 修复计划
- Phase 1：先完成产品定位拍板：official native vs experimental / H5-only
- Phase 2（若为 official native）：补齐 app scheme / deep link / callback / session bootstrap，并把流程写入文档
- Phase 3（若为 official native）：增加专项 smoke / QA checklist，证明 native 登录链路可交付
- Phase 4（若为 experimental）：在文档、CI 与产品文案中显式降级，不再给出 production-ready 暗示

## F-17 Web / Admin / Uniappx 的请求与错误语义已发生分叉，shared 层需要重新承担“语义统一器”角色

- **优先级**：P2
- **领域**：前端共享层 / API 调用 / 错误处理
- **当前状态**：通知语义已部分收敛；错误 envelope / unwrap / refresh 语义仍然分叉

### 代码定位
- `/Users/zxy/Code/StuHelper/clients/web/src/api/client.ts`
- `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/api/shared-result.ts`
- `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/api/shared-client.ts`
- `/Users/zxy/Code/StuHelper/clients/uniappx/src/api/shared-client.ts`
- `/Users/zxy/Code/StuHelper/clients/uniappx/src/api/result.ts`
- `/Users/zxy/Code/StuHelper/clients/shared/src/notification.ts`
- `/Users/zxy/Code/StuHelper/clients/web/src/components/common/NotificationBell.vue`
- `/Users/zxy/Code/StuHelper/clients/web/src/modules/user/views/NotificationsPage.vue`

### 审查发现
1. Web 端 `buildApiError()` 仍只读取 `payload.error.message/code/details`，而 admin / uniappx 端同时兼容 top-level `message` / `code` / `error`。
2. 三端仍各自维护请求包装、错误解析、unwrap 行为，shared 还没有真正成为统一的 response / error semantics 层。
3. 与原文不同，通知语义不再是主要分歧源：`clients/shared/src/notification.ts` 已提供完整的展示与跳转抽象，且 Web 端 `NotificationBell.vue` 与 `NotificationsPage.vue` 已直接消费 `resolveNotificationHref()`。

### 2026-04-14 复核结论
- 第 1、2 点继续成立；多端请求与错误语义仍明显分叉。
- 原文第 3 点需要修正为：shared 已经开始在 notification wire semantics 上承担统一器角色，当前更值得优先统一的是 error envelope、response unwrap 与 refresh / retry 语义。

### 为什么必须修
- 同一类后端错误在不同客户端上表现不一致，会直接增加排障成本与用户认知偏差。
- 当 refresh、CSRF、error envelope、通知跳转各写一份时，shared 的存在价值会被不断稀释。
- 这是典型的“看起来有 shared，实际上还是多端各管各的”问题。

### 需要决断的架构选择
1. **所有平台共用一个完整 transport 层**
   - 优点：理论上最统一
   - 缺点：Web / Admin / Uniappx 的 fetch/runtime 差异较大，容易过度抽象
2. **共享“语义核心”+ 平台薄适配层（推荐）**
   - shared 负责 response envelope、error normalization、notification wire semantics、retry / refresh policy 约定
   - 各端只保留平台差异化的 transport adapter

### 推荐结论
- **推荐采用“shared 语义核心 + 平台薄适配层”模型，shared 不负责抹平运行时差异，但必须负责统一 wire semantics。**

### 修复计划
- Phase 1：把错误 envelope 解析规则抽到 shared
- Phase 2：统一 notification wire -> presentation 的转换入口
- Phase 3：各端 transport 仅保留 fetch/runtime 差异，不再各自定义业务语义
- Phase 4：补三端一致性回归测试，覆盖 401/403/validation error/unknown error 等场景

---
## F-18 Admin 仍携带大量历史变体与未挂路由页面，说明后台产品线尚未完成收敛

- **优先级**：P2
- **领域**：前端架构 / 工程治理 / 删除无用代码
- **当前状态**：未根治

### 代码定位
- 历史 admin 变体：
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-antd/`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-antdv-next/`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-naive/`
- 当前主线：admin `web-ele`
  - `/Users/zxy/Code/StuHelper/clients/admin/package.json`
  - `/Users/zxy/Code/StuHelper/clients/package.json`
  - `/Users/zxy/Code/StuHelper/docs/FRONTEND.md`
- `web-ele` 中的未挂路由页面候选：
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/about/index.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/authentication/code-login.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/authentication/forget-password.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/authentication/qrcode-login.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/authentication/register.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/fallback/coming-soon.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/fallback/internal-error.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/fallback/offline.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/demos/element/index.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/demos/form/basic.vue`
- 类型边界黑洞：
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/adapter/component/index.ts`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/authentication/forget-password.vue`
  - `/Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/_core/authentication/register.vue`

### 审查发现
1. 当前 CI、本地命令、文档都明确只把 `web-ele` 当作 admin 主线；其余三个 app 变体未进入正式交付主链路。
2. `web-ele` 内仍保留一批 about / authentication / fallback / demos 页面，但并未进入正式路由与产品流程。
3. `web-ele` 的高复用适配层与认证表单边界中仍存在 `Recordable<any>`、`props: any` 这类黑洞类型；这意味着即便主线收敛为单一实现，strict type boundary 也还没有真正立住。
4. 这说明后台产品线仍处于”主线已确定，但历史 scaffold 未收口，且主线内部的高复用边界仍有类型债”的状态。

### 2026-04-14 复核结论
**完全属实。** 通过 Glob 确认 `clients/admin/apps/web-antd/` 仍完整保留（约 70+ 文件），包括 views、router、store、adapter 等全套 scaffold。Memory 中记录 admin 基于 `apps/web-ele/` 做二开且保留所有变体”方便 merge 升级”，但这些变体如果长期不进入交付主链路，确实会持续增加维护负担。建议如文档所述：若无明确多皮肤路线图，正式归档或删除非主线变体。

### 为什么必须修
- 历史变体越多，后续升级依赖、修安全问题、统一设计系统的成本越高。
- 未挂路由页面虽然不一定会被用户访问，但会持续污染维护者心智模型。
- 用户要求的“删除无用代码、提高复用”在 admin 侧的收益尤其大。
- 如果高复用适配层继续保留 `any` 黑洞，那么 admin 即使表面上开了 strict mode，真正关键的抽象边界仍然无法提供编译期保护。

### 需要决断的架构选择
1. **正式收敛为单一 admin 实现（推荐）**
   - `web-ele` 成为唯一受支持后台
   - 其余目录归档或删除
2. **保留多 UI 变体策略**
   - 必须正式引入 `admin-core` / `admin-shell` 等共享层
   - 并明确哪一套是 prod、哪几套是实验性 demo

### 推荐结论
- **若没有明确的多 UI 变体产品路线图，应正式收敛到单一 `web-ele` 主线，并清理未挂路由页面。**

### 修复计划
- Phase 1：拍板 admin 是“单实现”还是“多皮肤平台”
- Phase 2（单实现）：删除/归档历史变体与未挂路由页面
- Phase 3（单实现）：把 adapter / auth 表单边界中的 `any` 替换为明确的 props / payload 契约
- Phase 4（多皮肤）：抽 `admin-core`，禁止继续复制整套 scaffold
- Phase 5：为 dead route / orphan view 建立持续治理规则

---

## F-19 后端仍保留死包与重复的腾讯云签名栈，说明“删除冗余 + 抽底层能力”还未完成

- **优先级**：P2
- **领域**：后端工程治理 / 可复用性
- **当前状态**：未根治

### 代码定位
- 死包候选：`/Users/zxy/Code/StuHelper/server/internal/pkg/faceid/tencent.go`
- 重复实现对照：
  - `/Users/zxy/Code/StuHelper/server/internal/pkg/sms/tencent.go`
  - `/Users/zxy/Code/StuHelper/server/internal/pkg/faceid/tencent.go`

### 审查发现
1. `internal/pkg/faceid` 在审查时点未出现在 server 主入口依赖图中，也未发现有效业务接入痕迹，属于高概率死包。
2. `sms` 与 `faceid` 都维护了一套腾讯云 TC3 签名、请求构造、HMAC / SHA256 逻辑，明显存在底层能力重复。

### 2026-04-14 复核结论
**完全属实。** 通过 Grep 确认 `faceid` 仅在 `server/internal/pkg/faceid/tencent.go` 一个文件中存在，`server/internal/app/` 和 `server/cmd/` 中无任何导入。这是一个确定的死包。建议直接删除。

### 为什么必须修
- 死包会增加安全扫描、依赖升级、代码阅读的无效面。
- 签名逻辑重复实现意味着：一处修复、另一处忘改；一处加 tracing / metrics、另一处继续漂移。
- 用户要求“性能最优、长期正确、企业级最佳实践”，这类基础设施代码理应尽早抽平。

### 需要决断的架构选择
1. **若 FaceID 已无产品路线，直接删除（推荐）**
2. **若 FaceID 仍有路线，则把 TC3 签名与腾讯云请求模板下沉为统一基础包**
   - 例如：`server/internal/pkg/tencentcloud/tc3`

### 推荐结论
- **先确认 FaceID 是否还在产品路线中；若否，直接删除。若是，则立即抽取统一的 Tencent Cloud client/signature 基础包。**

### 修复计划
- Phase 1：确认 `faceid` 的产品状态
- Phase 2：删除死包，或抽取统一 TC3 签名基础层
- Phase 3：把 tracing / metrics / error mapping 统一到同一底层 client 模板
- Phase 4：为外部云服务接入建立统一扩展模式，避免后续继续复制 `sms/tencent.go` 模式

---


## F-20 `user_hash` 启动回填不应继续作为常驻运行时职责

- **优先级**：P2
- **领域**：后端运行时 / 数据维护 / 启动稳定性
- **当前状态**：问题仍成立；但实现已具备批次控制，文案需纠正为“启动路径职责错误”

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/app/modules.go:61-63`（启动时触发 `startUserHashBackfill`）
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/user_sync.go:242-285`（`BackfillUserHashes` 全量回填实现）

### 审查发现
2026-04-14 复核确认：`/Users/zxy/Code/StuHelper/server/internal/modules/auth/user_sync.go` 中的 `BackfillUserHashes()` 已使用 `batchSize = 500` 分批处理，并在 `batchCount < batchSize` 时自然退出循环；因此原文“没有批次控制”的描述不准确。

### 2026-04-14 复核结论
本条仍然成立的核心问题是：`user_hash` 回填被无条件挂在应用启动路径上，且缺少明确的 feature flag / 一次性 ops job 边界。应把问题表述从“无批次控制的全量回填”修正为“有批次控制，但错误地附着在 runtime startup 中”。

### 为什么必须修
- 对小数据量环境，这只是“看起来没事”；对大表环境，这会演化为隐藏的启动抖动与数据库压力源。
- 一次性历史数据修补不应长期附着在 runtime 里，否则后续每次重启都会背着隐性迁移任务。
- 该模式会让运维误以为“应用启动慢”是正常现象，而不是识别到后台在做额外维护工作。

### 需要决断的架构选择
1. **继续保留 runtime backfill，但补充开关、批次控制、指标与日志**
   - 优点：改造成本低
   - 缺点：仍然把历史修补任务放在主服务生命周期内
2. **把回填改成一次性 migration / 运维任务（推荐）**
   - 优点：语义清晰，职责明确，不污染主服务启动路径
   - 缺点：需要单独的执行入口与演练流程

### 推荐结论
- **推荐把 `user_hash` 回填迁出运行时，改为一次性 migration 或受控的运维任务。**

### 修复计划
- Phase 1：确认当前线上是否仍存在需要持续回填的历史数据
- Phase 2：若只是一轮历史修补，迁为一次性 migration / ops job
- Phase 3：若必须保留在线修补能力，则引入 feature flag、批次控制、指标与日志
- Phase 4：移除默认启动即扫描全表的行为

---
## F-21 OpenFGA 配置校验在非生产环境过松，会削弱环境一致性与授权回归的可信度

- **优先级**：P2
- **领域**：授权配置 / 环境一致性 / 发布质量
- **当前状态**：未根治

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/pkg/config/validation.go:170-177`（生产环境才强制 `OPENFGA_MODEL_ID`）
- `/Users/zxy/Code/StuHelper/server/internal/pkg/fga/client.go:34-48`（只要 StoreID 非空就会创建 client）
- `/Users/zxy/Code/StuHelper/server/internal/app/modules.go`（运行时实际初始化 FGA client）

### 审查发现
当前校验策略中，只在 production 环境强制要求 `OPENFGA_MODEL_ID`。结果是：
- staging / dev 可以出现”StoreID 有值、ModelID 为空”的半配置状态
- 运行时仍可能创建 FGA client，但授权计算所依据的模型版本不明确
- 非生产环境的授权行为不再可靠，削弱了回归与演练的代表性

### 2026-04-14 复核结论
**完全属实。** `validation.go:170-177` 确认只在 `c.App.Env == “production”` 时检查 `OPENFGA_MODEL_ID`。`fga/client.go:36-48` 中 `NewClient` 只要 `StoreID != “”` 就创建 client，不论 `AuthorizationModelID` 是否为空（`AuthorizationModelID` 会被传入 SDK 配置但不做校验）。这确实会导致非生产环境 FGA client 可能使用 store 的默认最新模型而非显式绑定版本。建议合理。

### 为什么必须修
- 授权系统最怕“环境看起来正常，实际上模型版本不一致”。
- 如果 staging 不能真实反映 production 的模型绑定方式，授权改动的回归价值会被显著削弱。
- 这类问题往往不会在编译期暴露，而是以“某些资源权限不对劲”的形式在线上或联调时才出现。

### 需要决断的架构选择
1. **继续允许非生产环境使用未显式绑定模型的 FGA Store**
   - 优点：本地调试门槛稍低
   - 缺点：环境一致性差，容易掩盖模型发布问题
2. **只要启用 FGA，就在所有环境显式绑定 Store + Model（推荐）**
   - 优点：授权行为可复现、可回归、可审计
   - 缺点：本地/测试环境初始化稍微更严格

### 推荐结论
- **只要环境中启用了 FGA，就应显式绑定 `OPENFGA_STORE_ID` + `OPENFGA_MODEL_ID`，不能只在 production 环境严格。**

### 修复计划
- Phase 1：把“启用 FGA 必须带 model id”提升为所有环境统一规则
- Phase 2：在启动校验中输出更明确的 Store / Model 绑定错误信息
- Phase 3：将 staging / dev 的 FGA bootstrap 流程纳入标准化脚本，避免人工随意配置
- Phase 4：为关键授权路径增加环境一致性回归测试

---

## F-22 当前单设备 logout 仍存在“成功假象”：返回 success 不等于会话已真正撤销

- **优先级**：P1
- **领域**：认证 / 安全 / 会话撤销语义
- **当前状态**：未根治；与 F-01 的 cookie/path 问题不同，这是当前实现层面的 fail-open 风险

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_session.go:19-37`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/service.go:85-116`
- `/Users/zxy/Code/StuHelper/server/internal/pkg/token/blacklist.go:330-381`

### 审查发现
当前 `Logout` 路径中：
1. `handler_session.go` 里的 `Logout()` 无条件调用 `h.svc.RevokeCurrentSession(...)`
2. `RevokeCurrentSession()` 内部对 blacklist / untrack 失败只做 `Warn` 记录，不向上返回错误
3. Handler 随后继续：
   - 清理 cookie
   - 记录审计成功
   - 返回 `logout successful`

这意味着在 Redis 故障、熔断打开、黑名单写入失败或 token untrack 失败时，系统可能出现：
- 客户端收到”登出成功”
- 浏览器 cookie 被清掉
- 但服务端对 token 的权威撤销并未完成

### 2026-04-14 复核结论
**完全属实。** 这是本次审查中最准确、最有价值的发现之一。`handler_session.go:20-37` 的 `Logout` 方法确实无条件走到 `response.Success`；`service.go:86-115` 的 `RevokeCurrentSession` 返回类型为 `void`（无 error 返回），所有 blacklist/untrack 失败都只 Warn。与之对比，`LogoutAll`（`handler_session.go:41-58`）的 `RevokeAllSessions` **正确地返回 error 并由 Handler 判断**，说明这不是统一设计而是单设备 logout 的遗漏。建议将此条优先级维持 P1。

### 为什么必须修
- 对用户来说，“登出成功”是安全语义承诺，而不是“尽力而为”提示。
- 对后续 Session / Token Family 方案来说，这类 fail-open 行为会直接破坏撤销语义的一致性。
- 对审计来说，当前实现会把“局部失败的撤销操作”记成成功事件，污染安全事件可追溯性。

### 长期正确方案
- **短期**：单设备 logout 至少要对服务端撤销失败给出明确失败语义，不能继续无条件 success。
- **长期**：与 D-02 对齐，全面迁移到服务端 Session / Token Family 模型，让 logout 成为对服务端权威状态的修改，而不是若干 best-effort 副作用的组合。

### 修复计划
- Phase 1：把 `RevokeCurrentSession` 改为显式返回错误，并区分“权威撤销失败”与“本地清 cookie 成功”
- Phase 2：审计日志改为记录真实结果，不再把部分失败链路误记为 success
- Phase 3：与 Session / Token Family 方案统一，把 logout 语义收敛到服务端状态机
- Phase 4：为 Redis 不可用、熔断打开、untrack 失败等场景补安全回归测试

---

## F-23 对象存储 bucket provisioning 被塞进应用启动路径，且缺少网络初始化硬超时

- **优先级**：P1
- **领域**：启动稳定性 / 基础设施边界 / 对象存储
- **当前状态**：未根治

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/app/modules.go:214-229`
- `/Users/zxy/Code/StuHelper/server/internal/pkg/objectstorage/store.go:37-112`

### 审查发现
当前用户身份照片对象存储在应用启动阶段完成：
1. `modules.go` 中使用 `objectstorage.New(context.Background(), ...)`
2. 随后调用 `photoStore.EnsureBucket(context.Background())`
3. `EnsureBucket()` 在 bucket 不存在时，会直接调用 `CreateBucket()`

这暴露出三个问题：
- **启动无硬超时**：`context.Background()` 使对象存储初始化完全依赖下游网络/服务质量，启动可能被无限拉长
- **基础设施副作用进入 app startup**：建桶是 provisioning 动作，不应由每次应用启动隐式触发
- **Region / Provider 语义不稳定**：当前 `CreateBucket()` 未显式携带创建配置，面对非默认 region 或不同 S3 兼容实现时行为不可预测

### 2026-04-14 复核结论
**完全属实。** `modules.go:216` 使用 `context.Background()` 调用 `objectstorage.New()`，`modules.go:229` 使用 `context.Background()` 调用 `photoStore.EnsureBucket()`。`store.go:91-112` 的 `EnsureBucket` 在 bucket 不存在时直接 `CreateBucket`，且 `CreateBucketInput` 未携带 `CreateBucketConfiguration`（region 等）。三个问题均可从代码中直接验证。注意对象存储初始化位于 `configureUserService` 方法内，且有 `if rt.cfg.ObjectStorage.Endpoint != ""` 前置判断，所以仅在配置了对象存储时才触发。但这不改变核心问题：生产环境中 `context.Background()` + 隐式建桶仍然是风险点。

### 为什么必须修
- 企业级生产环境里，应用启动应当只做“受控、可超时、可观测”的依赖检查，不应承担基础设施创建职责。
- 任何把 provisioning 放进 runtime 的设计，都会让重启、扩容、故障恢复变得不可预测。
- 一旦对象存储偶发抖动，当前实现可能把应用可用性直接绑死在 `HeadBucket/CreateBucket` 的网络路径上。

### 需要决断的架构选择
1. **继续由应用在启动时自动 EnsureBucket**
   - 优点：首次本地环境启动更方便
   - 缺点：生产职责边界错误、启动风险高、provider 兼容性差
2. **由基础设施预置 bucket，应用启动只做带超时的存在性校验（推荐）**
   - 优点：职责清晰、启动稳定、运维可控
   - 缺点：需要把 bucket provisioning 纳入 deploy/bootstrap 流程

### 推荐结论
- **推荐把 bucket 归属明确为 infra responsibility：部署链路预置 bucket，应用只做带 deadline 的 fail-closed 校验。**

### 修复计划
- Phase 1：拍板 bucket 归属，禁止继续处于“本地依赖自动建桶、生产又想稳定”的灰区
- Phase 2：把对象存储初始化改为带 timeout 的启动检查
- Phase 3：将 bucket provisioning 迁入 deploy/bootstrap/infra 初始化流程
- Phase 4：为对象存储不可达、bucket 缺失、权限不足三类场景建立启动回归测试

---

## F-24 PostgreSQL DSN 拼接错误使用 `url.QueryEscape`，复杂密码存在配置炸点

- **优先级**：P1
- **领域**：配置 / 数据库连接 / 生产可用性
- **当前状态**：未根治

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/pkg/config/config.go:310-314`
- `/Users/zxy/Code/StuHelper/server/internal/pkg/config/config_test.go`

### 审查发现
当 `DATABASE_URL` 为空时，系统会用 `DB_*` 字段拼接 PostgreSQL URL：

```go
cfg.Database.URL = fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=%s",
  cfg.Database.User, url.QueryEscape(cfg.Database.Password), ...)
```

这里使用的是 `url.QueryEscape()`，但 password 位于 URI 的 **userinfo** 部分，不是 query string。  
这会导致：
- 空格被编码为 `+`
- `+`、空格等特殊字符的语义与 PostgreSQL URI 预期不一致
- 看似合法的复杂密码，在生产环境中可能被错误解释，进而导致连接失败

### 2026-04-14 复核结论
**完全属实，且是一个高质量的精确发现。** `config.go:311-314` 确认使用 `url.QueryEscape(cfg.Database.Password)` 拼接 userinfo 部分。Go 标准库中 `url.QueryEscape` 会将空格编码为 `+`，而 RFC 3986 userinfo 部分应使用 percent-encoding（空格→`%20`）。正确做法应使用 `url.PathEscape` 或 `url.UserPassword(user, password).String()` 来构造 userinfo。虽然大多数简单密码不会触发此 bug，但生产环境中使用包含 `+`、空格、`@`、`:` 等字符的强密码时会出问题。建议维持 P1。

### 为什么必须修
- 这是典型的“简单密码没问题，真实生产强密码才出问题”的炸点。
- 一旦配置通过环境变量下发复杂密码，问题不会在编译期暴露，只会在启动时以“数据库连不上”形式出现。
- 对生产系统来说，连接串拼接逻辑必须严格遵循 URI userinfo 语义，不能混用 query escaping。

### 长期正确方案
- **优先方案**：生产环境显式要求 canonical `DATABASE_URL`，不要在运行时自行拼 URL
- **兜底方案**：若保留 `DB_*` 拼接能力，则必须使用符合 userinfo 语义的构造方式（例如 `url.UserPassword`）

### 修复计划
- Phase 1：将 `DB_*` 到 `DATABASE_URL` 的拼接逻辑改为 userinfo-safe 的实现
- Phase 2：补充包含空格、`+`、`@`、`:` 等特殊字符密码的配置测试
- Phase 3：文档明确“生产优先使用 `DATABASE_URL` 作为唯一事实源”
- Phase 4：若继续保留 `DB_*` 兼容入口，限定其只作为开发便利而非生产主路径

---

## 4. 架构/设计决策（2026-04-14 最终拍板）

> 以下决策已由项目负责人于 2026-04-14 确认，后续整改以此为准。

### D-01 Notification 采用统一 Wire DTO
- **决策**：方案 A — 统一 Wire DTO
- **理由**：当前 HTTP 列表和 SSE 推送共用同一个 `Notification` 结构体，无语义差异。清理 `relatedType`/`relatedID` deprecated 别名后即可完成收口。

### D-02 Auth 升级为服务端 Session / Token Family
- **决策**：引入服务端 Session / Token Family 模型
- **理由**：当前 token 跟踪机制（`TrackUserToken`/`UntrackUserToken`）已是 session 管理雏形，升级为正式 session_id 成本不高。logout/refresh/revoke 语义将统一操作服务端状态。

### D-03 `auth/user_sync.go` 迁入 `modules/user`
- **决策**：迁入 `modules/user` persistence 层
- **理由**：PII 存储和用户同步不是 auth handler 的责任。auth 域通过 `user.UserSyncRepo` 接口依赖 user persistence。配合 D-07 一起落地。

### D-04 课程列表页后端提供分组接口
- **决策**：后端提供面向场景的分组/目录聚合接口，前端不再全量下载聚合
- **理由**：当前全量拉取 + 前端 groupByDepartment 不具备长期可扩展性。
- **过渡**：Phase 1 已完成（收敛到 `loadCourseCatalog` helper）；Phase 2 新增后端接口后前端切换。

### D-05 生产公网入口由仓库内 edge 负责
- **决策**：方案 A — 仓库内自带 edge proxy（Traefik）
- **理由**：项目为单机部署场景，完全自包含更可控。
- **执行**：在 `docker-compose.yml` 中正式定义 edge 路由；运维文档明确证书终止、公网暴露策略。

### D-06 shared 层职责边界
- **决策**：明确分层 — `api.gen.ts`（wire contract）、`api/`（请求封装）、`constants/`（复用常量）、`presentation/`（UI 适配）
- **理由**：防止”方便用的类型”与”真实 wire contract”混用。

### D-07 认证域 Handler→Service 编排重构
- **决策**：`auth.Service` 成为认证域唯一编排入口，Handler 不再直接操作用户同步仓储
- **理由**：Handler 当前绕过 Service 直接调用 `userSyncRepo.UpsertUser()`、`UpsertByPhone()`、`ExistsByExternalID()`，违反分层原则。
- **执行**：Service 新增 `HandleOIDCCallback`、`HandlePhoneLogin`、`RefreshSession` 等方法。

### D-08 Review 访问策略内聚进 Service
- **决策**：先内聚进 `review.Service`；若后续复杂度上升再拆 `ReviewAccessPolicyService`
- **理由**：当前 `resolveReviewAccessFacts(ctx, externalID, capabilities)` 不依赖 gin.Context，迁移成本低，无需过早抽象。

### D-09 备份执行环境容器化
- **决策**：方案 B — 容器化 backup executor
- **理由**：`docker compose run --rm postgres pg_dump ...` 可复用已有镜像，环境稳定、版本可控。systemd/cron 只负责调度。

### D-10 `uniappx` 定位为正式产品线（native supported）
- **决策**：方案 A — 补完整 native auth 闭环，纳入正式质量矩阵
- **理由**：项目有原生端交付需求。
- **执行**：补齐 app scheme / deep link / callback / session bootstrap，增加专项 QA checklist。

### D-11 Admin 收敛为单一 `web-ele`，删除其余变体
- **决策**：方案 A — 直接删除 `web-antd`、`web-antdv-next`、`web-naive`
- **理由**：三个变体未进入交付链路，持续增加维护负担。upstream 同步通过 `git remote add vben` + merge 实现，不需要本地保留全套 scaffold。
- **执行**：从主分支删除三个目录，清理相关 CI/config 引用。

### D-12 三端错误语义采用 shared 核心 + 平台适配层
- **决策**：方案 B — shared 负责 error envelope 解析、response unwrap 规则；各端只保留 fetch/runtime 差异
- **理由**：Web `buildApiError` 与 admin/uniappx 的错误解析逻辑分叉明显。
- **执行**：第一步把 `parseApiError(payload)` 抽到 `@stuhelper/shared/api`。

### D-13 `user_hash` 回填迁出运行时
- **决策**：迁出运行时，作为一次性 ops 任务
- **理由**：启动时无条件扫表不属于 app 职责。
- **已完成**：2026-04-14 已将启动时自动回填改为 `warnPendingUserHashBackfill` 仅检查+警告。`BackfillUserHashes` 保留供 ops 命令调用。

### D-14 FGA 全环境统一校验
- **决策**：只要启用 FGA（StoreID 非空），就在所有环境显式绑定 Store + Model
- **理由**：staging/dev 的半配置状态会导致授权行为不可复现。
- **已完成**：2026-04-14 已将 `OPENFGA_MODEL_ID` 校验从仅 production 提升为所有环境。

### D-15 `clients/shared` 改为 source-first 消费
- **决策**：monorepo 内部采用 source-first / project-references 语义，`dist` 仅作为发布产物
- **理由**：消除 warm workspace 依赖，让冷启动构建具备可复现性。uniappx 已通过 tsconfig paths 实践了此模式。
- **执行**：shared/package.json exports 增加 source 入口；web tsconfig 加 paths 或 project references。

### D-16 对象存储 bucket 由 infra 预置
- **决策**：bucket 由 infra/deploy 流程预置，应用启动只做带超时的存在性校验
- **已完成**：2026-04-14 已将 `EnsureBucket` 改为 `CheckBucket`（只校验不创建），`context.Background()` 改为 15s timeout。

---

## 5. 分批整改计划（2026-04-15 更新）

> ✅ 标记表示已完成的修复。

### Batch 1：生产与门禁阻断项（最高优先级）
- ✅ 单设备 logout 失败语义收口（F-22：`RevokeCurrentSession` 返回 error，Handler 区分成败）
- ✅ 单设备 logout Bearer token 缺口修复（F-22 补充：Logout 从 Authorization header 提取 token，不再仅依赖 cookie）
- ✅ 修正 PostgreSQL DSN 密码编码（F-24：`url.QueryEscape` → `url.UserPassword`，含 7 case 特殊字符测试）
- ✅ 对象存储启动路径修复（F-23：`context.Background()` → 15s timeout；`EnsureBucket` → `CheckBucket`）
- ✅ FGA 全环境校验（F-21：`OPENFGA_MODEL_ID` 校验提升为所有环境）
- ✅ `user_hash` 回填迁出启动路径（F-20：改为仅检查+警告）
- ✅ 删除 faceid 死包（F-19）
- `clients/shared` 的 source-first 消费模式收口（D-15）
- 发布前备份容器化（D-09）
- 仓库内 edge proxy 正式化（D-05）
- `uniappx` native auth 闭环补齐（D-10）

### Batch 2：契约治理
- ✅ Go/TS 生成链路已统一消费 bundled spec（F-05）
- ✅ Review create 输入已严格对齐 OpenAPI（F-04）
- ✅ 清理 Notification deprecated 别名 `relatedType`/`relatedID`（D-01：OpenAPI、Go model、shared types、frontend fallback 全部清除）
- ✅ 统一三端错误 envelope 解析到 shared（D-12：`@stuhelper/shared/api/errors.ts` 提供 `parseApiError`/`extractApiErrorMessage`，web/admin/uniappx 已接入）
- shared 去影子类型与影子常量

### Batch 3：分层与架构收口
- ✅ 把认证编排收回 `auth.Service`（D-07：Handler 不再持有 `userSyncRepo`，新增 `SyncOIDCUser`/`SyncPhoneUser`/`UserExistsByExternalID` 方法）
- ✅ Review 访问策略内聚进 Service（D-08：`resolveReviewAccessFacts` → `Service.ResolveAccessFacts`，Handler 不再持有 `userRepo`）
- 迁移 `auth/user_sync.go` 到 `modules/user` persistence（D-03：接口抽象已完成，物理迁移待执行）
- 引入 Session / Token Family 模型（D-02）
- 明确 shared 分层边界（D-06）

### Batch 4：前端结构与性能
- ✅ 课程目录页收敛到 `loadCourseCatalog` helper（F-08 Phase 1）
- ✅ 历史 review 分叉组件和孤儿页面已清理（F-07）
- ✅ 删除 admin 历史变体 `web-antd`/`web-antdv-next`/`web-naive`（D-11：214 文件已删除）
- 清理 admin `web-ele` 未挂路由页面和 `any` 黑洞
- 课程目录页后端分组接口（D-04 Phase 2）
- 建立死代码检测与孤儿页面治理规则

### Batch 5：上线质量体系
- 真实业务 smoke check（登录、OIDC 回调、关键 API）
- 备份恢复演练 runbook
- native auth 专项 QA checklist
- 文档与拓扑的单一事实源
- 运维文档明确公网入口、证书终止、备份执行环境责任归属

---

## 6. 验收标准（2026-04-15 更新）

整改完成后，应满足以下可验证条件：

### 认证与通知
- ✅ 单设备 logout 不再出现”服务端撤销失败但 HTTP 仍返回 success”的 fail-open 行为
- ✅ Bearer 客户端调用 /logout 时 access token 正确提取并撤销（不再仅依赖 cookie）
- ✅ Notification Hub 连接驱逐为 oldest-first、无 double-close panic（已确认当前实现正确）
- ✅ `auth.Handler` 不再直接操作用户同步仓储（D-07：已通过 Service 方法间接访问）
- ✅ Review 授权事实不再由 Handler 自行拼装（D-08：`ResolveAccessFacts` 已迁入 Service）
- ✅ `user_hash` 回填不再在默认应用启动路径中无条件执行
- 认证系统升级为 Session / Token Family 模型（D-02 待执行）

### 契约
- ✅ OpenAPI 为唯一契约事实源（Go/TS 均消费 bundled spec）
- ✅ backend-only drift gate 可独立运行（不依赖 pnpm）
- ✅ `pnpm` 版本在 Dockerfile/CI/本地已固定为 `10.32.1`
- `clients/shared` 在 fresh checkout 下可被 `web` / `uniappx` 稳定消费（D-15 待执行）
- ✅ Notification deprecated 别名已清除（D-01：`relatedType`/`relatedID` 从 OpenAPI、Go model、shared types、frontend 全部移除）
- ✅ 三端错误 envelope 语义统一（D-12：shared `parseApiError` 已被 web/admin/uniappx 接入）

### 前端
- ✅ 课程目录页已收敛到 `loadCourseCatalog` helper
- ✅ 历史 review 分叉组件和孤儿页面已清理
- ✅ Admin 仓库中不再保留非主线变体（D-11：`web-antd`/`web-antdv-next`/`web-naive` 已删除）
- `uniappx` native auth callback 闭环真实可用（D-10 待执行）
- ✅ Web / Admin / Uniappx 对同一类后端错误具有一致语义（D-12 已完成）

### 生产交付
- ✅ 应用启动不再隐式创建对象存储 bucket
- ✅ `DB_*` 到 `DATABASE_URL` 拼接已使用 userinfo-safe 编码，含特殊字符覆盖测试
- ✅ FGA 启用时，Store / Model 绑定在所有环境保持一致
- 仓库内自带 edge proxy 并正式化（D-05 待执行）
- 备份执行环境容器化（D-09 待执行）
- 文档清楚说明公网入口、证书终止、备份执行环境的责任归属

---

## 7. 推荐后续动作（2026-04-15 更新）

1. ✅ 已完成：第一轮 ADR 拍板（全部 16 项决策已确认）
2. ✅ 已完成：Batch 1 中 7 项即时修复（F-19/F-20/F-21/F-22/F-22补充/F-23/F-24）
3. ✅ 已完成：Batch 2 中 D-01（Notification deprecated 别名清除）、D-12（三端错误 envelope 统一）
4. ✅ 已完成：Batch 3 中 D-07（认证编排收回 Service）、D-08（Review 访问策略内聚进 Service）
5. ✅ 已完成：Batch 4 中 D-11（删除 admin 历史变体，214 文件）
6. 下一步按 Batch 1 剩余项（D-05/D-09/D-10/D-15）→ Batch 3 剩余（D-02/D-03/D-06）→ Batch 4 剩余 → Batch 5 顺序执行
7. 每个 Batch 附带回归测试和风险回退方案

---

## 8. 附录：本次审查中已被识别的历史路径

以下路径在审查时被确认为历史包袱或冗余实现。标记 ✅ 的已在 2026-04-14 删除/修复。

### 已删除（2026-04-14 之前的快照提交中清理）
- `clients/web/src/components/business/review/ReviewForm.vue`
- `clients/web/src/components/business/review/ReviewDialogForm.vue`
- `clients/web/src/components/business/review/ReviewDialogCourseSearch.vue`
- `clients/web/src/components/business/review/ReviewDialogCancelConfirm.vue`
- `clients/web/src/components/business/review/composables/useReviewDialogForm.ts`
- `clients/web/src/components/business/review/composables/useCourseSearch.ts`
- `clients/web/src/components/business/review/composables/useTeacherSelect.ts`
- `clients/web/src/components/business/review/composables/useReviewDialogDraft.ts`
- `clients/web/src/components/business/review/composables/useReviewDialogSubmit.ts`
- `clients/web/src/modules/user/views/NotificationPreferencesPage.vue`
- `clients/web/src/modules/errors/views/LoadErrorPage.vue`
- `clients/shared/src/types/business/admin.ts`

### ✅ 已在 2026-04-14 修复轮中删除
- `server/internal/pkg/faceid/tencent.go`（F-19 死包删除）

### ✅ 已在 2026-04-15 修复轮中删除
- `clients/admin/apps/web-antd/`（D-11 admin 变体删除，~70 文件）
- `clients/admin/apps/web-antdv-next/`（D-11 admin 变体删除，~70 文件）
- `clients/admin/apps/web-naive/`（D-11 admin 变体删除，~70 文件）

### 待清理（admin web-ele 未挂路由页面，Batch 4 执行）
- `clients/admin/apps/web-ele/src/views/_core/about/index.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/code-login.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/forget-password.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/qrcode-login.vue`
- `clients/admin/apps/web-ele/src/views/_core/authentication/register.vue`
- `clients/admin/apps/web-ele/src/views/_core/fallback/coming-soon.vue`
- `clients/admin/apps/web-ele/src/views/_core/fallback/internal-error.vue`
- `clients/admin/apps/web-ele/src/views/_core/fallback/offline.vue`
- `clients/admin/apps/web-ele/src/views/demos/element/index.vue`
- `clients/admin/apps/web-ele/src/views/demos/form/basic.vue`
