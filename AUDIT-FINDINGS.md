# StuHelper 审计与修复台账

> 状态日期：2026-08-01
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

复核后没有 P0。已确认的 P1/P2 均已形成独立实施提交并有与风险相称的本地回归证据；当前
致命、高、中级**本地代码队列为空**。真实 Casdoor/OpenFGA 账号闭环、发布和故障演练仍是
`production-pending`，统一列在第 8 节，不能把“本地队列为空”误写成“已经上线”。

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

## 6. 低优先级、条件性和决策项

以下条目经过复核后不属于当前致命/高/中级队列。它们只有在产品决策、真实规模、运行指标或
明确发布范围支持时才进入实现，避免为了关闭审计数字而过度设计。

### 第一轮长尾（24 个唯一根因）

| 别名 | 最终处置 |
|------|----------|
| P2-4 | P3 先测：只优化仍可达的 dashboard 全扫，不重写聚合 API |
| P2-5 | P3 correctness：修空格 ID parsing 和真实命令名 |
| P2-8 | P4 可选：后台 i18n 标签 |
| P2-11 | P3：query array 契约与默认客户端对齐 |
| P2-17 | P3/P4 决策：preview knobs 是废弃还是受限恢复 |
| P2-19 | P3：先拆共享 sentinel，再补精确 review code |
| P3-1 | 可选：AdminContentLayout 无效 description 调用 |
| P3-2 | P3：min-width attrs/default 合并顺序 |
| P3-3 | P3：已审核行的 stale action 防护 |
| P3-4 | 可选：成员黑名单文档时态 |
| P3-5 | 可选：开发指南目录和入口校正 |
| P3-6 | P3：school SSO/mobile handoff OpenAPI security 与错误契约 |
| P3-8 | P3：三个 Handler 真正返回 204 No Content |
| R-3 | P3 UX：签名图片 URL 过期后的刷新/禁用 |
| R-4 | 可选：区分 array 的 payload guard 去重 |
| R-5 | P3：error-code reference exception/真源一致性 |
| R-6 | P3：四个限流端点补 429/503 契约 |
| R-7 | P4 hygiene：删除无业务语义的 `capturedAt`，不持久化客户端时间 |
| R-11 | P3 先测：outbox 固定 dedupe row lock-wait |
| R-13 | P3 hardening：Oracle expected identity 与 grant evidence |
| R-14 | P3 cleanup：删除未启用的 Renovate 配置 |
| R-15 | P3 hygiene：删除永不返回的 upload result 字段 |
| R-16 | 条件性：真实 academics connector 上线前再做 batching |
| R-17 | P3：review list page/pageSize 契约 |

### 第二轮长尾（18 个唯一根因）

| 别名 | 最终处置 |
|------|----------|
| #33 | P3：UniAppX custom navigation 安全区与真机验收 |
| #40 | P3：SearchPage moderation 后局部同步 |
| #41 | P3：teacher profile append error 保留已有列表 |
| #58 | P3：moderation route 渐进使用现有 capability |
| #66 | P3：OIDC ES256/RS256 配置与 verifier 行为对齐 |
| #68 | P3：OpenFGA `ReadTuples` continuation token 分页 |
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

当前审计分支上的修复尚不能统一宣称已发布。下列证据必须按实际发布范围逐项补齐：

- 真实 Casdoor organization admin 晋升、降权、lookup 故障关闭和 MFA step-up；
- OpenFGA projection applied、撤权 fence 生效时点、漂移 reconcile 和全量 rebuild；
- PostgreSQL 生产备份取回与 PITR；
- 受保护 GitHub environment 的发布/回滚；
- 真实 Redis、Oracle、对象存储、Koishi Console 热重载和 QQ admission 端到端；
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

截至本实施提交，已再次从源码、OpenAPI、migration、回归测试和本台账反向核对：不存在其他
未提交的 P0/P1/P2 根因。第 6 节条目均为已降级的 P3/P4、条件性、先测或产品决策项，不应为了
关闭审计数字自动进入实现；第 8 节生产门禁必须在真实环境单独留证。
