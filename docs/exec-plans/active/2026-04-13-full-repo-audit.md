# 2026-04-13 全库审查报告与整改计划

> 目的：把本次针对 `StuHelper` 仓库的全量审查结果沉淀为一份可执行、可追踪、可决策的工程文档。  
> 背景：本报告记录的是本轮审查期间识别出的主要问题、原因、风险、建议方案与需要拍板的架构决策。  
> 审查快照：分支 `codex/enterprise-remediation`，基线快照提交 `05704c4`（`chore: snapshot current remediation workspace`）。

---

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

---

## 2. 执行摘要

### 2.1 总结
- **P0：未发现明确 P0**
- **P1：存在多项影响生产稳定性、上线门禁、契约一致性的关键问题**
- **P2：存在较明显的架构债、重复实现、历史包袱和性能模型问题**
- **综合结论：仓库已有较完整的工程骨架，但在审查时点并不等价于“可长期稳态上线生产”**

### 2.2 最高优先级问题清单
1. 认证登出链路的会话吊销语义不完整
2. 通知 SSE Hub 存在连接管理实现风险
3. Notification / Review 契约层发生过明显漂移
4. `prod_init_contract` 与真实生产初始化脚本失配
5. 生产公网入口、备份对象存储、前端 SSO 构建配置缺少完整闭环

### 2.3 当前状态说明
本次审查之后，工作区已经出现一批战术性修复与代码清理；但这份文档的重点不是“本轮改了什么”，而是：
- **问题是什么**
- **为什么必须修**
- **长期正确的方案是什么**
- **哪些地方需要产品/架构拍板**

---

## 3. 详细审查发现

---

## F-01 认证登出链路的会话吊销语义曾不完整，长期仍应升级为服务端 Session 模型

- **优先级**：P1
- **领域**：认证 / 安全 / 会话管理
- **当前状态**：短期战术修复已落地；长期架构问题仍未完全解决

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_cookies.go`
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/handler_session.go`
- `/Users/zxy/Code/StuHelper/server/internal/pkg/middleware/auth.go`

### 审查发现
审查时，refresh cookie 仅在 refresh 路径下可见，而 logout 路由读取当前请求 cookie 来吊销当前会话。这样会导致：
- 用户发起 `/api/v1/auth/logout` 时，服务端无法稳定拿到 refresh token
- 最终只能吊销 access token，refresh token 可能继续有效

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

## F-02 Notification SSE Hub 的连接管理实现风险高，连接上限策略需要正式化

- **优先级**：P1
- **领域**：实时通知 / 高并发稳定性
- **当前状态**：已有短期修补；仍需做结构化设计收口

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/notification/hub.go`
- `/Users/zxy/Code/StuHelper/server/internal/modules/notification/handler.go`
- `/Users/zxy/Code/StuHelper/server/internal/modules/notification/service_test.go`

### 审查发现
审查时，Hub 的连接驱逐与 `Unsubscribe()` 之间存在二次关闭 channel 的风险，属于可触发 panic 的实现问题。同时，原始驱逐策略依赖 map 遍历，实际效果更接近“随机驱逐”，而不是注释中描述的“驱逐最老连接”。

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

## F-03 Notification 契约曾长期漂移；仍需决定 HTTP DTO 与 SSE DTO 是否正式分离

- **优先级**：P1
- **领域**：OpenAPI 契约 / 后端 DTO / 前端共享类型
- **当前状态**：本轮已有收口动作，但设计层决策仍需明确

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
审查时，Notification 在三个层面出现了不同语义：
- OpenAPI 只定义了窄字段与少量 type
- 后端真实通知里还有 `payload`、`sourceModule`、`sourceId`、`sourceUrl`
- shared 层又手工补了一套 superset 类型与跳转逻辑

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

## F-04 Review 创建请求边界曾被 shared API 放松；API 输入必须严格服从 OpenAPI

- **优先级**：P1
- **领域**：契约一致性 / 前端 API 设计
- **当前状态**：已有部分收口；原则需要写死

### 代码定位
- `/Users/zxy/Code/StuHelper/server/api/components/schemas/review.yaml`
- `/Users/zxy/Code/StuHelper/clients/shared/src/api/reviews.ts`
- `/Users/zxy/Code/StuHelper/clients/shared/src/types/business/review.ts`
- `/Users/zxy/Code/StuHelper/clients/web/src/components/business/review/reviewPayload.ts`
- `/Users/zxy/Code/StuHelper/clients/web/src/types/review.ts`

### 审查发现
审查时，OpenAPI 中 `PostReviewRequest` 明确要求 `courseID/title/termID/content/ratings` 必填，但 shared API 输入曾被定义成可选项，并在 API 包装层偷偷补空字符串。

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

## F-05 OpenAPI 生成链路曾存在 Go 用 bundled、TS 用 raw spec 的双源问题

- **优先级**：P1
- **领域**：生成链路 / 契约治理
- **当前状态**：已开始收敛；治理规则需要固化

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/api/gen/generate.go`
- `/Users/zxy/Code/StuHelper/server/Makefile`
- `/Users/zxy/Code/StuHelper/clients/package.json`
- `/Users/zxy/Code/StuHelper/server/api/openapi.bundled.yaml`

### 审查发现
此前 Go 生成链路使用 bundled spec，而 TS 生成链路直接读取 raw spec。两边虽然都来自 OpenAPI，但前处理输入并不一致，天然容易埋下差异。

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

## F-06 `auth/user_sync.go` 的 SQL 逃逸出 Repository 层，违反后端分层铁律

- **优先级**：P2
- **领域**：后端架构 / 分层边界
- **当前状态**：仍需根治

### 代码定位
- `/Users/zxy/Code/StuHelper/server/internal/modules/auth/user_sync.go`
- 关键函数：
  - `UpsertUser`
  - `FindByPhone`
  - `ExistsByExternalID`
  - `BackfillUserHashes`

### 审查发现
该文件混合了：
- 用户持久化 SQL
- 认证域流程
- 启动时 backfill 任务

这直接违背了仓库已声明的分层原则：
- SQL 只在 Repository
- 业务编排在 Service
- HTTP 协议在 Handler

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

## F-07 前端存在一整套历史 review 创建链路与孤儿页面，说明“删除无用代码”还未制度化

- **优先级**：P2
- **领域**：前端工程治理 / 复用 / 可维护性
- **当前状态**：本轮已删除一批；需要建立持续清理机制

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
这些实现与当前 `ReviewDialog.vue` / `PostReviewPage.vue` 所承载的正式链路重复，但未被真实路由或调用链消费，属于典型的：
- 历史原型残留
- 局部重构后未清尾
- 双实现并存

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
- `/Users/zxy/Code/StuHelper/clients/web/src/modules/course/views/CourseListPage.vue:60-115`
- 关键实现：
  - `pageSize = 100`
  - `Promise.all(...)` 拉取全部分页
  - `groupByDepartment(courses)` 前端聚合

### 审查发现
当前页面通过先拉第一页算出总页数，再把剩余页全部并发拉完，然后前端再分组。这个模式本质上把“课程目录页”做成了“全库批量下载器”。

### 为什么必须修
随着课程量增长，这个页面会出现：
- 首屏时延不可控
- 并发请求数不可控
- 移动端网络成本过高
- 服务端和客户端都承担不必要压力

### 需要决断的架构选择
1. **服务端直接返回按院系分组的数据（推荐）**
2. 保持分页课程列表，但前端按需展开 / 虚拟滚动
3. 继续全量拉取（不推荐）

### 推荐结论
- **如果页面目标是“按院系浏览全部课程”，最长期正确的做法是由后端直接提供面向场景的聚合接口。**

### 修复计划
- 新增专门的 course catalog / grouped listing API
- 前端改为分页或按分组懒加载
- 大列表页面统一引入虚拟滚动策略

---

## F-09 shared/business/admin.ts 这类手工业务类型曾与真实 API 漂移，说明 shared 的职责边界需要重新定义

- **优先级**：P2
- **领域**：前端共享层设计
- **当前状态**：已删除明显历史包袱；边界治理仍需继续

### 代码定位（历史发现，已在快照提交中移除）
- `clients/shared/src/types/business/admin.ts`

### 审查发现
该文件在审查时出现的问题包括：
- 与 OpenAPI 生成类型重复
- 枚举值与真实 API 不一致
- 仓库内几乎找不到可靠运行时消费者

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
- **当前状态**：已修一部分门禁问题；仍需正式上线架构决策

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
审查时发现：
1. `prod_init_contract` 与真实脚本失配，说明 CI 门禁本身不可信
2. 生产入口并未在仓库内闭环，大量服务仅绑定 `127.0.0.1`
3. 前端 SSO URL 存在构建时/部署时双源
4. 备份对象存储缺少完整硬校验
5. smoke check 只能证明服务起来了，不能证明关键业务链路可用

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
- 把 SSO / API / Grafana / Alertmanager / Prometheus 的入口策略写死
- 把备份对象存储配置纳入硬失败 preflight
- 把 smoke check 升级为真实业务冒烟：登录、OIDC 回调、关键 API、指标上报链路

---

## F-11 前端 grade 常量、通知跳转逻辑、类型守卫曾存在多份平行定义，说明“复用优先”未被系统执行

- **优先级**：P2
- **领域**：前端复用 / 单一事实源
- **当前状态**：已有收敛动作；需要作为长期规范固定下来

### 代码定位
- `clients/shared/src/constants/review.ts`
- `clients/shared/src/api/reviews.ts`
- `clients/shared/src/api/draft.ts`
- `clients/web/src/types/guards.ts`
- `clients/shared/src/notification.ts`
- `clients/web/src/components/common/NotificationBell.vue`
- `clients/web/src/modules/user/views/NotificationsPage.vue`

### 审查发现
同一业务概念曾在多个地方各写一份：
- Review grade 枚举
- 通知跳转解析
- 一些轻量业务校验

### 为什么必须修
这种重复不会立刻爆炸，但会在每次迭代里制造“轻微不一致”，最终演化成线上差异行为。

### 长期正确方案
- 共享常量只定义一次
- 共享 helper 只定义一次
- 业务守卫逻辑必须向 shared 收敛，不允许 web 再维护影子副本

---

## 4. 需要拍板的架构/设计决策

### D-01 Notification 是否采用统一 Wire DTO
- **推荐**：统一 Wire DTO
- **原因**：简化契约、生成、测试、前端消费

### D-02 Auth 是否升级为服务端 Session / Token Family
- **推荐**：是
- **原因**：单设备登出、logout-all、refresh 旋转都能语义稳定

### D-03 `auth/user_sync.go` 的归属
- **推荐**：迁入 `modules/user` persistence
- **原因**：PII 存储不是 auth handler 的责任

### D-04 课程列表页的产品/接口形态
- **推荐**：后端提供分组/目录接口，而不是前端全量下载聚合

### D-05 生产公网入口归属
- **推荐**：明确由仓库内或仓库外单方负责，禁止灰色地带

### D-06 shared 层职责边界
- **推荐**：`wire contract`、`api`、`constants`、`presentation` 明确分层

---

## 5. 分批整改计划

### Batch 1：生产与门禁阻断项（最高优先级）
- 会话吊销模型统一
- Notification Hub 稳定性兜底
- `prod_init_contract`、preflight、deploy、bundle 保证一致
- 前端 SSO / API build config 单一来源

### Batch 2：契约治理
- Notification DTO 最终定版
- Review create 输入严格化
- Go/TS/OpenAPI 生成输入统一
- shared 去影子类型与影子常量

### Batch 3：分层与架构收口
- 迁移 `auth/user_sync.go` 持久化逻辑
- 清理 auth service 冗余抽象
- 明确 shared 与 UI model 边界

### Batch 4：前端结构与性能
- 课程目录页性能重构
- 用户中心复用壳抽象
- review 投票/创建链路继续统一
- 建立死代码检测与孤儿页面治理规则

### Batch 5：上线质量体系
- 真实业务 smoke check
- 回滚与恢复演练
- 文档与拓扑的单一事实源

---

## 6. 验收标准

整改完成后，应满足以下可验证条件：

### 认证与通知
- 单设备 logout、logout-all、refresh 旋转均有明确语义和回归测试
- Notification 连接驱逐、重复注销、重连策略均可测试且无 panic

### 契约
- OpenAPI 为唯一契约事实源
- Go / TS 生成链路消费同一 bundled spec
- shared 不再保留漂移的影子 DTO

### 前端
- 不存在未接路由孤儿页面
- 不存在已确认无 consumer 的历史组件链路
- 课程目录页不再全量拉取后前端聚合

### 生产交付
- CI 门禁与真实脚本一致
- preflight 对关键配置 fail-closed
- 备份与恢复链路可验证
- 文档清楚说明公网入口和证书终止责任

---

## 7. 推荐后续动作

1. 先以本报告为依据，补齐一轮 ADR / 拍板：
   - Notification DTO
   - Session Model
   - Public Edge Ownership
2. 再按 Batch 1 -> Batch 5 的顺序执行，不要并行大爆炸式重构
3. 每个 Batch 必须附带：
   - 契约/设计说明
   - 回归测试
   - 风险回退方案

---

## 8. 附录：本次审查中已被识别的历史路径

以下路径在审查时被确认为历史包袱或冗余实现，后续阅读历史提交时可重点关注：
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

