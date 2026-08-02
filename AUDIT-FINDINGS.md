# StuHelper 审计与修复台账

> 状态日期：2026-08-02
> 性质：仓库根目录阶段性工作产物，不是运行时事实真源。代码、测试、OpenAPI 和 migration
> 与本文件冲突时，以前者为准。

本文件是 Claude 两轮审计和 Codex 交叉复核后的**唯一当前台账**。原始长报告
[`AUDIT-REPORT.md`](AUDIT-REPORT.md) 已应项目 owner 要求恢复，只作为历史取证和编号来源；
其中与本台账、源码、测试、OpenAPI、migration 或现行 ADR 冲突的内容均已失效。

删除前原文也仍可从 Git 历史独立取证：

```bash
git show 7f4849c3:AUDIT-REPORT.md
git show 7f4849c3:AUDIT-FINDINGS.md
```

原始报告中的编号继续作为别名使用，但原始严重度、修复方案和完成状态不再具有权威性。

## 1. 判定口径

| 状态 | 含义 |
|------|------|
| `confirmed` | 当前可达缺陷或明确违反契约，需要处置 |
| `partial` | 较窄根因真实，但原影响或方案被夸大 |
| `rejected` | 攻击链不可达、证据错误，或行为由产品/测试明确锁定 |
| `implemented` | 修复已进入提交并有与风险相称的回归证据 |
| `worktree` | 实现存在于当前工作树，但尚未形成独立提交 |
| `production-pending` | 本地实现完成，但真实生产账号、依赖或故障演练尚未验收 |
| `deferred` | P3/P4、条件性、先测或需产品决策，不属于当前致命/高/中级修复队列 |

完成代码修改不等于生产闭环。只有对应提交、回归测试和必要的真实运行证据全部存在，才能把
条目标为 `implemented`；生产依赖未验证时必须同时保留 `production-pending`。

## 2. 复核覆盖与去重

Claude 两轮共有 117 个原始标签，均已完成源码级复核：

| 轮次 | 原始标签 | Codex 去重/复核结果 | 当前用途 |
|------|---------:|---------------------|----------|
| 第一轮 | 61 | 连同 X-1/X-2 共 63 个标签；43 confirmed、13 partial、7 rejected；P2-14/P2-15 合并为同一根因 | 55 个需要实现、决策或观测；31 个完成核心处置，24 个降为长尾 |
| 第二轮 | 56 | 55 个唯一根因；33 confirmed、14 partial、8 rejected | 47 个需要处置；29 个 P1/P2 已完成核心处置，18 个 P3 长尾 |

两轮之间仍有重复根因，因此不得把两个分母或分子直接相加生成“全局完成率”。本台账按根因和
提交追踪，不再按 Claude 的 82 个“确认标签”创建 82 个任务。

## 3. 当前致命、高、中级队列

复核后没有 P0。Claude 两轮中已确认的 P1/P2 均已形成独立实施提交。PR #21 后续审查新增的
4 个线程也已再次按当前调用链交叉复核，均确认存在较窄但真实的 correctness / authorization
问题；对应最小完整修复均已形成独立提交并通过仓库级回归，详见第 5 节，当前本地 P0/P1/P2
代码队列已清空。真实 Casdoor/OpenFGA 账号闭环、发布和故障演练仍是
`production-pending`，统一列在第 8 节。

### IAM-01 不变量

- 只有配置的目标 Casdoor organization 中，owner 匹配、`IsAdmin=true`、未 forbidden/deleted
  的用户可以映射为 `super_admin`。
- 普通 OIDC/JWT `roles` claim、Casdoor role membership 和其他 organization 的 `IsAdmin`
  不参与授权。
- `school_admin`、`section_admin`、`section_moderator` 和 `section_reviewer` 继续以 PostgreSQL
  `authorization_grants` 为管理真源。
- login、native callback 和 refresh 在发放本地 session/token 前同步；已有 DB
  `super_admin` 的受保护请求实时复核 Casdoor 状态，lookup 失败时 fail-closed。
- Casdoor lookup 的依赖故障返回 503，不伪装成 401；refresh 不因该依赖故障清除客户端
  session cookie。跨 organization 或不可信主体仍按身份失败拒绝。
- 降权先提交 DB revoke fence，再由 outbox 精确删除并验证 OpenFGA tuple。
- provider 状态与 DB desired state 已一致但投影进入 terminal failed 时，下一次登录/refresh
  递增 revision、写 system audit，并把 dead-letter outbox 重新排队。
- 手工 grant API 不能创建或撤销 `super_admin`。
- 项目允许只有一个或没有 `super_admin`；MFA reset 不要求第二管理员，但 capability、step-up、
  审计和 self-disable 禁令继续生效。

## 4. 已关闭的致命/高优先级根因

| 别名 | 最终结论 | 提交 | 仍需真实环境验证 |
|------|----------|------|------------------|
| P0-1 | 受限管理员登录后落到不可访问首页；按 capability-filtered route 选择安全首页 | `c1cb9dfc`、`d9f84f45`、`771003c3` | 发布后 Admin 受控角色 |
| P1-1 | Koishi 缺 session/guild context 时可能越过群范围；改为 fail-closed | `f30f6ae5` | 真实私聊/群命令 |
| P1-2 | migration 文档把初始 schema 当唯一真源；改为完整有序 migration 集合 | `a8a3ac27` | 目标库升级窗口 |
| P1-3 | 物理备份缺少可靠校验和原子发布 | `109b1f44` | 生产对象存储、WAL/PITR |
| P1-4 | 部署包排除规则误删所需 env 模板 | `becc3ce6` | GitHub/远端部署 |
| P1-5 | 历史镜像回滚绕开时效门禁 | `96257b22` | 受保护 environment 回滚演练 |
| P1-6 | Admission SSE 阻塞优雅停机 | `e3fea6a1` | 真实 SIGTERM |
| P1-7 | replies ownership 依赖可选身份，但契约/路由未表达 | `fc6ddb19` | 发布路由 |
| P1-8 | 处理举报可能复活作者已删除的评课 | `84b64bf5` | 发布数据库 |
| P1-9 | Open Platform 在错误边界读取手机号 | `e5ffaacc` | 真实 Casdoor disclosure |
| P1-10 | 资源列表 `1 + 2N` 查询和单连接池阻塞 | `3bdbb44f` | 生产查询分布 |
| #3 + #4 | Koishi Console 全局/群 scope 和跨群数据隔离 | `7b448809` | scoped Console 操作员 |
| #9 | Web 评课 parser 丢失 `userVote` | `59322589` | 发布前端 |
| #18 | blacklist TTL 未使用已验证 token expiry | `0f8596ac` | 真实 Casdoor TTL/登出 |
| #44 | Bearer introspection 接受 refresh token purpose | `3d12d259` | 生产 introspection |
| U-1 | academics 管理链缺 MFA，且共享 step-up 状态码漂移 | `4b2f520b` | 真实 MFA |
| N-1 | Casdoor logout 参数 no-op，token family 未真正撤销 | `3d15c90d` | 受控账号 logout/logout-all |
| IAM-01 / #11 + #17 | 目标 Casdoor organization 用户对象 `IsAdmin` 成为 `super_admin` 管理权威；DB 保存带来源的 serving projection；登录/refresh 同步、受保护请求实时降权、投影失败自愈；移除双管理员 bootstrap、最后一名保护和 MFA reset 第二 reviewer | 本实施提交（`feat(iam): align super admin with Casdoor organization`） | 真实 Casdoor 账号晋升/降权、lookup 故障、OpenFGA applied/撤权和 MFA step-up |

旧的通用 role-claim reconcile 提交 `b199e19e` 已被最终窄 `IsAdmin` 方案取代，不单独作为
现行设计。

## 5. 已关闭的中优先级根因

| 别名 | 核心处置 | 提交 |
|------|----------|------|
| P2-1 | CI guard 变更触发真实消费 job | `8fa80a1f` |
| P2-2 | 供应链与 Koishi 包契约不可被 path filter 跳过 | `d1e99740` |
| P2-3 | repeat ledger 热查询下推 limit 并增加索引；retention 降为决策项 | `fc00b98e` |
| P2-6 | privileged Console listener 绑定作用域生命周期 | `6b4edc39` |
| P2-7 | 新生材料转发隔离单项 poison | `32966c6a` |
| P2-9 | Ansible deploy bundle 路径使用稳定绝对基准 | `d5566d99` |
| P2-10 | 普通学籍 importer 不再拥有身份证 enc/hash 列 | `c52ef537` |
| P2-12 | Oracle fan-out 端点共享用户级预算 | `ecefa044` |
| P2-13/P2-14 | claimed batch 批量上下文、单项隔离和可写时 lease 补偿；P2-15 为重复 | `81f4b62e` |
| P2-16 | 课程可空元数据贯穿 DB、OpenAPI、Go 和客户端 | `a81e5304` |
| P2-18 | 单实例敏感词快照在成功 mutation 后失效 | `3a69c5b5` |
| P2-20 | 教师投影刷新使用独立有界预算 | `214fd1ba` |
| P2-21/P2-22 | Oracle caller cancel/脏行与真实 backend failure 分类 | `541105d2` |
| P2-23 | outbox handler panic 进入既有 retry/dead-letter | `6ab9de00` |
| R-8 | Redis outage 不再冒充 OTP cooldown | `a4457e2a` |
| P3-7→P2 | 敏感批量导出每次请求写一条成功/失败审计 | `82eb3211` |
| P3-9→P2 | Redis 不可用时不再把未知 cache version 当 `v0` | `6bd5560d` |
| X-2/I-2 | runtime env 经 AST 分类进入模板或带理由 allowlist | `502d6671` |
| #5 | UniAppX H5 tabBar 资源进入输入树和产物契约 | `789017db` |
| 反向 mp-weixin | 删除虚假成功的微信构建声明，明确 H5-only 支持边界 | `61e610d7` |
| #6 | 草稿只在课程匹配时恢复，外课程草稿不误删 | `2739e97f` |
| #7 | provider-owned profile 字段进入可信 Casdoor account URL | `825f73cb` |
| #8 | 资源上传只接受窄枚举且经内容验证的 MIME refinement | `97b371a5` |
| #28 | Koishi 多域设置保存保留已确认成功的 baseline | `e5f9f79a` |
| #30 | Admission Console 明示 100 条窗口和完整 total | `b8413e72` |
| #34 | 重建粒子前释放旧 GSAP tween | `af8c8740` |
| #35 | 自有评课删除增加确认和 single-flight | `da17736e`、`700d0b7e` |
| #36 | ErrorBoundary 复用脱敏前端错误遥测 | `724620de` |
| #37 | AppShell 增加可聚焦 skip link | `798d7a50` |
| #38 | Toast timer 跨组件卸载保持有效 | `2a8e35a0` |
| #39 | 短暂 `/auth/me` 故障不清身份，投影轮询可恢复 | `27d71450` |
| #42 | 资料状态读取失败时不显示 false negative | `ffe6d0bf` |
| #43 | blacklist caller cancellation 对 breaker 记 neutral | `737a8f4c` |
| #45 | 日志只记录 route template，不记录 path credential | `9df1f3a9` |
| #51 | logout/refresh 竞争不再误报 token reuse | `0629b2f8` |
| #57 | scoped review grant 不再推出平台级 public full-content | `620d38be` |
| U-2 | 复用 long-lived JWKS cache，known key 可离线验证 | `7db400f0` |
| 反向 FGA | 非法 section tuple 按 grant 隔离、计数并告警 | `c589ae8a` |
| 反向 RatingBar | 可达评分条提供定性无障碍名称，不泄露精确值 | `b479bafa` |

这些提交均有对应定向回归；提交时记录的全量测试结果可从 Git 历史中的旧台账取证。生产、远端
或真实第三方依赖验证未发生的条目仍属于 `production-pending`，不能仅凭本表宣称已上线。

### 2026-08-01 PR #21 合并门禁复核

| 门禁 | 交叉复核结论 | 处置与证据 | 状态 |
|------|--------------|------------|------|
| Dependency review | `ansible-core==2.20.2` 命中 GHSA-w8p5-mx5w-cpqj，属于真实供应链风险 | 升级到修复后的稳定版 2.20.7；版本契约、三个 playbook 语法检查和全量基础设施契约通过 | `implemented` |
| Secret history scan | 两个命中均为历史生成 Go 文件中的 OpenAPI 压缩 Base64 分片，不是凭据 | 仅增加完整 commit/path/rule/line fingerprint；1545 个可达提交、约 40.50 MB 完整扫描为 0 未基线化泄漏 | `implemented` |
| Backend / Gosec G115 | 正常 claim 只有 1–5 次，实际不可溢出；但 `int` 到 PostgreSQL `INTEGER` 参数的转换缺少显式领域证明 | 转换前验证 claimed-action 范围，超界 fail-closed；边界测试、Admission 全包测试及 Gosec 412 文件/78,690 行扫描通过 | `implemented` |
| CodeQL excessive allocation | Service 最大 100、Repository 最大 200，实际内存放大链不可达，属于静态分析未识别字段钳位的误报 | Repository 使用独立归一化边界，响应空数组不再从请求值派生预分配容量；边界测试和 Authorization 全包测试通过 | `implemented` |

项目所有者同时决定整个 StuHelper monorepo 采用 GNU Affero General Public License v3.0 only
（SPDX：`AGPL-3.0-only`）。根 `LICENSE` 与 GNU 官方原文 SHA-256 一致；README、贡献规则、
OpenAPI、生成契约及 StuHelper 自有包元数据已对齐。`clients/admin/` 中源自 Vben 的代码继续
保留其 MIT 许可证与版权声明，第三方通知不因根许可证而被删除或替代。
OpenAPI License Object 使用 GNU 官方许可证 URL，而不是仅适用于 3.1 的 `identifier` 字段；
这样既保留明确的 AGPL-3.0-only 声明，也兼容当前生成后嵌入契约的运行时校验路径。

### 2026-08-01 PR #21 新增审查线程复核

| 线程 | 交叉复核 | 是否必须修改 / 是否过度设计 | 当前处置与证据 | 状态 |
|------|----------|-----------------------------|----------------|------|
| 首次启用 `authorization_grants` 会让旧 scoped operator 消失 | `confirmed`。migration 先创建空 DB 账本，而新读路径立即只信 DB；旧 Casdoor membership 与 OpenFGA direct tuple 没有迁移步骤，升级后有效操作者会被静默锁出 | 必须修改；不能靠发布说明手工补 grant。也不需要双写、2PC 或通用 IAM 迁移平台 | 新增 durable pending/completed marker 与一次性命令；仅导入“当前 Casdoor 身份/管理员状态”或“遗留 scoped membership ∩ OpenFGA direct tuple”的最小交集；未知/间接主体 fail-closed；grant/audit/outbox/marker 同事务，production-like 启动前强制门禁，重复部署幂等。提交 `5a84be64` | `implemented`、`production-pending` |
| refresh 先消费 provider refresh token，后做 Casdoor user lookup | `confirmed`。lookup 503 时 provider 可能已轮换 refresh token，本地仍保留旧 token，下一次重试可能永久失败 | 必须修改；无需重写 token service | 从已校验 session 取绑定 subject，先执行 Casdoor lookup；依赖失败不调用 token endpoint、不清 cookie。交换后再强制新 ID token subject 与 session subject 一致。故障注入断言 token endpoint 调用数为 0。提交 `d5ec46f4` | `implemented`、`production-pending` |
| Koishi admission 管理动作按内部 user ID 直接读 DB snapshot | `confirmed`。HTTP middleware 有 Casdoor 实时降权，但 service-credential / bot 调用经过 raw Authorization Service，陈旧 `super_admin` 可继续批准 admission | 必须修改；无需让所有普通 bot 请求都访问 Casdoor | admission gateway 改用 identity adapter；仅 DB snapshot 仍含 `super_admin` 的候选用户触发实时 lookup，降权先写 revoke fence 并重载，provider 故障 fail-closed。提交 `b8d417dc` | `implemented`、`production-pending` |
| grant list 空页把真实 total 返回为 0 | `confirmed`。`COUNT(*) OVER()` 只能从返回行扫描 total；`offset >= total` 时没有行，管理端得到错误总数 | 必须修改；这是 API correctness，不是无证据的全库分页优化 | 只把 Authorization grant list 改为同过滤条件的独立 COUNT + data query，越过末页返回非 nil 空数组和真实 total；集成测试覆盖 2 条数据、limit=1、offset=2。提交 `a9baffcd` | `implemented` |

为让首次授权切换能完整读取旧 direct tuple，原第二轮 `#68` 也从 P3 长尾提升并实施：OpenFGA
`ReadTuples` 现在显式设置 higher-consistency、page size，并遍历全部 continuation token；同时防御
重复 token，避免超过单页时漏导入或让 `WriteMissingTuples` 把既有 tuple 误判为缺失。该修改是
切换正确性的必要依赖，不扩展为通用批处理框架；独立提交为 `2fffa1b3`，状态为
`implemented`。

本轮共同验证证据：全量 Go race+coverage 通过，总语句覆盖率 64.3%，受保护包阈值全部满足；
最终 fail-closed 增量再次通过 Authorization race test、golangci-lint 与 gosec；全仓 gosec
扫描 417 个 Go 文件、0 issue，govulncheck 报告代码调用链 0 vulnerability；migration 按 CI
角色前置条件完成全量上行、最新 migration 回滚与重新上行；Koishi 全工作区 production build、
基础设施 contract、文档/自定义 Semgrep/OpenAPI drift 守卫和新增 cutover shellcheck 均通过。

### 2026-08-01 PR 合并与单维护者治理决策

PR #21 在提交 `84ad33f5` 上完成全部适用 CI、CodeQL 和 review conversation 后，由 `Xauryan`
使用仓库 ruleset 的 `pull_request` bypass 合并为 `main` 提交 `6fc03cce`。GitHub 拒绝作者对
自己的 PR 提交 `APPROVED` review（HTTP 422），所以该例外的准确语义是“允许唯一维护者在
保留 PR、检查、讨论和审计轨迹的前提下完成合并”，不是伪造作者自审。

ruleset 默认仍要求 1 个独立 approval、CODEOWNERS、最后推送者之外审批、线程解决、线性历史、
Required 与 Go/JavaScript CodeQL。bypass actor 只允许 GitHub 用户 `Xauryan`（user ID
`268165484`），最终模式固定为 `pull_request`；不授予组织管理员、仓库角色或其他用户，
不允许直接推送、force push 或 `exempt` 无审计绕过。为让两个长期分支保持相同历史，合并后
曾将该用户模式短暂切为 `always`，仅用 `force=false` 把 `develop` 从 `2da0d180` 非强制快进
到 `6fc03cce`，随后立即恢复为 `pull_request`；最终 `main`、`develop` 指向同一提交。

## 6. 低优先级、条件性和决策项

以下条目经过复核后不属于当前致命/高/中级队列。它们只有在产品决策、真实规模、运行指标或
明确发布范围支持时才进入实现，避免为了关闭审计数字而过度设计。

### 第一轮长尾（24 个唯一根因）

| 别名 | 最终处置 |
|------|----------|
| P2-4 | `implemented`：只修仍可达的两处全量读取：最近 moderation events 改为数据库内倒序加 limit 并增加 `createdAt` 索引，dashboard/review 的待处理成员直接按 `releasedAt/kickedAt IS NULL` 查询；同时删除未使用的 overview 全成员读取。未重写无规模证据的统计聚合 API。Store/model 专项 7/7 与两个相关 TypeScript 工程检查通过。本条所在提交。 |
| P2-5 | `implemented`：批量禁言只在第一个空白处分隔秒数，后续成员 ID 继续按空白/中英文逗号解析；格式提示改为真实命令 `群审禁言`。管理员命令专项 11/11 通过。本条所在提交。 |
| P2-8 | `implemented`：Admin 中英文补齐后端会返回的 `school_email_otp` / `school_sso` 学生认证方式，并用 locale 契约覆盖全部四种枚举。本条所在提交。 |
| P2-11 | `implemented`：OpenAPI 明确 `form + explode=true`，Handler 读取全部重复 `courseIDs` 并兼容旧逗号格式；共享客户端真实 wire test 与 Go parser 回归锁定三端一致性，删除未使用且错误扁平化 grouped response 的 Web adapter。本条所在提交。 |
| P2-17 | `implemented`：把误命名且被拿来限制游客正文的 `review_preview_title_chars` 无损迁移为 `review_guest_preview_content_chars`；游客仍只见首个非空行且默认上限保持 24 字，登录但无完整权限的用户改为真实应用正文字符数与百分比两个开关，标题保持可见。策略、用户配置和评课包测试通过。本条所在提交。 |
| P2-19 | `implemented`：拆分评课正文、回复正文与举报说明的长度 sentinel；只有评课正文使用 `A0110003/A0110004`，回复和举报仍保持通用范围错误，同时为危险内容、缺少评分维度和非法评分补上已有专用码。评课包全量测试通过。本条所在提交。 |
| P3-1 | `implemented`：`AdminContentLayout` 正式支持并显示可选页面说明，保留标题、总数和 action 布局；组件回归覆盖有/无说明。本条所在提交。 |
| P3-2 | `implemented`：持久化表格列的显式 `defaultMinWidth` 优先，否则只保留调用方透传的 string/number `min-width`，不再被 `undefined` 或非法 attribute 类型覆盖；组件回归锁定属性合并顺序，Admin 全量类型检查通过。提交 `743bcc3f` 及类型收尾提交。 |
| P3-3 | `implemented`：只有 `pending` 新生申请显示审核控件，已处理行只显示状态；提交入口再次检查当前行状态并阻断陈旧动作。专项回归覆盖 pending/approved/rejected。本条所在提交。 |
| P3-4 | `implemented`：把成员黑名单迁移计划重写为当前 PostgreSQL 权威、Admin/Bot API、Admission 联动、Koishi 故障语义和生产验收边界；移除 `blacklist.json` 仍为现状及“待移除/改为”等过时叙事。本条所在提交。 |
| P3-5 | `implemented`：前端目录/模块、后端 RBAC 边界已按当前源码校正；GitHub 时点状态迁入 `docs/internal` snapshot，现行 guide 只保留可重复验收步骤；automation/runbook 去除迁移叙事。文档卫生通过。本条所在提交。 |
| P3-6 | `implemented`：school SSO login/callback 继承全局 cookie/bearer 鉴权并补齐 Handler 可达错误响应；三个 token 型 mobile handoff 端点显式声明匿名访问。YAML 契约测试同时锁定两类相反语义。本条所在提交。 |
| P3-8 | `implemented`：取消收藏、删除教师、删除敏感词三个成功 Handler 改为真正的 204 且无响应体，与既有 OpenAPI 契约一致；集成断言锁定 status/body。本条所在提交。 |
| R-3 | `implemented`：保持后端签名前 `Stat/HEAD` 的 fail-closed 事实不变；Admin 详情窗口内任一已声明材料加载失败时立即禁用“通过”，提示链接可能过期并允许重新获取详情签名。桌面/移动 E2E 2/2 通过；原“缺对象仍可审批”仍为证伪。本条所在提交。 |
| R-4 | `implemented`：抽出共享 `isNonArrayRecord` 并替换 14 份语义相同的 JSON-object guard；保留 5 份有意接受 array 的 object-like guard，避免盲目统一改变边界。共享单测锁定 object/array/null/primitive。本条所在提交。 |
| R-5 | `implemented`：错误码参考明确八位码与六个已发布 `admission.*` 兼容码的双真源边界，禁止继续扩展 dotted code；补齐 OpenAPI 遗漏的 `member_blacklisted` 并以契约测试覆盖全部六项。本条所在提交。 |
| R-6 | `implemented`：复核所有 19 个挂载 endpoint/user/progressive limiter 的操作，不只修原报告少算的四项；每个 OpenAPI operation 现在都声明 limiter 可达的 429 与 Redis fail-closed 503，并由生成契约测试统一锁定。本条所在提交。 |
| R-7 | `implemented`：从 CameraCaptureRequest 和 Web payload 删除从未被服务端消费、且不能作为可信取证时间的客户端 `capturedAt`；服务端继续以接收/落库时间为准，E2E 明确断言 wire payload 不再携带该字段。本条所在提交。 |
| R-11 | `validated-no-change`：真实 PostgreSQL 并发测试证明固定 dedupe 行确会产生可观测 `Lock` wait，释放事务后两次写仍正确合并为一个 durable job；昂贵刷新不在锁内，因此无证据支持拆 key。新增 source-write p95 告警与“持续 SLO 违约且归因到该 key 才重构”的门槛，保留现有 supersession fence。本条所在提交。 |
| R-13 | `implemented`、`production-pending`：配置门禁要求应用 username 与 DBA provisioning 的 expected readonly username 相等；真实 smoke 从同一 Oracle 会话读取 `SESSION_USER` 和 `USER_*_PRIVS`，仅接受身份匹配、零 role、一个无 admin option 的 `CREATE SESSION`、目标表上一个无 grant/hierarchy option 的 `SELECT`、零列级授权。Evidence 只落身份哈希前缀、布尔值与授权计数。Go、配置、shell/readiness/init 合约均通过；本机没有真实 Oracle，生产 grant evidence 仍待发布验收。本条所在提交。 |
| R-14 | `implemented`：删除未启用且会与现行 Dependabot 更新策略形成双重权威的 `renovate.json`；依赖更新继续由 `.github/dependabot.yml` 单一治理。本条所在提交。 |
| R-15 | `implemented`：`IdentityPhotoUploadResult` 收敛为 Handler 实际且唯一返回的 `key`，删除永不返回的 `rejectionReason` / `createdAt` / `updatedAt` 可选字段，避免生成客户端承诺虚假能力。本条所在提交。 |
| R-16 | `implemented`：原风险在仅有 1/2/2/2/3 行内置 fixture 时不可达，但逐行 SQL 机制真实且不适合作为长期 connector 架构；现已把 terms/courses/teachers/offerings、教师关系、课表和成员关系改为事务内 `jsonb_to_recordset` set-based upsert/insert，往返次数不再随行数线性增长，同时保留引用校验、原子回滚与旧 snapshot 裁剪。1000 门课程/教师/开课/关系/课表/成员的 PostgreSQL 集成回归通过。本条所在提交。 |
| R-17 | `implemented`：四类 review list/grouped response 均在 OpenAPI 中把运行时已返回的 `page` / `pageSize` 声明为必填正整数；生成 Go/TS 契约不再把分页元数据隐藏成额外属性。本条所在提交。 |

### 第二轮长尾（17 个唯一根因）

| 别名 | 最终处置 |
|------|----------|
| #33 | `validated-no-change`：`navigationStyle=custom` 的状态栏/微信胶囊遮挡机制在 App/mp-weixin 成立，但仓库现行 UniAppX 发布契约是 H5-only，平台守卫明确禁止声明或构建 mp-weixin；H5 浏览器视口不包含原生状态栏/胶囊。删除 custom navigation 会无依据改变已验收 H5 布局，因此当前不改；将来新增 App/小程序正式目标时，必须随认证、构建、CI 和真机门禁一起重新选择原生导航或实现安全区。本条所在提交。 |
| #40 | `implemented`：SearchPage 订阅 ReviewCard 的 `moderated` 事件后，只重新读取用户已经加载的 review 页，不重跑 course 搜索、不清空结果也不滚回页首；多页结果按 ID 去重并更新真实 total，进行中的刷新 single-flight，新搜索会通过既有 AbortSignal 取消旧刷新。组件回归锁定审核后卡片更新、course 请求数与滚动位置。本条所在提交。 |
| #41 | P3：teacher profile append error 保留已有列表 |
| #58 | P3：moderation route 渐进使用现有 capability |
| #66 | P3：OIDC ES256/RS256 配置与 verifier 行为对齐 |
| #69 | P3：Native SSO 保存经校验的登录前 redirect |
| #70 | P3：通用评课列表从 OpenAPI 增加 `isOwner` |
| #71 | P3：inline edit 对齐 10–5000 字规则 |
| #72 | P3：Admission 登录按钮 single-flight |
| #73 | P3：OTP 重发 cooldown 和本地化错误 |
| #74 | P3：QQ bind polling deadline、终态和后台暂停 |
| #75 | P3：`sessionStorage` 副本失败不阻断登录 |
| #76 | P3：匿名 handoff operations 显式 `security: []` |
| #78 | P3：authorization 文档角色语义校正 |
| 反向 `/admin/stats` | P3 决策：保留 freshness carve-out 时是否只补基础 MFA enrollment/proof |
| 反向 Developer Connect | P3：issuer fallback 不使用 Web origin |
| U-3 | P3：StudentVerificationPanel 学籍邮箱流程国际化 |

## 7. 已证伪或明确不实施

- X-1“无 scope 的 school admin 自动获得全局评课权限”被 admin Entry capability gate 截断，
  不做授权模型紧急重写。
- 固定 admission hostname、关闭提醒渠道仍视为成功、内部 SMS route 不挂公网 limiter 等行为由
  当前部署/产品契约锁定，不按漏洞修改。
- 不把 OpenFGA、PostgreSQL 和 Casdoor 再抽象成第二套通用 IAM 平台。
- 不为局部保存、删除或重试引入 2PC、Saga、全局事件总线或通用调度框架。
- 不在单副本、无规模证据时引入 Redis pub/sub、通用 retention UI 或批量导入平台。
- 不把 H5 产物冒充微信小程序，也不在没有产品范围、compiler、认证和真机链路时建设微信发布。

## 8. 发布与生产验证边界

当前 `main` 上的修复尚不能统一宣称已发布到生产。下列证据必须按实际发布范围逐项补齐：

- 真实 Casdoor organization admin 晋升、降权、lookup 故障关闭和 MFA step-up；
- 首次授权账本切换的 digest/count/marker/audit/outbox evidence，以及旧 scoped operator 保留验证；
- refresh lookup 503 时 provider token endpoint 未被调用，以及 Koishi 已降权管理员被拒绝；
- OpenFGA projection applied、撤权 fence 生效时点、漂移 reconcile 和全量 rebuild；
- PostgreSQL 生产备份取回与 PITR；
- 受保护 GitHub environment 的发布/回滚；
- 真实 Redis、Oracle、对象存储、Koishi Console 热重载和 QQ admission 端到端；
- Oracle smoke 的 `runtimeIdentityMatched=true`、`leastPrivilegeGrantsVerified=true` 及严格授权计数；
- 前端生产构建在桌面/移动端的受控角色验收。

本地测试、健康检查、CI 绿灯和开发数据库样本都不能替代上述证据。

## 9. 后续执行协议

每个仍需实施的根因遵循同一闭环：

1. 先从当前代码、OpenAPI、migration 和测试重新确认根因仍存在；
2. 只修该根因的最小完整边界，不采用原报告中的宽泛重构方案；
3. 先改真源，再更新唯一当前态文档和本台账；
4. 运行定向测试、相关包/应用全量测试、生成漂移和文档守卫；
5. 复核负向路径和相邻调用方；
6. 一个唯一根因形成一个提交，不混入无关工作树文件；
7. 只有提交和验证证据齐全后，才把状态改为 `implemented`。

截至当前提交，已再次从源码、OpenAPI、migration、回归测试和本台账反向核对：第 5 节 4 个
PR 线程及其 OpenFGA 分页依赖均已实施，没有发现其他未实施的 P0/P1/P2 根因。第 6 节其余
条目均为已降级的 P3/P4、条件性、先测或产品决策项，不应为了关闭审计数字自动进入实现；
第 8 节生产门禁必须在真实环境单独留证。
