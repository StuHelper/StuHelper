# 全库审计发现汇总

本文件是本轮审计的工作产物,用于跟踪修复。按 [文档架构规范](docs/design/documentation-architecture.md),
审计报告不属于 `docs/`——这些条目最终应转成 GitHub Issues 与 PR 描述,不作为项目文档提交。

## 方法

12 个审计 agent 按维度并行扫描(后端授权 / 数据层 / 并发 / 业务模块 / 外部数据源 / 契约链路 /
Web / Admin / UniAppX+Koishi UI / Koishi / 基础设施 / 代码质量与文档)。

每条发现再交由独立的对抗验证 agent 复核,验证方提示词要求**默认证伪**,只有在代码明确证明缺陷存在时
才判定成立。P0/P1 用两个不同视角复核(能否复现 / 诊断与修复方案是否正确),P2/P3 用一个。

- 原始发现：61
- **Claude 原审计判定成立：43**
- Claude 原审计判定被证伪：18

> 上述数字仅表示 Claude 原审计的分类，不是本文件的最终处置结论。Codex 的独立复核发现：
> 原“确认成立”中存在重复计数、级别偏高和条件性风险；原“被证伪”中也有应重新进入待办的真实问题。
> 最终处置以紧随其后的“Codex 独立复核”章节为准。

## Codex 独立复核（2026-07-30）

### 复核结论

本次由主代理与 3 个并行只读复核代理，分别覆盖 P0/P1/X、全部 P2、全部 P3 与 18 条
“被证伪”记录；主代理另外复现了关键命令、检查了实际路由/授权链和当前工作树修复。
结论不是简单接受或否定原报告：

- 原报告 43 个“确认”标签中，39 个事实核心确认、4 个仅部分确认；P2-14 与 P2-15
  是同一根因的重复描述，因此对应 42 个唯一问题，而不是 43 个唯一问题。
- 原报告 18 个“被证伪”标签中，只有 6 个应维持驳回；4 个应重新进入待办，另 8 个属于
  低优先级、条件性或延期项，不能笼统写成“不应修改”。
- 主对话追加的 X-1 攻击链被证伪；X-2 的配置面缺口部分成立，但原来的 187/21 计数和
  “全部写入模板并做集合相等检查”的方案都不准确。
- 按 63 个标签（43 + 18 + X-1 + X-2）统计，当前结论是：43 个确认、13 个部分确认、
  7 个证伪。这个数字只适合描述标签，不能替代下面按唯一根因合并后的实施计划。
- 复核后不再保留 P0。P0-1 是真实核心流程缺陷，但影响是特定角色登录后落到 404、
  且可手输授权页面绕过，调整为 P1 更合适。

### 证据边界与文档完整性

- **已验证的是当前源码、契约、迁移、测试和本地容器行为。**本轮已在无网络隔离容器中
  实际启动恢复后的 PostgreSQL 18.4 base backup 并读回探针数据，但没有登录生产主机、
  检查生产备份目录或用生产 WAL 执行目标时间点恢复，也没有观察生产锁等待或真实大表延迟。
  因此“生产已经丢失 PITR”“生产已经 OOM/死锁/永久停更”等说法仍不得当作已证实事实。
- 当前工作树在复核开始前已经有未提交修改。Codex 已完成的修复均按问题或唯一根因独立提交，
  完整范围与验证证据见下方两张修复进度表；P2-15 等重复标签没有制造第二个提交。所有这些
  提交都尚未合并或发布，原有其他工作树修改没有混入修复提交。
- 原报告大量“修复方案”和全部 18 条驳回长文在句中截断。例如 P0-1 结尾是
  `only surface the 4`，P1-1 结尾是 `Promise.resolve<string[]>(`，P1-3 结尾是
  `both loc`；P3-1 至 P3-9 的方案也全部半句结束。第 1 条驳回理由还重复了两次。
  这些原文只能作为取证草稿，不能直接当作可执行 runbook。
- 下表的“最小处置”是本轮复核后的权威建议。原始长文保留用于追溯，但若与本节冲突，
  以本节为准。

### 处置标签

| 标签 | 含义 |
|------|------|
| 必须 | 已确认的授权、数据完整性、迁移、恢复能力或核心正确性问题，应进入近期修复 |
| 应改 | 缺陷真实且修复收益明确，但不应阻断所有其他发布 |
| 先测 | 机制真实，生产影响取决于规模、并发或部署形态；先补指标/压测再决定结构改造 |
| 决策 | 技术事实真实，但需要产品、安全或数据语义先定案 |
| 可选 | 低风险 UX、文档、清理或维护性改进 |
| 不改 | 攻击链/失败场景不成立，或当前行为是明确且有测试的产品设计 |

### P0、P1 与主对话追加项

| 编号 | Codex 结论 | 调整级别 | 必要性 | 最小处置与过度设计判断 |
|------|------------|----------|--------|------------------------|
| P0-1 | 确认，已完成本地修复与回归验证 | P0 → P1 | 已修复，待发布 | 守卫复用 capability 过滤后的 `accessibleRoutes`/`accessibleMenus` 推导首页，并只接受确实命中可访问路由的内部 redirect；未知、已过滤、核心登录、外部、scheme-relative 和坏编码路径均回退首页。没有放宽 dashboard 权限，也没有建立第二套 capability 判断。 |
| P1-1 | 确认，已完成本地修复与回归验证 | P1 | 已修复，待发布 | 缺 Session 时 fail-closed；私聊无群号仍按 command policy 的 `minAuthority` 校验，但不加载成员角色，随后在访问数据前返回群上下文提示；私聊显式群号继续按目标群角色策略校验，复核列表始终带 guild filter。没有禁用合法私聊，也没有重写命令权限框架。 |
| P1-2 | 确认，已完成本地修复与真实迁移验证 | P1 | 已修复，待发布 | 已将权威来源纠正为完整有序 migration 集合，明确 `000001` 是不可变初始基线、后续变更必须新增递增 `.up/.down` 文件，并同步 Make 帮助、数据库参考与 CI 步骤名。未普遍加入 `IF NOT EXISTS`，避免掩盖脏迁移和 schema drift；隔离 PostgreSQL 18 上按 CI 非超级用户权限实际完成 19 版 `up → down 1 → up`，最终 `dirty=false`。 |
| P1-3 | 确认，已完成本地修复与真实隔离恢复；生产状态未验证 | P1 | 已修复，待发布；生产 PITR 待演练 | 已改为 staging 内 `plain + wal-method=stream`，使用临时 replication slot 和 `pg_verifybackup`，压缩后经 `.partial` 原子发布；同步器排除临时工件，evidence 同时验证本地/取回的逻辑与物理备份、SHA256、可读性和新鲜度。固定 PostgreSQL 18.4 客户端真实生成并校验备份，无网络恢复实例成功读回探针，临时 slot/容器/卷/网络均已清理。没有引入持久 slot、备份平台或通用编排层；仍不能据此宣称生产 WAL 连续性/PITR 已验收。 |
| P1-4 | 确认，已完成本地修复与真实干净 worktree 打包验证 | P1 | 已修复，待发布 | 部署包改为只取 Git `HEAD` 跟踪文件，脏工作树在创建输出前拒绝；打包后断言根目录恰好包含 `.env.example`、`.env.prod.example`，不存在其他根 env 文件。临时干净仓库实测忽略的 secret/`node_modules` 不入包、两个模板存在、未跟踪文件会阻断。没有维护第二份易漂移 exclude 清单或引入新发布系统；仍需 CI 和真实远端部署/回滚验收。 |
| P1-5 | 部分确认，已完成窄范围修复；真实远端回滚未验证 | P1/P2 边界 | 已修复，待发布与远端演练 | 普通生产部署继续按当天 fail-closed。仅同环境成功发布记录、目标 tag、三个应用 digest、当时有效的完整 policy、当前生产基础镜像、操作人和理由全部匹配时，才复用原部署日审核窗口并写 0600 JSONL 审计。GitHub 回滚用当前 workflow SHA 的最小控制器覆盖旧 release 中的旧 validator；每日门禁提前 3 天告警。没有全局 report-only，也不允许未部署版本或镜像漂移绕过。 |
| P1-6 | 确认，已完成本地修复与真实 HTTP/数据库回归 | P1 | 已修复，待发布 | Admission handler 复用 runtime 现有 `bgCtx.Done()`，两个 SSE 在停机时发送 `end/shutdown` 并退出；ticker/keepalive 执行前二次检查，避免停机后再 claim。真实 HTTP + PostgreSQL 验证 bot 流可让 `http.Server.Shutdown` 在 2 秒内返回，camera 流同样主动结束。没有设置全局 `BaseContext` 或新增 10 分钟强制重连。 |
| P1-7 | 确认，已完成本地修复与真实路由/数据库回归 | P1 → P2 | 已修复，待发布 | OpenAPI 改为匿名、cookie、bearer 三种可选认证并声明 503，生成 bundle、Go 内嵌契约和 TS 类型；真实 GET 路由接入 optional auth 与健康门禁。PostgreSQL route-level 测试覆盖 owner、其他登录用户、匿名和认证后端故障。没有把公开列表改成强制登录、增加 ownership SQL 或让前端自行推断身份。 |
| P1-8 | 确认，已完成本地修复与真实 PostgreSQL 回归验证 | P1 | 已修复，待发布 | `ProcessReport` 对非删除态复用现有管理员转换白名单；作者已删除的 review 保持终态，只结案历史 report。缺失 review 统一映射 404。没有改 schema、增加新状态或在 Repository 注入静默 no-op。真实数据库覆盖 `hide`/`delete`、重复非法转换、计数和时间戳不变；评课包全量与定向 race 测试通过。 |
| P1-9 | 核心确认，已按既有安全模型完成本地修复 | P1/P2 边界 | 已修复，待真实 Casdoor 验收 | Open Platform 只传内部 user ID；app gateway 用 user repository 解析 Casdoor subject，再实时调用既有 `GetPhone`。本地只读取 `phone_enc IS NOT NULL` 作为验证状态，不再读取/解密掩码字节。真实 PostgreSQL/Redis 覆盖 phone API、identity-token、授权审计和 provider fail-closed；app adapter 覆盖身份解析失败不调用外部客户端。没有落库完整手机号、缓存 Casdoor 返回或让业务模块持有外部 subject。 |
| P1-10 | 机制确认，已完成本地修复与真实单连接池回归；生产影响未量化 | P1 → P2 | 已修复，待发布 | 先 drain 并显式关闭主结果，再以 2 条批量 SQL 取 tags/bindings，查询数由 `1 + 2N` 固定为 3。真实 PostgreSQL 在 `MaxConns=1` 下覆盖 1/2 条数据的固定查询数、详情查询与 6 个并发列表请求；无需 ORM/DataLoader。 |
| X-1 | 证伪 | P1 → 不成立 | 不改 | 无 school scope 的 `school_admin` 在 `ExpandRoleGrants` 已得到零 capability，admin Entry 会在 Handler 前返回 403；定向测试也覆盖该语义。Repository 的 nil/空切片确有可读性风险，但原攻击链不可达。最多补 route-level 防回归/显式空结果，按 P3 加固，不做紧急授权模型重写。 |
| X-2 | 部分确认，已完成分类修复与持续门禁 | P2 | 已修复，待 CI/发布验收 | 实施前用 Go AST 重新枚举得到 config 包 **184** 个运行时键，而不是先前复核误记的 181；17 个模板差集本身正确。13 个真正面向操作员的 DB/Redis mTLS、Casdoor scopes/introspection/admin-certificate 与邮件路由键已进入开发、生产模板；`LOG_SERVICE_VERSION` 是死配置，运行时始终使用 build version，现已删除。`AWS_CA_BUNDLE`、`LOG_SERVICE_NAME`、`LOG_ENVIRONMENT` 是已有主键的兼容覆盖项，保留但不写入模板，避免空值压掉 fallback。AST 回归只要求 config 运行时键进入至少一个模板或带理由的窄 allowlist，并拒绝动态 key、缺项和陈旧 allowlist；env 模板改动也会触发 Backend job。没有把 bootstrap/FGA 工具、`GIN_MODE`、Redis integration 开关暴露到运行模板，也没有要求模板与全仓 `getenv` 集合相等。 |

### P2 逐项复核

| 编号 | Codex 结论 | 调整级别 | 必要性 | 最小处置与过度设计判断 |
|------|------------|----------|--------|------------------------|
| P2-1 | 确认，已完成本地修复与静态契约验证 | P2 | 已修复，待 CI 验收 | 新增 `guards` 分类；文档、Vue UI、Semgrep 与 Node pin 分别触发实际消费它们的 job，Node 两个版本文件还必须一致。没有让所有脚本改动触发所有重型 job。原文“零 CI”已纠正为“零个相关门禁”，secret scan 原本仍会运行。 |
| P2-2 | 确认，已完成本地修复与全量 infra 回归 | P2 | 已修复，待 CI 验收 | 两个毫秒级静态合约改为 always-on 并纳入 Required；需要构建产物的 package contract 放进既有 Koishi job。Dockerfile/CI wiring 和机器人可部署包都不再依赖 infra filter；没有让所有机器人改动触发整套 infra/E2E。 |
| P2-3 | 热路径确认并已完成最小修复；容量影响未量化 | 热路径 P2；retention 决策 P3 | 热路径已修复，待发布；retention 待业务期限和容量证据 | repeat 检测已把排序/limit 下推 DB，并增加 `(guildId, createdAt)` 索引。开发库该表为 0 行，原 300k 行/OOM 是假设而非实测。账本还供迟到撤回按 messageId 取证；没有最长撤回/审计期限时自动删除会改变业务语义，因此不新增 retention WebUI/配置/清理任务，先观测规模与延迟。 |
| P2-4 | 部分确认；原报告引用了一条 dead 路径 | P2 → P3 | 先测 | 当前 dashboard 仍有 recent events 与 active guards 的全扫，应下推过滤/limit 后测延迟。原文所称 admission console 全量扫描不成立，五步聚合与全面 API 重写过重。 |
| P2-5 | 确认 | P2 → P3 | 应改 | 修复空格分隔 ID 的 destructuring，并同步真实命令名和回归测试。低频、低成本 correctness，不需要新 parser 框架。 |
| P2-6 | 确认，已完成本地修复与回归验证；真实 Console 热重载待验收 | P2 | 已修复，待发布与运行时热重载验收 | Group Guard 在 required console 子作用域注册；Group Guard 与 Core 的特权 listener 都交给 `ctx.effect` 管理，并只在 registry 仍指向本次 registration 时删除。authority 固定为 4，调用方不能降级。只做 injection 不能消除陈旧回调，无条件按 event 删除又会误删新作用域注册；无需全局 monkey patch 或修改上游 Console。 |
| P2-7 | 条件性确认，已完成 Koishi 交付阶段修复与全量回归 | P2 | 已修复，待发布与功能启用验收 | 功能开启时，delivery 单项失败按 application 隔离、记录 `phase=delivery` 后继续，健康项正常 ACK；批末仍重抛以保留 scheduler error。ACK 失败记录 `phase=ack` 后 fail-fast，避免控制面失联时扩大重复发送。立即增加 attempts/dead-letter/schema、并行转发或 claim/lease 都不是解除单 poison 队首阻塞的必要条件。 |
| P2-8 | 确认 | P2 → P4 | 可选 | 中英文增加 `school_email_otp`、`school_sso` 标签并可选保留 enum 原值 fallback。只是后台 i18n，不应占用 P2 修复预算。 |
| P2-9 | 核心确认、原故障机理部分错误；已完成本地修复与真实 controller 回归 | P2 | 已修复，待 CI/远端发布验收 | Ansible Core 2.20.2 的 localhost connection 会把 cwd 设为 playbook basedir，因此旧脚本路径本身能找到；真正失败是相对输出参数进入脚本后，内部 `git -C` 再改变 `archive --output` 的解析基准。干净隔离仓库以旧任务真实复现 `could not open '../../generated/deploy/...'`。现改为 `playbook_dir` 绝对脚本 `argv`、脚本默认仓库绝对输出和同一绝对 upload src；固定 controller 版本、内置 default callback 的 YAML result format、三 playbook syntax check、窄路径契约和独立 bundle tag smoke 已接入 CI。没有写通用 command/shell scanner或重构远端部署；真实 SSH 上传/部署仍待验收。 |
| P2-10 | 确认，已完成最小修复、真实 PostgreSQL 语义验证与全量 infra 回归 | P2 | 已修复，待发布 | 普通 importer 已拒绝 `sfzjh_enc`/`sfzjh_hash`，并从 normalize、临时表、copy、insert 和 conflict update 全链路移除两列：新行得到 `NULL/NULL`，重导保留既有 pair。当前仓库没有完整 pair 导入入口；在没有真实需求和密钥治理设计前不新增 CLI/API，也不能伪造空 `enc` 绕过约束。 |
| P2-11 | 确认潜在契约缺陷 | P2 → P3 | 应改 | 统一为 `explode: true`，Handler 用 `QueryArray` 并兼容旧逗号格式，重新生成；修正或删除未使用的 Web grouped adapter。OpenAPI 与 Handler 当前都按逗号语义，真正不兼容的是默认客户端，原文“三端各不兼容”不准确。 |
| P2-12 | 确认，已完成共享用户预算与真实 PostgreSQL/Redis 回归 | P2 | 已修复，待发布 | academic-match/request-otp 在 auth 后共用同一 Redis per-user 预算（合计 5/min），第 6 次 429 且不调用 Oracle；另一用户独立。Redis outage 在 handler 前 fail-closed 503。verify-otp 不访问 Oracle，未纳入预算。OpenAPI 为 academic-match 补真实 429/503 并完整生成。 |
| P2-13 | 确认，已完成主因最小修复与真实 PostgreSQL 状态机回归 | P2 | 主路径已修复，待发布；补偿失败与 dead-letter replay 为显式残余边界 | claim 后每批加载一次上下文；批量查询失败或 caller cancel 时，用独立 5 秒 context 一次性归还所有未公开 lease；缺 policy 等确定性单行 preparation failure 只消耗本行 attempt，stale/failure/abandon 均按 claimed attempt fenced，健康行继续返回。补偿写也失败时仍会保留 attempt；Admission 尚无 dead-letter replay API，不能宣称所有故障下都不会耗尽预算或 terminal 行已可运营恢复。 |
| P2-14 | 确认，属于 P2-13 的性能放大器；已合并修复 | 单独看 P3 | 已修复，待发布 | policy/failure contexts 已从逐行 `2N` 查询改为每批固定 2 条；真实 PostgreSQL 对 1 行与 8 行均测得 3 次 pool acquisition（1 次 claim 事务 + 2 次查询）。默认批是 50，修复前约 100 次额外查询，不是原文按 server 上限 200 推算的常态 400 次。 |
| P2-15 | 事实重复，已随 P2-14 覆盖 | 与 P2-14 重复 | 不单独立项/提交 | 与 P2-14 是同两个逐行 context query，不是第二个根因；保留原始记录用于追溯，不计入唯一问题数，也不建设第二套实现。 |
| P2-16 | 确认，已完成 nullable 语义修复与真实 PostgreSQL 回归 | P2 | 已修复，待发布 | migration 明确保留 `department_id`/`code`/`credits` 可空，客户端已有未分类/未提供语义，因此 NULL 是合法未知值；Go model、OpenAPI、生成契约和客户端按 nullable 收敛，学分排序 NULLS LAST。隔离 PostgreSQL 覆盖 detail/list/search/grouped/favorites，未用 `COALESCE(0/'')` 伪造事实，也未增加无依据的回填 migration。 |
| P2-17 | 确认两个配置当前未生效，但行为可能是刻意安全收紧 | P2 → P3/P4 | 决策 | 优先决定是否废弃 content preview knobs，并把 title knob 说明改成锁定首行 teaser；若恢复，只对已认证非 full tier 接线并保持 guest 收紧。直接“恢复配置生效”可能削弱访问控制。 |
| P2-18 | 确认，已完成单实例最小修复与真实 PostgreSQL 回归 | P2 | 已修复，待发布 | create/update/delete 只有在 Repository 成功后才调用进程内 `Filter.Invalidate()`；下一次内容检查重载 DB，失败继续映射 `ErrModerationUnavailable`，不会用旧词表放行。`Invalidate` 与 `Refresh` 共享一把窄 mutex，避免 mutation 与在途刷新交错后旧结果重新获得 5 分钟 freshness；matcher 读锁不覆盖 DB 查询。真实数据库覆盖已预热快照上的新增、改词/改级别、删除、失败 mutation 不失效和 reload failure fail-closed。当前 Compose 是单 app，不引入 Redis version/pub/sub；多副本前置要求已写入安全文档。 |
| P2-19 | 确认 | P2 → P3 | 应改 | 先为语义唯一的 dangerous/too-short/rating-required/invalid-rating 映射专用 code；共享的 content-too-long 需先拆 sentinel。不要把所有共享错误一刀切成 review code。 |
| P2-20 | 确认结构风险，已完成独立预算修复；当前样本未发生超时 | 条件性 P2 | 已修复，待发布与生产规模验收 | 普通查询 5 秒预算确实曾截断全量刷新，但开发库 15 名教师/13 条评课下两组共 6 次刷新仅 4.040–20.517 ms，outbox 为 `completed`、0 次失败，故“已经永久陈旧”没有证据。已增加继承 parent cancel 的专用 60 秒预算并限制在 5–90 秒，严格低于 2 分钟 lease；复用既有 DB duration、outbox retry/terminal 指标和告警。不抬高全局 DB timeout，不新增重复投影平台或无依据 cadence/age SLO。 |
| P2-21 | 确认，已完成最小修复与断路器状态回归 | P2 | 已修复，待发布 | `Probe`/`LookupStudent` 的错误统一进入局部 classifier：parent caller cancel/deadline 为 neutral，目录自身 query timeout 与真实 backend error 仍为 failure；neutral 必须调用 `RecordNeutral()`，因此 half-open probe 会被释放。没有改变全局 breaker 阈值、窗口或超时。 |
| P2-22 | 核心确认，已完成最小修复并保留 503 契约 | P2 | 已修复，待发布 | 无效/冲突单行保留 typed error 并记 neutral，不增加也不重置既有健康计数；学号与绑定参数错位拆成独立 sentinel，仍是 source failure。新增固定三类、无原始数据 label 的 integrity counter；app adapter 继续把所有源错误映射到既有 503。原建议用 `RecordSuccess()` 会错误清空此前真实失败，未采用。 |
| P2-23 | 核心确认、原影响范围夸大；已完成最小修复与真实 PostgreSQL 回归 | P2 | 已修复，待发布 | 共享 outbox 的 `process` panic 在 per-job 边界转成带 stack 的普通失败，复用现有 retry/dead-letter/terminal metric；同批后续 job 继续。真实生产调用面是 5 个共享 outbox worker，不包括两个手写 Admission expiry loop。没有加入无界 root supervisor；它会对无 durable attempt 的 poison 形成 panic loop。 |

P2 唯一根因应按以下修复簇合并，避免重复设计：

1. P2-1 + P2-2：CI 路径与静态契约触发，已按编号拆成两个独立提交完成。
2. P2-13 + P2-14 + P2-15：claim 后批量上下文、逐项隔离和 lease 安全释放，已合并完成本地修复。
3. P2-21 + P2-22：Oracle outcome 分类与 circuit-breaker neutral 语义，已合并完成本地修复。

### P3 逐项复核

| 编号 | Codex 结论 | 调整级别 | 必要性 | 最小处置与过度设计判断 |
|------|------------|----------|--------|------------------------|
| P3-1 | 确认 | P3 | 可选 | `AdminContentLayout` 增加可选 description/subtitle 渲染，或删掉无效调用；逐页判断重复文案。rich slot 不是必要修复。 |
| P3-2 | 确认 | P3 | 应改 | 修正 attrs/default min-width 合并顺序或统一 prop，并加 caller 提供 `min-width` 的测试。 |
| P3-3 | 确认 | P3 | 应改 | 已审核行隐藏/禁用通过与驳回按钮，点击前再检查 stale state；不需要全局改 409 语义。 |
| P3-4 | 部分确认 | P3 | 可选 | `member-blacklist-unification.md` 仍需把已落地内容改成当前时态；IAM 旧文在当前脏工作树已 rename/rewrite 为 `iam-architecture.md`，不要重复修改或覆盖现有改动。 |
| P3-5 | 部分确认 | P3 | 可选 | 只修 `frontend-development.md` 的缺失/不存在目录，以及 `backend-development.md` 过窄的 RBAC 入口。带明确日期的 GitHub 迁移快照和当前仍准确的 automation 叙事不应批量重写。 |
| P3-6 | 确认 | P3 | 应改 | 以 OpenAPI 为真源修 school SSO login/callback 和 mobile handoff 的 security/401/403，generate 后加路由契约测试；不要机械枚举未实现状态。 |
| P3-7 | 确认，已完成单事件审计与真实 PostgreSQL 取消回归 | P3 → P2 | 已修复，待发布 | 每次敏感批量导出恰好写一条 `data.export` success/failure 事件，记录规范化 format/status、已处理行数和 10,000 行上限；stream 返回 `(count, error)`，完成标记/写响应失败也归为 failure。取消请求下真实 DB stream 失败后审计仍用 WithoutCancel 上下文持久化。CSV 不含 moderation_reason，未重复调用 `h.logAdminOp`。 |
| P3-8 | 确认 | P3 | 应改 | 三个 Handler 改为真正的 204 No Content，保持 OpenAPI 真源；历史提交也表明 204 是有意契约，不应反向把 spec 改成 200。 |
| P3-9 | 确认，已完成最小修复与真实 Redis 故障回归 | P3 → P2 | 已修复，待发布 | 只有 Redis `Nil` 映射 `v0`；transport error/caller cancel 返回 unknown，版本化 get 为 miss、set 为 no-op且不本地 memoize。关闭并重启 miniredis 验证故障期不生成 `v0`、恢复后立即读取真实版本。没有引入 detached loader 或新 wrapper；singleflight 等待者在首 caller 取消时共同 bypass 属于可用性边界，不再是陈旧数据风险。 |

### 对原“被证伪”18 项的再复核

下表按原文出现顺序编号 R-1 至 R-18。原章节仍保留 Claude 的长篇驳回过程，但不再具有
“全部不应修改”的含义。

| 编号 | 原发现简称 | Codex 结论 | 必要性与建议 |
|------|------------|------------|----------------|
| R-1 | 两个 guild-member-request listener | 证伪成立 | 生产只存在 group-guard listener；core 是不可达 dead chain。可清 dead code，但不按并发授权故障修。 |
| R-2 | operation-logs 未绑定 page-size | 证伪成立 | 绑定和 size-change 都存在，原取证漏引关键行。不修改。 |
| R-3 | identity photo 只看 URL 即可批准 | 部分成立、P1 安全结论证伪 | 生产链先 Stat/HeadObject，批准时再次取证；仅页面打开后签名 URL 过期属于 P3 UX，可 onerror 禁用并刷新。 |
| R-4 | 19 份 payload guard 可纯搬迁 | 部分成立、无紧急缺陷 | 重复真实，但 `isRecord` 对 array 的语义分两类，不能盲替换。未来可抽 `isNonArrayRecord`，不是当前修复门槛。 |
| R-5 | error-code 文档 49/6 漂移 | 部分成立、P3 | 未引用常量可作为保留 vocabulary，不能批删；但文档声称唯一真源却漏掉 admission 的 6 个 dotted code，需记录 exception 或统一真源。 |
| R-6 | 4 个限流端点未声明 429 | **确认，应移回待办（P3）** | 真实 middleware 可返回 429/503，OpenAPI 必须声明并 generate；客户端容错不代表契约正确。 |
| R-7 | capturedAt 被静默丢弃 | 部分成立、P4 hygiene | 字段确实由客户端发送且服务端丢弃，但当前无业务语义。优先从 spec+client 删除，服务端时间才可信；不新增持久化。 |
| R-8 | Redis outage 被误报 cooldown | **确认，已完成最小修复与 transport 回归** | P2/P3 → 已修复，待发布。两处先判 `err`：只有 `redis.Nil` 或 nil-error 的非 `OK` 是 cooldown/429，其他 Redis error 保留底层 cause 并包装现有 unavailable sentinel/503。Admission request-otp 的 OpenAPI 503 与生成物已同步；未改 cooldown 时长、key 或限流预算。 |
| R-9 | 两段 rating stats SQL 重复 | 证伪成立 | 90 行重复真实，但不是运行时缺陷；动态表/列 helper 会增加 SQL 风险。共同修改时再抽取，当前只需等价测试。 |
| R-10 | QueryTimeout 让 connect timeout 不可达 | 证伪成立 | 外层是端到端 lookup SLA，内层 connect timeout 仍是更窄上限且都可配置。移除 query context 会回归。 |
| R-11 | 固定 outbox dedupe row 串行化写入 | **确认，应移回待办但先测（P3）** | 固定唯一键确会持锁到事务结束，且拿锁后仍有同步统计工作。先做并发测试/lock-wait 指标；证实影响后按事务或分片事件合并刷新，不能直接移除可靠投影。 |
| R-12 | 两提醒渠道关闭仍 success | 证伪成立 | 这是文档和测试明确锁定的产品语义，不修改。 |
| R-13 | Oracle runtime 账号未绑定只读账号 | 部分成立、P3 hardening | SYS/owner 已被拒绝，但其他高权账号仍可能误配。用户名相等不能证明只读；若加固，应把 expected identity 与 grant evidence 一起验。 |
| R-14 | Renovate 与 Dependabot 双机器人 | 部分成立、P3 cleanup | 当前 Renovate App 未安装则无竞争，但配置与“Dependabot 唯一权威”冲突，未来安装会激活。删除陈旧 `renovate.json` 是低成本清理。 |
| R-15 | upload result 三字段从不返回 | 部分成立、P3 hygiene | 字段可选所以不违约，但前端为永不返回字段承担复杂度。可从 spec 删除并 generate，不是运行时故障。 |
| R-16 | academics import 逐行 SQL + 15s tx | 部分成立、延期门槛 | 机制真实，但当前只有极小 fixture connector。COPY/batching 现在是过度优化；把 batching/专用 timeout 列为真实连接器上线前验收门槛。 |
| R-17 | review list 的 page/pageSize 未声明 | **确认，应移回待办（P3）** | runtime 返回字段而 OpenAPI/生成 TS 类型缺失。`additionalProperties` 允许额外字段不等于契约完整；补 schema 并 generate，或有意删除 runtime 字段。 |
| R-18 | 七个 exported helper 零调用 | 证伪成立 | 零调用事实真实但无运行时缺陷，部分是对称/兼容 API，审计 guardrail 也已约束 context 用法。最多登记清理债务并纠正原标题“six”。 |

### 建议实施顺序

#### 第一批：安全、恢复能力与不可逆正确性

1. P1-1 群命令 fail-open 已完成实现和本地验证。
2. P1-3 物理备份命令、原子产物、证据与隔离恢复演练已完成修复、验证和独立提交；
   生产对象存储与 WAL 目标时间点演练仍是发布验收项。
3. P1-8 删除态 review 状态机已完成修复、真实 PostgreSQL 回归验证和独立提交；
   已删除内容不会被举报处理复活，历史举报仍可正常结案。
4. P2-10 身份证 enc/hash 导入约束已完成最小修复、真实 PostgreSQL 验证和独立提交；
   当前仓库没有完整 pair 导入入口，不为本项额外建设未经需求驱动的 PII 导入 API/CLI。
5. P3-7 敏感导出审计已完成；每次请求只有一条可查询、可保留的 success/failure 事件。
6. P1-9 `phone.read` 已按既有安全模型完成实时 Casdoor 读取与 fail-closed 修复；
   仍需带真实 Casdoor 凭据执行一次发布环境验收。

#### 第二批：发布、运行前进性与核心契约

1. P0-1、P1-4 与 P1-5 已完成修复、验证和独立提交；P1-5 仍需在受保护
   GitHub environment 与真实目标机执行一次带审计记录的回滚演练。
2. P1-2 migration 指南与 P1-6 SSE shutdown 已完成修复、真实回归验证和独立提交。
3. P2-1 guard 路径分类、P2-2 always-on 静态供应链/Koishi package contract 与 P2-6
   privileged listener 生命周期均已完成修复、回归和独立提交；P2-6 的真实 Console
   服务热重载/插件卸载仍是发布环境验收项。
4. P2-7 Koishi 转发 poison、P2-13/14/15 claimed batch 与 P2-23 outbox handler panic
   均已完成隔离、全量/状态机回归和独立提交。
5. P2-21/P2-22 breaker 分类、R-8 Redis 错误分类与 P3-9 cache version unavailable
   均已完成最小修复和故障回归。
6. P2-9 Ansible 路径、P2-18 filter invalidation、P2-16 nullable course metadata、
   P3-7 敏感导出审计与 P2-12 外部 Oracle 用户级预算均已完成修复；第二批本地修复收口。

#### 第三批：可测量的性能与一致性

1. P2-3 ledger 逐消息热查询已完成有界读取和索引修复；无界留存降为需产品期限与容量证据的
   P3 决策项。继续 P2-4 dashboard 查询。
2. P1-10 resource N+1 与 P2-20 materialized-view 独立刷新预算均已完成修复、真实
   PostgreSQL 回归和独立提交；继续 P2-4 dashboard 查询。
3. R-11 outbox lock contention：只有指标或压测达到门槛后再改事件键/合并策略。
4. R-16 academics bulk import：真实 connector 上线前处理，不提前建设。

#### 第四批：低风险 UX、文档和清理

P1-7 已完成修复、真实路由/数据库回归和独立提交。继续 P2-5、P2-8、P2-11、P2-19、
P3-1 至 P3-6、P3-8、R-5 至 R-7、R-14、R-15、R-17。X-2 的配置分类治理已完成。

### 本轮执行的验证

- Admin：P0-1 的 capability-filtered 首页、可访问 redirect 和 6 类危险/无效 redirect
  定向 Vitest 10/10 通过；Admin 全量 Vitest 31 files / 153 tests 通过；`vue-tsc --noEmit
  --skipLibCheck`、Node 侧 `tsc`、`oxfmt --check`、`oxlint` 和 production build 均通过。
- 2026-07-31 累计完成性回归重新运行当前 Admin 全仓 `vsh lint` 时，发现 P0-1 新增的
  `route-resolution.ts` / `.test.ts` 留有 5 个 ESLint 门禁错误：import/named import 顺序、
  递归函数直接作为 `forEach` callback，以及测试使用 JSON 序列化深拷贝。业务行为和 153
  个测试仍通过，但先前“当前 HEAD 静态检查通过”的结论已不成立。现已只做机械收敛：
  callback 改为显式 wrapper，测试复用 Vben 现有 `cloneDeep`（保留 lazy route component，
  不使用无法克隆函数的 `structuredClone`）。修复后 Admin 全量 lint 为 0 warning / 0 error，
  两段 TypeScript 检查、31 files / 153 tests 与 production build 全部通过；未修改路由或授权
  语义，也未引入新 clone 工具。
- 同轮首次补跑 Admin production-preview 全量 Playwright 时，原有 capability route E2E
  仍断言受限管理员直达已过滤的 `/open-platform/apps` 后显示旧 404；桌面/移动均稳定失败。
  失败快照实际显示 P0-1 的预期安全行为：路由被回退到该角色可访问的 `/analytics`，且
  Open Platform API 请求数为 0，并非受限页面或数据被放行。现已把测试同步为同时断言
  可访问首页 URL/标题、无 404 和零越权 API 请求；production-preview 双视口定向 2/2
  通过。未为满足旧断言把安全回退改回 404，也未扩大路由授权模型。
- 首次 Admin 全量 Playwright 的另一个失败是移动端用户菜单偶发在点击“退出登录”前自行
  卸载；隔离重复可稳定得到 2 pass / 1 fail。根因不是认证请求，而是 click 模式仍创建
  `useHoverToggle`：初始化的 500 ms leave timer 在 watcher 被暂停后仍存活，较早打开的
  菜单会被旧 timer 关闭。`disable()` 现在只额外清理该 hook 自己的 pending timers；
  确定性 fake-timer 回归 1/1、移动退出连续 10/10、Admin unit 32 files / 154 tests、
  类型检查、全量 lint、production build 和最终双视口 Playwright 214/214 通过。没有修改
  logout/OIDC 语义、关闭全局动画或增加重试掩盖竞态；该项按低级/P3 UX 稳定性缺口记录。
- Koishi P1-1：缺 Session、私聊无 guild 的 authority 策略、显式 guild 越权和跨群数据隔离
  定向测试随相关 shared settings 共 14/14 通过；Koishi 全工作区 build、全量 unit
  593/593 和 Vue UI contracts 通过。
- Migration P1-2：文档卫生 Node 测试 5/5、shell 检查、CI/drift 契约和 Make 帮助/命令
  dry-run 通过；在无卷隔离 PostgreSQL 18 上复现 CI 的非超级用户 `stuhelper_app` 权限，
  实际完成 `000001` 至 `000019` 的全量前进、回滚最新一版、再次前进，最终
  `schema_migrations = 19, dirty = false`。验证容器已删除，未接触现有开发数据库。
- 部署包 P1-4：ShellCheck、`deploy-bundle-contract.sh`、CI drift/deploy security contracts 和
  文档卫生 5/5 通过。临时干净 Git 仓库实际运行打包脚本，确认被忽略的
  `.env.prod.secrets`/`node_modules` 不入包，根 env 文件恰好为两个模板；新增未跟踪文件后，
  脚本在创建输出前失败。临时仓库与 tar 已删除；尚未执行真实远端部署/回滚。
- PostgreSQL P1-3：固定 18.4 客户端确认原 `tar + gzip + stream + pgdata=-` 参数在解析阶段
  失败；最终实现没有采用依赖 WAL 保留窗口的 `fetch`，而是在独立 staging 使用
  `plain + stream` 和临时 slot。真实 PostgreSQL 18.4 源库完成 `pg_basebackup`，
  `pg_verifybackup` 成功且临时 slot 数回到 0；经 SHA256 与仓库恢复脚本提取后，无网络恢复
  实例启动为 `pg_is_in_recovery()=false` 并读回 `physical-backup-restored` 探针。故障注入确认
  失败不发布 final/sidecar、不残留 partial/staging；物理备份过期会阻断 evidence。
  ShellCheck（排除既有 source/字面量提示）、文档卫生 5/5、75 个全量 infra contracts 均通过。
  两个临时容器、匿名卷、网络和临时目录已删除；未接触现有开发数据库。
- 评课 P1-8：真实 PostgreSQL 顺序复现“先举报、后由作者删除、再处理举报”，分别覆盖
  `hide` 与 `delete`。处理后 review 始终为 `deleted`，`updated_at` 不变、课程计数保持 0，
  report 正常变为 `resolved`，后续 `restore` 被现有状态机拒绝；另测已由其他管理员隐藏的
  review 不可重复 `hide`，事务回滚后 report 仍为 `pending`。定向测试、定向 `-race`、
  `internal/modules/course/review` 全包测试、`go vet`、全服务端 `golangci-lint`（0 issues）
  与文档卫生检查均通过。
- 回滚 P1-5：用 2026-08-13 复现普通 policy 校验因 2026-08-12 审核窗口过期而失败；同环境
  2026-07-30 成功发布记录、相同 tag/三个应用 digest、完整历史 policy、执行人、理由和
  audit UUID 齐备时才通过。替换任一应用 digest 或使用过短理由均失败。隔离回滚控制器测试
  确认当前 policy 路径不创建例外记录；过期路径生成同一 audit ID、追加 mode 0600 的 JSONL
  后才调用部署。显式 source worktree 打包仍要求 clean HEAD 且保留 env 模板。ShellCheck、
  actionlint、文档卫生以及新增后全部 76 个 infra contracts 通过。尚未触发真实 GitHub
  protected environment、SSH 上传或生产回滚。
- SSE P1-6：两个定向测试均使用真实 PostgreSQL 和 `httptest.Server` 建立持续 HTTP 响应。
  取消应用 shutdown context 后，bot action 与 camera handoff 流都在 2 秒内写出
  `event:end`/`data:shutdown` 并结束；bot 用真实 `http.Server.Shutdown` 证明不会等到 deadline。
  定向 race、admission 全包（55.895 秒）、`go vet` 和全服务端 `golangci-lint`（0 issues）
  通过。尚未对已发布二进制发送真实 SIGTERM 或执行 Compose 滚动更新。
- 回复 P1-7：真实 PostgreSQL + Gin 完整路由注册覆盖同一列表中的 owner/非 owner 判断；
  owner 只看到自己的回复 `isOwner=true`，其他登录用户与匿名用户均为 false。注入 optional
  auth 后端故障时，健康门禁在 Handler 前返回 503。定向普通/race、review 全包（24.833 秒）、
  `go vet`、全服务端 `golangci-lint`（0 issues）、OpenAPI lint、全量生成漂移和文档卫生
  检查通过；未要求匿名用户登录，也未改变回复数据和删除授权语义。
- Open Platform P1-9：真实 PostgreSQL/Redis 中只放置已验证的掩码投影，fake authoritative
  reader 通过内部 user ID 返回完整 `+86` 手机号；phone API 与 identity-token 都得到标准化
  明文和掩码，并各写 granted 审计。随后注入 provider 故障，payload 不返回并写 denied 审计。
  app adapter 测试证明内部 ID 先由 user repository resolver 映射，解析失败时不调用 Casdoor。
  定向普通/race、Open Platform 全包（34.337 秒）、app/Casdoor 全包、`go vet`、Casdoor
  boundary guard 和全服务端 `golangci-lint`（0 issues）通过；没有使用真实 Casdoor 凭据或网络。
- 资源 P1-10：真实 PostgreSQL 上另建 `MaxConns=1` 的 pgxpool。列表分别返回 1 条和 2 条
  数据时，`Pool.Stat().AcquireCount()` 增量均为 3；详情查询同样为 3，证明查询数不随行数
  增长。6 个并发列表请求在同一单连接池内全部于 deadline 前完成；tags/bindings 排序保持，
  无关联项返回非 nil 空数组。定向普通/race、资源包全量、`go vet`、Casdoor boundary guard
  与全服务端 `golangci-lint`（0 issues）通过。
- CI P2-1：`ci-and-drift-contract.sh` 按具体 filter/job block 验证 `guards` output、
  `scripts/**`、`tools/**`、文档库、Vue UI contract 和 Node pin 的触发关系，并校验
  `.node-version == .nvmrc`。合约、Bash 语法与 actionlint 通过；ShellCheck 只有该脚本原有
  单引号 `$uri` 的 SC2016 info。尚未在 GitHub PR 上观察 dorny 输出和各 job 实际调度。
- CI P2-2：always-on `static-contracts` 定向执行 Dockerfile supply-chain 与 CI wiring，
  且成为 `required.needs`；Koishi job 在现有 build/test 后执行 deployable package contract。
  三个定向合约、actionlint 与全部 76 个 infra contracts（75 shell + 1 mjs）通过。未在真实
  GitHub runner 上验证 job 调度/耗时，也未把本地合约通过表述为 branch protection 已生效。
- Koishi P2-6：源码扫描确认 StuHelper Core 与 Group Guard 的 production listener 注册只剩
  两个受 `ctx.effect` 管理的 registrar；单测覆盖 authority 不能被调用方降到 4 以下、卸载时
  删除本作用域 registration，以及旧作用域 disposer 不会误删同 event 的新 registration。
  Koishi build、Vue UI contracts、595/595 unit tests、startup smoke 与 deployable package
  contract 通过。首次完整 `yarn test` 在 WebUI E2E 的无关“配置治理标签页导航”用例出现一次
  5 秒时序失败（45/46），随即在全新临时 Koishi 实例上复跑 `yarn test:ui` 为 46/46；
  因此如实记录为一次偶发复跑通过，而不把首次完整命令写成成功。现有 E2E 会真实调用 Console
  action，但不会在已连接客户端下切换 Console 服务或卸载插件，真实热重载/卸载仍待验收。
- Koishi P2-7：后端 SQL 静态复核确认队列按 `created_at` 取最老 100 条，且只有 ACK 后才设置
  `forwarded_at`。新增测试覆盖“旧 delivery poison + 新健康项”：失败项不 ACK，健康项仍发送并
  ACK，批次最终保持 rejected；另测 ACK 失败后立即停止后项，避免扩大未确认发送。既有测试继续
  覆盖全部管理群成功后才 ACK，以及任一管理群失败时全部目标均尝试但不 ACK。独立代理从
  scheduler、ACK、重复发送和 batch window 反证后确认分相方案无提交级 blocker。完整
  `yarn test` 一次通过：build、Vue contracts、597/597 unit、startup smoke 和 WebUI E2E
  46/46；deployable package contract 也通过。测试使用 fake bot/platform，没有向真实 QQ 群
  发送材料；功能当前默认关闭，尚未在发布环境开启并观察真实队列推进。
- Academic importer P2-10：迁移约束与 PostgreSQL 18 临时表实测均确认 hash-only 新行会被
  `chk_buaa_students_sfzjh_secure_pair` 拒绝，旧完整 pair 在旧 upsert 尝试清空 hash 时也会让
  整条 INSERT 失败；约束会阻止半对落库，真实后果是普通字段同样无法导入/更新。修复后分别实测
  旧行名称更新时 `sfzjh_enc` 字节和 `sfzjh_hash` 原样保留，新行得到 `NULL/NULL`。契约测试覆盖
  两个禁用 header、SQL 不再写 pair 及生产手册边界；ShellCheck、75 个 shell + 1 个 Node
  infra contracts、文档卫生单测 5/5 和全量文档检查通过。独立代理另外在用后即销毁的
  PostgreSQL 18 容器复现旧两种失败并逐列核对修复后的 15 列链路，最终只要求纠正文案中不存在
  的“现成受控工具”，纠正后无代码 blocker。主代理一次长复合数据库验证调用无输出并以 139
  退出，拆成只操作临时表的短事务后全部通过；未向持久表导入测试数据，也未处理真实身份证件号。
- Admission claimed batch P2-13/14/15：两个独立只读代理分别复核状态机与测试设计，确认
  claim 事务提交后 enrichment 失败会遗留整批 `dispatched` 并消耗 attempt，且旧
  `MarkBotActionStale` 没有 attempt fence；同时确认 P2-14/P2-15 是同一 N+1 根因。
  实现后真实 PostgreSQL 覆盖 1/8 行固定 3 次 pool acquisition、policy 表故障后的整批归还与
  可重领、父 context 取消后的 detached cleanup、旧 attempt stale/abandon 不覆盖新 lease、
  缺 policy poison kick 与健康 release 同批隔离及 poison 第 5 次单独 dead-letter。
  Admission 开启 race detector 的全包测试、全服务端 lint/build 与文档卫生检查通过；未模拟
  Service 返回后的网络丢包，因为那时动作可能已公开，按 at-least-once 语义不得自动归还。
  attempt 回退被限制为同步、未公开、single-shot cleanup；没有为当前不可达的异步 ABA
  场景提前增加 generation migration。
- image policy：2026-07-30 通过，2026-08-06 因 `review_by=2026-08-05` 失败，确认日历门禁。
- 授权：capability/RBAC/review 定向 Go 测试通过，确认 X-1 在 Handler 前 fail-closed。
- P2 早期基线：3 个 infra/import contract、Koishi 定向 29 tests，以及 outbox、externaldata、
  review、admission 定向 Go 测试通过；P2-7 已另补 poison/ACK 专项回归，P2-21/P2-22 已补
  caller cancellation、内部 timeout、half-open probe 和完整性分类专项回归。其余绿灯仍不覆盖
  panic-continuation、真实 Oracle 驱动/生产数据、真实 DB pair 与大表场景。
- P3/驳回项：Admin 相关 Vitest 9/9、Koishi reminder 7/7、externaldata、OTP、cache、
  review projection 定向 Go 测试通过；R-8 与 P3-9 已分别补真实 Redis transport error，
  P3-9 另覆盖恢复后读取真实版本与 caller cancel bypass；仍缺 min-width override、
  reviewed-row action 和 lock-wait 测试。
- P2-11 对 `openapi-fetch` 的实测确认数组被序列化为重复 `courseIDs` 参数。
- P2-9 后续已在隔离 venv 安装固定 `ansible-core==2.20.2`，补齐真实 controller 复现、
  三 playbook syntax check 和修复后 bundle task 执行；仅远端 SSH 上传/部署仍 pending。
- P2-20：开发库只读盘点为 15 名教师、8 个院系、13 条评课和 15 行投影；复核期间两组
  共 6 次真实 `REFRESH MATERIALIZED VIEW CONCURRENTLY` 为 4.040–20.517 ms，统一 outbox
  行为 `completed / attempt_count=0 / last_error=NULL`，因此没有把假设的生产永久陈旧写成
  现状。真实 PostgreSQL 回归另用 20 ms 普通预算证明 250 ms 显式预算可完成 60 ms statement，
  并证明显式 25 ms deadline 与已取消 parent 均能终止；Repository 在普通预算仅 1 ns 时仍可
  通过 2 秒专用预算刷新真实物化视图。生产行数、锁等待和 p95/p99 尚未验证。

### Codex 修复进度

| 编号 | 状态 | 实现与验证 | 独立提交 |
|------|------|------------|----------|
| P0-1 | 已修复，未发布；累计 lint 与浏览器契约回归已收口 | 按已过滤路由推导首页；redirect 必须命中可访问路由，否则回退。完成性审计发现并修复原实现文件的 5 个 ESLint 门禁错误；另把仍期待旧 404 的 Admin E2E 同步为断言受限路由安全回退到可访问首页且 Open Platform API 请求为 0。Admin 全量 lint 0 warning/error、两段 TypeScript 检查、production build、双视口定向 E2E 2/2 和最终全量 E2E 214/214 通过；unit 最终为 32 files / 154 tests（含 N-2 新回归） | `fix(admin): route users to an accessible home`；`fix(admin): keep route resolution checks lint clean`；`test(admin): align filtered-route E2E with safe fallback` |
| P1-1 | 已修复，未发布 | 缺 Session fail-closed；私聊无 guild 仍校验 policy 且禁止无范围查询；显式 guild 保留并按目标群授权。定向 14 tests、Koishi build、全量 593 tests 和 UI contracts 通过 | `fix(koishi): fail closed without a guild context` |
| P1-2 | 已修复，未发布 | 迁移权威改为有序 migration 集合；`000001` 锁定为初始基线，后续只新增递增 `.up/.down`；同步 Make/CI/数据库文档。文档与 CI 契约检查通过，隔离 PostgreSQL 18 实际 `up → down 1 → up` 后为 `19, dirty=false` | `docs(database): make migrations append-only` |
| P1-3 | 已修复，未发布；生产 PITR 待验收 | 物理备份在外部 staging 以 `plain + stream` 生成，经临时 slot、`pg_verifybackup`、SHA256 和 `.partial` 原子发布；同步排除临时工件，evidence 覆盖本地/取回的逻辑与物理备份及新鲜度。真实 18.4 隔离恢复启动并读回探针；ShellCheck、文档卫生和 75 个 infra contracts 通过 | `fix(backup): verify and atomically publish base backups` |
| P1-4 | 已修复，未发布 | 部署包只取干净 Git `HEAD`，生成后断言两个根 env 模板存在且无其他根 env；干净临时仓库实测忽略 secret/依赖不入包、未跟踪文件 fail-closed。ShellCheck、部署包/CI 契约与文档卫生通过 | `fix(deploy): preserve required env templates in bundles` |
| P1-5 | 已修复，未发布；真实远端回滚待验收 | 普通部署维持当前日硬门禁；历史窗口只对同环境成功记录和完全相同 digest 的审计回滚开放。当前 workflow 控制器兼容旧 release，每日 3 天提前告警。ShellCheck、actionlint、文档卫生及 76 个 infra contracts 通过 | `fix(rollback): audit expired image review exceptions` |
| P1-6 | 已修复，未发布；真实进程 SIGTERM 待验收 | Admission handler 接入既有 shutdown context；两个 SSE 写 `end/shutdown` 后退出，并在周期任务前优先检查停机。真实 HTTP/PostgreSQL、`http.Server.Shutdown`、定向 race、admission 全包、vet 与 lint 通过 | `fix(admission): release SSE streams during shutdown` |
| P1-7 | 已修复，未发布 | GET replies 的 OpenAPI 与真实路由均改为可选认证，认证后端故障 fail-closed 503；生成 bundle/Go/TS 契约。真实 PostgreSQL route-level 测试覆盖 owner、非 owner、匿名和故障，定向 race、review 全包、vet、lint、spec/drift 与文档检查通过 | `fix(review): preserve reply ownership on refresh` |
| P1-8 | 已修复，未发布 | Service 复用统一 review 状态机；作者删除态只结案 report，不改 review。真实 PostgreSQL 覆盖两种动作、重复转换、计数、时间戳和后续 restore；评课包全量、定向 race、vet、全服务端 lint 与文档卫生检查通过 | `fix(review): preserve deleted reviews during report handling` |
| P1-9 | 已修复，未发布；真实 Casdoor 待验收 | Open Platform 仅按内部 user ID 请求 authoritative phone；app gateway 在边界解析 Casdoor subject。真实 PostgreSQL/Redis 覆盖 phone API、identity-token、granted/denied 审计和 provider 故障；adapter、Casdoor client、race、全包、vet、边界门禁与 lint 通过 | `fix(openplatform): read disclosed phones from Casdoor` |
| P1-10 | 已修复，未发布；生产规模影响待观测 | 主结果集完整 drain 后显式关闭，tags/bindings 分别批量加载，查询数由 `1 + 2N` 固定为 3。真实 PostgreSQL 单连接池覆盖固定 query count、详情、空数组、排序和 6 并发；定向 race、资源全包、vet 与 lint 通过 | `fix(resource): batch related data loads` |
| P2-1 | 已修复，待 GitHub CI 验收 | `guards` 覆盖根 scripts/tools；文档库、Vue 合约、Semgrep 和 Node pin 精确触发消费 job，两个 Node 版本文件强制同步。CI contract、Bash 语法、actionlint 与文档检查通过 | `fix(ci): route guard changes to their checks` |
| P2-2 | 已修复，待 GitHub CI 验收 | always-on 静态 job 执行 Dockerfile/CI wiring 并纳入 Required；Koishi job 执行 deployable package contract。三个定向合约、actionlint 与全量 76 个 infra contracts 通过 | `fix(ci): make supply-chain contracts unskippable` |
| P2-6 | 已修复，未发布；真实 Console 热重载/插件卸载待验收 | Group Guard 使用 required console 子作用域；Core/Group Guard 特权 listeners 都由 `ctx.effect` 绑定生命周期，并用 registration identity 避免旧 disposer 删除新注册。authority 固定为 4。build、contracts、595 unit、startup 与 package contract 通过；WebUI E2E 首轮无关用例 45/46，立即复跑 46/46 | `fix(koishi): dispose privileged console listeners` |
| P2-7 | 已修复，未发布；真实材料转发待启用验收 | delivery failure 逐项记录并继续，成功项 ACK；批末重抛。ACK failure 独立记录并 fail-fast。poison→healthy 与 ACK failure 回归通过；完整 Koishi build/contracts/597 unit/startup/46 E2E 及 package contract 通过 | `fix(koishi): isolate freshman forward failures` |
| P2-10 | 已修复，未发布；完整 pair 导入能力不存在且不在本次范围 | 普通 TSV 拒绝两个身份证安全列，15 列 normalize/copy/upsert 链路不再写 pair；已有 pair 保留，新行 `NULL/NULL`。真实 PostgreSQL 临时表语义、定向契约、ShellCheck、76 个 infra contracts 与文档卫生通过 | `fix(import): preserve academic identity pairs` |
| P2-13/P2-14 | 主因已修复，未发布；P2-15 为重复标签；补偿失败/replay 边界已记录 | 每批固定两次上下文查询；补偿可写时，全批 enrichment failure/取消归还未公开 lease；单行确定性 preparation failure 独立 retry/dead-letter，stale/failure/abandon 使用 attempt fence。真实 PostgreSQL 覆盖固定查询数、故障/cancel cleanup、旧 lease fence、poison 隔离；Admission 全包 `-race`、全服务端 lint/build 与文档检查通过。未把补偿写同时不可用或 terminal replay 误报为已解决 | `fix(admission): isolate claimed action failures` |
| P2-21/P2-22 | 已修复，未发布；真实 Oracle/生产指标待验收 | parent cancel/deadline 与单条 invalid/ambiguous row 记 neutral 并释放 half-open probe；目录自身 timeout、backend failure 和 bind identity mismatch 仍记 failure。三类固定 integrity metric 已接入，typed error 经既有 adapter 保持 503。定向三包 race、全服务端 race、lint/build 与文档卫生通过 | `fix(externaldata): classify Oracle source outcomes` |
| P2-23 | 已修复，未发布；生产 poison 告警/replay 待验收 | `process` panic 转成带 stack 的普通 job error；第 5 次沿既有路径 dead-letter、增加 terminal metric，同批健康 job 继续。真实 PostgreSQL 18 直接验证 poison/healthy 最终状态；outbox 定向与全服务端 race、lint/build/docs 通过。未修改 runtime root supervisor，也未把两个 Admission expiry loop 误报为已覆盖 | `fix(outbox): isolate panicking job handlers` |
| P2-20 | 已修复，未发布；生产规模与锁等待待验收 | `ExecWithTimeout` 复用原 Exec 的 tracing/DB 指标且不重试，专用预算继承 caller/shutdown cancel；配置默认 60 秒、仅允许 5–90 秒并进入两份 env 模板。真实 PostgreSQL 验证显式预算覆盖普通预算、parent cancel 优先，以及 Repository 在 1 ns 普通预算下仍完成真实刷新；开发库 6 次实测 4.040–20.517 ms、outbox 无失败。全服务端 race/lint/build、docs 与全部 infra contracts 通过；未抬高全局 timeout、未新增重复指标/cadence 框架 | `fix(review): separate teacher projection timeout` |
| R-8 | 已修复，未发布 | Admission/User 两条 OTP cooldown reserve 对 `redis.Nil` 保持 429，对真实 miniredis server 关闭后的 transport failure 返回各自 unavailable sentinel 并映射 503；Admission request-otp OpenAPI 与 bundle/Go/TS 生成物同步。两包/全服务端 race、lint/build、spec、生成稳定性和 docs 通过 | `fix(otp): distinguish Redis outages from cooldowns` |

### 明确不建议实施的“修复”

- 不为 X-1 做紧急授权模型/Repository 类型大改；攻击链在 admin Entry 前已被截断。
- 不要求所有 `getenv` 必须同时出现在两个 env 模板；先按运行时、工具、标准变量、测试变量分类。
- 不把 image review freshness 在所有生产部署和回滚中降成 report-only。
- 不为 P1-6 顺带强制 SSE 10 分钟断线。
- 不用 Repository 静默 no-op 替代 P1-8 的 Service 状态机。
- 不让 P1-9 的 Open Platform 业务模块持有 Casdoor subject，不落库/缓存完整手机号，也不把
  authoritative provider 空值降级成成功响应。
- 不为 P1-10 引入 ORM、DataLoader 或通用关联加载框架；两个本地批量查询已经消除该根因。
- 不把 `bots/koishi/**` 整体加入 infra filter；构建型 package contract 复用 Koishi job，
  静态供应链合约 always-on，避免为每次机器人改动重复执行整套浏览器/生产运维合约。
- 不为 P2-6 monkey-patch 全局 Console、修改 `node_modules` 或只按 event 无条件删除 listener；
  后者会让旧作用域的延迟清理误删服务重载后同名的新 registration。也不为两个插件边界先建
  通用事件注册框架，局部 lifecycle helper 已覆盖当前根因。
- 不为 P2-7 立即新增 `forward_attempt_count`、dead-letter、按群持久化 delivery 或
  claim/lease schema，也不把最多 100 条顺序发送改成 `Promise.allSettled` 并发冲击 QQ；
  当前局部隔离已解除单 poison 队首阻塞，复杂治理应由真实启用后的失败率与重复率驱动。
- 不为 P2-10 伪造空 `sfzjh_enc`、用 `COALESCE` 模糊两列所有权，或在没有需求、密钥分发和
  审计设计时顺带新增完整身份证导入 API/CLI。当前正确边界是普通 importer 完全不拥有这两列；
  将来若确需导入，必须从同一明文原子生成 AES-GCM envelope 与 HMAC，并单独完成安全审计。
- 不用 `COALESCE('', 0)` 隐藏 P2-16 的 NULL 数据语义；schema 与客户端语义确认这些值可未知，
  应通过 nullable contract 传递。当前开发库 23 行虽没有 NULL，也不能据此收紧历史/未来导入约束。
- 不在当前单副本部署为 P2-18 先建 Redis pub/sub/version 系统。
- 不为 P2-23 增加无限自动 supervisor；panic 必须进入现有 retry/dead-letter。
- 不在无真实 connector、无规模证据时为 P2-3/P2-4/R-16 建完整管理 UI、聚合平台或 bulk framework。
- 不把已明确有文档和测试的 R-12 产品行为改成失败。

## Codex 对 `AUDIT-REPORT.md` 第二轮的独立复核（2026-07-30）

### 复核范围、完整性与口径

Claude 新生成的 `AUDIT-REPORT.md` 是一份新的汇总快照，不是对本文件的替换或严格超集：

- 报告所称“确认 82 条”由**第一轮 43 条**和**第二轮 39 条**相加而来，不代表又新增了
  82 条问题。第一轮的 43 个确认标签、18 个驳回标签以及 X-1/X-2，已经在本文件上方完成
  逐项复核。
- 在本节写入前，`AUDIT-FINDINGS.md` 与当前 `HEAD`（`81f4b62e`）中的版本逐字节一致，
  SHA-256 均为
  `8e6273a687a17cc18ee22104a61f1aa678efb045e4ed67bf5d5f9cf3d9717f0f`。因此此前 Codex
  写入的复核、修复进度和“不建议实施的修复”没有被 Claude 的新报告覆盖或丢失。
- 本轮没有只复核 Claude 列为“确认”的 39 条，而是覆盖了第二轮全部 **56 个原始标签**：
  39 个确认候选、14 个驳回候选和 3 个未完成验证候选。两条 `/admin/stats` MFA 记录是
  同一根因的重复标签，故合并后为 **55 个唯一根因**。
- 结论以当前源码、OpenAPI、迁移、配置、测试和本地构建行为为准。`AUDIT-REPORT.md`
  中 Claude 的原始取证保留用于追溯；若其严重度、影响范围或修复方案与本节冲突，以本节为准。

本节使用以下结论口径：

| 结论 | 含义 |
|------|------|
| 确认 | 存在当前可达缺陷或明确违反已声明契约；建议进入待办 |
| 部分确认 | 机制或较窄问题真实，但原报告夸大了影响、引用了错误路径，或修复前需要产品/安全决策 |
| 证伪 | 当前攻击链/失败场景不可达，或行为是有文档和测试锁定的设计；不按原问题修改 |

### 第二轮总结果

| Claude 第二轮分组 | 原始标签 | 唯一根因 | Codex 确认 | Codex 部分确认 | Codex 证伪 |
|--------------------|----------|----------|------------|----------------|------------|
| Claude 列为确认 | 39 | 39 | 30 | 9 | 0 |
| Claude 列为驳回 | 14 | 13 | 1 | 4 | 8 |
| Claude 未完成验证 | 3 | 3 | 2 | 1 | 0 |
| **合计** | **56** | **55** | **33** | **14** | **8** |

因此第二轮有 **47 个唯一根因需要处置或明确决策**，不是可以不加区分地照单全收的
“39 个新增确认问题”。按本轮重新校准后的处置级别统计：

| 级别 | 唯一根因数 | 说明 |
|------|------------|------|
| P0 | 0 | 没有证据支持立即全局阻断或已发生灾难性后果 |
| P1 | 8 | 授权边界、令牌语义、撤权一致性或核心管理链路 |
| P2 | 21 | 明确的发布正确性、可靠性、隐私日志、可用性和重要 UX |
| P3 | 18 | 局部 UX、可访问性、契约/文档和治理加固 |

严重度表示修复优先级，不表示当前生产已经发生相应后果。尚未执行真实生产 Casdoor、
真实 Console 操作员、原生 SSO、超过 100 条真实队列或生产 Redis
延迟验收，不能把静态机制直接写成生产事故。

### 对 Claude 第二轮 39 个“确认问题”的逐项复核

下表编号沿用 `AUDIT-REPORT.md` 的全报告编号，便于在两份文件之间交叉定位。

| 报告编号 | Codex 结论 | 调整级别 | 必要性与最小处置；不建议的过度设计 |
|----------|------------|----------|--------------------------------------|
| 3 | 确认，已完成本地修复、对抗复核与回归验证 | P1 | Group Guard 通过 Core service 复用唯一的权威 guild-scope resolver；保存 bot-wide runtime settings 前先解析必需 scope 并 `assertGlobal`，resolver 缺失、认证畸形、authority 不足或 binding 查询失败均 fail-closed。scoped 页面不再下发或渲染全局开关，绕过 UI 直接调用仍在任何保存/刷新前拒绝。没有搬迁整套 RBAC helper 或建设第二套 Console 权限框架。 |
| 4 | 确认，已完成本地修复、对抗复核与回归验证 | P1 | 成员、binding、被引用 template 和全部统计先按可管理 guild 过滤，再排序和截取 100 条；action 读取可信 record 后、在任何平台或 store 副作用前断言目标 guild。bot-wide 配置、平台信息和全部 Bot inventory 收拢为 `globalRuntime`，scoped response 返回 `null` 且不加载这些全局 store，客户端只显示权限说明。首次独立复核据此发现并阻断了“业务记录已过滤但全局运行态仍泄漏”的不完整修复；补齐后第二次复核 PASS。把 scope 下推 Repository 只有在真实规模/延迟需要时再做，当前继续改造属于过度优化。 |
| 5 | 确认 H5 缺陷并已完成修复；相邻 mp-weixin 假绿也已按不支持边界收口 | P2 | H5 的 8 个 tabBar 资源已移入输入树并由声明驱动的 source/output 契约保护。对 mp-weixin 再次实测：旧命令退出 0，却只生成 H5 `index.html/assets`，且仓库没有微信 compiler、认证或真机链路；因此没有把 H5 冒充微信产物，也没有无需求扩建微信平台。现已移除根/包级微信 dev/build 命令、manifest 微信 appid 声明并在 README 明确 H5 是唯一正式验证目标；零依赖 CI 契约持续拒绝重新误宣称。UniAppX 仍是实验性客户端，因此不升为 P1。 |
| 6 | 确认，已完成本地修复与回归验证；纠正恢复与清理语义 | P2 | PostgreSQL/OpenAPI 是按用户唯一的单槽 draft；页面原先不核对 `courseID`，会把 A 课程内容恢复到 B，并在 B 成功提交后删除未消费的 A 草稿。现在只恢复未绑定或 `courseID` 精确匹配当前课程的草稿；未绑定草稿不恢复 teacher。显式 cleanup eligibility 只在本页成功恢复或成功保存该槽后置 true，且仍只有 create 成功才 best-effort DELETE；外课程草稿不载入、不清理，保存响应不确定时也保守不授予清理资格。产品文档已纠正无 course path 的 GET/POST/DELETE 和单槽语义。没有改 DB/OpenAPI 为多草稿、增加 autosave 框架或为尚未证明的跨设备 ABA 引入 generation/CAS migration。 |
| 7 | 确认，已完成本地修复与回归验证 | P2 | 只有 provider-owned 的 username/email/avatar 原本被硬编码到只读 `/account/profile`；phone/identity/student/school 已指向真实可写页面。现在 Profile Completion 按字段复用 `/auth/me` 已校验的绝对 `accountSettingsUrl`，四类本地字段仍使用后端 action，URL 缺失时保守回退声明值。10 个单元用例覆盖三类 provider、四类本地和三类缺省回退，桌面/移动 Playwright 20/20 同时验证页面 href、刷新、非法 redirect 与完成后 Continue。没有重构 profile service、增加通用 action registry、复制 issuer fallback 或在只读摘要页建设资料编辑器。 |
| 8 | 部分确认，已完成本地修复与回归验证 | P2 | 只影响创建资源的 POST；metadata PATCH 不上传内容。原严格相等会误拒 OOXML/旧 Office、CSV/Markdown/JSON 等常见 refinement，但不是“每种文件都必然失败”。现在只在资源解码边界接受精确枚举的 ZIP 容器与文本细分；旧 Office 还要求 OLE 魔数，JSON 还要求语法有效，并将规范化有效 MIME 传入存储和版本记录。23 个正反边界子测试、真实 PostgreSQL 持久化用例和资源包全量 race 通过。任意 `text/*`、`vnd.*`、`*+zip`、无魔数 legacy 声明、无效 JSON 和真实类型矛盾仍拒绝；未改身份/准入图片链路，也未增加前后端重复 allowlist。 |
| 9 | 确认，已完成本地修复与回归验证 | P1 | review parser 已按 OpenAPI 的 `like`/`dislike` enum 保留可选 `userVote`，非法值继续 fail-closed；针对两种投票和非法值的定向回归通过。没有为了一个字段建设反射式“全 DTO 自动对齐”框架。 |
| 11 | 确认；owner 已选择长期架构，全面实施中 | P1（功能不可用，非越权） | 仓库没有受支持、可审计的 school/section admin tuple 发放与撤销流程；运维仍可直接写 OpenFGA tuple，所以不是物理上“无法写入”。实际后果是 scoped role 默认 fail-closed、角色不可用，不是自动获得全局权限。2026-07-31 owner 明确选择 PostgreSQL 授权账本作为唯一管理真源、OpenFGA 作为可重建运行时判定面、Casdoor 仅认证且不再承载业务角色。决策已写入 ADR-0008、IAM 架构与实施守卫；后续按 migration、grant/list/revoke、outbox projection、DB-derived snapshot、Casdoor role-sync 退役和闭环测试逐项实施。 |
| 17 | 确认，已完成本地修复、真实 OpenFGA 协议与回归验证 | P1 | `super_admin` tuple 原先只增不减，角色降级后 OpenFGA 全局权力会残留。现在只有 Web/native login 与 refresh 新签发、已验签且显式包含结构合法 roles claim 的 ID token 才设置 `RolesAuthoritative` 并执行 reconcile；`/auth/me` 的旧 access token、claim 缺失、`null` 或解析失败均不能增删 tuple。撤权使用 higher-consistency 的完整 direct tuple 精确读取，再以 `on_missing=ignore` 幂等删除并写 `iam.role.revoke` 审计；读取/删除失败时认证同步 fail-closed。真实 OpenFGA v1.18.1 临时 store 验证写入后为 1、首次和重复删除均为 200、最终为 0，store 已删除。没有顺手建设 #11 的 scoped provisioning 平台，也没有让 introspection 每请求写 OpenFGA。 |
| 18 | 确认，已完成本地修复与真实 Redis 验证 | P1 | blacklist TTL 不取 token 自身 `exp` 的问题真实；进一步确认 tracked logout 原先先按 session 写 blacklist，随后又用本地 5 分钟 TTL 对同一 key 二次 `SET`，会把较长 TTL 缩短。现在 Web/native login 与 refresh 都从已验签 ID token 保存 provider `exp`，refresh 在既有 Redis Lua 中原子更新 access hash + expiry；新 token 剩余寿命必须不超过 session lease 和 30 天 hard cap。logout/logout-all 对每个 session 按真实剩余寿命吊销，已过期不写 key，低于 1 秒只向上取整，超过上限不静默截断；tracked 撤销成功后直接返回，消除了二次覆盖。滚动升级旧 session 仅在仓库托管 Casdoor access=1h、不超过 session lease 的约束下使用真实 Redis PTTL，无 TTL 时 fail-closed。永久回归、四包定向/race、全服务端 race、lint/build/docs 均通过；一次性 Redis 8.8.1 容器验证原子轮换、50 分钟现代 session、20 分钟 legacy PTTL、10/40 分钟逐 session logout-all 和 hard-cap 拒绝均符合预期，容器已删除。没有给每个 Bearer 请求新增 session lookup/ID；当时独立保留的 N-1 provider no-op 后续已按下方 N-1 记录修复，没有混入本项。生产实际 Casdoor TTL 与真实登出仍待发布验收。 |
| 28 | 确认，已完成本地修复与失败路径回归 | P2 | 页面保存的是 7 个逻辑域，而非固定“7 次请求”：实际 mutation 数是 `6 + D` 次关键词删除 + `N` 次关键词 upsert。现在固定顺序记录 confirmed/unconfirmed/not-run，只对收到成功响应的 slice 推进 baseline；关键词域每个已确认删除/upsert 都立即推进，失败重试不会重复已确认删除。错误区展示逐域结果，并提供二次确认后的服务端 reload；missing keyword delete 幂等，已存在规则仍保留 guild scope 检查。WebUI 也复用服务端的安全正则校验。没有用 `Promise.all`、前端 rollback、2PC 或 saga 伪造跨 HTTP 事务。 |
| 30 | 部分确认，已完成本地修复与截断边界回归 | P2 | Admission runtime 确实在当前 guild scope 内把 active records 截断为 100 条，原页面同时显示未截断统计和“100 条”却无解释；但“其余记录在整个 Console 不可达、只能等待前 100 条老化”不成立。现在 API 返回同一 scope 快照的 `shown/total/limit/truncated`，服务端和客户端统一用 `(deadlineAt,id)` 稳定排序；页面显示 `shown / total`，截断时解释窗口并可直接进入处置中心按准入/成员/群号检索，也说明群内命令入口。103 条同截止时间回归固定前 100 条边界。处置中心和后台扫描仍不受窗口限制；没有提前增加 Repository pagination/search。 |
| 33 | 部分确认 | P3 | custom navigation 没有通用安全区处理，首页风险明确；仅凭静态代码不足以断言 login/callback 在所有设备都重叠。可恢复 native navigation，或按平台 status bar/capsule 做正确间距，并用微信真机验收。 |
| 34 | 确认，已完成本地修复、真实 Chromium 复现与生命周期回归 | P2 | 每个实际执行的 resize rAF 回调原先会丢弃当前 50 个粒子引用并新建 50 个 `repeat:-1` GSAP tween；同帧 resize 已合并，原报告按每个 event/帧推算的数量不是实测。Chromium 探针确认旧 tween 在离开首页后仍被 global timeline 强引用并继续推进。现在 `createParticles` 在丢弃数组前精确 `killTweensOf` 当前批次；回归固定 resize 合并、旧批先 kill、新批重建和卸载清理。没有增加 animation manager、坐标 clamp、Worker、ResizeObserver 或 Tween handle registry。 |
| 35 | 确认，已完成本地修复、删除边界与真实双视口回归 | P2（偏低） | `/user/reviews` 的 own-review 垃圾桶原先首击即发送 DELETE；后端是 soft delete，正文仍在库中，但用户和管理员都没有受支持的 restore 路径，且用户可重新发布造成旧状态恢复语义复杂。现在 `ReviewCard` 使用局部两阶段确认：首击/取消零请求，出现时聚焦取消、取消后恢复垃圾桶焦点，确认后才删除；composable 在同步置位后 fail-fast 保持 single-flight。累计 Web E2E 首轮发现旧测试仍假设首击立即 DELETE，现已改为先断言确认组可见且请求为零，再确认并断言唯一 DELETE；桌面/移动定向 2/2 和全量 363 passed / 1 designed skip 通过。内联块使用非模态 `role=group`，没有伪装成缺 focus trap 的 `alertdialog`；未新增 undelete API、migration、延时队列、undo、服务端 `confirm=true` 或全站确认框架。 |
| 36 | 确认，已完成本地修复、脱敏边界与退化路径回归 | P2 | `ErrorBoundary` 阻止异常继续传播，production 原先又不输出或上报，现有 frontend error telemetry 因而收不到组件错误。现在复用既有 `/api/v1/metrics/frontend-errors`，只新增固定枚举 `vue-error`；边界内异常由 `ErrorBoundary` 计数，边界外异常由 Vue 全局 handler 计数，前者继续返回 `false`，不会重复上报。组件异常只发送 `{"kind":"vue-error"}`，后端仍只读有限 `kind` 并增加 Prometheus counter，不存储 message、stack、props 或用户输入；未初始化 observability 的受限/E2E host 直接 no-op，transport 抛错也被隔离。没有新增第二个 endpoint、错误存储、全局 error bus 或 Sentry 类平台。 |
| 37 | 确认，已完成本地修复与双视口真实键盘回归 | P2 | AppShell 原先没有 skip-to-content，键盘用户必须逐项穿过固定导航。现在 DOM 第一个键盘停靠点是本地化的原生 `<a href="#main-content">`；目标为唯一稳定的 `main#main-content[tabindex="-1"]`，链接只在 `:focus-visible` 时出现在 header 之上，并处理顶部/左侧 safe area 与 reduced-motion。桌面和移动 Chromium 都验证首次 Tab 聚焦且显示链接、Enter 后焦点进入 main；同文件 Axe A/AA 基线保持全绿。原文“所有页恰好 11 个 tab stop”不是成立所必需，也未硬编码。没有增加 JS focus manager、路由 hook 或在每个业务页复制锚点。 |
| 38 | 确认，已完成本地修复与跨作用域回归验证 | P2 | toast 状态是全局的，但组件卸载 cleanup 原先只取消 timer、不移除全局 toast，能留下永久提示。现在删除错误的组件作用域 timer cleanup，让现有模块级 timer 在页面卸载后仍按原 duration 关闭；显式 `remove`/`clearAll` 继续同步取消 timer。没有把已有模块级状态重写成新 store/singleton，也没有在卸载时立即删掉用户尚未看见的成功提示。 |
| 39 | 确认，已完成本地修复与回归验证 | P2 | projection polling 原先遇到一次瞬时 `/auth/me` 失败即终止，且 `fetchUser` 会把任意非网络 ApiError（包括 5xx/403）当成登出并清空用户，最终把已验证用户送进不可恢复错误页。现在 `fetchUser` 只在 HTTP 401 明确拒绝会话时清本地身份，页面同时切换登录态；403、5xx、网络、超时和未知 refresh 错误保留当前用户并继续消耗既有 1/2/4/8/16 秒有限预算，耗尽后仍停在 `projectionPending` 且可手动重试。Abort 立即终止，capability predicate 的程序错误不被重试层吞掉。没有把 403 宽泛当作失效会话，也没有新增重试框架、无限轮询或放宽服务端授权。 |
| 40 | 部分确认 | P3 | SearchPage 确实忽略 `ReviewCard` emits，但当前可达的是 `moderated`；deleted/updated 控件并未启用。只接线 moderation 后 refetch/局部更新，避免为不可达事件建设通用同步总线。 |
| 41 | 确认 | P3 | teacher profile 的 load-more 失败会用错误面板替换已加载 review。区分 initial 与 append error，保留已有列表并提供重试当前页即可。 |
| 42 | 部分确认，已完成本地修复与失败重试回归 | P2 | fresh store 的 status fetch 失败原先会把 phone、QQ、实名、学籍及对应披露状态呈现为 false negative；Claude 所称邮箱和全部 base info 也必然受影响不成立，它们来自当前已认证用户。现在页面显式区分 loading/ready/error：保留可靠账号与邮箱，完整状态读取成功前不渲染“未验证/未绑定/缺失”结论，失败提供内联重试，成功后才展示服务端确认的正负状态。没有给整个 verification store 增加第二套状态机或全局重试框架。 |
| 43 | 确认，已完成本地修复与回归验证 | P2 | 每请求 blacklist 查询现在用保留 request values、忽略客户端取消的 50 ms bounded detached context。进一步复核发现同一全局 breaker 还被 revoke 写入、refresh reservation/release 共用，因此 5 个 Redis error 分支统一分类：`context.Canceled` 调用 `RecordNeutral`，不增加失败数但释放 half-open probe；内部 deadline exceeded 与真实 Redis 错误仍计 failure，所有错误路径仍 fail-closed。永久回归覆盖 closed/half-open neutral、直接 caller cancellation、deadline、真实 backend error 和 canceled Gin request。没有无遥测依据地把 50 ms 改成 200 ms，也没有把 30 秒 open window 猜成 3 秒。 |
| 44 | 确认，已完成本地修复与回归验证 | P1 | Bearer introspection 现在发送 `token_type_hint=access_token`，并在 active 后对同一原始 JWT 强制校验 Casdoor `tokenType == "access-token"`；refresh、claim 缺失、opaque 和 malformed token 均 fail-closed。用户认证还要求已登记应用和非空 `sub`。没有信任 response 的通用 `token_type=Bearer`，也没有为每个 Bearer 请求再建一套 JWKS 验签。 |
| 45 | 确认，已完成本地修复与回归验证；影响面比报告列举更广 | P2 | 实际有 5 条 admission `:token` 路由；raw path 原先不只进入 RequestLogger/Recovery，还进入两条请求体超限告警，共 4 类日志点。现在全部复用单一 `requestLogRoute`：匹配路由只记录 `c.FullPath()` 模板，404/405 固定记录 `unmatched`，绝不回退 raw URL。永久测试证明全局 middleware 在 handler 前后都能得到不含 credential 的模板，并覆盖 404/405；静态扫描确认 4 个旧 raw-path 日志点均消失。保留既有 query masking，没有枚举敏感 path 参数、遍历 Params 或做字符串替换。 |
| 51 | 部分确认，已完成本地修复与回归验证 | P2 | browser cookie 场景存在 logout 与在途 refresh 竞争，原实现会仅凭历史 attribution 误判 reuse 并撤销整个 family；native 常在更早的 tracked-session 检查失败。现在加载 referenced session，只有 session 存在、当前 refresh hash 非空且与提交 token hash 不同时才判定真 reuse；session 缺失或 hash 相同只返回 revoked，不撤销其他设备、不增加 reuse metric、也不记录虚假 reuse 审计。测试覆盖成功轮换、logout 已完成、blacklist→delete 竞争窗口和无 attribution 四种状态。没有新增 revoke reason schema。 |
| 57 | 确认，已完成本地修复与回归验证；收窄泄漏叙事与修复范围 | P2 | course/latest/search/batch 四类 public read 原先共用只接收 flat capability 的 access facts，把合法 scoped `admin:reviews:manage` 错当平台级 full-content 标志，导致范围外已发布正文由 preview 升为 full。现在 Handler 同时传入完整与 global capability 集合：普通学生的全文/发布/本人编辑删除能力仍读完整集合，只有 global `admin:reviews:manage` 能推出 `CanManageReviews/CanViewFull`。永久回归覆盖 school scope、section scope 和 global grant，并验证实际正文裁剪。public SQL 仍只读 `published`，没有 hidden-row 泄漏证据；`Review` 也没有报告声称的 `SchoolID`，因此未扩 DTO/SQL/scan/cache 去实现新的“scope 内公共全文”产品语义。 |
| 58 | 部分确认 | P3 | `admin:reports:manage` 未被直接使用，moderation route 仍按 role gate；但当前 raw-role 集合与 capability catalog 等价，下游 scope 也仍检查，所以尚无可利用的授权漂移。可逐步改用已有非全局 capability 并保留 scope，不需要新建 content-edit capability 或一次性重写全部 RBAC。 |
| 66 | 确认 | P3 | 本地 alg pre-check 接受 ES256，而实际 go-oidc verifier 默认只接受 RS256，配置/行为不一致。显式把本地允许集收窄为 RS256，或明确配置 verifier 支持经决策允许的 ES256；不要无界信任 discovery 返回的任意算法。 |
| 68 | 确认 | P3 | `ReadTuples` 只读取第一页，忽略 continuation token，后续 `WriteMissingTuples` 可能重写已存在 tuple 并失败。循环分页即可；改成对每条 tuple 单独 `hasTuple` 会把问题变成 N 次网络调用。 |
| 69 | 确认 | P3 | Native SSO 丢失登录前 redirect，成功后固定跳 profile。以 best-effort 方式保存经校验的内部 redirect，callback 消费后清除并保留安全 fallback。 |
| 70 | 部分确认 | P3 | 通用列表 payload 不含 ownership，相关控件无法可靠显示；但 “My Reviews” 自有页面可用，原文“整个模块全死”过度。通过 OpenAPI 真源增加 `isOwner`，服务端用 optional-auth context 计算；是否禁止 self-report 是独立产品决策。 |
| 71 | 确认 | P3 | inline edit 只检查非空，与 API 10–5000 字符规则不一致。共享常量/校验并设置 maxlength，避免提交后才收到通用 400。 |
| 72 | 确认 | P3 | admission login/signup/re-login 没有 single-flight，连续点击可轮换 OIDC state。组件内 busy guard 和禁用按钮足够；全局 OAuth 调度器可后置。 |
| 73 | 确认 | P3 | resend 忽略 `cooldownSeconds` 且直接展示英文错误。接入倒计时和已有本地化错误映射，不需要重写 OTP 状态机。 |
| 74 | 确认 | P3 | QQ bind 每 3 秒无限 polling，不考虑 token expiry 或 document visibility。增加 deadline、隐藏页暂停和终态停止。 |
| 75 | 确认 | P3 | `sessionStorage` 不可用会阻断登录，尽管受支持部署中的后端 state/cookie 才是权威。前端副本改为 best-effort，同时保留 SPA callback 的校验逻辑。 |
| 76 | 确认 | P3 | 3 个 public handoff route 继承 OpenAPI 全局鉴权，与运行时匿名访问矛盾。给 operation 显式 `security: []` 后按规范 regenerate；已有全局/IP/body/single-use 控制，新增每 token limiter 可另立加固项。 |
| 78 | 确认 | P3 | authorization 文档漏 `freshman_provisional`、引用错误映射文件，也没有解释 `section_reviewer` 当前无能力。修正文档即可；“typical capabilities” 本来就不是机器可比对的完整 catalog，不应为此强建全量文档 parser。 |

这 39 个标签经重新分级后为 **P1 × 7、P2 × 17、P3 × 15**。其中 9 条是部分确认，
意味着应处理较窄根因，不能直接采用原报告中更宽的故障叙事或重构方案。

#### #11 scoped role provisioning 的最终复核与决策边界（2026-07-31）

**结论：缺口真实，但 Claude 的标题和修复范围都过宽。**当前没有受支持的
`school#admin`、`section#section_admin`、`section#section_moderator` 人员授权写路径；
因此只在 Casdoor 给用户分配同名扁平角色不会产生任何 scoped capability。该用户会在后台
Entry capability gate 被 403 拒绝，资源级 OpenFGA check 也会拒绝，不会因为 scope 为空而
获得全局权限。OpenFGA 运维人员仍能直接写入合法 tuple 并让角色生效，所以“所有 scoped
role 物理上都无法工作”不准确；真正缺少的是**受支持、可审计、可撤销、可恢复的生命周期**。

本轮对当前工作树重新做了以下交叉验证：

- `RoleScopeResolver` 只读取 `school#effective_admin`、`section#section_admin` 和
  `section#section_moderator`，并先把 Casdoor subject 映射为内部 `users.id`；tuple 的 user
  不能使用 Casdoor subject。
- 非测试写路径只覆盖 ecosystem/school/section 结构关系、资源 owner/author/school 关系、
  profile/admission 投影和平台 `super_admin`；`fga-setup` 也只写 school parent 与每校一个
  review-moderation synthetic section 的结构 tuple。Casdoor bootstrap 只创建扁平角色目录。
- 永久测试覆盖：没有 tuple 时 resolver 不产生 scope，非法 section grant 被忽略并计数，
  OpenFGA 查询错误向上传播并保持 fail-closed。原 X-1 的“空 scope 变全局读”攻击链已被
  admin Entry 前置 capability gate 截断。
- 当前实现只接受 `ReviewModerationSectionID` codec 生成的 synthetic section；#11 不能借机
  扩成任意业务 section 管理平台。`section_reviewer` 当前既不进入 resolver，也没有有效
  capability，仍按 #78 的 P3 文档/产品语义项处理，不混入本 P1。
- 已提交的
  [`iam-implementation-guardrails.md`](docs/design/iam-implementation-guardrails.md)
  明确规定“当前仓库尚未确定 scoped role 的权威 provisioning 来源”，且禁止读路径猜测并
  自动删除 tuple。工作区尚未提交的 `iam-architecture.md` / ADR 草稿倾向 OpenFGA 关系权威、
  受 step-up MFA 保护的管理面和“DB/OpenFGA 可审计事实”，但没有定案期望状态存储、
  Casdoor 扁平角色协同、reconcile 或灾后重建；本轮未修改、未提交这些用户草稿，也没有把
  它们冒充当前权威来源。
- Claude 的原方案第 4 点在 `AUDIT-REPORT.md` 中以
  `Short-term, low-risk mi …` 截断，不能作为完整实施说明；其前三点又同时要求 migration、
  outbox、API、CLI、MFA 和 audit，超出了“先补一条可用写路径”的最小修复。

需要 owner 选择的是**期望授权事实的真源**，不是 OpenFGA 是否参与最终授权：

| 方案 | 最小实现 | 收益与代价 | Codex 建议 |
|------|----------|------------|------------|
| A. StuHelper DB 为期望状态真源 | 受限的 grant/list/revoke Service + Repository；同一事务写 grant、审计与既有 outbox，worker 投影精确 OpenFGA tuple；Casdoor 不承载业务 role | 可审计、可重试、可从 DB 重建 OpenFGA，最符合多操作员和灾后恢复；需要新 migration、OpenAPI、事务语义及投影 | **owner 已选择**。按 ADR-0008 全面实施，不建设任意 tuple API 或第二套作业框架 |
| B. OpenFGA tuple 本身为真源 | 受 step-up MFA 和 `user:system:update` 保护的窄 grant/list/revoke API，直接写固定 relation/object 并落 `audit_events`；Casdoor role 仍需定义协同规则 | 改动较小、立即可用；但 DB 不能重建授权，备份、漂移、角色 claim 与 tuple 的部分成功语义必须另行定案 | 仅当 owner 明确接受 OpenFGA store/backup 为唯一期望状态时采用；不得暴露自由填写 relation/object 的通用写接口 |
| C. 受保护的运维清单为真源 | 受代码评审和最小权限保护的外部 desired-state 清单，窄 reconcile 命令只计算/应用允许的三类 relation，并生成审计证据 | 最小应用面，适合少量 bootstrap；身份 ID、清单保存位置、撤权时效和生产审批依赖运维治理 | 可作为小规模过渡方案；不能把含用户标识的清单随意提交到公开仓库，也不能把 shell 历史当持久审计 |
| D. 从 Casdoor role 名或 metadata 推导 scope | 在身份层编码 school/section | 看似少一个数据源，但会把业务授权事实塞回 IDP，破坏扁平 role 与 Authorization Service 边界 | **拒绝** |

无论选择 A、B 还是 C，实施时都必须保持以下最小验收边界：

1. 只允许 `school_admin`、`section_admin`、`section_moderator` 与受支持的 school /
   review-moderation section 组合；user 统一使用内部 `users.id`，不提供任意 tuple 写入能力。
2. grant/revoke 幂等，撤权失败 fail-closed；OpenFGA 不可用不能返回授权已生效。Casdoor
   业务 role claim 必须证明不影响授权；grant revision、撤权栅栏及部分失败恢复必须有明确测试。
3. 操作需要全局 `user:system:update`、现有 step-up MFA chain，并记录 actor、target、role、
   scope、reason、outcome；不能依赖客户端隐藏按钮。
4. 永久负向测试至少覆盖：无 scope 为零 capability、跨 school/section 拒绝、非法 ID 拒绝、
   重复 grant/revoke、OpenFGA/Casdoor/audit/outbox 失败、最后一条与非最后一条 grant 撤销。
5. 使用真实 PostgreSQL 与临时 OpenFGA store 做 grant → login/refresh → resolve/check →
   revoke → deny 闭环；发布后再以受控生产账号验收，不能用本地测试声称生产授权已收敛。

**Owner 决策（2026-07-31）**：选择 A 的企业级长期形态，并进一步取消 Casdoor 业务角色
投影。PostgreSQL 授权账本是唯一管理真源，OpenFGA 是运行时关系判定面，Casdoor 只负责
认证与登录层 MFA；后台入口、Capability 和 scope 全部从 DB-derived access snapshot 派生。
ADR-0008 固定了 grant/revoke 状态机、撤权栅栏、revision fencing、最后一名 super_admin
保护、迁移和回滚边界。#11 从 **P1 decision-blocked** 转为 **P1 implementation-in-progress**。

**实施阶段 1：DB 授权账本与投影状态机（2026-07-31）**

- 新增 `000020_authorization_grants` migration：固定五类管理员 role/scope 组合、DB 外键与
  check、`desired_state` / `projection_status`、单调 revision、active/projection 索引和
  `NULLS NOT DISTINCT` 唯一约束；没有创建任意 relation/object 存储。
- 新增 `modules/authorization` Repository/Service：grant/revoke 在同一 PostgreSQL 事务内
  写账本、`audit_events` 与既有 `domain_event_outbox`；重复 applied grant/revoke 幂等，
  pending/failed 重试递增 revision；最后一名 applied super_admin 由事务级 advisory lock
  和同事务计数保护。
- 新增专用 `iam_authorization_grant_projection` stream，worker 只映射固定 tuple：
  ecosystem `super_admin`、school `admin`、三类 section relation。授予必须
  write + higher-consistency exact read 后才 applied；撤销在 DB desired state 提交后已被
  access snapshot 排除，再执行 `on_missing=ignore` delete + exact absence verify。
- DB access snapshot 当前已能从 applied grant、`user_profiles` 与有效 freshman credential
  派生角色和 scope；尚未接入认证 middleware，因此本阶段不声称运行时已经切换。
- 真实 PostgreSQL 18 migration/事务回归覆盖 grant → pending deny → projection → allow →
  revoke fence deny → tuple delete、并发版本覆盖、最后一名 super_admin、重复操作以及故意
  删除 `audit_events` 后账本/outbox 整体回滚；模块与 outbox race 测试通过。

### 对 Claude 第二轮 14 个“驳回标签”的反向复核

Claude 的两条 `/admin/stats` 记录是同一位置、同一 middleware 顺序和同一影响，合并为一个
唯一根因。以下按 13 个唯一根因给出最终结论：

| 原驳回发现简称 | Codex 最终结论 | 级别 | 必要性与建议 |
|----------------|-----------------|------|----------------|
| A11yButton 键盘触发两次 | 维持证伪 | 不立项 | H5 probe 与 Playwright 均只触发一次 action。最多增加 `defaultPrevented`/key repeat 防御，不能按“控件无法打开”修复。 |
| mp-weixin 无可达 SSO | **重新确认，已按“明确不支持”完成修复** | P2 | 修复前再次真实运行 `build:mp-weixin`：退出 0，但 `dist/build/mp-weixin` 只有 `index.html`、单个 JS/CSS 与静态图标，没有 `app.json`/WXML/WXSS；依赖中也没有 `@dcloudio/uni-mp-weixin`，login 只有 `plus`/`window` 分支。根 `AGENTS.md` 将 UniAppX 定义为实验性跨端，故选择不虚构平台支持：删除两级微信命令、manifest 声明与 README 支持说法，并接入静态目标契约。将来只有同时补真实 compiler、认证、CI 和开发者工具/真机验收后才能重新声明支持。 |
| `/admin/stats` 跳过 MFA（两个重复标签） | **部分重新确认** | P3 | 跳过 5 分钟 step-up freshness 是有提交和测试锁定的刻意 carve-out，不能直接把整个 group middleware 移上去；但该独立 route 也绕过基础 MFA context/enrollment，而 IAM 文档要求 `super_admin` 使用 MFA。应明确记录例外，或只补基础 MFA proof/enrollment gate、不要求 freshness。 |
| academics 使用 `UserSchool*` capability | 维持证伪 | 不立项 | 路由调用 `RequireGlobalCapability`，scoped school admin 不会通过；当前与 `UserSystem*` 的主体集合相同，仅是命名债务。 |
| `/internal/sms/send` 无 `/api/v1` limiter | 维持证伪 | 不按公网漏洞立项 | shipped ingress 不暴露 `/internal`，服务绑定 loopback 且需要 internal key。可在部署加固中验证私网 ingress 和 Casdoor 侧预算，但不应把内部 route 机械搬进 public API/global limiter。 |
| 非法 FGA section 让该用户请求 503 | **部分重新确认，已完成本地修复、告警接线与回归验证** | P2 | 非法/陈旧的手工 tuple 原先会毒化持有对应 scoped role 的那个用户，不是“所有用户”。现在逐项过滤无法按 review-moderation codec 解析的 section：无效 grant 不生成 capability，合法 grant 继续工作，只有无效 grant 时得到零 scoped grant；真正的 OpenFGA 查询错误仍返回 503。每项无效 grant 增加无 label 的 `iam_invalid_role_scope_total` 并记录内部 FGA user、固定 role 与 section ID warning；Prometheus 告警要求人工 reconcile。运行时没有在尚未确定 scoped provisioning 权威来源时擅自删除 tuple。 |
| section school 从 synthetic ID 解析 | 维持证伪 | 不立项 | 应用从同一 schoolID 同时生成 section ID 与 tuple，原报告所述不一致无法由当前写路径产生。可在 provisioning/reconcile 时做完整性检查，不应在每次请求额外读 tuple。 |
| GetAdminStats 无 school scope/cache key | 维持证伪 | 不立项 | route 只允许 global dashboard grant，全局聚合和 scope-free cache 与授权语义一致。 |
| RatingBar 无无障碍文本 | **部分重新确认，纠正位置并已完成 live surface 修复/浏览器验证** | P2 | 被引用的 `RatingBar/DimensionBars` 是 dead code，但真实 `CourseDetailPage` 中的可视化 bars 原先同样只靠宽度/颜色传递信息。现在每个可达维度行是具名 `role=img`，名称由“本地化维度 + 既有五级 face 定性文案”组成，纯视觉 bar/dot 对辅助技术隐藏；没有把原始 `avgRating` 写入 aria/title/隐藏文本。桌面/移动 Chromium 对 4.6 的样例只读出“教学质量：超赞”，页面仍无精确数值。未修改 dead component，也未新增第二套阈值。 |
| OtpCodeInput 非数字残留 | 维持证伪 | 不立项 | Vue 会强制同步 input `value`，且组件当前无生产使用。可补测试，不按运行时故障改。 |
| admission hostname 不可配置 | 维持证伪 | 不立项 | 固定 join domain 是文档与 smoke test 锁定的安全/部署契约；改成任意 env host 反而会破坏负向隔离。 |
| “继续入群认证”未强制 reauth | 维持证伪 | 不立项 | wrong-account 会被服务端拒绝，页面已有 switch-account；每次强制登录会回归正常续办。可增加更显眼的切换账号入口。 |
| Developer Connect issuer fallback | **部分重新确认** | P3 | 支持的正式 build 都注入 issuer，故没有已证实生产故障；但 `configured -> current web origin -> default SSO` 的 fallback 顺序在 Web/SSO 分域模型中确实错误，并让安全默认值不可达。改为 configured issuer、已知 SSO 默认值或明确配置错误，不需要动态 discovery framework。 |

这 14 个标签对应 13 个唯一根因：**1 个确认、4 个部分确认、8 个维持证伪**。因此
`AUDIT-REPORT.md` 中“以下条目不应修改”的总括语不再准确。

#### Codex 对 mp-weixin 假绿的最终复核与实施记录（2026-07-31）

**结论与产品边界**

- 旧 `pnpm --filter @stuhelper/uniappx build:mp-weixin` 再次真实退出 0，并提示导入微信开发者
  工具；但目标目录只有 `index.html`、一个 JS、一个 CSS 和 H5 静态资源，没有小程序入口
  `app.json` 或任何 WXML/WXSS。该“成功”不能被解释为微信产物。
- `clients/uniappx` 没有 `@dcloudio/uni-mp-weixin` 依赖；登录页只实现浏览器
  `window.location` 与原生 `plus.runtime.openURL`，没有小程序登录/授权码/回调链路。旧
  manifest appid 占位、README 和脚本只是宣称，不构成实现。
- 根 `AGENTS.md` 明确把 UniAppX 定义为“实验性跨端”，而 H5 已有正式 build、产物契约、
  类型/单元和双视口浏览器回归。基于现有证据，最小、诚实的产品选择是“当前不支持
  mp-weixin”，而不是为了关闭报告而引入 compiler、微信认证和发布体系。

**最小实现与防回归**

1. 删除 monorepo `build:uni:mp`、包内 `dev:mp-weixin` / `build:mp-weixin`，以及 manifest
   的 `mp-weixin` appid 声明；调用旧两级命令现在均明确以 missing-script 非零退出，不再假绿。
2. README 将 H5 标为当前唯一有正式构建与回归的目标，并列出未来重新声明微信支持的最小
   门槛：真实 compiler、`app.json`/WXML/WXSS 产物契约、小程序认证、CI、开发者工具与真机
   验收。原生 App 代码仍是实验性源代码，不借本项扩展或宣称发布支持。
3. 新增零依赖 `check-uniappx-platform-contract.mjs`：要求 H5 两级 build 命令存在，同时拒绝
   微信两级命令与 manifest 声明；repository-policy job 永久执行，现有 CI wiring contract
   锁定接线。未来若正式支持微信，必须在一个受审变更中更新实现、契约和支持声明。

**验证**

- 修复前真实命令以 0 退出且输出仅为 H5 结构；修复后根/包两条旧命令均非零，
  supported-target contract 通过。
- H5 正式 monorepo build 和 8/8 tabBar source/output 契约通过；UniAppX type-check、
  8 files / 58 unit 与已跟踪 `surface.spec.ts` 桌面/移动 72/72 通过。未跟踪的临时
  `probe.spec.ts` 属于用户工作树，未纳入提交或以其替代受控回归。Actionlint、
  CI/drift 定向与全部 infra contracts、文档卫生均通过。
- 微信开发者工具/真机不再是当前发布验收缺口，因为仓库不再声称支持微信；将来重新声明时，
  这些验证是硬门槛，H5 绿灯不能替代。

### 补齐 Claude 未完成验证的 3 项

`AUDIT-REPORT.md` 只记录了“3 条未完成验证”，没有保留标题。Codex 从本轮审计执行记录中
恢复候选项，并回到当前代码逐项验证：

| 编号 | 候选问题 | Codex 结论 | 级别 | 最小处置与边界 |
|------|----------|------------|------|----------------|
| U-1 | `/admin/academics` 缺少 privileged MFA/admin-entry | 部分确认，MFA 缺口已完成本地修复与回归验证 | P1 | `/admin/academics` 三个 operation 已复用现有 production/prod-parity MFA context + privileged MFA chain，并保留精确 global capability gate；development 的既有 no-op 语义不变。独立复核还发现共享 step-up middleware 实际返回 428、而 OpenAPI/错误码/管理端约定 412；现已在共享 RBAC 边界统一为 412，并更新所有真实 step-up 断言。缺少通用 admin-entry 没有额外影响，因此没有增加重复 entry gate或建设第二套 MFA。 |
| U-2 | 每 5 分钟丢弃 JWKS `RemoteKeySet` 缓存 | 确认，已完成代码、守卫文档与回归修复；用户工作中的架构稿旧段待收敛 | P2 | 固定 go-oidc v3.17.0 的 `RemoteKeySet` 本身就是 long-lived cache：已知 `kid` 本地验签，未知 `kid` 才立即回源，回源失败不清旧 cache。外层 5 分钟 wrapper 反而销毁这些 key，令 Casdoor 短暂不可用时已知 token 也 503。现在 verifier 进程内只复用一个 `RemoteKeySet`，薄包装仅保留 remote-fetch → provider-unavailable 分类；未知 `kid` 失败仍 fail-closed，已知 key 不做 stale claim 降级，Cookie 路径的 session hash/blacklist/iss/aud/exp 均不变。没有把 TTL 猜成另一个数字或自建缓存。`docs/design/iam-implementation-guardrails.md` 与 `auth-and-session.md` 已固定此语义；当前用户暂存重命名且继续编辑的 `docs/design/iam-architecture.md` 仍含旧“5 分钟重建”段，本次为避免吞入用户改动未触碰，合并该稿前必须去掉冲突。 |
| U-3 | StudentVerificationPanel 学籍邮箱流程硬编码中文 | 确认 | P3 | 标签、placeholder、状态和错误均有硬编码中文，英文 locale 会混合显示。增加 scoped 中英文 key，并按稳定状态/错误码映射；不需要引入 CMS 或新 i18n 引擎。 |

### 深挖复核新增的独立问题

下列问题不是 Claude 第二轮 56 个标签之一，因此**不回写也不改变**上方“55 个唯一根因、
47 个需要处置”的原报告复核统计；它是在实现级交叉验证既有条目时发现的 Codex 补充项。

| 编号 | Codex 结论 | 级别 | 真实机制、必要修复与边界 |
|------|------------|------|--------------------------|
| N-1 | 确认，已完成本地修复、固定版本源码/运行实例与真实 Redis 交叉验证 | P1 | 当前固定镜像证据对应 Casdoor v3.125.0，而旧代码注释中的 3.31.1 已过时；结论本身仍真实。运行中的固定 digest discovery 只有 `/api/logout`，实测旧 `token=<refresh>&token_type_hint=refresh_token` 在无浏览器 session 时返回 `status=ok` no-op，非法 `id_token_hint` 则以 HTTP 200 返回 `status=error`。现在 session 分别加密保存 provider access/refresh token，refresh Lua 原子轮换两份密文；仅对与 discovery issuer 同源且路径精确为 `/api/logout` 的 endpoint 发送 `id_token_hint=<provider access>`，并要求 2xx + JSON `status=ok`，真正的 `revocation_endpoint` 仍独立走 RFC 7009。正常 Casdoor refresh 已删除旧 row，不再错误 logout 已消失的旧 family；refresh 阶段未提交的新 family 会补偿撤销。旧 session 的当前设备 logout 复用已匹配 hash 的 access，logout-all 先完成本地撤销，再用加密 refresh rotation 后立即撤销替代 family；两种 provider 凭据都缺失会明确失败，仅精确 `invalid_grant` 可视为已失效，其他错误 fail-closed。本地 contract 忠实模拟 `expires_in=0` 共享 row，证明 access introspection 与 refresh grant 同时失效；一次性真实 Redis 8.8.1 验证两份密文原子轮换和 logout 取最新 family，容器已删除。没有引入通用 IdP logout 框架、provider token 表、迁移任务或每请求额外查询；生产受控账号端到端仍待发布验收。 |
| N-2 | 确认，已完成本地修复与长套件交叉验证 | P3 | Admin 用户下拉框在 click 模式下仍初始化 hover watcher；watcher 虽立即暂停，但初始化排队的 500 ms leave timer 未被取消，用户较早打开菜单时会被旧 timer 关闭。首次全量 Playwright 的移动退出用例因此超时，隔离重复 3 次也复现 1 次。最小修复只让 `useHoverToggle.disable()` 清理自身 pending timers；确定性 fake-timer 回归、移动连续 10 次、Admin 154 unit、类型/lint/build 与双视口 214 E2E 全部通过。未改 logout/OIDC、全局动画、菜单组件架构或测试超时。 |

### 对新版报告 I-1、I-2、I-3 的复核

| 报告项 | Codex 结论 |
|--------|------------|
| I-1 无 scope 的 `school_admin` 全量可见 | 不是新问题，与本文件 X-1 相同，维持证伪。`ExpandRoleGrants` 对无 scope 角色不给 capability，admin Entry 在 Handler 前返回 403。#11 的真实问题是“缺少受支持的 scoped tuple provisioning，角色 fail-closed 不可用”；#57 的真实问题是“已有合法 scoped grant 被 public content path 错当全局 full-content grant”。两者都不能证明 I-1 的默认全局泄漏。 |
| I-2 21 个 env 变量缺模板 | 不是新问题，与 X-2 相同，维持“部分确认”并已完成分类修复。实施前 AST 口径为 config runtime key 184 个、模板差集 17 个；本文件早先的 181 是计数错误，现已纠正。13 个 operator-facing 键进入两个模板，死配置 `LOG_SERVICE_VERSION` 删除，3 个兼容覆盖项进入带理由 allowlist；另 4 个原报告键分别属于 bootstrap/FGA 工具、`GIN_MODE` 和 Redis integration test，不进入运行模板。门禁不要求所有 `getenv` 与两个模板严格相等。 |
| I-3 契约测试锁死 deploy bundle 缺陷 | 不是新问题，是旧 P1-4 的测试反模式补充；相关修复和本地验证已在上方进度表记录。新版报告第 7 节刻意不判断当前实现状态，所以不能据其原始正文把已修复项重新标成“未处理”。 |

### 本次增量完整性核对

在写入本次新增复核前再次做了文件级核验：

- 当前工作树中的 `AUDIT-REPORT.md` 实际为 **4343 行**，SHA-256 为
  `e02f0b318f51ec306c586537732b4912756d85a51a0c140e390a772662679079`；这比用户转述的
  4337 行多 6 行，当前文件末尾已包含 I-1、I-2、I-3 和第 7 节实施说明。
- 写入前的 `AUDIT-FINDINGS.md` 为 **3198 行**，SHA-256 为
  `a26a68759306f03400f4928efbe337df7d44049823fa6b67cb11ada697cc6964`，与当时
  `HEAD=f446b969` 逐字节一致（`git diff --quiet -- AUDIT-FINDINGS.md` 返回 0）。
- 第一轮复核、第二轮 56 个标签、I-1/I-2/I-3、深挖新增 N-1、实施顺序、测试边界和修复进度
  均仍在本文件中；Claude 的新版报告是独立未跟踪文件，没有覆盖或删除 Codex 既有内容。

新版报告的 #47/#48/#49 也分别重列了旧 P2-13/P2-14/P2-15。当前 `HEAD` 已包含
P2-13/P2-14 的合并修复和回归记录，P2-15 是同一逐行上下文查询根因的重复标签；保留报告条目
用于历史追溯，不应重新创建三个待办或覆盖本文件已有实施状态。

### 建议实施顺序与过度设计边界

#### 第一批：授权、令牌和撤权一致性

1. #3 + #4：Koishi Console 全局/群 scope 和跨群数据隔离。
2. #44：access/refresh token 类型边界与非空 subject。
3. #17：`super_admin` tuple 的权威 reconcile（已完成本地修复）；#11 的代码、测试、已提交
   guardrail 与未提交 IAM 草稿均已复查，当前唯一缺口是 owner 选择 scoped grant 真源及
   Casdoor membership ownership，详见上方决策矩阵。在此之前保持 P1 decision-blocked，
   不由审计修复擅自建设授权平台。
4. #18：**已完成本地修复**；按 token 自身 `exp` 计算 blacklist，并以受约束的 Redis
   PTTL 完成旧 session 滚动升级边界。
5. N-1：**已完成本地修复**；在 #18 之后以独立改动实现并验证 Casdoor
   logout/token-family 撤销契约。
6. U-1：academics import 补复用现有 MFA chain。
7. #9：恢复 `userVote` 契约，避免用户动作语义反转。

不要把这些问题合成一个“重写 IAM/Casdoor/OpenFGA”的大型项目。每项都应先固定权威输入、
fail-closed 语义和撤权测试，再做局部实现。

#### 第二批：隐私、可靠性和核心前进性

1. #45 path credential 日志脱敏（已完成本地修复），#51 refresh reuse 误判（已完成本地修复），#57 scoped grant 的 public-content 边界（已完成本地修复）。
2. #39 projection polling（已完成本地修复）、#43 breaker cancellation（已完成本地修复）、U-2 JWKS 缓存策略（已完成代码与守卫文档修复；用户架构稿旧段待收敛），以及非法 FGA section 的单 grant 隔离/告警（已完成本地修复）。
3. #5 的 H5 资产产物契约与相邻 mp-weixin 假绿均已完成：H5 保留真实产物门禁，
   mp-weixin 按当前实验性产品范围明确为不支持并由静态契约防止再次误宣称。
4. #6、#7、#8、#28、#30、#34、#35、#36、#37、#38、#42（均已完成本地修复）等会让用户状态错误、流程卡死、异常不可见、键盘流程受阻或操作结果不可信的问题。

#### 第三批：局部 UX、可访问性、契约和文档

#33、#40/#41、#58、#66、#68 至 #78，以及反向复核中重新进入待办的 issuer fallback。
live rating bar 已完成可达表面的最小修复。优先做一处根因、一组回归测试的窄修复；不先建设动画框架、全局事件总线、
通用 OAuth 调度器、动态 OIDC discovery 平台或文档全量 parser。

### 并行实现级深挖复核记录

“逐项复核”分为两层：上方 55 个唯一根因已经全部完成源码级确认/证伪与重新分级；下表是在此
基础上，针对优先队列继续做的实现级调用链、负向测试、第三方契约和修复冲突复核。没有列入
下表的条目不是“未审计”，而是尚未完成同等深度的实施前验证；后续每个条目在修改前仍须补齐
这一层，不能仅凭初始静态结论直接改代码。

| Workflow / 条目 | 深挖结论 | 实施决策与避免过度设计边界 |
|-----------------|----------|--------------------------------|
| WF01/WF19：#3 + #4 | 两项均为真实 P1，已修复；首次实现复核因残留全局运行态读取判 FAIL，补齐后最终 PASS | Core service 暴露既有 resolver；global settings 写入前 `assertGlobal`；页面先 scope filter 再统计/排序/limit；mutation 在副作用前断言 guild。最终把所有 bot-wide 字段收拢为仅 global scope 返回的 `globalRuntime`，scoped 分支不加载、不下发且 UI 不渲染。继续把过滤下推 Repository 只在真实规模/延迟需要时做；不搬迁整套 RBAC helper，不与 #30 队列 UX 混改。 |
| WF02：#44 | 真实 P1，已修复 | Casdoor active introspection 不能只看 `active` 或相信 hint；运行时已强制 3 段 JWT、`tokenType=access-token`、registered app 和非空 subject。没有增加每请求第二套 JWKS 验签。 |
| WF03：#17 | 真实 P1，已修复；协议假设经真实 OpenFGA 纠正 | `RolesClaimPresent` 只表示 claim 明确存在且结构有效，Web/native login 与 refresh 才把新 token 提升为 `RolesAuthoritative`；`/auth/me` 明确保持 false。精确 direct tuple 使用 higher-consistency Read，存在时通过 SDK `on_missing=ignore` 删除，避免并发 refresh 的 read/delete 竞态把已撤权状态误报为失败；实际删除后记录 `iam.role.revoke`。Claude 所引 `client_edge_test.go` 只证明空列表 no-op，并未证明缺失 tuple 默认幂等；真实 v1.18.1 验证确认必须显式使用 ignore。没有实现 #11 的 DB/outbox/API/CLI provisioning 平台。 |
| WF04：#9 | 真实 P1，已修复 | 所有 `readReviewPagePayload` consumer 都经过同一 parser；保留 `like/dislike` enum、缺失可选、非法 fail-closed 足够，不建设反射式 DTO 自动同步框架。 |
| WF05/WF10：U-1 | MFA 缺口真实；“缺通用 admin-entry”证伪；交叉审查从 FAIL 修到 PASS | 三条 `/admin/academics` route 已复用现有 admin MFA chain。反向复核发现共享 RBAC step-up 曾返回 428、与 OpenAPI/管理端 412 契约冲突，已中央对齐 412并扩展真实链路断言；非 MFA 的 Open Platform 428 未被机械替换。精确 global capability 已比 admin-entry 更严格，无需重复 gate。 |
| WF06：#18 + N-1 | 两项均为真实 P1，已分别修复；Redis、固定 Casdoor 源码/运行实例与 contract 交叉验证通过 | #18 已保存已验签 expiry、原子轮换 hash+expiry、逐 session 计算剩余 TTL，并修复 tracked logout 的二次短 TTL 覆盖。N-1 又将 provider access/refresh 密文纳入同一 Lua 轮换；仅对 issuer 同源的精确 `/api/logout` 使用 `id_token_hint` 并校验 JSON 业务状态，RFC 7009 端点保持独立。正常 refresh 不重复 revoke 旧 row，refresh 阶段未提交 family 才补偿；logout-all 先做本地撤销，legacy family 使用受约束的 refresh-rotate-logout bridge。没有把两项揉成 IAM 重写，也没有增加 provider token 数据库、后台迁移或每请求 session lookup。 |
| WF07：#51 | 较窄 P2 真实，已完成本地修复 | referenced session 存在、当前 refresh hash 非空且与 presented hash **不同**才是真 reuse；session 缺失或 hash 相同是已撤销/并发 logout，只返回 revoked，不触发 family revoke、metric 或 audit。永久回归覆盖 rotated、logout-complete、blacklist-before-delete 与 missing attribution；无需为此新增 revoke-reason schema。 |
| WF08：#57 | 真实 P2，已完成本地修复；原报告的 hidden-row 与 `Review.SchoolID` 证据不成立 | Handler 将完整与 global capability 分开传给统一 access facts；public full-content 的管理 entitlement 只接受 global grant，普通学生能力保持原语义。school/section/global 三类回归通过；未改 capability producer、管理路由、Review DTO/OpenAPI/SQL，也未实现新的“scope 内 public full”产品语义。 |
| WF09：#45 | 真实 P2，已完成本地修复；报告漏掉 1 条 token route 和 2 类 body-limit 日志 | RequestLogger、Recovery 和两类 body-limit 告警统一走 `requestLogRoute`：匹配路由用 `FullPath()`，未匹配 404/405 固定 `unmatched`；永久回归验证 handler 前后模板和负向路由，静态扫描无旧 raw-path logger。保留 query masking，未维护敏感参数黑名单或 token 字符串替换器。 |
| WF20：#39 | 真实 P2，已完成本地修复；原报告把 403 与 401 一并视为 hard auth failure 的建议过宽 | `fetchUser` 只在 `/auth/me` 返回 401 时清身份；403/5xx/网络/超时均保留已有用户但继续向调用方抛错。投影轮询只捕获 refresh request 失败，非 401 在原五次预算内继续，Abort/401 立即停止；耗尽后页面保留 `projectionPending` 与手动重试。服务端 capability/OpenFGA 仍是授权真值，不增加轮询次数、全局 retry abstraction 或客户端授权降级。 |
| WF21：U-2 | 真实 P2，已完成实现与回归；用户工作中的架构稿仍有旧 TTL 文案 | 选择仓库已声明的“已知 key 离线验证、未知 kid 回源失败 503”策略，直接复用固定 go-oidc 的 long-lived `RemoteKeySet`；薄包装只做错误分类。测试证明初次加载只请求一次，provider 下线后已知 key 不回源、未知 key 返回 provider unavailable，失败后已知 key 仍可验签。紧急移除已泄漏 known key 通过 provider 撤 key、撤 session 与滚动重启 verifier 明确处置，不用任意 TTL 猜窗口。 |
| WF11：#5 + mp-weixin 反向项 | H5 缺图与 mp build 假绿均为真实 P2，已分别完成最小修复 | 8 个原 blob 移到 `src/static`，H5 build 强制执行声明驱动的源/产物契约。mp 旧命令再次复现退出 0、仅输出 H5 结构；当前没有 compiler/auth/真机证据，故删除虚假命令与 manifest 声明、README 明确不支持，并由 CI 静态契约固定。没有增加资源副本，也没有为实验性客户端擅自建设微信发布体系。 |
| WF12：#6 | 真实 P2，已完成本地修复与回归；“create 失败也删除”证伪 | 只恢复匹配或未绑定课程草稿，未绑定不恢复 teacher；显式记录当前页是否成功恢复/保存服务端单槽，create 成功后才有条件清理。外课程草稿和不确定保存结果均保守保留。产品文档同步真实无 course path 单槽契约；不改 DB/OpenAPI 为多草稿，也不在本项解决跨设备 CAS/If-Match。 |
| WF13：#7 | 真实 P2，已完成本地修复与回归 | Profile Completion 对 username/email/avatar 优先使用 `/auth/me` 已给出的绝对 `accountSettingsUrl`，phone/identity/student/school 保留后端本地 route，缺失外部 URL 时回退声明 action。单元和双视口浏览器回归覆盖字段归属与既有 Continue 链路。未为静态 catalog 注入整套 config/service、复制 issuer fallback 或把本地只读资料页改造成第二个身份提供方编辑器。 |
| WF14：#8 | 部分确认 P2，已完成本地修复与回归；问题是有限的媒体类型兼容，不是通用 MIME 绕过 | 只在资源 POST 的内容解码边界做窄映射并返回 effective MIME；ZIP/Office/OLE/text refinement 精确枚举且保留 sniff，OLE/JSON 另做内容验证。负向测试固定任意 `text/*`、`vnd.*`、`*+zip`、无魔数 legacy 与无效 JSON 仍拒绝。没有把策略复制到身份/准入图片路径，也没有增加易漂移的前端 allowlist。 |
| WF15：#28 | 真实 P2，已完成本地修复与回归；“7 endpoint”已纠正为 7 个逻辑域、`6 + D + N` 次 mutation | 顺序 orchestrator 区分 confirmed/unconfirmed/not-run；成功 slice 与关键词子 mutation 逐步推进 baseline，失败后可确认 reload，missing delete 幂等且已有规则继续做 scope 检查。客户端和运行时共用安全正则校验。没有引入 rollback、2PC、saga 或无界并发。 |
| WF16：#30 | 部分确认 P2，已完成本地修复与回归；100 条静默窗口真实，但全系统不可处理和等待老化叙事被证伪 | API 和页面已显示同一授权 scope 下 `shown/total/limit/truncated`，以 `(deadlineAt,id)` 稳定选取窗口并链接处置中心；103 条同 deadline 回归固定 100 条边界。处置中心、群内命令、backend-sync/time-code 仍覆盖窗口外记录；没有在缺乏规模/SLO 证据时建设 Repository cursor pagination/search。 |
| WF17：#34 | 真实 P2，已完成最小修复；Chromium 与永久生命周期回归均通过 | 在 `createParticles()` 丢弃当前 targets 前 kill；保留现有 rAF 合并与 unmount 清理。永久测试证明同帧两个 resize 只重建一次、旧 targets 在新 `gsap.to` 前被清理、unmount 清当前批次。无需 GSAP context 重构、坐标 clamp、Tween handle registry、ResizeObserver、Worker 或全局 animation manager。 |
| WF18：#35 | 真实但偏低端 P2，已完成局部确认、single-flight 与双视口 E2E 收口 | `ReviewCard` 首击只进入 inline `role=group` 确认态并把焦点移到取消；取消零请求并恢复焦点，确认才调用原 DELETE。`useReviewDelete` 在 await 前同步检查/置位，程序级重复调用也只有一个请求。累计回归同步了仍按旧单击语义编写的 E2E，并明确断言确认前零请求、确认后唯一请求；桌面/移动定向 2/2、Web 全量 363 passed / 1 designed skip。没有新增 restore schema/API、延时删除、undo 平台或全站 modal manager。 |
| WF22：#38 | 真实 P2，已完成最小修复与组件卸载交叉验证 | Toast 列表、timer map 与 ID 本来就是模块级状态，唯一错误是创建组件销毁时取消 timer 却保留列表项。删除该 dispose hook 后，timer 闭包仍安全调用既有 `remove`；永久 Node 回归用 `effectScope.stop()` 固定跨作用域自动关闭，Claude 的 jsdom mount/unmount 探针也转为通过。无需把函数整体提升重写成第二套 singleton/store，也不能在卸载时立即移除提示。 |
| WF23：#42 | 部分真实 P2，已完成最小页面状态修复与失败重试验证 | `/auth/me` 的账号/邮箱与三条 verification 状态请求不是同一事实来源；只保护 phone、QQ、实名、学籍和相关披露字段。页面本地状态在 ready 前隐藏未知负面结论，pending 显示轻量 loading，失败保留可靠字段并提供 single-flight 重试。无需修改共享 store 数据模型、添加全局 error bus，或在一次子请求失败后清空已有 store 投影。 |
| WF24：#36 | 真实 P2，已完成既有 telemetry 的最小扩展与隐私边界回归 | `ErrorBoundary` 返回 `false` 后异常不会进入 Vue 全局 handler，所以必须在边界本身调用 reporter；没有边界的组件异常继续由全局 handler 兜底。两处只传固定 `vue-error`，后端沿用现有有限 label counter，不接收为诊断而扩建的 stack/props 存储。reporter 只在既有 bootstrap 明确初始化后生效并吞掉 transport 自身异常；无需第二个 API、错误数据库、全局事件总线或第三方采集平台。 |
| WF25：#37 | 真实 P2，已完成 AppShell 级最小修复与真实键盘回归 | 按最新 Web Interface Guidelines 复核后，缺口限定为 shell 的 bypass block；skip link 使用原生 anchor 和 fragment，不用 click handler 模拟导航，`main` 只增加稳定 id 与 `tabindex=-1`。样式沿用现有 token、全局 focus-visible 和 reduced-motion，z-index 高于 sticky header。桌面/移动 Chromium 实测首个 Tab 可见、Enter 后 main 获得焦点；无需在各页面复制链接或构建 focus manager。 |
| WF26：反向复核 / 非法 FGA section | 部分真实 P2，已完成单 grant 隔离、可观测性与失败关闭回归 | `ListObjects` 成功返回的每个 section ID 独立经过既有 codec；解析失败项不再把 resolver 整体变成 dependency error，但也绝不进入 `orgScopedRoles`。混合列表保留合法 scope，纯无效列表展开为零 capability；OpenFGA transport/server error 在过滤前返回，继续映射 503。无效项用无 label counter 控制基数、结构化 warning 定位，并由告警要求人工 reconcile；由于 #11 的权威来源尚未决定，读路径不自动删除 tuple。 |
| WF27：反向复核 / live rating bar | 部分真实 P2，已完成可达表面的定性可访问名称与双视口回归 | Claude 引用的通用 RatingBar 是 dead code，不能靠修改它关闭问题；真实 CourseDetailPage 的维度条改为一个具名图像语义，复用 `normalizeRatingLevel` 和现有 `review.rating.face1..5`。helper 与 policy 测试固定“不含原始 4.6”，桌面/移动 Chromium 读取“教学质量：超赞”且页面找不到精确数值。没有改评分 API、产品显示策略、dead component 或新增 ARIA 数值进度条。 |
| WF28：X-2 / runtime env 模板差集 | 较窄 P2 真实，已完成 AST 复枚举、分类修复和 CI 接线 | 实施前 config 包有 184 个字面量运行时键、17 个未进入任一模板；Claude 的 187/21 和本文件早先的 181 都不是准确的 config 包计数。13 个 operator-facing 键进入两个模板，`LOG_SERVICE_VERSION` 删除后当前为 183 个；`AWS_CA_BUNDLE` 与两个 `LOG_*` fallback override 用显式理由保留在 3 项 allowlist。Go AST 测试遍历整个 config 包，要求字面量 key、模板/allowlist 覆盖并拒绝陈旧 allowlist；CI backend filter 同时覆盖两个模板。没有扫描或模板化全仓工具/测试变量，也没有建立一套新配置 schema/generator。 |
| WF29：P2-9 / Ansible bundle path | 真实 P2，已完成故障机理纠正、真实 controller 修复验证和 CI 门禁 | Ansible Core 2.20.2 源码与 cwd probe 都证明 localhost task 从 playbook basedir 执行，所以旧 `../../ops` 可找到脚本；旧任务真实失败在脚本内部 `git -C` 令相对 output 再换基准。干净 clone 先复现 `git archive` 无法打开输出，再以候选改动真实执行唯一 `deploy-bundle` tag 成功，产物含两个 env 模板。实现只使用 `playbook_dir` 绝对 argv、脚本默认输出、同源 copy src 和无用 facts 禁用；固定 requirements、core-compatible callback、syntax/bundle CI 与窄契约覆盖，不扫描其他 shell task。 |
| WF30：P2-18 / sensitive-word filter invalidation | 真实 P2，已完成并发失效边界和真实 DB 回归 | 预热后的五分钟 snapshot 原先不会感知管理端 mutation。现在成功 create/update/delete 只把本进程 snapshot 标记 stale，下一检查仍复用既有 detached/singleflight reload；失败 mutation 保留当前快照。额外用 refresh/invalidation mutex 防止在途旧刷新覆盖失效标记，但 DB 查询不持有 matcher lock。PostgreSQL 回归从空快照依次验证新增 block 立即命中、更新后旧词消失/新 warn 生效、删除后消失；closed pool 时返回 moderation unavailable。单副本不接 Redis，多副本要求留在安全文档。 |

### 本轮交叉验证与尚未验证边界

本轮并行代理均以只读复审为主，主代理对关键链路再次交叉检查。执行结果：

#### 2026-07-31 主对话累计完成性门禁

为避免把不同提交时点的局部绿灯误当成当前整体状态，主对话在所有 P0/P1/P2 处置完成后
重新执行了一轮跨域回归，并对文档中的提交引用做闭包检查：

- 提交闭包：从本文件提取 62 个 Conventional Commit subject，与当前 `git log` 精确比对，
  `missing=[]`。本轮完成性检查发现的 Admin lint、评价删除旧 E2E、Admin 受限路由旧 E2E
  和用户菜单 timer 竞态均已分别更新本文并独立提交，没有用一次聚合提交掩盖问题来源。
- 后端：`make test` 的 `go test -race -p 1 ./...`、`make lint`（0 issues）与
  `make build` 通过；OpenAPI lint、生成代码 drift、文档/API 同步检查也通过，后者覆盖
  194 paths / 17 prefixes。构建未产生跟踪文件差异。
- Shared / Web：Shared 12 files / 71 tests 通过；Web 当前全部跟踪 unit 为
  82 files / 518 tests，类型检查、跟踪文件 ESLint、production build 均通过。
  production-preview Playwright 最终为 363 passed / 1 designed skip；#35 首击确认前
  DELETE=0、确认后唯一 DELETE 的桌面/移动场景包含在该结果中。
- Admin：32 files / 154 unit、两段 TypeScript 检查、全仓 lint 0 warning / 0 error、
  production build 均通过。第一次全量 E2E 的 3 个失败已逐一分型：两个是 P0-1 后仍期待
  旧 404 的测试契约，一个是 N-2 真实 500 ms timer 竞态；分别修复后最终双视口
  Playwright 214/214 通过。
- UniAppX：只纳入 Git 跟踪的 8 files / 58 unit 和 `surface.spec.ts`，type-check、正式
  H5 build、8 个 tabBar 资源契约与桌面/移动 72/72 E2E 通过。仓库已明确不支持
  mp-weixin，因此没有用 H5 结果冒充微信开发者工具或真机验收。
- Koishi：全工作区 build、Vue UI contracts、611/611 unit、startup smoke 与真实
  Console Chromium 46/46 通过；日志中的 401/connection-refused 为测试固定的故障分支，
  对应断言均通过，不是被忽略的套件失败。
- 基础设施：`make check-infra-contracts` 当前实际执行 77 个 shell/Node 契约并全部通过，
  覆盖 admission、Casdoor/OpenFGA、CI/deploy、镜像供应链、备份恢复、可观测性、
  PostgreSQL/Redis、生产部署/回滚和公开认证浏览器 smoke。
- 隔离边界：用户未跟踪的 UniAppX/Web probe、`AUDIT-REPORT.md`、LICENSE 与正在编辑的
  文档稿均未纳入受控测试计数或提交；两个用户已暂存的文档重命名在每次 `--only` 提交后
  仍保持 staged。Web 全仓聚合 lint 会扫描用户未跟踪的临时测试并因其中的 unused import
  失败，因此 Web lint 证据明确使用 Git 跟踪文件集合，未修改或删除用户探针来制造绿灯。

这轮本地回归没有替代 GitHub Required checks、真实生产部署、受控 Casdoor/OpenFGA 授权
闭环或生产指标观察。按本报告重新分级后的 P0/P1/P2 根问题，当前唯一没有授权代码实现的
  是 #11；其运行时保持 fail-closed，但受支持的 scoped grant 生命周期仍不可用。owner 已于
  2026-07-31 选择 PostgreSQL 唯一管理真源、OpenFGA serving projection 与 Casdoor
  authentication-only，当前进入全面实施。P3 产品/体验项仍按优先级另行处理，不能为了
  “待办归零”混入本轮高、中级修复。

#### 分项交叉验证记录

- 后端：U-1/412 修复后执行 `go test ./... -count=1` 全部通过；`rbac`、`app`、
  `academics`、`auth`、`user`、`course/review` 的定向 race 也通过。#18 深挖的
  `token`/`oidc`/`auth` 基线、#45 的 `middleware`/`metrics` 基线以及 #57 的
  `course/review`/`capability` 基线均通过；这些现有正向测试本身不覆盖已确认的负向场景。
- Web：第一组 2 files / 7 tests、补充组 5 files / 20 tests 均通过；专门的 jsdom probe
  在修复前复现了 #38 的永久 toast。修复后新增 Node 环境永久回归覆盖创建作用域销毁、
  原定时长边界、显式移除和持久提示清理；同一 jsdom mount/unmount 探针也转为通过。
- Koishi、Web、UniAppX 初始定向回归分别为 34/34、54/54、36/36。#6 修复后 UniAppX
  Vitest 8 files / 58 tests、类型检查、四类草稿边界的桌面/移动 Playwright 8/8、
  完整跟踪版 H5 E2E 72/72 和 monorepo 正式 build 均通过。永久负向用例证明 foreign-course
  draft 不恢复且成功发布后 DELETE=0；匹配 draft、本页成功保存和未绑定 draft 可清理，
  未绑定 draft 的 teacher 不恢复。H5 A11yButton 键盘 probe 1/1 通过，支持维持证伪“双触发”。
- #5 修复前正式 H5 build 以 0 退出但 8 个声明资源都不在产物；新增契约在移动前按预期列出
  8 个 source miss + 8 个 output miss。8 个 Git blob 原样移入 `src/static/tabbar` 后，
  package 与 monorepo 正式 `build:uni:h5` 都通过声明驱动契约；本机 built preview 从
  `pages.json` 派生 8 个 URL，逐项返回 200 `image/png`。UniAppX 8 files / 58 unit、
  type-check、桌面/移动定向 2/2 和完整跟踪版 `surface.spec.ts` 68/68 均通过。
  README 已固定输入树和门禁语义。后续单独复现 mp-weixin 命令仍以 0 退出却只生成 H5
  结构；现已按实验性产品范围明确为不支持，删除虚假命令/manifest 声明并接入静态目标契约。
  H5 再次完成正式 build、58/58 unit、type-check 和已跟踪双视口 72/72 E2E；没有用 H5
  绿灯替代微信验证，也没有把用户未跟踪的临时 probe 混入回归或提交。
- #3/#4 修复最终执行 4 个测试文件 / 22 个定向测试和 Koishi 全量 602/602 unit；Core/Group Guard
  两个 `tsc --noEmit`、Vue UI contracts、完整 build、startup smoke 与 46/46 Chromium UI smoke
  均通过。独立代理第一次复核发现 scoped response 仍携带 bot-wide 配置/Bot inventory 而判
  FAIL；改为 `globalRuntime: null`、scoped 分支不加载全局 store 且客户端不渲染后，第二次复核
  PASS。提交为 `7b448809`。
- #17 修复的 OIDC claim provenance、Web/native/refresh 权威输入、`/auth/me` 非权威输入、
  exact tuple higher-consistency Read、幂等删除、FGA 故障 fail-closed 和撤权审计均有负向测试；
  `oidc`、`fga`、`auth`、`user` 四包定向与 race、全服务端 `make test`（`-race -p 1 ./...`）、
  Casdoor 边界检查、`golangci-lint` 和 build 均通过。真实 OpenFGA v1.18.1 一次性独立 store
  验证写入 200、精确读取 `1`、首次删除 200、重复删除 200、最终读取 `0`；store 以 204 删除并
  确认不存在。没有读取或修改现有业务 store。
- #18 修复的 OIDC expiry provenance、login/native/refresh 持久化、Lua hash+expiry 原子轮换、
  session lease/hard-cap 拒绝、tracked logout 无二次覆盖、逐 session logout-all、legacy PTTL
  与无 TTL fail-closed 均有负向测试；`token`、`oidc`、`middleware`、`auth` 四包定向与 race、
  全服务端 `make test`（`-race -p 1 ./...`）、Casdoor 边界检查、`golangci-lint`、build 和
  文档卫生均通过。一次性 Redis 8.8.1 容器实测现代 token 约 50 分钟、legacy PTTL 约 20 分钟、
  logout-all 两条 token 约 10/40 分钟，超 30 天拒绝且不写 key；测试容器已删除。
- N-1 修复前对运行中的固定 Casdoor digest 做无状态协议探针：discovery 无
  `revocation_endpoint`、`end_session_endpoint=/api/logout`；旧 RFC 7009 表单返回
  HTTP 200 `status=ok` no-op，非法 `id_token_hint` 返回 HTTP 200 `status=error`，与
  v3.125.0 固定源码的 `ExpireTokenByAccessToken`、refresh `expires_in <= 0` 拒绝逻辑一致。
  永久 contract test 模拟同一 token row，验证 adapter 成功后 access introspection inactive、
  refresh grant `invalid_grant`，并覆盖跨 issuer origin、业务错误、畸形响应、错误 endpoint、
  真实 RFC 7009 分支、legacy refresh-rotate-logout、双凭据缺失与 unexpected 4xx 不得被
  掩盖；logout-all 还有“本地 session 全部撤销后才调用 provider”的顺序断言。
  `oidc`/`token`/`auth` 定向与 race、全服务端 `make test`（`-race -p 1 ./...`）、
  Casdoor 边界检查、`golangci-lint`、build 与文档卫生均通过；一次性 Redis 8.8.1 验证
  provider access/refresh 密文随 hash/expiry 原子轮换、logout 使用最新 family，容器和
  临时测试文件均已删除。
- #45 重新枚举当前 admission routes 得到 5 条 `:token` 路由，并逐点确认 RequestLogger、
  Recovery、已知 Content-Length 超限和流式读取超限四类日志原先都使用 raw URL path。
  统一 helper 的路由级回归验证匹配请求在 handler 前后都只得到模板，404/405 固定为
  `unmatched`；静态扫描确认 middleware 内不存在旧的 raw-path zap 字段。middleware
  定向/全包 race、全服务端 race、lint、build、文档卫生与差异检查均通过；全量门禁中
  一次无关的 course PostgreSQL fixture 就绪超时，经 fixture 回收后该包独立无缓存 race
  和完整 `make test` 复跑均通过，未通过放宽 timeout 或改动课程代码规避。
- #57 用 `BuildUserAccessSnapshot` 直接制造合法的 school-scoped 与 section-scoped
  `admin:reviews:manage`：两者的 flat capabilities 均含该名称、global capabilities 均为空，
  修复后 `CanManageReviews/CanViewFull` 都为 false，第二行正文被裁剪；无 scope 的 global
  grant 则保留完整两行。评课模块定向 race 和包含真实 PostgreSQL fixture 的全包 race
  均通过；全服务端 race、lint、build、文档卫生与差异检查也通过。四类公共 repository
  查询仍只选择 `published`，且没有为修复增加逐行 scope DTO/SQL。
- #43 的 deterministic Redis dialer 依次返回 caller cancellation、deadline exceeded 和
  backend error：三条调用都保持 fail-closed，breaker failure 计数分别为 0、1、2。
  CircuitBreaker 回归进一步把 breaker 推到 half-open，证明 neutral 不改变成功/失败计数且
  会释放单一 probe；取消 Gin request 的 optional-auth 回归则证明 blacklist 查询使用
  detached 预算后 failure 仍为 0。`circuitbreaker`、`token`、`middleware`、`auth` 四包
  全量 race、全服务端 race、lint、build、文档卫生与差异检查均通过。
- #39 的轮询单测覆盖瞬时 503 后成功、五次网络失败后可恢复超时、401 首次即停止和 Abort
  不被吞掉；页面状态回归证明瞬时 polling rejection 仍显示 `projectionPending`、超时提示和
  手动重试，401 则进入 `needsLogin`，两者都不落入通用 error。Store 回归分别验证 503/403
  保留 cached user、401 清空。三组定向 52 tests、全部 76 个已跟踪 Web unit 文件
  493 tests、Web type-check、定向 ESLint、production build 和文档卫生均通过；用户未跟踪的
  `zzToastScope.tmp.test.ts` 因缺 jsdom 单独失败，未修改、未计入本项回归。
- #8 用真实 `http.DetectContentType` 覆盖 17 个允许和 6 个拒绝边界：OOXML、
  OpenDocument、EPUB/JAR、Windows ZIP、带 OLE 魔数的旧 Office、CSV/Markdown/TSV 和
  有效 JSON 保留规范化 MIME；伪装图片、任意文本/ZIP 细分、无魔数旧 Office、无效 JSON
  与畸形 MIME fail-closed。真实 PostgreSQL 用例确认 DOCX 的有效 MIME 写入版本记录；
  资源包全量 `-race` 通过。未对身份材料或 admission 上传复用该兼容策略。
- #28 的纯状态机回归在第二个逻辑域抛出超时，确认结果稳定为
  `confirmed / unconfirmed / not-run`，后续域没有被调用；关键词回归先确认删除 A、再让 B
  upsert 失败，baseline 只保留 B 的旧值，重试只执行 B/C upsert、不再删除 A。服务端回归
  证明 missing delete 返回成功且不调用 delete，已存在的外群规则仍在副作用前拒绝。WebUI
  静态契约固定逐域状态、成功 slice baseline、确认 reload 和共享安全正则接线。Koishi
  production build、UI contracts、606/606 unit、startup smoke 与 46/46 Chromium UI
  smoke 全部通过；浏览器全局设置的保存、放弃和恢复默认真实 Console action 未回归。没有
  声称这些测试提供跨 7 个逻辑域的原子事务。
- U-2 先对固定 go-oidc v3.17.0 源码交叉验证：`RemoteKeySet` 文档要求复用 long-lived
  verifier，已知 `kid` 先读 cache，只有不匹配才回源，remote failure 不替换
  `cachedKeys`。永久 httptest 依次证明首次 known key 请求一次、provider 下线后 known
  key 零回源、unknown key 返回 `ErrProviderUnavailable` 且请求数加一、随后 known key
  仍可验证。OIDC 包普通/race、middleware+auth race、app、全服务端
  `go test -race -p 1 ./...`、Casdoor boundary、golangci-lint、build 与文档卫生均通过。
  尚未用生产 Casdoor 故障演练验证运行实例；用户暂存的 `iam-architecture.md` 冲突段也未
  纳入本提交。
- #51 的永久 handler 回归分别制造四种 Redis 状态：成功 rotation 后旧 hash 与当前 hash
  不同，判定真 reuse 并撤销全部 session；logout 完成后 referenced session 不存在；
  logout 的 blacklist→delete 竞争窗口仍保留相同当前 hash；以及历史 attribution 缺失。
  后三者都只返回 revoked，其他设备 session 保留且 reuse counter 不增加。实现未增加
  revoke reason 字段或改变 native tracked-session 门禁。auth 全包与 race、全服务端
  `make test`（`-race -p 1 ./...`）、Casdoor 边界检查、`golangci-lint`、build 和文档卫生
  均通过。
- #30 的服务端回归在同一 global scope 构造 103 条相同 deadline、逆序 ID 的 active records，
  结果元数据为 `shown=100/total=103/limit=100/truncated=true`，窗口稳定为 `member-000`
  到 `member-099`；既有 scoped 回归同时确认 foreign 101 条不会污染 allowed guild 的总数、
  截断状态或 payload。客户端模型对相同 deadline 也按 ID 稳定排序，组件契约固定
  `shown / total`、截断说明和处置中心导航。Koishi build、UI contracts、609/609 unit、
  startup smoke、46/46 Chromium UI smoke 与 package contract 通过；浏览器 smoke 只证明
  既有 Admission 页面/真实动作未回归，未伪称以生产规模浏览器数据验证了截断提示。处置中心、
  群内命令和两类后台扫描不受 100 条窗口影响；尚无生产数据证明队列经常超过 100。
- #38 的永久回归在 Vue `effectScope` 中创建 3 秒 Toast，销毁创建作用域后于 2999 ms 仍
  可见、3000 ms 准时消失，并验证显式 `remove` 不影响 duration=0 的持久提示、
  `clearAll` 可确定清空。全部 78 个已跟踪 Web unit 文件 505/505、type-check、定向 ESLint、
  production build 与文档卫生通过；用户未跟踪的 jsdom 组件卸载探针也单独 1/1 通过，
  但未修改、未提交、未计入正式 505 个用例。
- #42 的组件回归先让 verification 请求保持 pending，确认账号邮箱仍可见而 phone/QQ/
  实名/学籍的“未验证、未绑定、缺失”均不出现；随后制造首次请求失败，确认内联 error/retry
  保留可靠字段，第二次成功后恢复 verified/bound 状态。另一路成功空响应证明真正未认证用户
  只有在完整读取完成后才显示负面状态。全部 79 个已跟踪 Web unit 文件 507/507、
  type-check、定向 ESLint、production build 与文档卫生通过；未改 verification store、
  OpenAPI 或后端状态语义。
- #34 用真实 Chromium 观察 GSAP global timeline：初始 50，跨帧 resize 后逐批增加，离开首页
  只清除最后一批且旧 target 仍变化；在丢弃数组前 kill 的对照探针始终稳定 50、卸载归零。
  探针只修改 HTTP 响应中的临时 bundle，没有写工作树。实现后永久 jsdom 回归用 3 个
  target 精确固定初始批、同帧 resize 合并、旧批 kill、新批创建和卸载 kill；全部 80 个
  已跟踪 Web unit 文件 508/508、type-check、定向 ESLint、production build 与文档卫生通过。
- #35 精确 Playwright 用例确认 own-review 首击立即触发 DELETE；服务端与迁移证据确认是
  soft delete，但现有用户和管理员状态机都没有 restore。该用例输出在工作树外，未制造新文件。
  修复后的永久组件回归覆盖首击/取消零请求、确认后唯一请求、deleted ID、成功 toast、焦点
  转移/恢复及 composable 重复调用 single-flight，共 3/3；全部 81 个已跟踪 Web unit 文件
  511/511、type-check、定向 ESLint、production build 与文档卫生通过。没有把前端确认误报为
  服务端 restore/undo 保证。
- #36 的组件回归制造包含私有 detail 的 setup 异常，确认 `ErrorBoundary` 只调用一次
  `reportFrontendError('vue-error')` 并接管 fallback UI；transport 回归确认初始化前 no-op、
  初始化后 payload 精确为 kind-only，且 `sendBeacon` 自身抛错不会形成第二个应用异常。
  后端回归确认 `vue-error` 被固定白名单接受并计数，未知 kind 仍丢弃，label 基数固定为 3。
  OpenAPI 连续两次生成的 bundled spec、Go 与 TypeScript 输出 SHA-256 完全一致；82 个受控
  Web unit 文件 514/514、333 个受控 Web lint 输入（仅 3 个既有 CSS ignore warning）、
  type-check、production build、metrics race、app、Go lint、spec lint 与文档卫生均通过。
  package 默认 lint 唯一失败来自用户未跟踪的 `zzProbeAdminEditDialog.test.ts` 未使用 import，
  未修改、未计入本项门禁。
- #37 在 desktop/mobile Chromium 各执行完整 accessibility 文件 7 项，共 14/14：新增用例
  证明 skip link 是首个 Tab 目标、获得焦点时可见、Enter 后 `main#main-content` 获得焦点；
  home/about/privacy/terms/404 及 dark home 的 Axe WCAG A/AA 基线仍无可检测违规。全部 82 个
  受控 Web unit 文件 514/514、type-check、333 个受控 lint 输入（仅 3 个既有 CSS ignore
  warning）、production build 与文档卫生通过。首次 Playwright 命令使用了仓库不存在的
  `chromium` project，未启动测试；随后按真实 `desktop-chromium`/`mobile-chromium`
  完整重跑并通过，未将工具参数错误混作产品失败。
- 反向复核的非法 FGA section 回归覆盖纯无效、合法/无效混合和 school/section 两类真实
  dependency error：前两者分别得到零 scope/只保留合法 scope，并按无效 grant 增加 counter；
  dependency error 仍可 `errors.Is` 原始 OpenFGA 错误。authorization/metrics 定向 race、
  全服务端 `make test`（`-race -p 1 ./...`）、Casdoor boundary、`golangci-lint`、静态 build、
  全部 infra contracts、告警 contract 与文档卫生均通过。指标没有外部 ID label，日志不含
  Casdoor subject；没有自动删除 tuple 或把依赖故障降级成零权限。
- 反向复核的 live rating bar 用纯 helper 与显示策略测试固定本地化维度、五级 face 归一化和
  不泄露 4.6 的约束；真实 CourseDetailPage mock 在 desktop/mobile Chromium 2/2 中得到
  `role=img` accessible name“教学质量：超赞”，精确 `4.6` 文本计数为 0。定向 2 files /
  10 tests、隔离基线 2 files / 10 tests、最终单独执行的全部 82 个受控 Web unit 文件
  516/516、type-check、受控 ESLint、production build 和文档卫生均通过。第一次把全量 unit
  与 build 并行时 3 个无关基线测试因 5 秒 timeout/级联状态失败；隔离及无资源争用全量复跑
  均通过，未放宽 timeout、未修改 locale/observability。
- X-2 先以 AST 对实施前 `b479bafa` 的 config 包复枚举，得到 184 个运行时字面量键和 17 个
  模板差集；删除 `LOG_SERVICE_VERSION` 并补齐 13 个操作员配置后，永久门禁记录当前 183 个
  runtime key、378 个模板/基础设施 key 和 3 个有理由的兼容别名。门禁同时验证新增配置调用
  必须使用可审计的字面量 key，allowlist 项必须仍被运行时读取且不得已进入模板。config 全包
  race、全服务端 `make test`（`-race -p 1 ./...`）、Casdoor boundary、`golangci-lint`、静态
  build、Actionlint、文档卫生、开发/生产 env 初始化 contract 和全部 infra contracts 均通过；
  新增模板空值保持现有 scopes default、OIDC discovery、邮件默认 failover 和 TLS 非 mTLS
  行为，没有改变运行时默认值。
- P2-9 使用隔离 venv 中固定的 Ansible Core 2.20.2 做了两层反证：通用 localhost cwd probe
  返回 playbook basedir，固定源码也明确在 local connection 上设置
  `_connection.cwd = _loader.get_basedir()`，因此原报告“脚本必然找不到”不成立；但在干净 clone
  原样执行旧 task 时，脚本进入后因相对 output 配合 `git -C` 返回 128 和
  `could not open '../../generated/deploy/bundle.*.tar.gz'`。候选修复的三个 playbook syntax
  check 全部通过，单独执行 `--tags deploy-bundle` 为 1/1，生成的目标 tar 确实包含
  `.env.example` 与 `.env.prod.example`。窄路径 contract、CI/drift、Actionlint、全部 infra
  contracts 和文档卫生通过；没有连接 staging/production SSH，也没有执行 upload、preflight
  或远端 deploy，因此这些仍是发布验收边界。
- P2-18 的真实 PostgreSQL 状态序列先显式预热空 filter，再创建 `block` 词并由下一次检查立即
  拦截；随后同一记录更新为新词和 `warn`，旧词不再命中、新词得到 warn；删除后新词也不再命中。
  create/update/delete 每一步都断言 snapshot 被标记 stale。缺失 ID 的 update/delete 返回错误且
  `lastRefresh` 不变；失效后关闭连接池则重载返回 `ErrModerationUnavailable`、结果为 nil、快照
  仍保持 stale。另一个 race 用例固定 `Invalidate` 必须等待 refresh critical section，防止旧
  refresh 覆盖失效。评课定向 5 项与全包 race、全服务端 `make test`（`-race -p 1 ./...`）、
  Casdoor boundary、`golangci-lint`、静态 build 和文档卫生均通过。当前没有多 app replica
  的受支持 Compose 拓扑，因此没有扩展为 Redis version/pub/sub。
- P2-21/P2-22 通过 `database/sql` scripted driver 实际驱动 `LookupStudent`/`Probe` 状态机：
  parent cancel 在 closed 状态保持 0 failure，在 half-open 状态返回原 `context.Canceled` 并释放
  `probe_in_flight`；parent deadline 同属 neutral，而目录自身 10ms timeout 保留
  `ErrStudentSourceUnavailable` 并打开 threshold=1 的 breaker。NULL 姓名和冲突重复姓名分别
  返回原 typed integrity error、保持 breaker closed 并增加固定 reason counter；合法但与绑定参数
  不同的学号返回新 identity-mismatch sentinel、保留 unavailable 包装并打开 breaker。adapter
  专项测试确认 raw integrity error 仍嵌套 `ErrAcademicLookupUnavailable`，所以现有 HTTP 503
  契约未改变。externaldata/metrics/app 三包定向 race、全服务端 `make test`
  （`-race -p 1 ./...`）、Casdoor boundary、`golangci-lint`、静态 build 和文档卫生均通过。
  本地没有连接真实 Oracle，也未验证生产指标抓取、告警或真实坏行。
- P2-23 的 unit 状态机把 attempt=4、`MaxAttempts=5` 的 poison panic 转为 terminal failure，
  验证 `outbox_job_failures_total{terminal="true"}` 增量、stack 保留和同批 healthy job
  `markDone`。另用 testcontainers 启动真实 PostgreSQL 18 并应用完整 migration：同一 stream
  中先插 poison、再插 healthy，执行一次 `ProcessBatch` 后逐行读回
  `dead_letter/attempt_count=1/last_error contains stack` 与
  `completed/attempt_count=0/locked_at=NULL`；容器与 Ryuk 均已自动清理。outbox 定向 race、
  全服务端 `make test`（`-race -p 1 ./...`）、Casdoor boundary、`golangci-lint`、静态 build
  和文档卫生均通过。源码调用面复枚举为 5 个共享 outbox polling worker；两个 Admission
  expiry loop 不调用 `ProcessBatch`，本修复不声称覆盖。生产告警投递、真实 poison replay 和
  外部副作用幂等仍是发布验收边界。
- R-8 分别启动 miniredis 后关闭 server，使 go-redis 的 `SET NX` 经真实 socket 得到
  connection-refused transport error；Admission/User 两条路径都断言 unavailable sentinel
  存在、cooldown sentinel 不存在，错误文本保留具体 reserve operation。独立 HTTP mapping
  用例确认 unavailable 为 503，既有第二次请求用例继续确认真实 NX 冲突为 429。Admission
  request-otp 先补 OpenAPI 503，再执行完整 Go/TS generate；第二次 generate 前后对
  bundle、嵌入 Go spec 和 TS 类型的 binary diff hash 相同，证明生成稳定。Admission/User
  两包完整 race、全服务端 `make test`（`-race -p 1 ./...`）、spec lint、Casdoor boundary、
  `golangci-lint`、静态 build 和文档卫生均通过。没有改变 Redis key、60 秒 cooldown、
  endpoint limiter 或 OTP 状态机；生产 Redis 故障注入仍待发布验收。
- P3-9 先向 miniredis 写入真实 cache version `7`，再关闭 socket，确认 `GetVersion` 和
  `BuildVersionedKey` 返回 unknown/空 key，而不是 `v0`；本地版本 map 不产生条目，raw/typed
  get 均为 miss，set 在 Redis 已关闭时仍为 no-op。重启同一 miniredis 后立即读取真实 `v7`，
  证明故障结果没有被本地 memoize；预取消 context 也覆盖同一安全降级。cache 全包与
  course/review 调用方 race、全服务端 `make test`（`-race -p 1 ./...`）、Casdoor boundary、
  `golangci-lint`、静态 build 和文档卫生均通过。未引入 detached 200ms loader；首 caller
  取消时其他 singleflight waiter 同样 bypass 只影响缓存命中率，不再允许陈旧 `v0` 数据复活。
- P2-16 对运行中的开发 PostgreSQL 只读统计 23 门课程，当前 nullable 三列均为 0 个 NULL；
  该结果只证明当前样本干净。隔离 PostgreSQL 18 另插入三列均为 NULL 的课程，真实执行
  detail/list/search/grouped/favorites，全部保留 nil 并成功返回；HTTP 详情明确为
  `departmentID:null`/`credits:null`。OpenAPI 完整 generate 后 Go/TS nullable 类型一致，
  Web 正向/null/错误类型/负数解析、UniAppX 58 unit、三前端类型检查、两业务包与全服务端 race、
  spec/drift、`golangci-lint`、静态 build 和文档卫生均通过。没有写生产数据、migration、
  `COALESCE(0/'')` 或伪造“未分类院系”记录。
- P3-7 在隔离 PostgreSQL 18 对 NDJSON/CSV 各执行成功和预取消 context 的真实 DB stream
  failure，共 4 次请求、4 条且每次恰好 1 条 `data.export`；成功行数为 1，失败行数为 0，
  actor/resource/filter/10,000 行上限与 success/failure reason 均从落库记录复读验证。失败
  请求没有完成标记，但审计通过 `context.WithoutCancel` 成功持久化。评课包与全服务端 race、
  `golangci-lint`、静态 build 和文档卫生均通过。没有双写 `h.logAdminOp`、新增表/队列，或把
  `row_count` 误称为客户端可靠接收证明。
- P2-12 在隔离 PostgreSQL 18 与 miniredis 中，让同一认证 subject 连续执行 4 次
  academic-match + 1 次 request-otp，真实 academic gateway 调用合计 5 次；第 6 次跨回
  academic-match 返回 429、带 `Retry-After: 60`，gateway 计数不变，证明两路由共享一个
  budget。第二个 subject 立即成功且只增加一次，证明用户隔离。关闭 miniredis 后，匿名请求
  先被 auth 拒绝为 401，认证请求在 handler/Oracle 前 503，gateway 为 0。Admission/app
  定向与全服务端 race、OpenAPI lint/drift、生成稳定、共享 TS type-check、`golangci-lint`、
  静态 build 和文档卫生均通过；没有限制不查 Oracle 的 verify-otp、重排 OTP cooldown，
  或引入第二个 limiter 实现。
- P2-3 先只读检查开发 SQLite：数据库文件为 204 KiB，消息账本为 0 行且只有主键索引；
  旧查询的执行计划仍明确为全表 `SCAN` 加临时 B-tree 排序，证明机制真实但没有本地规模影响
  证据。候选实现用锁定版 Minato 的 cursor 把 `guildId`、`createdAt DESC` 和原始 limit
  下推，并由模型自动创建 `(guildId ASC, createdAt DESC)` 索引。隔离 Koishi SQLite 实例
  实际生成该索引；插入 10,000 行（目标群 9,000 行）后，`EXPLAIN QUERY PLAN` 改为按复合
  索引 `SEARCH`，最新 3 行顺序正确。定向 8/8、Koishi 全工作区 build/contracts、
  611/611 unit、startup 和 46/46 UI smoke 通过。没有删除开发数据，也没有在缺少最长撤回/
  审计期限时加入自动 retention。

测试通过只说明现有正向契约未被破坏，不能覆盖报告指出的所有负向场景。以下仍需在真正处置时
单独验收：

- 生产 Casdoor 的实际 access/refresh TTL、introspection、logout/revocation 和 JWKS 故障行为；
- #18 已完成本地实现和 Redis 协议验证，但生产发布仍须确认 `bootstrap-platform.sh prod`
  收敛 access TTL、旧 session PTTL 前提成立，并以真实 login → logout/logout-all → token
  拒绝链路留证；不能用本地绿灯替代。
- N-1 已完成本地实现、固定镜像无状态探针和 contract/Redis 验证，但尚未用生产受控账号留存
  “旧 access introspection inactive + 旧 refresh grant invalid_grant”的真实 logout 与
  logout-all 证据，也未验收滚动升级旧 session 的 refresh-rotate-logout bridge；HTTP 200、
  cookie 清除或本地 session 删除均不能替代这项 provider 验收。
- 生产 Casdoor 实际降级 `super_admin` 后的新 ID token、OpenFGA tuple 撤权、审计落库以及各
  school/resource 权限立即失效的端到端验收；
- 真实 scoped Console operator 的跨群读写、插件重载与全局设置权限；
- 生产无效 section tuple 的 warning、`StuHelperInvalidOpenFGARoleScope` 告警投递与人工
  reconcile 流程；本地验证只覆盖规则与契约，没有写入或删除生产 tuple；
- 原生 SSO redirect 和浏览器 storage 受限模式；mp-weixin 已明确不支持，不再伪列为当前验收项；
- 超过 100 条的真实 restricted-member 队列、生产 Redis p95/p99 与 breaker 行为；
- 任何真实部署、生产数据迁移或生产 OpenFGA tuple reconcile。

### Codex 第二轮修复进度

| 报告编号 | 状态 | 验证证据 | 提交 |
|----------|------|----------|------|
| 3 + 4 | 已修复，待发布；真实 scoped Console 操作员验收待补 | Canonical scope resolver、global settings fail-closed、scoped records/stats/filter-before-limit、foreign action side-effect=0、scoped `globalRuntime=null` 和全局 loader=0 均有负向测试；22 定向、602 全量 unit、双插件 typecheck、contracts、build、startup 与 46 UI smoke 通过；独立复核经历 FAIL→修正→PASS | `7b448809` `fix(koishi): enforce admission console guild scope` |
| 9 | 已修复，待发布 | Web review payload reader 保留 `like`/`dislike`，缺失字段保持可选，非法 enum fail-closed；定向 payload/voting 回归、Web 类型检查与相关静态检查通过 | `59322589` `fix(review): preserve current user vote state` |
| 11 | 复核闭环；P1 decision-blocked，未改授权代码 | 非测试写路径、`fga-setup`、Casdoor bootstrap、resolver/capability/Entry 及永久负向测试交叉确认：缺少受支持生命周期，但空 scope 为 403/deny 而非全局权限。已提交 guardrail 明确来源未定；未提交 IAM/ADR 草稿也未完成期望状态、Casdoor 协同和恢复模型。文档现给出 DB/OpenFGA/运维清单三方案、推荐与共同验收边界；待 owner 选择真源及 role membership ownership | `docs(audit): bound scoped role provisioning` |
| 17 | 已修复，待发布；生产 Casdoor 降级与 OpenFGA 撤权待验收 | 新 ID token 的合法 roles claim 才能触发 reconcile，`/auth/me`/缺失/畸形 claim 均不 mutation；exact direct tuple higher-consistency Read、`on_missing=ignore` 删除、失败关闭和 `iam.role.revoke` 有负向测试。四包定向/race、全服务端 race、lint/build 与真实 OpenFGA v1.18.1 临时 store 写入/重复撤权/清理验证通过 | `fix(auth): reconcile authoritative super admin roles` |
| 18 | 已修复，待发布；生产 Casdoor TTL/真实登出待验收 | 已验证 provider `exp` 随 session 保存并在 refresh 原子轮换；新 token 强制不超过 session lease/30 天，logout/logout-all 逐 token 写真实 TTL，legacy 仅按受约束 PTTL 回退，无 TTL fail-closed；并消除 tracked logout 的 5 分钟二次覆盖。四包定向/race、全服务端 race、lint/build/docs 与真实 Redis 8.8.1 PTTL/hard-cap 验证通过 | `fix(auth): revoke access tokens through verified expiry` |
| N-1 | 已修复，待发布；生产 Casdoor 受控账号 token-family 撤销待验收 | session 加密保存并原子轮换 provider access/refresh；issuer 同源且路径精确的 Casdoor `/api/logout` 使用 `id_token_hint` 并校验 JSON `status=ok`，RFC 7009 路径保持独立；正常 refresh 不重复 revoke 旧 row，refresh 阶段未提交 family 补偿撤销，logout-all 先做本地撤销，legacy family 通过 refresh-rotate-logout bridge。固定镜像源码/运行实例无状态探针、忠实 contract、三包定向/race、全服务端 race、lint/build/docs 与真实 Redis 8.8.1 轮换验证通过 | `fix(auth): revoke Casdoor token families correctly` |
| N-2 | 已修复，待发布；低级 UX 稳定性项 | click 模式暂停 hover watcher 时同步清理已排队的 enter/leave timers，避免早期开启的用户菜单被旧 500 ms timer 关闭。修复前移动退出隔离 2 pass / 1 fail；修复后确定性 unit 1/1、移动连续 10/10、Admin 全量 32 files / 154 tests、类型/lint/build 和双视口 E2E 214/214 通过 | `fix(admin): cancel disabled hover timers` |
| 45 | 已修复，待发布 | 5 条 admission token route 的 RequestLogger、Recovery 与两类 body-limit 告警统一记录 route template；404/405 固定 `unmatched`。handler 前后/未匹配负向测试与 raw-path logger 静态扫描通过；middleware 定向/全包 race、全服务端 race、lint/build/docs/diff 检查均通过 | `fix(logging): keep dynamic path credentials out of logs` |
| 51 | 已修复，待发布 | blacklisted token 只有在 referenced session 存在、当前 refresh hash 非空且与提交 hash 不同时才是真 reuse；session 缺失、hash 相同或 attribution 缺失只返回 revoked。rotated/logout-complete/blacklist-before-delete/missing-ref 回归验证其他设备和 metric 语义；auth 全包/race、全服务端 race、lint/build/docs 通过 | `fix(auth): distinguish revoked refresh tokens from reuse` |
| 57 | 已修复，待发布 | 公共评课访问事实同时接收完整与 global capability 集合，只有 global `admin:reviews:manage` 能推出平台级管理/全文；普通学生能力不变。school scope、section scope、global grant 的正文裁剪回归、评课全包 race、全服务端 race、lint/build/docs/diff 均通过，未扩 DTO/SQL | `fix(authz): preserve scoped review access boundaries` |
| 43 | 已修复，待发布 | blacklist lookup 使用 50 ms detached context；5 个共享 breaker Redis error 分支将 caller cancellation 记为 neutral 并释放 half-open probe，deadline/backend error 仍计 failure 且 fail-closed。closed/half-open、canceled/deadline/backend、Gin request 负向测试、四包与全服务端 race、lint/build/docs/diff 均通过；未猜测修改预算/阈值/window | `fix(auth): classify blacklist cancellations as neutral` |
| 39 | 已修复，待发布 | `fetchUser` 对 `/auth/me` 只有 401 清本地身份；403/5xx/网络/超时保留用户并由投影轮询在既有五次预算内继续。耗尽后页面仍可手动重试，Abort/401 提前停止。定向 52、全部已跟踪 Web unit 493、type-check、ESLint、production build 与文档卫生通过；未跟踪临时探针不在提交/回归范围 | `fix(web): keep admission projection polling recoverable` |
| 44 | 已修复，待发布 | active introspection 只接受 Casdoor `access-token` purpose，refresh/missing/malformed/opaque token 均拒绝；Bearer 用户路径拒绝空白 subject，provider unavailable 与 inactive 分类保持不变；OIDC、middleware、app/auth 定向回归与服务端静态检查通过 | `3d12d259` `fix(auth): reject refresh tokens on bearer paths` |
| U-1 | 已修复，待发布 | 三个 academics admin operation 消费现有 admin MFA middlewares；共享 step-up 响应从错误的 428 对齐既有 412 契约，OpenAPI/生成物同步；blocking route contract、真实 MFA chain、相关包/全量 Go 回归、race、spec/drift、lint/build 与文档检查通过 | `4b2f520b` `fix(academics): require MFA for import administration` |
| U-2 | 已修复，待发布；用户架构稿旧 TTL 段待收敛 | verifier 复用单一 long-lived go-oidc `RemoteKeySet`，known key 离线验证、unknown kid 单次回源失败 503，失败不清旧 cache；session/blacklist/claims 门禁不变。OIDC 普通/race、middleware+auth race、app、全服务端 race、lint/build/docs 通过；生产故障演练待发布验收 | `fix(auth): preserve cached JWKS during provider outages` |
| 5 | 已修复，待发布；相邻 mp-weixin 假绿已另行收口 | 8 个 tabBar blob 原样移入 `src/static`；H5 build 从 `pages.json` 派生并强制检查 8 个 source/output 文件，built preview 8/8 为 200 PNG。58 unit、type-check、桌面/移动 68 E2E、package/monorepo build 通过；未改 URL/publicDir、未复制资源、未把 H5 当微信产物 | `fix(uniappx): include declared tab bar assets in H5 builds` |
| 反向-mp-weixin | 已修复，待发布；当前明确不支持微信小程序 | 修复前命令退出 0 但只有 H5 `index.html/assets`；当前无微信 compiler/auth。已删除根/包微信命令和 manifest 声明，README 固定 H5-only 验证边界，零依赖 target contract 接入 repository-policy。旧命令现均非零；H5 build/8 资产、58 unit、type-check、已跟踪双视口 72 E2E、Actionlint、docs 与全部 infra contracts 通过。未建设未经需求驱动的微信发布体系 | `fix(uniappx): stop advertising unsupported WeChat builds` |
| 6 | 已修复，待发布；跨设备 ABA 未提前设计 | 当前页只恢复 course 匹配/未绑定草稿，未绑定不恢复 teacher；成功恢复或保存才获得 create 成功后的单槽清理资格，外课程和不确定保存结果均保留。产品文档纠正为无 course path 的每用户单槽。58 unit、type-check、草稿边界 8 E2E、完整 H5 72 E2E、正式 build/docs 通过 | `fix(uniappx): preserve drafts across course submissions` |
| 7 | 已修复，待发布 | provider-owned 三个字段使用已验证的 `accountSettingsUrl`；四个 StuHelper 字段保留后端本地 action，外部 URL 缺失时不猜测 issuer。10 个单元测试、Web type-check/ESLint、桌面和移动完整 Open Platform 流程 20/20 通过，既有 Continue 与安全 redirect 行为未回归 | `fix(web): route provider profile completion to account settings` |
| 8 | 已修复，待发布 | 资源创建对精确 ZIP/OLE/text refinement 返回并持久化有效 MIME；OLE/JSON 增加内容验证，未知 octet-stream、任意文本/ZIP 派生和真实冲突继续拒绝。23 个媒体边界子测试、真实 PostgreSQL 持久化、资源全包 race、OpenAPI spec/drift、全服务端 race/lint/build/docs 通过 | `fix(resource): preserve verified upload media types` |
| 28 | 已修复，待发布；跨请求原子性仍不承诺 | 逐域状态机和关键词部分成功 baseline 负向测试通过；missing delete 幂等、existing foreign rule 无副作用；Koishi build、UI contracts、606 unit、startup 与 46 Chromium UI smoke 通过。失败后可按剩余差异重试或确认 reload，未引入并发 fan-out/rollback/2PC/saga | `fix(koishi): preserve partial settings save results` |
| 30 | 已修复，待发布；生产持续截断率仍待观测 | 同 scope 103 条等 deadline 逆序数据稳定返回前 100 条及完整窗口元数据；foreign records 不污染 scoped total。客户端排序/提示/导航契约、Koishi build、609 unit、startup、46 UI smoke 与 package contract 通过。未新增分页/search，生产队列规模与 SLO 尚未验证 | `fix(koishi): disclose truncated admission queues` |
| 38 | 已修复，待发布 | 删除与全局 Toast 生命周期冲突的组件 scope timer cleanup；创建 scope 销毁后仍于原 duration 自动关闭，显式 remove/clearAll 语义保持。永久 Node 回归 2/2、用户 jsdom 卸载探针 1/1、全部已跟踪 Web unit 505/505、type-check、ESLint、production build 与 docs 通过 | `fix(web): keep toast dismissal across navigation` |
| 42 | 已修复，待发布 | 资料页本地 `loading/ready/error` 只保护 verification 来源字段；账号和邮箱在失败时保留，状态成功前不显示未知的未验证/未绑定结论，内联重试成功后恢复真实状态。组件负向/重试 2/2、全部已跟踪 Web unit 507/507、type-check、ESLint、production build 与 docs 通过 | `fix(web): distinguish unknown profile verification state` |
| 34 | 已修复，待发布 | `createParticles` 在覆盖 target 数组前 kill 当前 GSAP tweens，保留既有 resize rAF 合并和卸载清理。真实 Chromium 前后对照、永久生命周期回归 1/1、全部已跟踪 Web unit 508/508、type-check、ESLint、production build 与 docs 通过 | `fix(web): release particle tweens before rebuild` |
| 35 | 已修复，待发布；累计 E2E 契约已同步，仍无 restore/undo 产品能力 | own-review 删除改为局部两阶段确认并管理键盘焦点；取消零请求，确认后唯一 DELETE，composable 程序级 single-flight。组件/边界回归 3/3、当前全部跟踪 Web unit 82 files / 518 tests、type-check、跟踪文件 ESLint、production build、桌面/移动定向 2/2、Web 全量 363 passed / 1 designed skip 与 docs 通过 | `fix(review): confirm own-review deletion`；`test(review): align deletion E2E with confirmation` |
| 36 | 已修复，待发布；生产 dashboard/告警消费仍待验收 | `ErrorBoundary` 与 Vue 全局 handler 复用既有 kind-only telemetry，固定 `vue-error` label；未初始化时 no-op，transport failure 被隔离，后端不存原始异常。组件/transport/基数负向回归、全部受控 Web unit 514/514、type-check、ESLint、production build、metrics race、app、Go lint、spec/generate stability 与 docs 通过 | `fix(web): report captured Vue component errors` |
| 37 | 已修复，待发布 | AppShell 首个键盘停靠点增加本地化原生 skip link，唯一 main target 可聚焦；链接 focus-visible 时位于 fixed header 之上并遵守 safe-area/reduced-motion。双视口真实键盘与 Axe 14/14、全部受控 Web unit 514/514、type-check、ESLint、production build 与 docs 通过 | `fix(web): let keyboard users skip application navigation` |
| 反向-FGA | 已修复，待发布；生产告警投递与人工 tuple reconcile 待验收 | 无效 section grant 独立忽略、计数并 warning，合法 grant 保留，纯无效得到零 scope，真实 OpenFGA error 仍失败关闭；Prometheus warning alert 已接线。定向 race、全服务端 race、lint/build、全部 infra contracts 与 docs 通过 | `fix(authz): isolate invalid OpenFGA role scopes` |
| 反向-RatingBar | 已修复，待发布 | 只修改可达 CourseDetailPage：维度条以定性 `role=img` 名称暴露，视觉层隐藏，复用既有 face bucket，精确 avg 不进入辅助文本。helper/policy 10/10、双视口 Chromium 2/2、全部受控 Web unit 516/516、type-check、ESLint、production build 与 docs 通过 | `fix(web): describe course rating bars qualitatively` |
| X-2 / I-2 | 已修复，待 CI/发布验收 | 实施前 AST 精确枚举 184 runtime keys / 17 missing；13 个 operator-facing 键补入两模板，删除死 `LOG_SERVICE_VERSION`，3 个兼容 override 留在带理由 allowlist。当前 183 runtime keys 全覆盖；全服务端 race/lint/build、Actionlint、docs、dev/prod env 初始化和全部 infra contracts 通过，模板改动已接入 Backend job | `fix(config): keep runtime env templates complete` |
| P2-9 | 已修复，待 CI/远端发布验收 | 旧任务在干净 clone 的真实 Ansible Core 2.20.2 中找到脚本后，于相对 output + `git -C` 处稳定失败；候选改动三 playbook syntax 通过，`deploy-bundle` tag 1/1 并生成含两个 env 模板的目标 tar。固定 controller requirements、core-compatible callback、窄路径/CI contract、Actionlint、全部 infra contracts 和 docs 通过；未连接远端 | `fix(deploy): anchor Ansible bundle paths` |
| P2-18 | 已修复，待发布 | 成功敏感词 mutation 使本地快照 stale，失败 mutation 不失效；refresh/invalidation 串行避免旧查询恢复 freshness，重载失败 fail-closed。真实 PostgreSQL 覆盖 create/update/delete 和依赖失败；评课/全服务端 race、lint/build/docs 通过，单 app 范围与多副本前置要求已写入安全文档 | `fix(review): invalidate sensitive-word snapshots` |
| P2-21/P2-22 | 已修复，待发布；真实 Oracle 与生产指标/告警待验收 | parent cancel/deadline 和 invalid/ambiguous row 均记 neutral，不增加或重置 failure，并释放 half-open probe；内部 query timeout、backend failure 与 bind identity mismatch 仍计 failure。三类固定 integrity counter、typed error 与既有 503 adapter 契约均有回归；三包定向 race、全服务端 race、lint/build/docs 通过 | `fix(externaldata): classify Oracle source outcomes` |
| P2-23 | 已修复，待发布；生产告警与 poison replay 待验收 | per-job recover 将 handler panic 转为带 stack 的普通失败，沿既有 attempt/dead-letter/terminal metric 路径处理；同批健康 job 继续。真实 PostgreSQL 18 验证 poison=`dead_letter`、healthy=`completed`，定向/全服务端 race、lint/build/docs 通过；未增加 runtime supervisor，未覆盖 Admission 手写 expiry loop | `fix(outbox): isolate panicking job handlers` |
| R-8 | 已修复，待发布；生产 Redis 故障注入待验收 | 两条 OTP reserve 只有真实 NX miss 才返回 cooldown/429，transport error 包装现有 unavailable/503 并保留 cause；miniredis socket 关闭与 HTTP 映射回归通过。Admission request-otp 补 503 后完整生成，两个业务包/全服务端 race、lint/build/spec/codegen stability/docs 通过 | `fix(otp): distinguish Redis outages from cooldowns` |
| P3-9 | 已修复，待发布；生产 Redis 故障注入待验收 | 版本 key 缺失仍为合法 `v0`；transport error/caller cancel 改为 unknown，`BuildVersionedKey` 返回空 key，版本化 get miss、set no-op且不写本地版本缓存。真实 miniredis 关闭/重启验证故障阶段不读写 `v0`、恢复后立即读到预存 `v7`；缓存全包 race、调用方两包 race、全服务端 race、lint/build/docs 通过 | `fix(cache): bypass versioned data when Redis is unavailable` |
| P2-16 | 已修复，待发布；目标库 NULL 分布与同批发布待验收 | 可空课程元数据使用指针/nullable contract，必有的 departmentID/credits 明确为 null，code/departmentName 缺失时省略；未分类 group 保留 null，credits sort NULLS LAST。开发库只读 23/0 NULL 盘点，隔离 PostgreSQL 覆盖五条读取路径；Go/TS generate、Web nullable parser、UniAppX fallback、三前端 type-check、两包/全服务端 race、spec/drift/lint/build/docs 通过 | `fix(course): preserve unknown catalog metadata` |
| P3-7 | 已修复，待发布；生产审计查询/留存待验收 | NDJSON/CSV 成功与请求取消各 1 次，真实 PostgreSQL 验证每个 request_id 恰好一条 `data.export`，包含 actor、normalized filter、row_count/row_limit 和 success/failure；取消后仍持久化。评课/全服务端 race、lint/build/docs 通过；未双写 operation log 或新增异步审计系统 | `fix(audit): record review export outcomes` |
| P2-12 | 已修复，待发布；真实 Oracle 池与生产限流指标待验收 | 两个 Oracle fan-out route 同一 subject 合计 5/min，第 6 次 429 且 gateway 计数不变；第二 subject 独立。miniredis outage 下 auth 先于 limiter，认证请求 503 且 gateway=0。academic-match 429/503 进入真源并完整生成；Admission/app/全服务端 race、spec/drift/type/lint/build/docs 通过 | `fix(admission): bound academic email lookups per user` |
| P2-3 | 逐消息热路径已修复，待发布；自动 retention 降为 P3 决策 | 开发库 0 行但旧计划仍为 scan+临时排序；cursor 下推保持原 limit，模型声明复合索引。隔离 SQLite 10,000/9,000 行验证索引实际落地、查询计划按 guild 搜索和最新三行顺序；8/8 定向、build/contracts、611/611 unit、startup、46/46 UI smoke 和 docs 通过。未臆定删除期限 | `fix(koishi): bound repeat ledger reads` |
| P2-20 | 已修复，待发布；生产刷新 p95/p99 与锁等待待验收 | 独立 60 秒预算只用于教师投影刷新，范围 5–90 秒且 parent cancel 优先；DB 执行路径复用现有 tracing/metrics 并保持不重试。真实 PostgreSQL 覆盖 20 ms 普通预算下的长 statement、deadline/cancel，以及 Repository 在 1 ns 普通预算下成功刷新；开发库两组 6 次为 4.040–20.517 ms、outbox 无失败。全服务端 race/lint/build、docs 与全部 infra contracts 通过；未抬高全局预算或增加无 SLO 的 cadence/age 平台 | `fix(review): separate teacher projection timeout` |

## Claude 原审计的确认问题分布（保留原始记录）

> 本节及后续“确认问题明细”是 Claude 的原始分类与取证草稿，不代表 Codex 复核后的级别、
> 唯一根因计数或修复方案。请先阅读上面的“Codex 独立复核”。

| 区域 | P0 | P1 | P2 | P3 | 合计 |
|------|----|----|----|----|------|
| 评课与内容 |  | 2 | 5 | 2 | 9 |
| Koishi 机器人 |  | 1 | 5 |  | 6 |
| Admin 前端 | 1 |  | 1 | 3 | 5 |
| 基础设施与运维 |  | 3 | 2 |  | 5 |
| 入群认证 |  | 1 | 4 |  | 5 |
| 文档 |  | 1 |  | 2 | 3 |
| CI/CD |  |  | 2 |  | 2 |
| OpenAPI 契约 |  |  | 1 | 1 | 2 |
| 外部数据源 |  |  | 2 |  | 2 |
| 后端公共包 |  |  | 1 | 1 | 2 |
| 开放平台 |  | 1 |  |  | 1 |
| 资源 |  | 1 |  |  | 1 |

## Claude 原审计的确认问题明细（保留原始记录）

### P0（1 项）

#### P0-1. Admin landing page requires admin:dashboard:view, so every non-super_admin role lands on a full-page 404 after login

- **位置**：`clients/admin/apps/web-ele/src/preferences.ts:12`
- **区域**：Admin 前端　**类别**：broken-core-flow　**验证票数**：2/2

**证据**

```
preferences.ts:12 → `defaultHomePath: '/analytics',`
router/routes/modules/dashboard.ts:24 → `authority: [ADMIN_DASHBOARD_VIEW],` on the `/analytics` route
api/core/user.ts:31 → `homePath: preferences.app.defaultHomePath,` (no role awareness)
router/routes/core.ts:34 → Root `path: '/'` has `redirect: preferences.app.defaultHomePath`
router/guard.ts:178-186 → `const redirectPath = (from.query.redirect ?? (to.path === preferences.app.defaultHomePath ? userInfo.homePath || preferences.app.defaultHomePath : to.fullPath)); return { ...router.resolve(decodeURIComponent(redirectPath)), replace: true };`
packages/utils/src/helpers/generate-routes-frontend.ts:14-16 → `const finalRoutes = filterTree(routes, (route) => hasAuthority(route, roles));` (unauthorized routes are REMOVED, not 403-ed, because none set `menuVisibleWithForbidden`)
server/internal/pkg/capability/catalog.go:38-73 → only `super_admin` is granted `AdminDashboardView`; `school_admin`/`section_admin`/`section_moderator` are not.
```

**失败场景**

A `school_admin` (capabilities: admin:reviews:manage, admin:reports:manage, user:student:read/review, user:school:read/update) signs in. `/auth/me` returns canAccessAdmin=true (admin:reviews:manage is in AdminEntryCapabilities), so the session is accepted. login.vue:20-27 sends the OIDC redirect to `preferences.app.defaultHomePath` = `/analytics`. The access guard generates routes, `/analytics` is filtered out because the user lacks `admin:dashboard:view`, so `router.resolve('/analytics')` falls through to the top-level catch-all `/:path(.*)*` → `views/_core/fallback/not-found.vue`. That route is registered outside BasicLayout, so the user gets a bare full-page 404 with no sidebar/menu, and the Fallback component's "back" button uses `homePath: '/'` (packages/effects/common-ui/src/ui/fallback/fallback.vue:20,121) which redirects to `/analytics` again — an infinite 404 loop. The admin is only reachable if the operator hand-types e.g. `/content/reviews`. Same for `section_admin` and `section_moderator`.

**修复方案**

Make the landing path capability-derived, and add a guard-level safety net. Do NOT "fix" it by widening `authority` on dashboard.ts or by adding `meta.menuVisibleWithForbidden`: `GET /admin/stats` is gated on GLOBAL `admin:dashboard:view` (server/internal/app/admin_authorizers.go:72 → `rbac.RequireGlobalCapability(capability.AdminDashboardView)`), so a school_admin would render the dashboard shell and then eat a 403 — a different broken page.

1. New helper `clients/admin/apps/web-ele/src/router/resolve-home-path.ts`: `resolveHomePath(capabilities: string[]): string`. Derive it from `accessRoutes` itself (walk the tree ordered by `meta.order`, reuse `hasAuthority` from `@vben/utils`, pick the first authorized leaf that has a `component` and is not `hideInMenu`) so it can never drift from the route table; fall back to `preferences.app.defaultHomePath` only when nothing matches.

2. `clients/admin/apps/web-ele/src/api/core/user.ts:31`: `homePath: resolveHomePath(me.capabilities)`. `mapMeToUserInfo` already receives the whole `MeResult`, so no signature change. Update `src/api/core/user.test.ts` accordingly.

3. `clients/admin/apps/web-ele/src/router/guard.ts:178-186`: after `accessStore.setIsAccessChecked(true)`, resolve the candidate first and reject a dead target — if `resolved.name === fallbackNotFoundRouteName` (or `resolved.matched` contains only the catch-all), retry with `userInfo.homePath`, then with the first leaf path found in `accessibleMenus`, and only surface the 4


### P1（10 项）

#### P1-1. ensureAdminCommandAccess returns "allowed" when the guild id is empty, and `群审复核` then dumps every guild's review queue

- **位置**：`bots/koishi/plugins/stuhelper-admin/src/command-access.ts:29`
- **区域**：Koishi 机器人　**类别**：authorization　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P1

**证据**

```
command-access.ts:27-33
```ts
  const targetGuildId = input.targetGuildId ?? session?.guildId
  const guildId = targetGuildId
  if (!session || !guildId) return          // <- undefined == access granted
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    store.getMemberRoles(guildId, session.userId),
  ])
```
`resolveGuildId` yields `''` off-guild (command-access.ts:45: `return guildId?.trim() || session?.guildId || ''`), and `registerReviewListCommand` passes it straight through (commands.ts:95-107) then calls `deps.moderationStore.listPendingReviews(targetGuildId)`. store.ts:173-177 treats an empty id as "all guilds":
```ts
  async listPendingReviews(guildId?: string) {
    const query = guildId ? { guildId } : {}
```
```

**失败场景**

An operator whose account has Koishi authority >= 3 (e.g. bound to the console admin, authority 5) but who is deliberately excluded from the `guardReviews` command policy DMs the bot `群审复核` with no argument. `session.guildId` is undefined in a private chat, so `targetGuildId` is `''`; `ensureAdminCommandAccess` returns before ever reading the `guardReviews` policy or the caller's member roles, and `listPendingReviews('')` returns the pending review queue for **every** guild — member IDs, action types and free-text reasons — instead of just the guilds the operator governs.

**修复方案**

Three edits; deliberately do NOT apply two parts of the original proposal.

1. `bots/koishi/plugins/stuhelper-admin/src/commands.ts` — `registerReviewListCommand` (lines 91-108). Load-bearing fix for the leak. After the `if (denial) return denial` block, mirror `registerWarningCommand:80` and reuse the existing message key (no schema change needed, `guardWarningMissingContext` = '请在群聊中执行，或显式传入群号和成员 ID。'):
```ts
      if (denial) {
        return denial
      }
      if (!targetGuildId) {
        return adminMessage(messages, 'guardWarningMissingContext')
      }
      return formatPendingReviews(await deps.moderationStore.listPendingReviews(targetGuildId), messages)
```

2. `bots/koishi/plugins/stuhelper-admin/src/command-access.ts` lines 27-33. Close the policy bypass so the policy is always authoritative. Replace the permissive early return with a deny-on-no-session plus an empty-roles evaluation (keeps `minAuthority` enforced off-guild instead of skipping the whole check, and avoids regressing the friendlier group-only messages that `registerBatchMuteCommand`/`createReviewRequest` emit after the access call):
```ts
  if (!session) {
    return renderMessageTemplate(resolveAdminMessages(input.messages).commandAccessDenied)
  }
  const guildId = (input.targetGuildId ?? session.guildId ?? '').trim()
  const [policy, memberRoles] = await Promise.all([
    store.getCommandPolicy(commandId),
    guildId ? store.getMemberRoles(guildId, session.userId) : Promise.resolve<string[]>(

#### P1-2. Migration guides instruct editing the baseline schema file, which silently drops schema changes on any migrated database

- **位置**：`docs/guides/database-migrations.md:20`
- **区域**：文档　**类别**：doc-accuracy　**验证票数**：2/2

**证据**

```
docs/guides/database-migrations.md:18-21
  - 当前项目按绿地 schema 管理，不保留增量迁移兼容链路。
  - 结构变更直接更新 `000001_initial_schema.up.sql`。
docs/guides/database-migrations.md:31  # 仅本地可用：删除当前 schema baseline
  DATABASE_URL='postgres://...' make migrate-down-one
docs/guides/backend-development.md:73-74
  - 结构变更直接更新 `server/migrations/000001_initial_schema.up.sql`
  - `server/migrations/000001_initial_schema.up.sql` 是唯一 schema 权威来源
docs/README.md:70  | 某张表的列 / 索引 | `server/migrations/000001_initial_schema.up.sql` |

Reality: server/migrations/ holds 000001 through 000019 (…000019_notification_idempotency.up.sql adds notifications.idempotency_key). server/Makefile:19 uses golang-migrate v4 and server/cmd/migrate-runtime calls m.Up(). docs/reference/database.md:11 says the opposite: "`000001_initial_schema.up.sql` 只是基线，不代表后续演进后的完整 schema".
```

**失败场景**

A backend dev follows docs/guides/backend-development.md:73 and adds a column by editing 000001_initial_schema.up.sql, then runs `make migrate-up` against dev/staging. schema_migrations already records version 19, so golang-migrate returns ErrNoChange and migrate-runtime exits 0 with no DDL applied. The developer sees a green migration, the column never exists, and the first query against it fails at runtime in every already-provisioned environment (dev, staging, production). The doc-recommended rollback `make migrate-down-one` is also described as "删除当前 schema baseline" but actually only reverts 000019.

**修复方案**

Docs-only change, plus one Makefile help correction. Do NOT "fix" this by regenerating 000001 into a full schema dump — 000017 and 000019 use bare ADD COLUMN without IF NOT EXISTS, so a fattened baseline would make fresh installs fail with duplicate-column and mark schema_migrations dirty.

1. docs/guides/database-migrations.md
   - Line 5 frontmatter: `authoritative-source: server/migrations/` (not the 000001 file). Bump last-verified.
   - Lines 13-14: describe 000001 as the initial baseline only, explicitly noting it does not reflect later evolution; keep 000001.down.sql as local-only baseline teardown.
   - Lines 19-21 维护规则, replace wholesale with: schema changes are added as the next numbered up/down pair (currently 000020_*); 000001 and every already-applied migration is immutable, because golang-migrate does not checksum migration files — editing an applied file is silently ignored (m.Up() returns ErrNoChange and migrate-runtime exits 0) so the change never reaches any provisioned database; every up must ship a matching down; prefer additive DDL and use IF NOT EXISTS / DROP CONSTRAINT IF EXISTS so re-runs are safe.
   - Line 34 comment and line 80 rollback text: change "删除当前 schema baseline" to "回滚最新一个 migration（仅本地）", and note it reverts one version only (today: 000019, which drops notifications.idempotency_key plus its constraint and unique index).
   - Lines 46, 49, 67: replace "应用当前 baseline schema" with "按版本顺序应用 server/migrations/ 中全部未应用的 migration".

2. docs/guid

#### P1-3. pg_basebackup invocation is unconditionally rejected, so physical base backups and PITR never exist

- **位置**：`infra/ops/backup-postgres.sh:63`
- **区域**：基础设施与运维　**类别**：backup-integrity　**验证票数**：2/2

**证据**

```
compose run --rm --no-deps -T \
    postgres-client \
    pg_basebackup \
      --dbname "${replication_url}" \
      --format=tar \
      --gzip \
      --wal-method=stream \
      --checkpoint=fast \
      --pgdata=- \
    >"$output_file"
```

**失败场景**

PostgreSQL refuses `-X stream` when tar output goes to stdout. Verified against the exact pinned image used in production (`cgr.dev/chainguard/postgres:latest@sha256:dc2f04037c1044a22af76cee4de70b9111885b17c561b939d7ed70103d100759`): `pg_basebackup: error: cannot stream write-ahead logs in tar mode to stdout` (exit 1). This is argument validation, so it fires before any connection — it can never succeed. Consequences: the weekly `stuhelper-postgres-basebackup.timer` installed by `infra/ops/install-backup-timers.sh:78-88` runs `run-scheduled-backup.sh basebackup`, which aborts at line 36 under `set -e`, so `backups/postgres/base/` only ever accumulates 0-byte `.tar.gz` files created by the `>` redirect (and the 15-minute sync timer then uploads those empty files to object storage). `infra/ops/restore-postgres-basebackup.sh` therefore has nothing to restore and PITR is impossible. Nothing catches it: `infra/ops/postgres-backup-evidence.sh:170-190` only inspects `${logical_dir}` for `*.dump` and never looks at the base directory, and `remote-preflight.sh:150-160` only warns about the logical directory, so `make prod-backup-evidence` and the deploy-time gate stay green.

**修复方案**

Three changes:

1. `infra/ops/backup-postgres.sh` — fix the invocation and stop leaving poisoned artifacts.
   In `run_basebackup()` (lines 53-67) replace `--wal-method=stream` with `--wal-method=fetch`, which is the only WAL method PostgreSQL permits with `--format=tar --pgdata=-` (verified: `fetch` passes arg validation on the pinned image; `stream` is fatally rejected). Keep `--format=tar --gzip --checkpoint=fast --pgdata=-` so `restore-postgres-basebackup.sh`'s `tar -xzf` stays compatible (fetch writes the required WAL into base.tar, which extracts into `pg_wal/`).
   Also make both `run_dump` and `run_basebackup` redirect to `"$output_file.partial"` and `mv` it into place only after the command succeeds (or `trap`-remove the partial on failure), so an aborted run never leaves a 0-byte `.tar.gz`/`.dump` in `backups/postgres/{base,logical}` for the 15-minute sync timer to upload. Optionally pass `--slot`/`--create-slot` (or ensure a non-zero `wal_keep_size`) so a long `fetch`-mode backup cannot lose WAL to recycling.

2. `infra/ops/postgres-backup-evidence.sh` — close the blind spot so this class of failure cannot stay green.
   Add `base_dir="${BACKUP_BASE_DIR:-${REPO_ROOT}/backups/postgres/base}"`, call the fetcher for the base artifacts as well as `logical`, and run the existing `verify_sha256_sidecar` + a freshness check (e.g. mtime within `BACKUP_BASE_RETENTION_DAYS`/8 days, matching the weekly `Sun 03:45` timer) against `latest_file "${base_dir}" '*.tar.gz'` both loc

#### P1-4. Deploy bundle strips .env.example / .env.prod.example, which every remote deploy and rollback requires

- **位置**：`infra/ops/build-deploy-bundle.sh:33`
- **区域**：基础设施与运维　**类别**：deployment-correctness　**验证票数**：2/2
- **级别修正**：验证方将 P0 修正为 P1

**证据**

```
tar \
    --exclude='.git' \
    --exclude='.claude' \
    --exclude='.run' \
    --exclude='.tools' \
    --exclude='.env*' \
    --exclude='.env' \
... (GNU tar matches --exclude patterns against unanchored path suffixes, so '.env*' also drops '.env.example' and '.env.prod.example')
```

**失败场景**

Verified empirically. (1) tar behaviour: building a fixture tree with `.env.example`, `.env.prod.example`, `normal.txt` and running the exact exclude list produces an archive containing only `./normal.txt` — both template files are gone. (2) Consumers require them: `infra/ops/lib/common.sh:753-761` `ensure_env_file()` does `[[ -f "${template_file}" ]] || die "missing env template: ${template_file}"` unconditionally (before it checks whether ENV_FILE already exists), and ENV_TEMPLATE_FILE defaults to `${REPO_ROOT}/.env.example` (common.sh:7); `.deploy/remote.env` written by `init-remote-deploy-config.sh` never sets ENV_TEMPLATE_FILE. (3) `infra/ops/validate-runtime-image-scan.py:229-230` calls `parse_env_file(repo_root / env_path)` for every image whose `env_files` is `[".env.example", ".env.prod.example"]` (all 17 registry images), and `parse_env_file` raises PolicyError on OSError. Reproduced: a repo root containing only `infra/security/runtime-images.json` + the validator gives `[runtime-image-scan][error] cannot read /tmp/bundleroot/.env.example: [Errno 2] No such file or directory` (exit 1). `infra/ops/bootstrap-ubuntu2404.sh:58-84` only `install -d`s directories and `touch`es

**修复方案**

Three coordinated edits.

1) /home/wztxy/Code/StuHelper/infra/ops/build-deploy-bundle.sh — delete line 33 (`--exclude='.env*' \`). The nine explicit exclusions already present at lines 34-42 (`.env`, `.env.generated`, `.env.generated.secrets`, `.env.prod.local`, `.env.prod.shared`, `.env.prod.secrets`, `.env.prod.secrets.local`, `.env.prod.generated`, `.env.prod.generated.secrets`) cover every real root runtime env file. Add the ones the enumeration is missing, all of which are gitignored and therefore invisible to the clean-worktree gate:
    --exclude='.env.casdoor-bootstrap.local' \
    --exclude='.env.local' \
    --exclude='**/.env.local' \
    --exclude='**/.env.development.local' \
    --exclude='**/.env.production.local' \
Do NOT try to re-add the templates as extra tar operands — GNU tar applies --exclude to command-line names too, so that silently does nothing.

2) Same file, after the `tar` subshell (before `mv "${tmpfile}" "${OUTPUT_FILE}"` at line 63) — add a fail-closed contents assertion so this can never regress in either direction:
  bundle_entries="$(tar -tzf "${tmpfile}")"
  for required in ./.env.example ./.env.prod.example; do
    grep -Fxq -- "${required}" <<<"${bundle_entries}" \
      || die "deployment bundle is missing required env template: ${required}"
  done
  if leaked="$(grep -E '(^|/)\.env($|\.)' <<<"${bundle_entries}" | grep -v '\.env\.example$' | grep -v '\.env\.prod\.example$')"; then
    die "deployment bundle contains real env files: ${lea

#### P1-5. Calendar-expiring image pin reviews hard-block production deploy and make rollback to older releases impossible

- **位置**：`infra/security/runtime-images.json:109`
- **区域**：基础设施与运维　**类别**：release-engineering　**验证票数**：2/2

**证据**

```
"id": "CASDOOR_DEV",
      "image": "casbin/casdoor:latest@sha256:d7658640...",
      "scope": "optional loopback-only local identity-provider development fixture",
      "pin_review": {
        "verified_on": "2026-07-29",
        "review_by": "2026-08-05",   <-- 6 days from today (2026-07-30)
```

**失败场景**

`infra/ops/validate-runtime-image-scan.py:236-246` enforces `require(end >= today, f"{end_field} expired on ...")` for every moving-tag pin inside `validate_policy`, i.e. also under `--policy-only`. Both `infra/ops/prod-deploy.sh:38-41` and `infra/ops/remote-preflight.sh:56-59` run `--policy-only`. Reproduced: `validate-runtime-image-scan.py --policy-only --today 2026-08-13` → `[runtime-image-scan][error] images[0].pin_review.review_by expired on 2026-08-12` (exit 1). Two concrete failures: (a) On 2026-08-05 the *development-only* Casdoor fixture pin lapses and every production deploy aborts, even though `casdoor` is `profiles: [dev-full]` and never runs in prod (7 pins total lapse by 2026-08-12; the CASDOOR_DEV exception at line 299-302 expires the same day). (b) `.github/workflows/rollback.yml:52-56` checks out `inputs.commit_sha` (the OLD release commit) and `build-deploy-bundle.sh` packages that commit's `infra/security/runtime-images.json`. Because `max_pin_review_days` is capped at 30, rolling back to any release older than ~30 days is permanently blocked: during an incident, `prod-rollback.sh` → `prod-deploy.sh:38` fails with an expired-review error and there is no override

**修复方案**

Split the calendar-freshness control away from the deploy-time immutability control.

1. `infra/ops/validate-runtime-image-scan.py`: add `--review-windows {enforce,report}` (default `enforce`). Thread an `enforce_freshness: bool` through `validate_policy` into `validate_review_window` and make ONLY the `require(end >= today, ...)` assertion at line 110 conditional — when `report`, print `[runtime-image-policy][warn] {end_field} expired on {end}` to stderr and continue. Keep every other assertion unconditional in both modes: `start <= today`, `end >= start`, the `(end - start).days <= maximum_days` cap, `DIGEST_REF_RE` immutability, the `.env.example`/`.env.prod.example` exact-match check, `upstream_evidence`, and `validate_effective_environment`.

2. `infra/ops/prod-deploy.sh:38-41` and `infra/ops/remote-preflight.sh:56-59`: append `--review-windows report`. Keep `--policy-only --effective-environment production` so digest immutability and the production env-match gate still fail closed and `infra/ops/tests/runtime-image-security-contract.sh:135-136` still passes.

3. Keep `enforce` (the default) in `infra/ops/scan-runtime-images.sh:74-77` and `:144-148`, i.e. the CI `runtime-image-security` job stays the hard gate for stale pins.

4. 原方案建议给 CI 增加每日 schedule，使日历窗口失效不再只在部署时暴露；原始英文在此处
   截断，没有给出完整告警时机和旧提交回滚兼容方案。

**Codex 修复与复验（2026-07-30）**

原方案把 `prod-deploy.sh` 和 `remote-preflight.sh` 一律切到 report-only，会让所有新生产部署
继续使用已经失效的漏洞例外/VEX，也没有解决 GitHub workflow checkout 旧提交后实际执行旧
validator 的问题，因此没有采纳。最终实现保持默认和普通生产部署的当前日期硬门禁，只给
历史成功回滚增加显式证据路径：

- validator 新增 `--minimum-review-days-remaining` 作为定时预警门槛；默认仍为 0，不改变
  deploy 的 fail-closed 语义。每日独立 workflow 要求至少还剩 3 天，避免为此触发整套 CI。
- rollback 先按今天正常校验。只有失败后，才读取目标环境
  `.deploy/releases/<target-tag>.env`；目标 tag、三个应用 digest 和记录必须完全一致，
  release 的 `DEPLOYED_AT` 必须是过去时间，且完整 policy 在该日有效。当前生产基础镜像仍由
  `--effective-environment production` 逐字匹配；任何非日历错误在历史日期重验时仍失败。
- 例外要求合法操作人、12–500 字符理由和控制器生成的 UUID，并在部署前向
  `.deploy/rollback-review-exceptions.jsonl` 追加 mode 0600 的 JSONL 事件，记录 policy
  SHA256、目标 tag、三个应用 digest 和当前门禁失败原因。它表示“例外已授权/尝试”，不伪称
  部署成功；最终成功仍由既有 release record 和 smoke 记录。
- GitHub workflow 以 `github.workflow_sha` checkout 当前可信控制器，同时把目标 release
  checkout 到独立目录。最新打包脚本从目标 clean HEAD 创建 bundle；远端只覆盖
  `prod-rollback.sh` 和 validator 两个控制文件，所以已经发布的旧提交也能获得修复，而不会
  用当前应用源码替换目标 release。`reason` 是必填输入，并以 base64 安全传到远端 shell。
- 2026-08-13 普通校验按预期因窗口过期失败；同环境 2026-07-30 成功记录的严格证据路径通过。
  应用 digest 漂移和无意义理由均失败。隔离控制器测试确认当前 policy 不创建例外记录；
  过期路径的审计 ID 贯穿到部署子进程且日志权限为 0600。显式历史 source worktree 打包继续
  拒绝 dirty tree。ShellCheck、actionlint、文档卫生和全部 76 个 infra contracts 通过。

**验证边界**：没有运行真实 GitHub protected environment、SSH/SCP 或生产目标机回滚，故状态
是“实现已修复、待发布与远端演练”，不能称生产回滚能力已经验收。

#### P1-6. Bot action SSE stream has no shutdown release, so every SIGTERM stalls 30s and the process exits 1

- **位置**：`server/internal/modules/admission/handler_bot_queries.go:137`
- **区域**：入群认证　**类别**：graceful-shutdown　**验证票数**：2/2

**证据**

```
handleStreamBotAdmissionActions loops forever with the request context as its only exit:

```go
	ticker := time.NewTicker(2 * time.Second)
	defer ticker.Stop()
	keepalive := time.NewTicker(15 * time.Second)
	defer keepalive.Stop()

	for {
		select {
		case <-c.Request.Context().Done():
			return
		case <-keepalive.C: ...
		case <-ticker.C:
			if !h.writeQueuedAdmissionActions(c, filter) { return }
		}
	}
```

Unlike the notification stream, nothing releases it at shutdown. `server/internal/app/modules.go:78` registers the *only* shutdown hook in the process — `rt.addShutdownHook(notifHub.Stop)` — and `admissionHandler` (modules.go:205-206) registers none. `server/internal/app/server.go:56-59` documents the intent that is not met here: "先停止后台取件并释放 SSE 等长连接… http.Server.Shutdown 本身不会主动取消活动中的长连接处理器". No `http.Server.RegisterOnShutdown` / `BaseContext` is set anywhere (`grep -rn 'RegisterOnShutdown|BaseContext' internal/` → no hits). Go 1.26 `net/http/server.go:3150` Shutdown only polls `closeIdleConns()`; it never cancels an active handler's context. I reproduced it with a minimal server whose handler waits on `r.Context().Done()`: `Shutdown returned after 3s err=context deadline exce
```

**失败场景**

A Koishi bot holds GET /api/v1/bot/admission/actions/stream open (bots/koishi/packages/shared/src/platform/index.ts:45 dials exactly this path). Operator sends SIGTERM for a rolling deploy. server.go:52 computes shutdownTimeout = DB_QUERY_TIMEOUT(5)*3 + 15s = 30s. `srv.Shutdown` cannot drain because the SSE handler is still active, so after 30s it returns context.DeadlineExceeded → server.go:60 logs "Server forced to shutdown" → server.go:71 `errors.Join` returns a non-nil error → app.Run returns it → cmd/stuhelper/main.go:19-20 prints "Application error" and calls os.Exit(1). Result: every clean shutdown takes 30 extra seconds and reports failure (non-zero exit → k8s/compose treats a normal deploy as a crash). Worse, during those 30s Shutdown has already closed the listeners while the SSE handler keeps claiming actions every 2s and pushing them to the bot; the bot performs the kick/release in QQ but its POST /api/v1/bot/admission/actions/{id}/events ack is refused (listener closed, idle keep-alive conns closed), so the action stays `dispatched`, is re-claimed 30s later on the new instance, and the same user is kicked/released twice.

**修复方案**

Give the admission handler a shutdown release symmetric to `notification.Hub`, reusing the `bgCtx` the runtime already cancels first in `beginShutdown`.

1. `server/internal/modules/admission/handler.go`: add `streamStop <-chan struct{}` to the `Handler` struct plus an option `WithStreamShutdown(ctx context.Context) HandlerOption` that stores `ctx.Done()`. Leave it optional — a nil channel blocks forever, preserving current behavior for `handler_errors_test.go:15` and `handler_user_test.go:176`.

2. `server/internal/app/modules.go:199`: pass `admission.WithStreamShutdown(bgCtx)` to `admission.NewHandler`. `router.go:37-39` already creates `bgCtx` and `runtime.go:172-175` cancels it before `shutdownHTTPServer` runs, so no new lifecycle wiring is needed. (Do NOT set `srv.BaseContext` to that context instead — it would cancel every in-flight ordinary request mid-write and defeat draining.)

3. `server/internal/modules/admission/handler_bot_queries.go:137` and `server/internal/modules/admission/handler_user.go:215`: add `case <-h.streamStop:` to both select loops, emitting `c.SSEvent("end", "shutdown")` + `c.Writer.Flush()` before `return` so the bot sees a clean close and reconnects to the new instance rather than logging a stream error.

4. 原方案还建议给 bot stream 增加固定 10 分钟最大寿命。Codex 不采纳：客户端定期重连是另一项
   可靠性策略，不能替代进程 shutdown 信号；在没有代理 idle-timeout、负载均衡或连接老化证据时
   强制断开所有正常流，属于与本缺陷无关的行为变更。

**Codex 修复与复验（2026-07-30）**

- `Handler` 新增可选 `WithStreamShutdown(context.Context)`，只保存 `ctx.Done()`；未传 option
  时 nil channel 永久阻塞，现有单元构造和非应用装配行为不变。
- 应用装配把现有 `bgCtx` 传入 handler。`Runtime.beginShutdown()` 已先调用 `bgCancel()`，
  再执行 hooks 和 `http.Server.Shutdown`，因此没有新增 lifecycle、goroutine 或全局
  `http.Server.BaseContext`。
- bot action 与 camera handoff 两个 SSE loop 都监听停机 channel，写出 `end: shutdown` 后返回；
  初始事件后以及 ticker/keepalive/timeout 分支实际工作前再次检查，避免多个 case 同时 ready
  时继续 claim action 或查询状态。
- 新增真实 PostgreSQL + `httptest.Server` 回归。bot 流建立后取消应用 context，再调用真实
  `http.Server.Shutdown`，2 秒 deadline 内成功返回并读到 `event:end`/`data:shutdown`；
  camera handoff 流也在同一窗口主动结束。定向 race、admission 全包、`go vet` 和全服务端
  `golangci-lint`（0 issues）通过。
- 修复没有增加 bot 10 分钟 timer、没有取消普通请求、没有改 SSE/OpenAPI 契约。原标题的
  “every SIGTERM”仅在存在这些活跃 SSE 时成立；没有在已发布二进制或 Compose 中发送真实
  SIGTERM，故生产滚动更新仍是待验收边界。

#### P1-7. GET /course/review/reviews/{reviewID}/replies has no optional-auth middleware, so isOwner is always false and users lose the delete button on their own replies

- **位置**：`server/internal/modules/course/review/handler.go:116`
- **区域**：评课与内容　**类别**：correctness　**验证票数**：2/2

**证据**

```
handler.go:116 — no auth middleware at all:
	r.GET("/reviews/:reviewID/replies", h.GetReplies)

compared with the sibling read routes, e.g. handler.go:95:
	r.GET("/courses/:courseID/reviews", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetCourseReviews)

review_read/review_reply.go:21 calls h.resolveOptionalUserHash(c), which is request_identity.go:26-30:
	userID := middleware.GetUserID(c)
	if userID == "" { return "", true }

middleware.GetUserID reads only CtxKeyUserID, which is set exclusively by setClaimsToContext in the auth middleware (pkg/middleware/auth_context.go:122,150). service_interaction.go:434-437 then never sets ownership:
	if params.UserHash != "" { list[i].IsOwner = list[i].UserHash == params.UserHash }

The group is created without .Use(): internal/app/modules_course_metrics.go:74 `api.Group("/course/review")`.
```

**失败场景**

A logged-in student posts a reply (POST .../replies returns Reply{IsOwner: true}, so the delete button appears). They reload the page; the client calls GET /api/v1/course/review/reviews/{id}/replies. GetUserID returns "" because no auth middleware ran, so every reply comes back with isOwner:false. clients/web/src/components/business/review/ReplyCard.vue:13 renders the delete control under `v-if="reply.isOwner"`, so the user can never delete their own reply after a refresh — even though DELETE /replies/{replyID} would succeed. The contract encodes the same bug: server/api/openapi.bundled.yaml:4152 declares `security: []` for getReplies while requiring `isOwner` in the response, whereas optional-auth endpoints use `security: [{}, cookieAuth: [], bearerAuth: []]` (line 3687).

**修复方案**

1. server/internal/modules/course/review/handler.go:116 — put the replies list route on the same optional-auth chain as its siblings:
   `r.GET("/reviews/:reviewID/replies", optionalAuthMiddleware, middleware.RequireHealthyOptionalAuth(), h.GetReplies)`
   This makes `setClaimsToContext` populate CtxKeyUserID for authenticated callers, so `resolveOptionalUserHash` returns a hash and service_interaction.go:436 sets IsOwner correctly; anonymous callers still pass through (auth.go:178-182) and get isOwner:false.

2. server/api/paths/review-reply.yaml:6 — replace `security: []` on getReplies with the optional-auth triple used by getCourseReviews (review-crud.yaml:6-9):
   security:
     - {}
     - cookieAuth: []
     - bearerAuth: []
   Add `'503': $ref: '../components/responses/common.yaml#/ErrorResponse'` to getReplies' responses, since RequireHealthyOptionalAuth can now return 503 (auth.go:243).

3. Regenerate the contract with `cd server && make generate`. The 503 response changes the generated
   TypeScript response union, while the Go output updates its embedded bundled spec. Do not hand-edit
   `openapi.bundled.yaml`, `api.gen.ts`, or `internal/api/gen`.

4. Add a route-level regression test that registers the real route graph with an instrumented optional-auth
   middleware and checks owner, authenticated non-owner, anonymous, and auth-backend-unavailable requests.

**Codex 修复与复验（2026-07-30）**

- OpenAPI GET replies 的 security 现为 `{}`、`cookieAuth`、`bearerAuth` 三种替代方案，并声明
  optional auth 健康门禁可能返回 503；`make generate` 更新 bundled spec、Go 内嵌契约和
  TypeScript 503 response union，没有手改生成代码。
- 真实路由接入 `optionalAuthMiddleware` 和 `RequireHealthyOptionalAuth()`。匿名请求仍然公开；
  携带有效身份时，既有 `resolveOptionalUserHash`/Service 比较链恢复 `isOwner`，删除端点的
  强制认证与所有权校验没有改变。
- 新增真实 PostgreSQL route-level 回归：同一回复列表中，owner 仅拥有自己的回复，另一登录
  用户和匿名用户均不拥有；optional auth 后端故障在 Handler 前返回 503。定向普通测试、
  定向 race、review 全包、`go vet`、全服务端 `golangci-lint`（0 issues）、spec lint、
  `make check-drift` 和文档卫生检查通过。
- 没有增加 ownership 查询、没有把公开 GET 改成强制认证、没有让前端根据本地用户信息猜测
  owner，也没有改数据库或回复模型。原 P1 等级过高；它是刷新后操作入口丢失的 P2 UX/契约
  正确性问题，不是权限绕过、数据泄漏或删除授权失效。

#### P1-8. ProcessReport applies review status changes with no state-transition guard, letting a user-deleted review be resurrected and later republished

- **位置**：`server/internal/modules/course/review/service_report.go:188`
- **区域**：评课与内容　**类别**：correctness　**验证票数**：2/2

**证据**

```
service_report.go:182-198 (the "hide" branch) reads the current status but never validates the transition:
	case "hide":
		reportStatus = ReportStatusResolved
		currentStatus, currentCourseID, currentTeacherID, err := s.repo.GetReviewStatusCourseTeacherTx(ctx, tx, report.ReviewID)
		if err != nil { return err }
		if err := s.repo.UpdateReviewStatus(ctx, tx, report.ReviewID, StatusHidden); err != nil { return err }

The sibling entry point for the same mutation does guard it — service_admin.go:27-31 / 161-168:
	var validTransitions = map[string]map[string]bool{
		"hide":    {StatusPublished: true, StatusPendingReview: true},
		"restore": {StatusHidden: true, StatusPendingReview: true},
		...
	if !allowed[currentStatus] { return "", fmt.Errorf("%w: cannot %s from %s", ErrInvalidTransition, action, currentStatus) }

repository_review_query.go:162 has no status filter:
	_, err := tx.Exec(ctx, `UPDATE reviews SET status = $2, updated_at = NOW() WHERE id = $1`, reviewID, status)
and reviewCoreFieldsBaseQuery (repository_review_query.go:15) selects by id only.
```

**失败场景**

1) User A publishes a review; User B reports it (ReportReview requires status='published', so the report row exists with status=pending). 2) User A deletes their own review → reviews.status='deleted'. 3) A moderator calls PUT /admin/reports/{reportID} with action="hide". ProcessReport only checks `report.Status != ReportStatusPending`, so it proceeds and flips the review from 'deleted' to 'hidden'. 4) Any moderator then calls PUT /admin/reviews/{reviewID} with action="restore": validTransitions["restore"] permits StatusHidden, so the review is set back to 'published' and IncrementCourseReviewCount runs — content the user deliberately deleted is publicly visible again and counted in the course review total. The direct PUT /admin/reviews path correctly rejects both hide-from-deleted and restore-from-deleted.

**修复方案**

Primary fix in server/internal/modules/course/review/service_report.go, inside the ProcessReport transaction (lines 179-216):

1. Hoist the review-state read out of the switch so both "hide" and "delete" read `currentStatus, currentCourseID, currentTeacherID` from GetReviewStatusCourseTeacherTx once (it already takes FOR UPDATE), and wrap pgx.ErrNoRows as ErrReviewNotFound for consistency with applyAdminReviewActionTx.

2. For action "hide": if `currentStatus == StatusDeleted`, do NOT touch the review at all — set `reportStatus = ReportStatusResolved` and fall through to UpdateReport. The reported content is already gone, so the report is legitimately resolved; this preserves the author's deletion. For every other source status, call `validateAdminReviewTransition("hide", currentStatus)` (service_admin.go:161) and return its ErrInvalidTransition error unchanged so respondProcessReportError (http_errors.go:182) maps it to the same 4xx the direct PUT /admin/reviews path returns.

3. For action "delete": same shape — if `currentStatus == StatusDeleted`, skip SoftDeleteReview and just resolve the report (avoids pointless updated_at churn); otherwise validate with validateAdminReviewTransition("delete", currentStatus).

4. 原方案还建议在 Repository 增加 `status <> 'deleted'` 作为 defense in depth。Codex
   **不采纳这一项**：Repository 当前只返回 `error`，无法区分“成功更新”与“条件不匹配的
   0 rows affected”，会把状态竞争变成静默成功；而且它不能约束 `SoftDeleteReview`，也不能
   统一 `hidden` 等其他非法源状态。正确边界是事务内 `FOR UPDATE` 读取后，由 Service 明确
   校验和决定是否变更。

**Codex 修复与复验（2026-07-30）**

- 已在 `ProcessReport` 的同一事务内对 `hide`/`delete` 只读取一次 review 状态；底层查询已有
  `FOR UPDATE`。不存在的 review 映射为 `ErrReviewNotFound`，HTTP 层新增对应 404 映射。
- 对非 `deleted` 状态直接复用 `validateAdminReviewTransition`，因此举报入口与直接管理员入口
  使用同一份转换白名单；非法转换会回滚，report 保持 `pending`。
- 对 `deleted` 特意不调用 `UpdateReviewStatus` 或 `SoftDeleteReview`，只把待处理 report
  结案为 `resolved`。这是必要的业务例外：内容已经不可见，举报已达到处置目的，但作者删除
  仍是 review 的不可逆终态。
- 新增真实 PostgreSQL 集成回归：先创建举报，再走真实作者删除服务，最后分别执行 `hide` 和
  `delete`。两条路径均确认 review 仍为 `deleted`、`updated_at` 未变化、课程计数不二次扣减、
  report 正常记录处理人/备注/时间；后续 `restore` 返回 `ErrInvalidTransition`。另覆盖
  “其他入口已经隐藏后再次 hide”必须失败且 report 保持 pending。
- 定向测试、定向 race 测试、整个 `internal/modules/course/review` 包、`go vet`、全服务端
  `golangci-lint`（0 issues）和文档卫生检查均通过。修复没有修改 OpenAPI、迁移、
  Repository SQL 或状态集合，也没有引入幂等框架/新锁，因此不属于过度设计。

#### P1-9. Open-platform phone.read disclosure can never succeed: users.phone_enc holds a masked phone, but the disclosure path requires 11 consecutive digits

- **位置**：`server/internal/modules/openplatform/service_disclosure.go:598`
- **区域**：开放平台　**类别**：integration-gap　**验证票数**：2/2

**证据**

```
service_disclosure.go:594-604
	phone, err := s.phoneCipher.Decrypt(projection.PhoneEnc)
	...
	normalized, ok := normalizeCasdoorMainlandPhone(phone)
	if !ok {
		return fmt.Errorf("%w: phone projection is unavailable", ErrDisclosureUnavailable)
	}
	out["phone"] = normalized

with `var mainlandPhoneDigitsPattern = regexp.MustCompile(`1[3-9]\d{9}`)` (line 14).

The only writer of users.phone_enc encrypts the MASKED value — server/internal/modules/user/service_phone.go:29-32:
	func (s *Service) buildPhoneProjection(phone string) (string, []byte, string, error) {
		trimmed := strings.TrimSpace(phone)
		masked := phoneutil.Mask(trimmed)
		phoneEnc, err := s.docCipher.Encrypt(masked)

Both sides use the same cipher (internal/app/modules.go:307 `user.NewService(userRepo, crypto.GetHMACKey(), piiCipher, ...)` and internal/app/modules_openplatform.go:39 `openplatform.WithPhoneDecryptor(piiCipher)`), and openplatform/repository_projection.go:18 reads `u.phone_enc` from the same `users` table. `grep -rn "phone_enc"` confirms no other writer.
```

**失败场景**

用户绑定 `13812345678` 后，本地 `users.phone_enc` 实际保存
`Encrypt("138****5678")`。已获批并取得 consent 的第三方调用 phone API 时，旧实现解密出的
仍是掩码，无法匹配 11 位手机号，因此稳定返回 `ErrDisclosureUnavailable`/503；OIDC
identity-token 的 `phone` scope 复用同一 payload builder，也同样失败。Profile completion 只看
`PhoneVerified`，而非明文可用性，所以无法提前发现。OpenAPI 本来就同时声明 200 与 503，
因此不是“只承诺 200”的契约遗漏；真实缺陷是合法、已完成资料的用户也没有可达的成功披露
路径。旧 denied/`payload_unavailable` 审计符合当时的错误结果，但持续记录的是集成缺口，而非
真实 provider 故障。

**修复方案**

Implement the documented real-time authority read without moving Casdoor identity into a business module:

1. Open Platform 的 provider 接口只接受内部 `userID`。`UserProjection` 只带内部 ID 和
   `phone_enc IS NOT NULL` 派生的验证状态，不加载/解密 `phone_enc` 内容。
2. app 组合根的 gateway 先调用 user repository 的 `GetCasdoorSubject(userID)`，再调用既有
   `platform/casdoor.UserProfileClient.GetPhone`。Casdoor subject 只存在于允许的 user/app/platform
   边界，不进入 `internal/modules/openplatform`。
3. 本地未验证仍沿用 `phoneVerified:false`；一旦本地标记为已验证，reader 缺失、内部身份无效、
   subject 解析失败、Casdoor 调用失败、返回空值或格式无法标准化都返回
   `ErrDisclosureUnavailable`。不能把 Casdoor 空手机号降级成成功响应。
4. 覆盖 phone API 与 OIDC identity-token 两条调用链，以及成功/失败审计、provider 故障和
   app adapter 身份解析失败。

**Codex 修复与复验（2026-07-30）**

- 删除 Open Platform 的 `phoneDecryptor`/`WithPhoneDecryptor` 和 projection 中的 `PhoneEnc`；
  新增 authoritative reader，只按内部 user ID 获取手机号。本地投影继续只表达验证状态。
- 复用启动时已经创建的 Casdoor user-profile gateway；gateway 在 app 边界用
  `userRepo.GetCasdoorSubject` 解析外部身份，再调用现有 `UserProfileClient.GetPhone`。这与当前
  Casdoor Go SDK 的 `GetUserByUserId`/`User.Phone` 能力一致，没有新增 SDK 或 HTTP 客户端。
- 真实 PostgreSQL/Redis 集成测试故意把掩码字节写入本地 `phone_enc`，reader 返回另一份
  `+86` 完整号码；phone API 和 identity-token 都只返回 reader 的标准化号码并写 granted
  审计。随后 provider 报错，payload 不返回、错误保持 `ErrDisclosureUnavailable` 并写 denied
  审计。单元测试另覆盖本地未验证、reader 未配置、内部 ID 缺失、空值和非法格式。
- app adapter 测试验证内部 ID 到 subject 的映射和 trim；映射失败时 Casdoor client 调用数为
  0。定向普通/race、Open Platform 全包、app/Casdoor 全包、`go vet`、Casdoor boundary guard
  与全服务端 `golangci-lint`（0 issues）通过。
- 原方案要求把 `CasdoorSubject` 加入 Open Platform projection，违反
  `check-casdoor-boundary.sh` 的 IAM 边界，Codex 不采纳；“Casdoor 无手机号时返回 200”也与
  `security-model.md` 的 fail-closed 约束冲突。没有落库完整手机号、没有缓存 authoritative
  响应、没有新增身份服务或更改 OpenAPI。尚未使用真实 Casdoor 凭据/网络执行发布环境验收。

#### P1-10. Resource list issues 2 nested queries per row while the outer cursor still holds a pooled connection (N+1 + pool starvation)

- **位置**：`server/internal/modules/resource/repository.go:228`
- **区域**：资源　**类别**：efficiency　**验证票数**：2/2

**证据**

```
func (r *Repository) scanItems(ctx context.Context, rows pgx.Rows) ([]Item, int, error) {
	items := make([]Item, 0)
	total := 0
	for rows.Next() {
		item, err := scanItem(rows, &total)
		...
		item.Tags, err = r.loadTags(ctx, item.ID)         // r.db.Query -> acquires another pool conn
		...
		item.Bindings, err = r.loadBindings(ctx, item.ID) // r.db.Query -> acquires another pool conn
```

**失败场景**

GET /api/v1/resources?pageSize=100 (public, optionalAuth, handler.go:30) -> ListResources (repository.go:62) holds one pooled connection for the outer cursor for the entire scan and issues 2 more queries per row = 200 extra round trips. Confirmed in pgx v5 pgxpool/pool.go: Pool.Query does p.Acquire(ctx) and only releases the connection on Rows.Close(), so every request needs 2 pool connections at once. With DB_MAX_CONNS default 20 (config.go:370), 20 concurrent /resources requests each pin a connection on their outer cursor, then every loadTags() blocks in Acquire until the 5s DB_QUERY_TIMEOUT expires -> all requests return 500 'failed to list resources' and the entire DB pool is deadlocked for 5 seconds. GetResourceByID, UpdateResource and CreateResource all funnel through the same scanItems.

**修复方案**

Rewrite `scanItems` in /home/wztxy/Code/StuHelper/server/internal/modules/resource/repository.go (lines 220-299) as a drain-then-batch, mirroring the pattern already used in /home/wztxy/Code/StuHelper/server/internal/modules/openplatform/repository_apps.go (ListApps + loadScopeRequests/loadRedirectURIRequests).

1. Drain the outer cursor first, collecting ids and an index map, and check `rows.Err()` BEFORE issuing any nested query (so we never attach children to a partially-read result set):

```go
func (r *Repository) scanItems(ctx context.Context, rows pgx.Rows) ([]Item, int, error) {
	items := make([]Item, 0)
	ids := make([]int64, 0)
	index := make(map[int64]int)
	total := 0
	for rows.Next() {
		item, err := scanItem(rows, &total)
		if err != nil {
			return nil, 0, err
		}
		index[item.ID] = len(items)
		ids = append(ids, item.ID)
		items = append(items, item)
	}
	if err := rows.Err(); err != nil {
		return nil, 0, err
	}
	// Cursor is exhausted here, so pgxpool has already returned the outer
	// connection (pgxpool/rows.go: Next() closes on !n). The batch loads below
	// therefore reuse a single pool connection instead of needing a second one.
	if err := r.attachTags(ctx, ids, items, index); err != nil {
		return nil, 0, err
	}
	if err := r.attachBindings(ctx, ids, items, index); err != nil {
		return nil, 0, err
	}
	return items, total, nil
}
```

2. Replace `loadTags` (line 260) and `loadBindings` (line 278) with batched versions keyed on `resource_id` (they have no ot

**Codex 独立复核与处置（2026-07-30）**

- **事实核心确认，原生产结论降级。**pgx v5 官方 API 说明 `Pool.Query` 取得的连接会在
  `Rows.Close()` 后归还，`Next()` 返回 false 也会自动关闭 rows。原实现确实在外层 rows
  尚未耗尽、连接仍被占用时逐行调用 `loadTags` 和 `loadBindings`，因此查询数为
  `1 + 2N`，并且每个请求在扫描期间需要再取得一个连接。连接池饱和时会形成等待乃至超时。
  但本轮没有生产并发、池占用、接口 p95 或 5 秒超时记录，不能把“20 个请求必然让整个池
  死锁并全部 500”写成已发生事实；该项维持真实问题，级别由 P1 调整为 P2。
- **采用最小修复。**`scanItems` 先完整扫描主结果并收集 resource IDs，检查迭代错误后显式
  `Close()`；随后分别使用 `resource_id = ANY($1::bigint[])` 批量加载 tags 与 bindings，
  并按资源 ID 回填。每个 Item 在扫描时初始化非 nil 空切片，保持 OpenAPI 必填数组语义；
  两个批量查询继续显式排序，保持原先单行加载的 tag、binding 顺序。空结果不发关联查询。
- **真实数据库验证。**测试在迁移后的 PostgreSQL 上建立 `MaxConns=1` 的独立 pgxpool。
  分页返回 1 条与 2 条数据时，连接获取增量都严格为 3；`GetResourceByID` 也是 3，证明
  SQL 数量不随行数增长。6 个并发列表请求共用该单连接池仍全部在 deadline 内成功；该场景
  在旧实现中首个请求会持有唯一连接并等待自己的关联查询。另验证 tags/bindings 排序和
  无关联资源返回 `[]` 而非 `null`。
- **回归结果。**新增定向普通测试与 `-race` 均通过，资源模块全包、`go vet`、Casdoor
  boundary guard 和全服务端 `golangci-lint`（0 issues）通过。未修改 schema、OpenAPI、
  Handler 或 Service。
- **过度设计判断。**不引入 ORM、DataLoader、通用预加载器、缓存或新索引。当前两个窄范围
  batch helper 已将该路径固定为三次查询并解除嵌套连接获取；是否还需要合并成单条 JSON
  聚合 SQL，应只在发布后的真实 p95/数据库指标证明有必要时再评估。


### P2（23 项）

#### P2-1. No CI path filter covers scripts/ or tools/, so the custom Semgrep security rules can be changed with zero CI

- **位置**：`.github/workflows/ci.yml:63`
- **区域**：CI/CD　**类别**：ci-coverage-gap　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
docs:
              - 'AGENTS.md'
              - 'README.md'
              - 'docs/**'
              - 'scripts/check-docs-hygiene*'
...
  sast:
    if: >-
      github.event_name == 'workflow_dispatch' ||
      needs.changes.outputs.clients == 'true' ||
      needs.changes.outputs.backend == 'true' ||
      needs.changes.outputs.workflows == 'true'
```

**失败场景**

The seven filters (ci.yml:42-76) match only `server/**`, `clients/**`, `bots/koishi/**`, `docs/**`, `infra/**`, `.github/**`, `Makefile`, `docker-compose*.yml`, `.env.example`, `.env.prod.example`, `.gitleaksignore`, `AGENTS.md`, `README.md` and `scripts/check-docs-hygiene*`. Nothing matches `tools/**`, `scripts/lib/**`, `scripts/check-semgrep-custom-rules.sh`, `scripts/check-vue-ui-contracts.mjs`, `.node-version` or `.nvmrc`. Concretely: a PR that deletes the `stuhelper.go.raw-phone-log-field` rule from `tools/semgrep/stuhelper-security.yml` (the only file `scripts/check-semgrep-custom-rules.sh:5` scans `server/internal` with) sets every filter output to `false`, so `repository-policy`, `backend`, `contract`, `clients`, `client-e2e`, `koishi`, `infra`, `runtime-image-security` and `sast` all skip. `required` (ci.yml:652-659) only fails on `*failure*`/`*cancelled*`, so `CI / Required` — the sole non-CodeQL required check per docs/guides/github-migration.md:128 — reports success and the weakened security rule is never executed. Same for `scripts/lib/docs-hygiene-lib.mjs`, which holds all of `validateDocsTree()` while only `scripts/check-docs-hygiene.mjs` matches the `docs` filter; a

**修复方案**

Two changes, both mechanical.

A) `.github/workflows/ci.yml` — make guard code and the toolchain pin trigger the jobs that execute them.
1. Add a `guards` output at line 30 area: `guards: ${{ steps.filter.outputs.guards }}` (alongside `backend`..`workflows`, lines 24-30).
2. Add a `guards` filter in the `filters:` block (lines 42-76):
   ```
   guards:
     - 'scripts/**'
     - 'tools/**'
     - '.node-version'
     - '.nvmrc'
   ```
3. Extend the `docs` filter (line 59-63) with `- 'scripts/lib/**'` so `repository-policy` fires when `validateDocsTree()` itself changes (keep the existing `scripts/check-docs-hygiene*` line).
4. Add `needs.changes.outputs.guards == 'true'` to the `if:` of `repository-policy` (ci.yml:81-85) and `sast` (ci.yml:598-603). `sast` is the only executor of `scripts/check-semgrep-custom-rules.sh`, so this is the line that actually closes the `tools/semgrep/**` hole.
5. Add `.node-version` and `.nvmrc` to the `clients` (45-48), `contract` (49-58) and `koishi` (71-73) filters — every one of those jobs resolves Node via `node-version-file: .node-version`, so a major bump must build and test.

B) `infra/ops/tests/ci-and-drift-contract.sh` — assert the coverage so a future guard cannot be added outside a filter. Next to the existing `assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- '\.github/\*\*'$"`, add:
   ```
   assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- 'scripts/\*\*'$"
   assert_contains "${GITHUB_CI_FILE}" "^[[:space:]]+- 'tools/\*\*'$"


**Codex 独立复核与处置（2026-07-30）**

- **问题确认，但原文“零 CI”不准确。**`secret-scan` 没有 `if`，只改 `scripts/**` 或
  `tools/**` 时仍会运行；真正缺失的是消费这些文件的相关门禁。尤其
  `tools/semgrep/**` 和 `scripts/check-semgrep-custom-rules.sh` 不触发 SAST，
  `scripts/lib/docs-hygiene-lib.mjs` 不触发 repository-policy，
  `scripts/check-vue-ui-contracts.mjs` 不触发 clients/Koishi。Required 会把 job-level skip
  当作非失败，这是 GitHub Actions 的正常语义，不能依靠聚合 job 猜测某个 skip 是否合理。
- **采用按消费者分类的最小修复。**`changes` 新增 `guards` output，统一覆盖
  `scripts/**`、`tools/**`、`.node-version` 和 `.nvmrc`；repository-policy 与 SAST
  接入该 output。与此同时，文档卫生库只进入 docs filter，Vue UI contract 只进入
  clients/Koishi filter，Node pin 进入实际使用 Node 的 repository-policy、clients、
  contract、Koishi 与 infra 路径。没有把 root scripts 变化接到 backend、E2E 等所有重型任务。
- **补上可执行契约。**`ci-and-drift-contract.sh` 不再只检查某个路径字符串是否在 YAML
  任意位置出现，而是提取各 filter 和 job block，逐项断言 output、路径和 `if` 接线；
  同时要求 `.node-version` 与 `.nvmrc` 内容非空且完全一致，防止只改兼容入口却仍用旧版本
  跑 CI。
- **验证与边界。**CI wiring contract、Bash 语法和 actionlint 均通过；ShellCheck 只报告
  文件既有的单引号 `$uri` SC2016 info，新增代码无告警。文档卫生检查通过。由于本地不能
  模拟 GitHub PR 的 dorny/paths-filter 事件，本项仍需由本提交对应的真实 GitHub Actions run
  验收调度结果。P2-2 的 always-on 静态供应链契约与 Koishi packaging 接线单独处理，避免
  把两个审计编号混成一个不可追溯提交。

#### P2-2. The infra filter omits the Dockerfiles and Koishi sources that infra contracts assert, so supply-chain pinning gates skip the very change they guard

- **位置**：`.github/workflows/ci.yml:64`
- **区域**：CI/CD　**类别**：ci-coverage-gap　**验证票数**：1/1

**证据**

```
infra:
              - '.env.example'
              - '.env.prod.example'
              - '.github/**'
              - 'Makefile'
              - 'docker-compose*.yml'
              - 'infra/**'
```

**失败场景**

`infra/ops/tests/run-infra-contracts.sh` (only run by the `infra` job) executes 74 contracts that assert on files outside `infra/**`: `dockerfile-supply-chain-contract.sh:23-44` requires digest-pinned `ARG *_IMAGE=` and rejects mutable base tags in `server/Dockerfile`, `server/Dockerfile.dev`, `clients/web/Dockerfile` and `clients/admin/scripts/deploy/Dockerfile`; `ci-and-drift-contract.sh:111-142` asserts on those Dockerfiles plus `clients/.dockerignore`, `clients/package.json`, `clients/pnpm-workspace.yaml` and `server/Makefile`; `koishi-stuhelper-package-contract.sh` validates `bots/koishi` packaging. A PR that changes `clients/web/Dockerfile` from `ARG NGINX_IMAGE=nginx:1.30.4-alpine@sha256:...` to `nginx:latest` matches only the `clients` filter, so the `clients` job runs lint/type-check/test/build (which never touch Docker) while `infra` is skipped — the mutable-base-image gate does not run and `CI / Required` is green. The violation then surfaces later, on an unrelated `infra/**` PR by a different author. Same for `bots/koishi/**` changes versus the Koishi packaging contract.

**修复方案**

Apply the finding's second option, not the filter-extension option.

1. In .github/workflows/ci.yml, add a new job with NO `if:` gate on `needs.changes` (both scripts use only grep/sed/find — no apt, pnpm, Playwright, or Koishi setup — so this costs ~20s):

  static-contracts:
    name: Static file contracts
    runs-on: ubuntu-latest
    steps:
      - name: Check out the repository
        uses: actions/checkout@d23441a48e516b6c34aea4fa41551a30e30af803 # v6
        with:
          persist-credentials: false
      - name: Assert supply-chain pinning and CI wiring
        run: |
          bash infra/ops/tests/dockerfile-supply-chain-contract.sh
          bash infra/ops/tests/ci-and-drift-contract.sh

Leave run-infra-contracts.sh untouched; these two re-running inside the infra job costs milliseconds and keeps `make check-infra-contracts` complete for local use.

2. Add `- static-contracts` to the `required` job's `needs:` list (ci.yml:637-649) so a failure turns `CI / Required` red. Without this the new job is advisory only, since `required` is what branch protection observes.

3. Extend the `infra` filter at ci.yml:64 with `'scripts/**'` and `'bots/koishi/**'`. `scripts/**` is the bigger hole: scripts/check-secrets.sh, check-semgrep-custom-rules.sh and check-uniappx-shadow-files.sh currently match no filter whatsoever, so a PR touching only one of them runs zero gated jobs. `bots/koishi/**` is needed because koishi-stuhelper-package-contract.sh exercises infra/ops/package-ko

**Codex 独立复核与处置（2026-07-30）**

- **问题确认。**`dockerfile-supply-chain-contract.sh` 与 `ci-and-drift-contract.sh` 的确读取
  `server/`、`clients/` 等 infra filter 之外的文件，旧 workflow 只在 infra job 中运行它们；
  因此仅把 Web/Admin/Server Dockerfile 改成 mutable base tag 时，消费该约束的 job 会 skip。
  `koishi-stuhelper-package-contract.sh` 同样只在 infra 全量合约中执行，而 Koishi 源码变化只
  触发 Koishi job。现有 Required 只聚合 job result，不可能判断这些 skip 是否遗漏了消费者。
- **静态约束改为不可跳过。**新增没有 `needs: changes`、没有 job-level `if` 的
  `static-contracts`，每次 workflow 都 checkout 并运行 Dockerfile supply-chain 与 CI wiring
  两个纯静态脚本；`required.needs` 显式加入该 job，失败会传递到现有 branch-protection
  聚合检查。保留它们在 `run-infra-contracts.sh` 中，保证本地 `make check-infra-contracts`
  仍是完整集合。
- **构建型约束放在已有消费者。**Koishi job 的 `corepack yarn test` 已先执行 workspace
  build；随后直接运行 deployable package contract，验证本地 workspace、编译产物、浏览器
  dist、归档 manifest 和 secret 排除。没有把 `bots/koishi/**` 加入 infra filter，避免每次
  机器人源码变化再重复安装 Web Playwright、浏览器依赖并执行全部生产运维合约。
- **契约防回归。**`ci-and-drift-contract.sh` 验证 static job 同时调用两个脚本、没有
  changes 条件、已进入 Required，并验证 Koishi job 调用了 package contract。因此后续误删
  任一接线会由 always-on job 自身失败。
- **验证与边界。**Dockerfile supply-chain、CI wiring、Koishi package 三个定向合约及
  actionlint 通过；完整 `run-infra-contracts.sh` 的 75 个 shell contract 与 1 个 mjs contract
  全部通过。未触发真实 GitHub runner，所以 job 的实际排队、容器内工具可用性和远端 Required
  check 仍须由 PR CI 验收。没有修改 Dockerfile、发布产物或运行环境。

#### P2-3. Repeat detection loads a guild's entire message ledger into memory on every message, and the ledger is never pruned

- **位置**：`bots/koishi/packages/moderation-core/src/store.ts:84`
- **区域**：Koishi 机器人　**类别**：efficiency　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
store.ts:83-86
```ts
  async listRecentMessages(guildId: string, limit: number) {
    const records = await this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { guildId })
    return records.sort(sortByCreatedDesc).slice(0, limit)
  }
```
Called once per message from message-guard.ts:232 (`const records = await this.deps.store.listRecentMessages(input.guildId, moderation.repeatWindowSize)`), right after message-guard.ts:63 inserts a new row via `saveMessage`. The model has no index on `guildId` and no ordering/limit pushdown (models.ts:88-101, `{ primary: 'messageId' }`), and the ledger content columns are `'text'`. A repo-wide grep for `database.remove` finds deletions only for keyword rules and reports — there is no retention job for `moderation_message_ledger`.
```

**失败场景**

An operator enables moderation in the Koishi WebUI (`moderationEnabled`). In a busy 2000-member QQ group at ~5k msgs/day, after two months the ledger holds ~300k rows for that guild. Every subsequent incoming message then executes `SELECT * FROM moderation_message_ledger WHERE guildId = ?` with no index, materializes ~300k JS objects (each carrying full `content` + `normalizedContent`), sorts them, and throws away all but `repeatWindowSize` (default ~10). Message handling latency grows linearly with history until the Koishi process stalls and eventually OOMs, taking down admission reminders and the action stream with it.

**修复方案**

Three changes, all in the koishi workspace.

1) Push ordering + limit into the query — bots/koishi/packages/moderation-core/src/store.ts:83-86. Use the cursor overload (not `select()`) so the store's dependency surface stays `database.get`, keeping the existing test double in store.test.ts:34-46 valid:

  async listRecentMessages(guildId: string, limit: number) {
    return this.ctx.database.get(MODERATION_MESSAGE_LEDGER_TABLE, { guildId }, {
      sort: { createdAt: 'desc' },
      limit,
    }) as Promise<MessageLedgerRecord[]>
  }

Keep `limit` exactly as passed (do NOT use limit + 1): message-guard.ts:233-234 filters the just-saved current message out of the window, so today's effective window is limit-1, and changing that would silently shift repeat-detection thresholds. Drop the now-unused JS sort for this method only (`sortByCreatedDesc` is still used by listRecentEvents).

2) Add the supporting index — bots/koishi/packages/moderation-core/src/models.ts:88-101, third argument of `ctx.model.extend`:
  { primary: 'messageId', indexes: [{ keys: { guildId: 'asc', createdAt: 'desc' } }] }
@minatojs/driver-sqlite creates this on migration (createIndex, lib/index.cjs:558); the live DB currently has only the primary-key autoindex.

3) Add config-driven retention. Add `ledgerRetentionDays` to
`GroupGuardModerationSettings` in
`bots/koishi/packages/shared/src/guard/behavior-settings.ts`。**Claude 更新后的原报告在此处截断，
没有给出完整实现、保留期限依据或迟到撤回语义。**

#### Codex 对 P2-3 的最终复核与实施记录（2026-07-31）

- **结论拆分。**
  - 逐消息热路径缺陷成立：`saveMessage` 后每条消息都会调用
    `listRecentMessages`；旧实现读取该群全部文本行、在 JS 排序后才截断，且没有
    `guildId/createdAt` 索引。这一部分按 P2 修复。
  - 原报告的规模与 OOM 影响没有仓库运行证据。只读检查当前开发
    `bots/koishi/data/koishi.db` 得到文件 204 KiB、账本 0 行、0 个 guild；它只能证明当前
    样本尚未累积数据，不能反向证伪热路径。原文“5k msgs/day、两个月 300k 行直至 OOM”
    是风险推演，不是实测事实。
  - “从不清理”是独立的容量/产品策略问题。账本还被
    `handleMessageDeleted -> getMessage(messageId)` 用于撤回取证；仓库没有权威的最长撤回
    到达期、审计留存期或生产表规模。此时加入固定天数或 WebUI 配置会擅自改变迟到撤回语义，
    因此降为 P3 决策项，不能为了把报告标成 fixed 而删除历史数据。
- **最小实现。**
  - `listRecentMessages` 继续使用既有 `database.get`，但将
    `{ sort: { createdAt: 'desc' }, limit }` 传给 Minato cursor，数据库只返回窗口内记录。
  - 模型声明 `(guildId ASC, createdAt DESC)` 复合索引；锁定版本的 Minato
    `prepareIndexes()` 会在启动准备阶段为已有/新建 SQLite 表补齐索引。
  - `limit` 原样保留，没有改成 `limit + 1`：调用方随后排除刚保存的当前消息，改上限会扩大
    既有复读窗口并改变阈值语义。
  - 开发指南记录热路径、索引和 retention 前置决策；没有新增设置字段、控制台页面、定时器、
    第二张表或清理框架。
- **交叉验证。**
  - 变更前对开发库执行计划为
    `SCAN stuhelper_moderation_message_ledger` + `USE TEMP B-TREE FOR ORDER BY`。
  - 隔离 Koishi SQLite 启动后实际生成
    `index:stuhelper_moderation_message_ledger:guildId+createdAt`；插入 10,000 行、其中目标群
    9,000 行后，相同查询改为
    `SEARCH ... USING INDEX ... (guildId=?)`，且 `LIMIT 3` 返回 `9000, 8999, 8998`。
  - 单元测试锁定 cursor 参数和模型索引声明，并刻意让数据库桩返回非时间顺序，证明 store
    不再把全量结果拉回 JS 重排。复读/撤回真实 SQLite 定向测试共 8/8 通过。
  - Koishi 全工作区 build、Vue UI contracts、611/611 unit、startup smoke、46/46
    Playwright UI smoke 与文档卫生通过。生产表规模、p95/p99 和索引迁移锁等待仍须发布后
    观测，不能由 10,000 行隔离样本替代。

#### P2-4. Dashboard overview loads the whole moderation-event and guard-member tables to compute counters

- **位置**：`bots/koishi/packages/moderation-core/src/store.ts:61`
- **区域**：Koishi 机器人　**类别**：efficiency　**验证票数**：1/1

**证据**

```
store.ts:60-63
```ts
  async listRecentEvents(limit = 20) {
    const records = await this.ctx.database.get(MODERATION_EVENT_TABLE, {})
    return records.sort(sortByCreatedDesc).slice(0, limit)
  }
```
store.ts:356-363
```ts
    const [events, reviews, reports, warnings, guards] = await Promise.all([
      this.listRecentEvents(20),
      ...
      this.ctx.database.get(GUARD_MEMBER_TABLE, {}),
    ])
```
`appendEvent` (store.ts:53-58) inserts a row for every join, kick, warn, keyword hit, repeat hit, recall and report, and nothing ever deletes from `MODERATION_EVENT_TABLE`. `buildAdmissionRuntimePageData` similarly calls `deps.guardStore.listActive()` and `deps.moderationStore.listAllKeywordRules()` unbounded (admission-console-api.ts:107-114).
```

**失败场景**

After a few months of normal operation the moderation event table holds hundreds of thousands of rows (every `guild-member-added` writes one). Opening the StuHelper console dashboard triggers `getOverview`, which SELECTs the entire event table plus the entire guard-member table (including long-since released/kicked rows) into JS objects and sorts them, just to show a 20-row 'recent events' list and four integers. The console request takes seconds to tens of seconds and spikes RSS by hundreds of MB, and repeated refreshes can OOM the bot process.

**修复方案**

Five concrete changes; items 1-2 are the finding's real substance, 3-4 replace its incorrect suggestions.

1. bots/koishi/packages/moderation-core/src/store.ts:60-63 — push the sort and limit into the driver:
   async listRecentEvents(limit = 20) {
     return this.ctx.database.get(MODERATION_EVENT_TABLE, {}, {
       sort: { createdAt: 'desc' },
       limit,
     }) as Promise<ModerationEventRecord[]>
   }
   Keep the `sortByCreatedDesc` helper — `listRecentMessages` (store.ts:83-86) still uses it; apply the same cursor treatment there (`{ guildId }` + `sort`/`limit`), since message-guard.ts:232 calls it on EVERY message and the ledger grows per message (message-guard.ts:63) with no retention. That hot path is worse than the dashboard one.

2. bots/koishi/packages/moderation-core/src/models.ts:52-67 — add an index so the new ORDER BY/LIMIT does not degenerate into a full scan + sort in SQLite:
   }, { primary: 'id', indexes: [{ keys: { createdAt: 'desc' } }] })
   Do the same for MODERATION_MESSAGE_LEDGER_TABLE on `{ guildId: 'asc', createdAt: 'desc' }`.

3. bots/koishi/plugins/stuhelper-core/src/core/api/page-api-runtime.ts:181-184 — filter at the DB instead of in JS, mirroring GuardMemberStore.listActive:
   async function listActiveGuardMembers(ctx: Context) {
     return ctx.database.get(GUARD_MEMBER_TABLE, { releasedAt: null, kickedAt: null }) as Promise<GuardMemberRecord[]>
   }
   This stops the 30s dashboard poll from materializing every historical released/kicked r

#### P2-5. Batch-mute payload parser silently drops all but the first member id when ids are space-separated, and the error hint names a command that does not exist

- **位置**：`bots/koishi/plugins/stuhelper-admin/src/commands.ts:320`
- **区域**：Koishi 机器人　**类别**：correctness　**验证票数**：1/1

**证据**

```
commands.ts:315-330
```ts
function parseBatchMutePayload(payload: string | undefined) {
  const source = (payload || '').trim()
  ...
  const [secondsText, memberIdsText] = source.split(/\s+/, 2)
```
JavaScript's `String.prototype.split(sep, limit)` truncates the result array — it does not append the remainder — so `'600 111 222 333'.split(/\s+/, 2)` is `['600','111']`. `parseMemberIds` (commands.ts:308-313) nonetheless splits on `/[\s,，]+/`, i.e. it is written to accept whitespace-separated ids. The hint the operator is shown on a parse failure points at a non-existent command: `guardBatchMuteInvalidPayload: '请提供禁言秒数和成员 ID 列表，例如：群审批量禁言 120 10001,10002'` (packages/shared/src/message-template.ts:36) while the registered command is `'群审禁言 <payload:text>'` (commands.ts:112).
```

**失败场景**

An operator runs `群审禁言 600 10001 10002 10003` (whitespace-separated, which `parseMemberIds` implies is supported). `memberIdsText` is `'10001'`, so only member 10001 is muted and only one `action_executed` moderation event is written; 10002 and 10003 are never muted and no warning is emitted. If the operator instead gets the format wrong, the bot tells them to type `群审批量禁言 …`, which resolves to no command at all.

**修复方案**

Three changes, all in tracked source (no generated files touched):

1. `bots/koishi/plugins/stuhelper-admin/src/commands.ts:320` — stop truncating the payload. Replace

    const [secondsText, memberIdsText] = source.split(/\s+/, 2)

with

    const [secondsText, ...rest] = source.split(/\s+/)
    const memberIdsText = rest.join(' ')

Behavior is preserved for every currently-working input (`'120 10011,10012'` still parses to two ids) and for the reject paths: payload `'600'` gives `rest === []` -> `memberIdsText === ''` -> `parseMemberIds` returns `[]` -> `null` -> `guardBatchMuteInvalidPayload`, and a non-numeric first token still fails the `Number.isInteger` check at 322. `parseMemberIds`'s existing `/[\s,，]+/` split then makes `600 10001 10002`, `600 10001, 10002`, and `600 10001，10002` all resolve fully, so its `\s` branch stops being dead code. No change needed at the call site (128-158).

2. `bots/koishi/packages/shared/src/message-template.ts:36` — point the hint at the real command and show both accepted separators:

    guardBatchMuteInvalidPayload: '请提供禁言秒数和成员 ID 列表，例如：群审禁言 120 10001,10002（也可用空格分隔）',

3. `bots/koishi/plugins/stuhelper-core/client/components/SettingsView.vue:1248` — update the duplicated WebUI default to the identical string, otherwise an operator who opens and saves the settings form re-persists the old wrong hint into `AdminRuntimeSettingsStore`.

Regression test: extend `bots/koishi/plugins/stuhelper-admin/src/index.test.ts` alongside the existin

#### P2-6. Console listeners are never disposed, so privileged admission actions stay callable after the plugin is unloaded

- **位置**：`bots/koishi/plugins/stuhelper-group-guard/src/admission-console-api.ts:91`
- **区域**：Koishi 机器人　**类别**：lifecycle　**验证票数**：1/1

**证据**

```
admission-console-api.ts:91-103 registers three listeners and discards the registration entirely:
```ts
  ctx.console.addListener(ADMISSION_RUNTIME_ACTION_EVENT, async function (input) {
    return handleAdmissionRuntimeAction(ctx, deps, input, this as ConsoleActionClient)
  }, { authority: CONSOLE_AUTHORITY })
```
The installed console implements `addListener` as a bare assignment with no disposer (node_modules/@koishijs/console/lib/index.js:242): `addListener(event, callback, options) { this.listeners[event] = { callback, ...options }; }`. Neither this file nor `stuhelper-core/src/core/api/page-api.ts:30-44`, `governance-actions.ts:62/69`, `review-actions.ts:28/36` register a `ctx.on('dispose', ...)` or `ctx.effect` to unregister. `console` is an *optional* injection (index.ts:46-49), and cordis only restarts scopes for **required** services (`if (!runtime.inject[name]?.required) continue` — @cordisjs/core/lib/index.cjs:459/470), so the stuhelper plugins are never reloaded when the console is.
```

**失败场景**

An admin disables `stuhelper-group-guard` from the Koishi WebUI plugin list. The scope is disposed (streams closed, `ctx.on` handlers removed), but `stuhelperGroupGuard/action/admission-member` remains in `console.listeners` bound to the disposed plugin's closure. Any authority-4 console client can still invoke `skip` / `release-blacklist` / `regenerate`, executing real backend admission mutations and `muteGuildMember` calls through the dead plugin's `platform` client and `guardStore`, while the WebUI reports the plugin as off. The closure also pins the whole plugin graph in memory. Symmetrically, reloading the console plugin creates a fresh `listeners` object and permanently kills the StuHelper console pages until the stuhelper plugins are manually reloaded.

**修复方案**

Two-part fix; part 1 is the high-value one.

PART 1 — make console a required injection for the registration sub-scope (fixes the reload/never-registered half, matches the pattern stuhelper-core already uses).
In `bots/koishi/plugins/stuhelper-group-guard/src/index.ts:150`, replace the bare call with:
```ts
ctx.inject(['console'], (consoleCtx) => {
  registerAdmissionConsoleAPI(consoleCtx, { config, platform, runtimeSettings, behaviorSettings, messageProvider, guardStore, policyStore, moderationStore, admissionSubjectCoordinator, onRuntimeSettingsChanged: () => actionStreams.refresh() })
})
```
This mirrors `plugins/stuhelper-core/src/setup/register-console-api.ts:27` and `register-console-entry.ts:11`, and makes the `if (!ctx.console) return` bail at `admission-console-api.ts:87-89` dead code — delete it (or keep as a type narrow only). Effect: the three `stuhelperGroupGuard/*` listeners are re-registered whenever the console service reappears, so reloading `@koishijs/plugin-console` from the WebUI config editor no longer leaves the admission page permanently answering `{"error":"not implemented"}`.

PART 2 — add a real disposer so the listeners die with the scope (fixes the stale-privileged-action half).
Add a shared helper, e.g. `bots/koishi/packages/shared/src/console/disposable-listener.ts`:
```ts
export function addDisposableConsoleListener<K extends keyof ConsoleEvents>(
  ctx: Context, event: K, callback: ConsoleEvents[K], options?: DataService.Options,
) {
  ctx.effe
```

> Claude 原修复方案在 `ctx.effe` 处截断；上面的残缺片段仅为原始取证记录，不是最终实现。

**Codex 独立复核与处置（2026-07-30）**

- **问题真实，P2 与“必须修复”判断合适。**仓库安装的 Console `addListener` 只覆盖
  `console.listeners[event]`，本身不返回 disposer；原代码既没有 `ctx.effect`，Group Guard
  又只把 `console` 声明为顶层 optional service。插件作用域被释放后，authority-4 listener
  仍可持有并调用旧的 mutation closure；Console 服务重建后，原作用域也不会因此可靠地重注册。
  这不是未授权提权：调用者仍需 authority 4，所以不升为 P1；但“插件显示已停用而特权动作仍可
  调用”属于真实生命周期与运维边界缺陷，不能按低价值优化驳回。
- **采用了两部分最小修复。**Group Guard 在
  `stuhelper-group-guard/src/index.ts` 中用 `ctx.inject(['console'], ...)` 建立 required
  registration 子作用域，服务出现时注册、消失或替换时释放/重建。StuHelper Core 原本已有
  required `console/database/stuhelperGroupCenter/auth` 子作用域，因此保留该结构，只修正其
  listener 的释放行为。
- **所有相关注册现在跟随作用域释放。**Core 的 page、review、governance、blacklist 及其他
  WebSocket API listener 统一经过 `createAuthority4ListenerRegistrar(ctx)`；Group Guard 的
  page/action/settings 三个 listener 经过本地 registrar。两个 registrar 都在 `ctx.effect`
  内注册，并捕获 Console registry 中本次写入的 registration。
- **清理必须做 identity check。**disposer 只有在 `console.listeners[event]` 仍严格等于
  自己捕获的 registration 时才删除。若服务重载或新作用域已覆盖同名 event，旧作用域随后释放
  不会把新 listener 一并删掉。Core registrar 还把 `authority: 4` 放在 options 展开之后，
  防止调用方通过 options 意外降低特权门槛。
- **没有采用原文建议的共享包抽象。**Group Guard 与 Core 的 Console event 类型分别由不同
  模块扩展，当前只有两个明确注册边界；在各自边界放一个很小的 registrar 比新增跨包通用框架
  更直接。实现没有 monkey-patch Console、没有改 `node_modules`，也没有维护第二份全局 listener
  registry。
- **验证结果。**新增单测模拟同 event 的两个作用域，确认先释放旧作用域不会删除新 registration，
  再释放新作用域后 registry 为空；另测调用方传入 `authority: 1` 仍实际注册为 4。Koishi
  build、Vue UI contracts、595/595 unit tests、startup smoke 与 deployable package contract
  通过。首次完整 `yarn test` 的 WebUI 阶段在无关配置治理导航用例发生一次 5 秒时序失败
  （45/46），在全新临时实例立即复跑为 46/46。
- **验证边界。**startup smoke 和 WebUI E2E 证明完整 Koishi 实例可启动且真实 Console actions
  可调用，但测试没有在保持 authority-4 WebSocket 客户端连接的同时卸载 Group Guard、重载
  Console 服务并再次调用旧/新 action。故本地实现与模拟生命周期已验证，发布环境的真实
  Console 热重载/插件卸载仍标记为 pending，不能声称已完成运行时热重载验收。

#### P2-7. One unforwardable freshman application permanently blocks all later material forwards

- **位置**：`bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts:471`
- **区域**：Koishi 机器人　**类别**：correctness　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
member-guard.ts:469-476
```ts
    const items = await this.deps.platform.listPendingFreshmanForwards()
    const messages = await this.getMessages()
    for (const item of items) {
      const bot = resolveFreshmanForwardBot(forwardBots, item)
      await forwardFreshmanMaterial(bot, item, messages)
      await this.deps.platform.markFreshmanForwarded(item.application.id)
    }
```
No try/catch inside the loop, and `scanPendingMembers` (member-guard.ts:215) does not guard the call either — the only catch is the scheduler's `.catch(...)` in events.ts:59-61. Both `resolveFreshmanForwardBot` (freshman-forward.ts:18/22) and `forwardFreshmanMaterial` (freshman-forward.ts:31/34/50) throw. The backend returns the queue oldest-first and only clears items on ACK: `repository_bot_scan.go:96-100` → `WHERE app.status = 'pending' AND app.forwarded_at IS NULL ... ORDER BY app.created_at ASC`.
```

**失败场景**

`forward_raw_material_to_qq` and `freshmanForward.enabled` are turned on. The oldest pending application belongs to a policy whose `management_guild_ids` includes a group the bot was since removed from, so `bot.sendMessage(guildID, ...)` throws and `forwardFreshmanMaterial` raises an `AggregateError` before `markFreshmanForwarded` runs. `forwarded_at` stays NULL, so this item is returned first on every subsequent scan and throws again — every newer freshman application behind it is never forwarded to the review group, and the only symptom is one 'group guard scheduled scan failed' log per scan interval. Freshman review silently stops for all applicants.

**修复方案**

Isolate each queue item WITHOUT swallowing the batch error, in `forwardFreshmanMaterials` at /home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/member-guard.ts:471-475:

```ts
const failures: unknown[] = []
for (const item of items) {
  try {
    const bot = resolveFreshmanForwardBot(forwardBots, item)
    await forwardFreshmanMaterial(bot, item, messages)
    await this.deps.platform.markFreshmanForwarded(item.application.id)
  } catch (error) {
    failures.push(error)
    this.deps.logger.warn('group guard freshman forward failed', {
      applicationID: item.application.id,
      error: formatAdmissionActionError(error), // already imported at line 21
    })
  }
}
if (failures.length === 1) throw failures[0]
if (failures.length) throw new AggregateError(failures, 'freshman forward batch failed')
```

Rethrowing the single failure unchanged keeps freshman-forward.test.ts:73-88 and :109-118 green (Node matches a RegExp validator against `String(error)`, so the original message must survive) while every other application in the batch is still attempted. Add a regression test with an old poison item plus a newer healthy item asserting the healthy one is sent and marked.

Note this only stops starvation *within* a batch; the poison item is still re-served first on every tick and still logs each interval. To actually drain the queue, add server-side failure tracking: a `forward_attempt_count`/`last_forward_error`/`last_forward_attempt_at` set of columns on

> Claude 原修复方案在 `set of columns on` 处截断；上面的 schema 扩展不是完整设计，也不是本轮
> 最小修复的前置条件。

**Codex 独立复核与处置（2026-07-30）**

- **核心问题确认，但触发条件必须写准确。**只有后端 policy 的
  `forward_raw_material_to_qq` 和 Koishi runtime 的 `freshmanForwardEnabled` 都开启时才进入
  该链路；当前默认值和运行文档均为关闭，因此不能描述成已发生的生产事故。功能开启后，后端按
  `created_at` 升序返回最多 100 条 `pending + forwarded_at IS NULL` 记录，旧实现遇到第一条
  `resolveFreshmanForwardBot`、URL 校验或发送失败就退出，后续项确实完全不尝试。
- **最小修复只隔离 delivery 阶段。**每个 item 的 bot 解析和对全部管理群的发送放在独立
  `try/catch` 中；失败时记录 `applicationID`、`phase=delivery` 和规范化错误，保留该 item
  未 ACK 并继续后项。成功项仍严格在所有目标群发送成功后调用
  `markFreshmanForwarded`。批次处理完后，单错误原样重抛、多错误用 `AggregateError` 重抛，
  因此 scheduler 仍产生 error 级信号，不会把坏项静默伪装成整批成功。
- **ACK 与 delivery 必须分相。**独立代理指出，把 ACK 网络/后端失败也当作 item-local continue
  会在控制面不可确认时继续发送其余最多 99 条，使下轮重复范围从一条放大到整批。最终实现对
  ACK 失败记录 `phase=ack` 后立即 fail-fast，保持旧行为的安全边界；如果此前已有 delivery
  failures，则把它们与 ACK 错误聚合后退出。
- **单 poison 不再卡住跨窗口队列。**失败项每轮仍占一个位置，但同批健康项会 ACK，下一轮新的
  健康项可进入剩余窗口，因此原报告的“一条坏记录永久阻断所有后续记录”已解除。没有通过 ACK
  坏项、吞掉错误或无界并发来换取表面推进。
- **回归与交叉验证。**新增 poison→healthy 测试确认发送顺序、失败项不 ACK、健康项 ACK、
  structured warning 和批末 reject；新增 ACK failure 测试确认第一项发送后 ACK 失败会阻止
  第二项发送。既有测试继续覆盖全健康 ACK 和同一申请多个管理群部分失败。完整 `yarn test`
  一次通过：build、Vue UI contracts、597/597 unit tests、startup smoke 与 WebUI E2E 46/46；
  deployable package contract 通过。独立只读代理复查 scheduler、backend list/ACK、重复语义
  和 batch window 后，确认当前分相实现没有提交级 blocker。
- **仍存在但不应夸大为本提交已解决的边界。**服务端 `freshmanForwardItems` 为任一记录生成
  material URL 失败时仍会让整个 list API 返回错误，Koishi 拿不到后续项；同一申请对多个群
  部分成功时，下一轮按既有 at-least-once 语义可能重复发给已成功群；极慢扫描没有 single-flight
  保护；100 个永久 poison 仍可占满窗口。本次结论只限于“Koishi delivery 阶段的一条 poison
  不再阻塞同批及后续窗口”，不能声称端到端 poison、重复投递或调度重叠都已闭环。
- **过度设计判断。**功能当前默认关闭，且没有真实失败率、重复率或扫描耗时证据。现在引入
  attempts/dead-letter、per-target delivery 状态、claim/lease 或 scheduler single-flight 会把
  一个局部循环修复扩成跨服务状态机。它们应作为正式启用前或监控达到阈值后的治理项，而不是
  本次 P2-7 的必要代码。

#### P2-8. Student verification method labels miss school_email_otp and school_sso, so the review queue renders the raw i18n key

- **位置**：`clients/admin/apps/web-ele/src/locales/langs/zh-CN/admin.json:432`
- **区域**：Admin 前端　**类别**：i18n-coverage　**验证票数**：1/1

**证据**

```
zh-CN/admin.json:432-435 → `"method": { "ldap": "LDAP", "manual": "人工审核" }` (en-US/admin.json:432-435 is identical in coverage).
views/users/student-verification/index.vue:200-206 → `const verificationMethodLabel = (method) => method ? $t(`admin.users.studentVerification.method.${method}`) : $t('admin.common.notSet');` — no fallback for unknown values (contrast content/reports/index.vue:101-108, which does `return label === key ? reason : label`).
clients/shared/src/types/api.gen.ts:5536 → `verificationMethod?: "ldap" | "manual" | "school_email_otp" | "school_sso" | null;`
server/internal/modules/user/service_student_email_otp.go:225-231 → `method := VerifyMethodSchoolEmailOTP; profile := &Profile{ ..., VerificationStatus: StatusPending, VerificationMethod: &method, ... }`
server/internal/modules/admission/repository_verified_profile.go:160-169 → `profileVerificationMethodForCredential` returns `"school_email_otp"` / `"school_sso"`.
```

**失败场景**

A student at a school whose `approvalPolicy` is `manual` verifies via school-email OTP (`POST /api/v1/user/profile/school-email/verify-otp`). service_student_email_otp.go writes `verification_status='pending', verification_method='school_email_otp'` (the auto-approve branch at line 239 is skipped for manual schools). The row appears in the admin 学生认证 pending queue, and the 认证方式 column renders the literal string `admin.users.studentVerification.method.school_email_otp` instead of a label (vue-i18n returns the key path on miss). Same for `school_sso` rows produced by the admission SSO credential path. Both locales are affected.

**修复方案**

Three edits.

1. clients/admin/apps/web-ele/src/locales/langs/zh-CN/admin.json:432-435 — extend the `users.studentVerification.method` map to cover the full enum, reusing the wording already used elsewhere in the same file (schoolConfig.schoolEmailOtp = "学校邮箱 OTP", schoolConfig.schoolSso = "学校 SSO") so the two screens stay consistent:
  "method": { "ldap": "LDAP", "manual": "人工审核", "school_email_otp": "学校邮箱 OTP", "school_sso": "学校 SSO" }

2. clients/admin/apps/web-ele/src/locales/langs/en-US/admin.json:432-435 — same, matching its own schoolConfig entries ("School Email OTP" / "School SSO"):
  "method": { "ldap": "LDAP", "manual": "Manual Review", "school_email_otp": "School Email OTP", "school_sso": "School SSO" }

3. clients/admin/apps/web-ele/src/views/users/student-verification/index.vue:200-206 — make the lookup degrade to the raw enum value instead of leaking the key path, mirroring `reasonLabel` in clients/admin/apps/web-ele/src/views/content/reports/index.vue:101-108:

const verificationMethodLabel = (
  method: StudentVerification['verificationMethod'],
) => {
  if (typeof method !== 'string' || method.trim() === '') {
    return $t('admin.common.notSet');
  }
  const key = `admin.users.studentVerification.method.${method}`;
  const label = $t(key);
  return label === key ? method : label;
};

This keeps the existing `notSet` behavior for null/undefined (and now also for an empty string), and any future enum addition on the server shows `school_xyz` rather than `admi

#### P2-9. Ansible deploy playbook builds the bundle with playbook-relative paths through ansible.builtin.command, which never resolves them

- **位置**：`infra/ansible/playbooks/deploy.yml:22`
- **区域**：基础设施与运维　**类别**：deployment-correctness　**验证票数**：1/1

**证据**

```
- name: Build deploy bundle on control node
      delegate_to: localhost
      ansible.builtin.command:
        cmd: ../../ops/build-deploy-bundle.sh ../../generated/deploy/stuhelper-deploy-bundle.tar.gz
```

**失败场景**

`ansible.builtin.command` executes a raw process with no `chdir`, so relative paths resolve against the controller process's working directory — never against the playbook directory (only file-lookup modules like `script`, `copy`, `template` do that). `Makefile:233` runs `cd infra/ansible && ansible-playbook -i inventory/production.ini playbooks/deploy.yml`, so the cwd is `<repo>/infra/ansible` and `../../ops/build-deploy-bundle.sh` resolves to `<repo>/ops/build-deploy-bundle.sh`, which does not exist → the task fails immediately with "No such file or directory", so `make ansible-deploy-prod` / `make ansible-deploy-staging` never reach the upload step. The neighbouring lines prove the inconsistency: line 6 uses `lookup('pipe', 'git -C ../..')` (correct for cwd `infra/ansible`) while lines 22 and 32 use `../../ops/...` and `../../generated/...` (only correct for cwd `infra/ansible/playbooks`, which is what `ansible.builtin.copy` on line 32 actually uses). Even if the second arg were reachable it would write the tarball to `<repo>/generated/deploy/`, where the `copy` task on line 32 (resolving to `infra/generated/deploy/`) would not find it.

**修复方案**

1) infra/ansible/playbooks/deploy.yml:19-22 - anchor the command and drop the redundant output argument, since infra/ops/build-deploy-bundle.sh already self-anchors via REPO_ROOT (infra/ops/lib/common.sh:5) to exactly the path the upload task reads:

    - name: Build deploy bundle on control node
      delegate_to: localhost
      ansible.builtin.command:
        argv:
          - "{{ playbook_dir }}/../../ops/build-deploy-bundle.sh"
      changed_when: true

(Equivalent acceptable form: `chdir: "{{ playbook_dir }}/../.."` plus `cmd: ./ops/build-deploy-bundle.sh ./generated/deploy/stuhelper-deploy-bundle.tar.gz` - note that anchor is <repo>/infra, not repo root.)

2) infra/ansible/playbooks/deploy.yml:32 - make the upload src explicitly playbook-anchored so build output and upload input are provably the same file: `src: "{{ playbook_dir }}/../../generated/deploy/stuhelper-deploy-bundle.tar.gz"`.

3) Add regression coverage, which is absent today: create infra/ops/tests/ansible-playbook-path-contract.sh (auto-discovered by infra/ops/tests/run-infra-contracts.sh, which globs *.sh) that, for every infra/ansible/playbooks/*.yml, (a) fails if any `command`/`shell` task uses a `cmd:`/`argv:` path beginning with `./` or `../` without a `chdir:`, and (b) resolves every `{{ playbook_dir }}`-anchored path plus every `ansible.builtin.script:` / `copy: src:` relative path against infra/ansible/playbooks and asserts the target exists on disk. That catches both the missing script and any

#### P2-10. Academic student import violates the sfzjh enc/hash pair constraint, aborting the documented fallback import

- **位置**：`infra/ops/import-buaa-academic-students.sh:232`
- **区域**：基础设施与运维　**类别**：data-integrity　**验证票数**：1/1

**证据**

```
INSERT INTO academic.buaa_students (
  xh, xm, sfzjlxdm, sfzjh_hash, yxdm, zydm, bjdm, xznj, rxnj, pyccdm,
  xslbdm, sjh, dzxx, xjztdm, sfzx, sfzj, synced_at
)
...
  NULLIF(btrim(sfzjh_hash), ''),
...
ON CONFLICT (xh) DO UPDATE
SET ...
    sfzjh_hash = EXCLUDED.sfzjh_hash,
```

**失败场景**

server/migrations/000001_initial_schema.up.sql:71 enforces CONSTRAINT chk_buaa_students_sfzjh_secure_pair CHECK ((sfzjh_enc IS NULL AND sfzjh_hash IS NULL) OR (sfzjh_enc IS NOT NULL AND sfzjh_hash IS NOT NULL)). The script's own usage text (line 45) advertises sfzjh_hash as a supported optional column while line 48 states sfzjh_enc is deliberately never imported, so the INSERT always leaves sfzjh_enc NULL. An operator following docs/guides/production-go-live.md:180 runs BUAA_ACADEMIC_STUDENTS_TSV=<tsv with a populated sfzjh_hash column> ./infra/ops/import-buaa-academic-students.sh: validate-only passes, then psql (-v ON_ERROR_STOP=1) aborts the whole transaction with 'new row for relation "buaa_students" violates check constraint "chk_buaa_students_sfzjh_secure_pair"' and zero rows are imported. The ON CONFLICT branch has the mirror-image bug: for any existing row that has both enc and hash, re-importing a TSV without sfzjh_hash sets sfzjh_hash = NULL against a non-NULL sfzjh_enc and fails the same constraint.

**修复方案**

Apply option (a) from the proposal - it matches the script's own stated contract that encrypted identity columns are not loaded from this TSV. In infra/ops/import-buaa-academic-students.sh: (1) remove "sfzjh_hash" from the Python `columns` list at line 121, and add a fail-fast right after `header` is built (around line 145) that raises SystemExit if the header contains sfzjh_hash or sfzjh_enc, e.g. `forbidden = [c for c in ("sfzjh_hash", "sfzjh_enc") if c in header]` -> `raise SystemExit(f"BUAA academic TSV must not supply {', '.join(forbidden)}: academic.buaa_students enforces chk_buaa_students_sfzjh_secure_pair, so encrypted identity columns must be written as a pair by the dedicated encrypted identity sync path")`; (2) delete `sfzjh_hash text,` from the TEMP TABLE at line 214, delete sfzjh_hash from the \copy column list at line 229, delete it from the INSERT column list at line 232, and delete the `NULLIF(btrim(sfzjh_hash), ''),` select item at line 239 - all four must change together so the normalized header and the copy list stay aligned; (3) delete `sfzjh_hash = EXCLUDED.sfzjh_hash,` from the ON CONFLICT SET list at line 257 so a re-import can never null a hash that is paired with an existing sfzjh_enc (simpler and strictly safer than the proposed COALESCE); (4) drop `sfzjh_hash` from the "Supported optional columns" usage text at line 45 and extend the note at lines 48-49 to say both sfzjh_enc and sfzjh_hash are never imported here because the schema enforces chk_buaa

> Claude 原修复方案在 `schema enforces chk_buaa` 处截断；其中“使用现成 dedicated sync path”
> 的假设也不成立。下面的 Codex 结论与已实施边界为本项权威处置。

**Codex 独立复核与处置（2026-07-30）**

- **问题真实，但不会持久化半对。**`academic.buaa_students` 的 CHECK 要求
  `sfzjh_enc/sfzjh_hash` 同时为空或同时非空。旧脚本把 hash-only 候选行送入 INSERT 时，
  PostgreSQL 会在冲突更新前拒绝该候选；不带 hash 重导一个已有完整 pair 时，旧
  `sfzjh_hash = EXCLUDED.sfzjh_hash` 又会尝试形成 `enc != NULL/hash = NULL` 并被拒绝。
  约束因此保护了存量数据，真实故障是整个多行 INSERT 原子失败，任何新行和普通字段更新都不落库，
  不是“数据库成功保存了孤立 hash”。
- **普通 importer 已撤销两列所有权。**规范化列从 16 个收窄为 15 个，并同步删除临时表、
  `\copy`、INSERT/SELECT 和 `ON CONFLICT` 中的 `sfzjh_hash`；`sfzjh_enc` 本来就不在写入 SQL。
  规范化、copy 和 SQL 的 15 列顺序已逐项对齐。新记录自然得到 `NULL/NULL`；冲突更新完全不引用
  两列，因此已有加密 envelope 与 HMAC 保持字节级不变。
- **输入契约 fail-fast。**离线 validate-only 与真实导入共用同一规范化器；标准
  `sfzjh_enc` 或 `sfzjh_hash` header（包括首尾空白）会在任何 Docker、环境文件或数据库操作前
  失败。帮助文本和生产 go-live 手册不再把 hash 列为支持字段，并明确普通 upsert 只保留既有 pair。
- **不存在可复用的完整 pair 写入口。**独立全库搜索确认 user academic repository 只读，
  seed 只显式写 `NULL/NULL`，prod-parity 也省略两列；通用 PII cipher 不能等同于一个可操作的
  学籍导入工具。最终文案明确“当前仓库不提供该入口”。确需该能力时，必须另行实现并审计从同一
  明文原子生成 AES-GCM envelope 与使用服务端一致密钥的 HMAC 的专用工具；本次直接新增 API/CLI
  会扩大 PII、密钥和审计边界，属于过度设计。
- **回归与交叉验证。**契约测试新增两个禁用 header 的动态失败用例，并静态保证 SQL 不再声明、
  选择或冲突更新 hash；真实 PostgreSQL 18 临时表复现旧两种失败，并证明修复语义为“旧 pair
  保留、新行 `NULL/NULL`”。ShellCheck、75 个 shell + 1 个 Node infra contracts、文档卫生
  单测 5/5 与全量文档检查通过。独立只读代理逐列复核 Python fieldnames、normalized TSV、
  TEMP、`\copy`、INSERT/SELECT 和 conflict update，纠正了“已有受控工具”的文案后确认无 blocker。
- **验证边界。**没有执行生产导入、没有使用真实身份证件号或生产密钥，也没有实现完整 pair 导入。
  当前提交只保证普通 fallback TSV 导入不再违反安全配对约束、不会清空存量 pair；未来专用 PII
  导入能力必须作为独立需求重新做威胁模型、密钥一致性、最小权限、审计和回滚验收。

#### P2-11. Batch reviews `courseIDs` array serialization is incompatible on all three legs (spec explode:false vs. client explode:true vs. handler reading only first value)

- **位置**：`server/api/paths/review-crud.yaml:195`
- **区域**：OpenAPI 契约　**类别**：contract-mismatch　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
Spec (server/api/paths/review-crud.yaml:185-196):
      - name: courseIDs
        in: query
        required: true
        schema:
          type: array
        style: form
        explode: false

Backend (server/internal/modules/course/review/review_read.go:178-184):
	idsStr := c.Query("courseIDs")   // gin returns only the FIRST repeated value
	parts := strings.Split(idsStr, ",")

Shared client (clients/shared/src/api/client.ts:28-32) creates openapi-fetch with no `querySerializer`, so the library default (`style: form, explode: true`) applies. Verified empirically with the installed openapi-fetch:
  client.GET('/api/v1/course/review/reviews/batch', { params: { query: { courseIDs: [1,2,3], pageSize: 5 } } })
  -> 'http://x/api/v1/course/review/reviews/batch?courseIDs=1&courseIDs=2&courseIDs=3&pageSize=5'
The uniappx transport has the same behaviour (clients/shared/src/api/session-client.ts:144-149 pushes one pair per array item).
```

**失败场景**

`api.review.getBatchCourseReviews([101, 202, 303])` (clients/shared/src/api/reviews.ts:42) sends `?courseIDs=101&courseIDs=202&courseIDs=303`. `c.Query("courseIDs")` yields "101", so the handler fetches reviews for course 101 only and returns `{"101": {...}}`; courses 202 and 303 are silently missing with HTTP 200 — the exact N+1 avoidance the endpoint exists for is defeated with no error. The one wrapper that consumes it, clients/web/src/api/review.ts:42, then feeds the course-keyed map into `readReviewPagePayload`, which requires top-level `list`/`total` (clients/web/src/modules/review/reviewListPayload.ts:187-195) and therefore throws `Invalid review list response` for every successful call. No view calls it yet, so the break is latent but 100% reproducible the moment it is wired up.

**修复方案**

Apply the spec+handler side only; do NOT touch the shared client's serializer.

1. `server/api/paths/review-crud.yaml:195` — change `explode: false` to `explode: true` so the published contract matches what every generated client actually emits, mirroring the already-working `scope` param at `server/api/paths/open-platform.yaml:641`. Then regenerate (`server/api/openapi.bundled.yaml`, `server/internal/api/gen/`, `clients/shared/src/types/api.gen.ts`) via the normal codegen task — do not hand-edit generated files.

2. `server/internal/modules/course/review/review_read.go:178-184` — replace the single-value read with an array read that still accepts the legacy comma form, so both the new client format and any existing spec-conformant third-party caller keep working:

    raw := c.QueryArray("courseIDs")
    if len(raw) == 0 {
        response.BadRequest(c, "courseIDs is required")
        return
    }
    parts := make([]string, 0, len(raw))
    for _, v := range raw {
        parts = append(parts, strings.Split(v, ",")...)
    }
    if len(parts) > 20 { ... }

   Keep the existing per-item TrimSpace / ParseInt / id<=0 validation and the `len(courseIDs)==0` guard unchanged.

3. `clients/web/src/api/review.ts:41-44` — `getBatchCourseReviewsPage` must stop calling `readReviewPagePayload` on a course-keyed map. Either change its contract to return `Record<string, PaginatedResult<Review>>` and map each entry through `readReviewPagePayload`, or drop the adapter entirely until a view

#### P2-12. Admission school-email endpoints fan out to the external Oracle source with no per-user rate limit

- **位置**：`server/internal/modules/admission/handler.go:81`
- **区域**：入群认证　**类别**：missing-throttle　**验证票数**：1/1

**证据**

```
admission.POST("/school-email/academic-match", authMW, h.handleMatchSchoolEmailAcademicStudent)
admission.POST("/school-email/request-otp", authMW, h.handleRequestSchoolEmailOTP)
// compare server/internal/modules/user/handler.go:114-115
// user.POST("/profile/school-email/academic-match", middleware.EndpointRateLimitMiddleware(h.verifyLimiter, ...), ...)
// verifyRateLimitPerMinute = 5
```

**失败场景**

The admission Handler has no limiter fields at all, while the equivalent user routes are capped at 5 req/min/user. MatchSchoolEmailAcademicStudent -> resolveAcademicStudentEmail -> GetAcademicInfo issues one Oracle query per call, and RequestSchoolEmailOTP performs the same lookup before reserveEmailOTPCooldown, so the 60s OTP cooldown does not throttle it either. One authenticated user with a linked admission session loops academic-match up to the only remaining cap (API_IP_RATE_LIMIT, default 100/min since .env.prod.example does not set it). With EXTERNAL_STUDENT_SOURCE_ORACLE_MAX_OPEN_CONNS=4, concurrent loops saturate the pool; genuine lookups then block on pool acquisition until the 3s ctx expires, returning DeadlineExceeded, which is recorded as a breaker failure (line 220) -> breaker opens -> 30s of school-wide 503s. The school's DBA also sees unbounded query volume from StuHelper.

**修复方案**

Apply the limiter half only; drop the cooldown reorder.

1. `server/internal/modules/admission/handler.go` — add a limiter field and a functional option, mirroring the user module:
   - add `schoolEmailLimiter *middleware.RedisRateLimiter` to the `Handler` struct (line 17-22);
   - add `func WithSchoolEmailRateLimiter(l *middleware.RedisRateLimiter) HandlerOption` next to `WithAdminAuthorizers` (avoids changing the 3-arg `NewHandler` signature; alternatively accept `*redis.Client` as `user.NewHandler` does and build the limiter internally);
   - in `RegisterRoutes`, replace lines 81-83 with the nil-safe branch used at `user/handler.go:112-122`: when `h.schoolEmailLimiter != nil`, wrap each route in `middleware.EndpointRateLimitMiddleware(h.schoolEmailLimiter, "admission-school-email-academic-match" | "-request-otp" | "-verify-otp")`; otherwise register as today. Distinct endpoint labels matter — the key is `"rl:endpoint:"+endpoint+":user:"+userID`, so a shared label would make the three routes share one bucket.
   - Keep `authMW` first so `GetUserID(c)` is populated and the limiter keys per-user rather than falling back to per-IP.

2. `server/internal/app/modules.go:199-204` — pass `admission.WithSchoolEmailRateLimiter(middleware.NewRedisRateLimiter(rt.redisClient.GetClient(), 5, time.Minute))`, matching `verifyRateLimitPerMinute = 5` in `user/handler.go:46`. Define the constant in the admission package (e.g. `schoolEmailRateLimitPerMinute = 5`) rather than hardcoding it at t

#### Codex 对 P2-12 的最终复核与实施记录（2026-07-31）

**最终结论**

缺少昂贵查询的用户级限流真实存在。全局 10,000/min、IP 100/min 以及 user 200/min 只能限制
一般流量；`academic-match` 每次查询一次外部学籍源，`request-otp` 也在 60 秒 OTP cooldown
之前查询，因此 cooldown 不能保护 Oracle。报告把系统描述成“完全无限”不准确，但单账号可在
现有通用预算内持续占用默认 4 个 Oracle 连接，P2 级依赖保护是必要的。

Claude 建议给三条邮箱路由不同 key，其中 `verify-otp` 不访问 Oracle，而两个昂贵路由使用不同
key 又会把总查询预算翻倍。最终按风险根因只限制 `academic-match` 与 `request-otp`，并让它们
在同一认证 subject 下共用 `admission-school-email-academic-lookup` key，合计 5 次/分钟。

**已实施的最小修复**

1. Admission Handler option 从既有 Redis client 构造一个 5/min sliding-window limiter；
   runtime 显式接线。两条昂贵路由均保持 `authMW` 在前、同一 endpoint key 在后，因此不会
   匿名回退到 IP key，也不会被两条路由各自获得 5 次。
2. limiter 满额返回现有 429 与 `Retry-After`；Redis/entropy/context 故障沿用现有
   fail-closed 503，不进入 handler，也不访问 Oracle。没有修改全局/IP/user 默认预算、Oracle
   breaker、连接池或 OTP cooldown/attempt 状态机。
3. `verify-otp` 只读 Redis OTP 记录并写验证结果，不访问学籍源，因此保持原路由，不为了表面
   一致性额外消耗 Oracle 查询预算。
4. OpenAPI 真源为 academic-match 增加实际可能的 429/503，完整 generate 后 bundle、嵌入 Go
   spec 与共享 TypeScript 响应联合类型一致；没有手改生成文件。

**交叉验证**

- 同一认证用户依次执行 4 次 academic-match 和 1 次 request-otp，真实 PostgreSQL admission
  session/config 与 counting academic gateway 证明恰好 5 次外部调用；第 6 次回到另一条昂贵
  路由仍为 429，gateway 不增加。
- 第二个认证 subject 在第一用户满额后仍可成功，证明 Redis key 以已认证 subject 隔离。
- miniredis socket 关闭后，匿名请求先由 auth 返回 401；认证请求由 limiter 返回 503，handler
  未执行且 gateway 调用为 0。该测试验证了 middleware 顺序与故障策略，不把 Redis outage
  当作额外额度。
- 当前测试使用真实 PostgreSQL/miniredis 和受控 gateway，不等同于生产 Oracle 连接池、DBA
  查询量或限流指标验收；发布后仍需受控账号验证 429/503，并观察 Oracle p95、pool wait 和
  breaker 指标。

#### P2-13. One failing row discards the whole claimed bot-action batch, permanently dead-lettering kick/release actions

- **位置**：`server/internal/modules/admission/service_bot_actions.go:53`
- **区域**：入群认证　**类别**：correctness　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
```go
	rows, err := s.repo.ClaimDueBotActions(ctx, normalized, now)   // already flipped every row to 'dispatched', attempt_count+1
	...
	actions := make([]AdmissionPendingAction, 0, len(rows))
	for i := range rows {
		action, stale, err := s.pendingActionFromQueuedRow(ctx, &rows[i], now)
		if err != nil {
			return nil, err          // <-- discards rows[0..i-1] that were already claimed and are already ack-pending
		}
		if stale {
			if err := s.repo.MarkBotActionStale(ctx, rows[i].ID, now); err != nil {
				return nil, err      // <-- same loss
			}
			continue
		}
```

The claim already consumed the lease (repository_bot_action_outbox.go:139-147): `SET status = 'dispatched', attempt_count = attempt_count + 1, next_attempt_at = $5`. There is no per-row recovery: the batch is up to `maxPendingActionLimit = 200` (repository_bot_scan.go:12) and spans every guild that bot serves. At `admissionBotActionMaxAttempts = 5` (repository_bot_action_outbox.go:14) the claim SQL's `terminal` CTE (lines 110-124) flips them to `dead_letter`. That state is terminal: `QueueBotActionTx`'s ON CONFLICT explicitly preserves it — `WHEN admission_bot_action_outbox.status IN ('dispatched','succeeded','dea
```

**失败场景**

Postgres is slow for ~3 minutes (a vacuum, failover, or connection-pool saturation). `pendingActionFromQueuedRow` issues 2 queries per row (see the N+1 finding), each bounded by DB_QUERY_TIMEOUT=5s, so at least one row in every claim errors. Each 2s SSE tick claims up to 200 queued kick/remind/release rows, burns one attempt on every one of them, then returns HTTP 500 / an SSE `error` frame and delivers none. After 5 such rounds (~2.5 min, spaced by next_attempt_at = now+30s) every queued action is `dead_letter`. Concretely: a student who just finished verification has a `BotActionRelease` row keyed `<sessionID>:release` (repository_bot_action_outbox.go:281-282); once it dead-letters the bot never un-mutes them, and because the key is stable and ON CONFLICT preserves `dead_letter`, neither `MarkVerified` nor `ProjectStudentVerification` re-queueing can ever revive it — the verified student stays muted in the QQ group forever without manual DB surgery.

**修复方案**

1) server/internal/modules/admission/service_bot_actions.go (ClaimQueuedAdmissionActions, lines 49-63): make the loop per-row fault-isolated, mirroring outbox.ProcessBatch (server/internal/pkg/outbox/worker.go:104-137). On pendingActionFromQueuedRow error: logger.L().Warn with action_id/session_id/error, call the new repo lease-release below, then continue. On MarkBotActionStale error: log and continue instead of returning. Only propagate an error when ClaimDueBotActions itself fails. Add a `if ctx.Err() != nil` check at the top of each iteration that breaks out and releases the leases of the remaining rows using a fresh short context derived from context.Background() (as outbox.abandonClaimedJobs/finalizeContext does), so a client disconnect mid-batch does not burn attempts.

2) server/internal/modules/admission/repository_bot_action_outbox.go: add a lease-release method modeled on outbox.MarkJobAbandoned (repository.go:348-361), fenced on the claimed attempt so a concurrent newer claim is never clobbered:
   func (r *Repository) AbandonBotActionLease(ctx context.Context, actionID int64, dispatchAttempt int, now time.Time) error, executing
   UPDATE admission_bot_action_outbox SET status = 'failed', attempt_count = GREATEST(attempt_count - 1, 0), next_attempt_at = $2, updated_at = $2 WHERE id = $1 AND status = 'dispatched' AND attempt_count = $3
   (use withDBTable(ctx, admissionBotActionOutboxTable); treat RowsAffected()==0 as a lost lease that is logged, not returned, so i

#### P2-14. Bot action claim issues 2 extra DB queries per claimed row instead of one batched lookup

- **位置**：`server/internal/modules/admission/service_bot_actions.go:137`
- **区域**：入群认证　**类别**：efficiency　**验证票数**：1/1

**证据**

```
```go
func (s *Service) pendingActionFromQueuedRow(ctx context.Context, row *AdmissionBotActionOutboxRow, now time.Time) (...) {
	...
	seeds := pendingActionSeeds([]AdmissionSession{session}, now)
	contexts, err := s.pendingActionContexts(ctx, []AdmissionSession{session})   // per row!
```

`pendingActionContexts` (service_bot_action_contexts.go:35-49) always runs two queries — `ListPoliciesByGuildKeys` and `ListAdmissionFailuresByKeys` — and it is called once per claimed row from the loop at service_bot_actions.go:50-62. The sibling read path shows the intended batched shape: `ListPendingAdmissionActions` (service_bot_actions.go:26) calls `pendingActionContexts(ctx, sessions)` exactly once for the whole slice, and the repository helpers are already written to accept key slices (`unnest($1::text[], $2::text[])`, repository_bot_contexts.go:57-72).
```

**失败场景**

A bot connected to /api/v1/bot/admission/actions/stream ticks every 2s (handler_bot_queries.go:132). After a mass join event the queue holds 200 due actions (maxPendingActionLimit). One tick then executes 1 claim transaction + 400 point queries, each taking a pooled connection with a 5s timeout, and the next tick fires 2s later. With DB_MAX_CONNS at its default this alone saturates the pool, slows every user-facing request, and — because a single one of those 400 queries timing out aborts the whole batch (see the batch-discard finding) — it is also the trigger that burns the retry budget on all 200 actions.

**修复方案**

Hoist the context lookup out of the loop in /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_bot_actions.go, mirroring `ListPendingAdmissionActions`.

In `ClaimQueuedAdmissionActions`, after the `len(rows) == 0` early return (line 46-48):

    sessions := make([]AdmissionSession, len(rows))
    for i := range rows {
        sessions[i] = rows[i].Session
    }
    seeds := pendingActionSeeds(sessions, now)
    contexts, err := s.pendingActionContexts(ctx, sessions)
    if err != nil {
        return nil, err
    }
    actions := make([]AdmissionPendingAction, 0, len(rows))
    for i := range rows {
        action, stale, err := s.pendingActionFromQueuedRow(&rows[i], &sessions[i], seeds[i], contexts)
        // ... unchanged stale / MarkBotActionStale / append handling

Then change the signature to drop `ctx` and `now` and accept the shared data:

    func (s *Service) pendingActionFromQueuedRow(
        row *AdmissionBotActionOutboxRow,
        session *AdmissionSession,
        seed pendingActionSeed,
        contexts pendingActionContexts,
    ) (AdmissionPendingAction, bool, error) {
        if row == nil || session == nil {
            return AdmissionPendingAction{}, true, nil
        }
        if !sessionCanDispatchQueuedBotAction(session) {
            return AdmissionPendingAction{}, true, nil
        }
        action, err := s.pendingActionFromSession(session, seed, contexts)
        if err != nil {
            return AdmissionPendingAction{}, fals

#### P2-15. ClaimQueuedAdmissionActions reloads policies and failure counters once per claimed row instead of batching

- **位置**：`server/internal/modules/admission/service_bot_actions.go:137`
- **区域**：入群认证　**类别**：efficiency　**验证票数**：1/1

**证据**

```
func (s *Service) pendingActionFromQueuedRow(ctx context.Context, row *AdmissionBotActionOutboxRow, now time.Time) (...) {
	...
	seeds := pendingActionSeeds([]AdmissionSession{session}, now)
	contexts, err := s.pendingActionContexts(ctx, []AdmissionSession{session})  // 2 queries, per row
// invoked from the ClaimQueuedAdmissionActions loop at service_bot_actions.go:50-62
```

**失败场景**

The Koishi guard polls POST /api/v1/bot/admission/actions/claim. ClaimDueBotActions returns up to filter.Limit rows (maxPendingActionLimit = 200, repository_bot_scan.go:12). For each row the loop calls pendingActionContexts, which runs ListPoliciesByGuildKeys and ListAdmissionFailuresByKeys (repository_bot_contexts.go:17 and :41), so a full batch costs 400 extra round trips per poll instead of 2, re-reading the same handful of policy rows 200 times. The sibling read path ListPendingAdmissionActions (service_bot_actions.go:26) already batches this correctly via pendingActionContexts(ctx, sessions), so the claim path is an unintended divergence.

**修复方案**

Batch the context load once per claim in /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_bot_actions.go, mirroring `ListPendingAdmissionActions`.

1. In `ClaimQueuedAdmissionActions` (lines 33-64), after the `len(rows) == 0` early return at line 46, build the session slice and load contexts once:

    sessions := make([]AdmissionSession, 0, len(rows))
    for i := range rows {
        sessions = append(sessions, rows[i].Session)
    }
    contexts, err := s.pendingActionContexts(ctx, sessions)
    if err != nil {
        return nil, err
    }

Building the slice from all rows (rather than pre-filtering with `sessionCanDispatchQueuedBotAction`) is fine and simpler: `pendingActionLookupKeys` dedupes both key sets through maps, so a few extra keys cost nothing and it stays a single query.

2. Change the helper signature at line 124 from
   `func (s *Service) pendingActionFromQueuedRow(ctx context.Context, row *AdmissionBotActionOutboxRow, now time.Time)`
   to
   `func (s *Service) pendingActionFromQueuedRow(row *AdmissionBotActionOutboxRow, contexts pendingActionContexts, now time.Time)`
   and delete lines 137-140 (the per-row `s.pendingActionContexts` call and its error branch). The `context.Context` parameter becomes unused — drop it. Keep the per-row `pendingActionSeeds([]AdmissionSession{session}, now)` at line 136 as-is (it is pure and does no I/O), or index into a batched `pendingActionSeeds(sessions, now)` for symmetry with `pendingActionsFromSessio

#### Codex 对 P2-13/P2-14/P2-15 的最终复核与实施记录（2026-07-30）

**最终结论**

- P2-13 的正确性核心真实存在：`ClaimDueBotActions` 的事务一旦提交，整批已经成为
  `dispatched` 且 `attempt_count + 1`；随后任一批量上下文错误、逐行映射错误或 caller
  cancellation 都不会回滚 claim。旧实现返回错误时，Bot 一个 action 也收不到，但整批会在
  30 秒后继续消耗 attempt，持续故障最终可让稳定 key 的 release 进入无法自动重排的
  `dead_letter`。
- 原文的“一次失败即永久丢失”和“约 2.5 分钟”过强。实际需要持续失败耗尽 5 次 claim，
  总耗时还受 30 秒 lease、SSE 断线重连和抖动影响；kick key 含 scheduled time，也不能扩大成
  “该 session 永远不能再产生任何 kick”。release key 稳定且 upsert 保留 dead letter 的后果则成立。
- P2-14 的 `2N` 查询真实存在，但单独属于 P3 性能问题；它会放大 P2-13。常规 Koishi
  `limit=50` 对应约 100 次额外查询，只有显式最大 `limit=200` 才会达到 400 次。
- P2-15 与 P2-14 指向完全相同的 `pendingActionContexts` 调用，是重复标签，不是第二个根因。
- 原审计还漏掉一个真实竞态：旧 `MarkBotActionStale` 仅按 ID 且
  `status <> 'succeeded'` 更新，旧 attempt 在 30 秒后可能把新 attempt 标成 stale。

**已实施的最小状态机**

1. claim 后从所有返回行构造 sessions/seeds，整批只调用一次 `pendingActionContexts`：
   policy 与 failure 各一条批量 SQL；不把两类数据硬拼成怪异的单条查询。
2. 批量查询失败或 caller 在 Service 返回前取消时，Handler 尚未写出任何 action。服务端用
   detached、5 秒有界 context 一次性批量归还仍匹配 `id + status=dispatched +
   attempt_count` 的 lease；补偿写成功时退还本次 retry budget，并把下次领取延后 30 秒
   避免热循环。若 outbox 补偿写本身也超时/断连，原 claim 不会被原子回滚，仍保留此次
   attempt；方法返回包含原错误与 cleanup 错误的合并错误。
3. 确定性单行 preparation failure（当前可复现的是 kick 缺 policy）只把本行按 fence 标成
   `failed`、写 `last_error` 并 backoff；第 5 次只让该 poison 行 `dead_letter`。健康
   release/remind 继续返回，不能因 poison 行丢弃整批。
4. stale 与 preparation failure 都要求 exact attempt 且检查 `RowsAffected`。逐行最终化遇到
   基础设施错误时先对本行做一次同步 abandon；只有最终化和补偿都失败，才返回批次错误并归还
   其他尚未公开的健康/未处理 lease。
5. Service 成功返回后的 HTTP/SSE 丢包存在“客户端是否收到”的模糊窗口，仍按 at-least-once
   投递保留 `dispatched`；此路径绝不回退 attempt。

**ABA 与过度设计边界**

归还 helper 会把 attempt 从 N 退回 N-1，下一次 claim 会再次得到 N，因此它不是可重试的通用
lease token。当前实现安全依赖三个同时成立的约束：仅 Service 内部可调用、动作尚未向 Bot
公开、cleanup 同步 single-shot 且返回前完成；发生模糊错误也不后台重试。当前链路不存在携带
旧 N 的外部 ACK 或延迟 cleanup 回调，故不为本项增加 migration。若未来改成逐行流式写出、
异步重试 cleanup、多阶段 action preparation，必须把单调 `dispatch_generation`/lease token
与 retry count 拆开；继续回退并复用 `dispatchAttempt` 将不再安全。

**残余风险与后续建议**

- 这是补偿式恢复，不是跨 claim、enrichment 与 cleanup 的原子事务。真实 PostgreSQL 测试稳定
  证明的是“policy/failure 查询失败或 caller cancel，且 outbox 仍可接受补偿写”时不消耗
  attempt；它没有证明连接池/网络/outbox 同时不可用时补偿必定成功。在极端间歇故障中，
  claim 提交成功而 enrichment 与 cleanup 连续失败，仍可能逐次消耗预算并最终 dead-letter。
  若产品要求“内部 enrichment 失败在任何存储故障下绝不消耗预算”的硬保证，需要重新设计
  retry count 与单调 lease generation/事务边界；这不是本次局部批次修复可以诚实保证的性质。
- 确定性 poison 现在不会阻塞健康行，但达到第 5 次后仍进入 terminal `dead_letter`；
  `QueueBotActionTx` 会保留该状态，仓库目前没有 Admission 专用 replay API。补齐 policy 后动作
  不会自动复活。显式、授权、带审计的 dead-letter replay 是独立的运营恢复待办；本次没有顺带
  扩建 OpenAPI、Admin UI 和授权流，也不把“隔离并终止”误写成“已经可恢复”。

**交叉验证与测试**

- 两个独立只读代理分别审查了事务边界、poison 分类、fence/ABA 和测试注入方式；均确认应修
  P2-13、合并 P2-14、删除 P2-15 独立计数，并反对用长事务、generic outbox 重写或单条巨型 SQL。
- 真实 PostgreSQL 测试确认 1 行与 8 行均固定为 3 次 pool acquisition（claim 事务 +
  2 次上下文查询），旧实现则随 N 增长为 `1 + 2N`。
- 临时 rename policy 表稳定注入 claim 提交后的 batch lookup failure，确认两行均恢复为
  pending/attempt 0，30 秒后可重新领取；表锁 + caller cancel 确认 cleanup 不继承父取消。
- 回归还覆盖旧 attempt 的 stale/abandon 不能覆盖新 attempt、缺 policy kick 在第 5 次单独
  dead-letter 且同批健康 release 正常 ACK，以及既有迟到 ACK、dispatch timeout 和 stale 路径。
- 测试没有破坏 outbox 写入或整个连接池，因此不声称补偿写同时失败时也能归还 lease；也没有
  replay terminal 行，因为该 API 当前不存在。这两项按上述残余风险保留。
- 最终提交前再次运行 Admission 全包 `go test -race ./internal/modules/admission -count=1`，
  并通过全服务端 `golangci-lint`、production binary build、文档卫生单测与全量文档检查。

本次没有引入新队列/worker、没有把 claim/enrichment 放进长事务、没有修改 OpenAPI 或迁移，
也没有做 200 行压力测试或为精确 wire SQL 数引入 tracer；这些都不是消除当前根因的必要条件。

#### P2-16. Nullable courses columns are scanned into non-nullable Go fields; one NULL department_id/code/credits 500s every course endpoint

- **位置**：`server/internal/modules/course/repository.go:175`
- **区域**：评课与内容　**类别**：schema-mismatch　**验证票数**：1/1

**证据**

```
SELECT c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count
FROM courses c
LEFT JOIN departments d ON d.id = c.department_id
WHERE c.id = $1
// scanned into model.go:19 DepartmentID int64, model.go:21 Code string, model.go:23 Credits float64
// live schema: department_id bigint NULL, code character varying(50) NULL, credits numeric(4,1) NULL
```

**失败场景**

Reproduced against the live dev database (migrations at version 19): inserting one row with INSERT INTO courses (school_id, name, code, department_id, credits, category) VALUES ($1,'NULL-AUDIT',NULL,NULL,NULL,'x') and running the exact GetCourseByID query with the exact Scan destinations returns "can't scan into dest[2] (col: department_id): cannot scan NULL into *int64". Because no Go code inserts into courses (catalog rows come from operator SQL import), a single imported course lacking a department, course code, or credit value makes GET /courses/:id, GET /courses (line 131), GET /courses/search (line 154) and ListCoursesGroupedByDepartment (line 284) all return 500 — the list endpoints break for every user, not just that one course. The LEFT JOIN d.name into DepartmentName string fails the same way, while sibling repositories in the same module do COALESCE it (course/review/repository_interaction.go:90 uses COALESCE(c.name, '')).

**修复方案**

Normalize NULLs in SQL (do NOT switch to pointer types — that breaks the generated TS contract).

1. `server/internal/modules/course/repository.go` — in all four SELECTs (lines 131, 154, 175, 284) replace the projection
   `c.id, c.school_id, c.department_id, d.name, c.code, c.name, c.credits, c.category, c.review_count`
   with
   `c.id, c.school_id, COALESCE(c.department_id, 0), COALESCE(d.name, ''), COALESCE(c.code, ''), c.name, COALESCE(c.credits, 0), c.category, c.review_count`.
   Leave the WHERE/ORDER BY clauses untouched (they already handle 0 as the "no department filter" sentinel at lines 96, 136).

2. `server/internal/modules/course/review/repository_interaction.go:267` (`ListFavorites`) — same defect, missed by the finding. Change
   `SELECT c.id, c.name, c.code, c.credits, c.department_id, d.name, ...`
   to
   `SELECT c.id, c.name, c.code, COALESCE(c.credits, 0), COALESCE(c.department_id, 0), d.name, ...`.
   `Code`/`DepartmentName` are already `*string` in `review/model.go:144,147` so they need no COALESCE.

3. `server/internal/modules/course/model.go:23` — drop `omitempty` from `Credits float64 \`json:"credits,omitempty"\`` → `json:"credits"`. Required because `server/api/components/schemas/course.yaml:3` marks `credits` as **required**; COALESCEing NULL→0 with `omitempty` still present would silently drop a required field and produce a response that violates the OpenAPI contract (`api.gen.ts:3608` types it `credits: number`). `DepartmentID` already lacks `omi

#### Codex 对 P2-16 的最终复核与实施记录（2026-07-31）

**最终结论**

扫描崩溃真实存在，但 Claude 的 `COALESCE(department_id, 0)` / `COALESCE(credits, 0)` 方案不应
实施。权威 migration 从初始 schema 起就把 `courses.department_id`、`code`、`credits` 定义为
可空，后续 18 个 migration 没有收紧；Web/UniAppX 也已经存在“未分类院系”“未提供课程代码”
展示。开发库只读聚合盘点为 23 行、三列当前均无 NULL，说明样本干净，不足以把数据库明确允许
的未知元数据重新解释成非法。`0` 不是合法院系 ID，`0` 学分也可能是一个真实值，强行
COALESCE 会让“未知”和“已知为零”不可区分。

**已实施的最小语义修复**

1. `Course` 与 `FavoriteCourse` 对可空数据库列使用指针扫描；`code`/`departmentName` 继续是
   可选字段，`departmentID`/`credits` 保持响应必有但值可为 JSON `null`。未分类
   `DepartmentGroup` 的 ID/name 同样可为 null，Repository 用 `(present,id)` 作为内部分组键，
   没有把 null 借用成业务 ID 0。
2. OpenAPI 3.1 真源把 `departmentID`、`credits` 及未分类 group 元数据声明为 nullable，
   完整 generate 后 Go 类型为 `*int64`/`*float64`，共享 TypeScript 类型为
   `number | null`；没有手改生成文件。
3. Web payload reader 只接受 null 或原有合法数值，仍拒绝 department ID 0、负学分和错误类型；
   收藏 reader 同步。UniAppX 对 null 显示“未分类院系/学分未提供”，不会把 null 插值成
   `Department #null` 或 `null 学分`。
4. 按学分降序改为 `NULLS LAST`，使未知值不抢占已知高学分课程；其余过滤、分页和排序不变。
   没有创建“未分类院系”伪记录、猜测学分、回填数据或增加 migration。

**交叉验证**

- 隔离 PostgreSQL 18 插入一门三列均为 NULL 的课程，detail/list/search/grouped 四条真实
  Repository/Service 路径全部成功；HTTP 详情返回 200，包含
  `"departmentID":null`、`"credits":null` 且不输出可选 `code`。
- 独立收藏 Repository 回归确认同一课程可被列出且四项可空元数据均保持 nil，不再 Scan 500。
- Web nullable/负向 payload 测试、UniAppX 58 个 unit、Web/Shared/UniAppX type-check、
  OpenAPI lint/drift、course/review 两包与全服务端 race、lint/build/docs 均通过。当前开发库
  盘点不是生产数据验收；发布前仍应对目标库只读统计 NULL 分布并确认旧客户端与服务端同批发布。

#### P2-17. review_preview_content_chars / review_preview_content_percent are validated, parsed and cached but never applied; content previews are truncated with the title budget

- **位置**：`server/internal/modules/course/review/access.go:241`
- **区域**：评课与内容　**类别**：integration-gap　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
access.go:236-259 — both preview branches pass the TITLE budget and hardcode percent=100:
	case !facts.Authenticated:
		result[i].Content = previewFirstContentLine(result[i].Content, facts.PreviewTitleRunes)
		result[i].Title = ""
	case !facts.CanViewFull:
		result[i].Content = previewFirstContentLine(result[i].Content, facts.PreviewTitleRunes)
	...
	func previewFirstContentLine(value string, maxRunes int) string {
		for _, line := range strings.Split(value, "\n") {
			preview := previewText(line, maxRunes, 100)

facts.PreviewContentRunes / facts.PreviewContentPct are populated (access.go:180-181) from admin config (access.go:136-147) and validated on write (user/service_admin.go:470-477), but `grep -rn "PreviewContentRunes\|PreviewContentPct"` shows zero non-test production readers. previewText's percent branch (`percent > 0 && percent < 100`, access.go:268) is therefore dead in production.
```

**失败场景**

An operator sets review_preview_content_chars=400 and review_preview_content_percent=30 via PUT /api/v1/admin/system-configs/{key} to widen previews for unverified visitors. The value passes validation, is stored, InvalidateReviewAccessPolicySnapshot fires, and buildReviewAccessPolicy loads it into the snapshot — yet GET /course/review/courses/{id}/reviews still truncates content at PreviewTitleRunes (default 24) runes with no percentage applied. The knob has no observable effect at any value, and content previews are 5x shorter than the configured/documented content budget of 120.

**修复方案**

Stop shipping a half-wired config surface. Pick one of the two directions below; do not leave the current state.

Direction A (wire the knobs, preserving the single-line safety from 58243689) — preferred:
1. server/internal/modules/course/review/access.go:251 — change to `func previewFirstContentLine(value string, maxRunes int, percent int) string` and pass `previewText(line, maxRunes, percent)` instead of the hardcoded 100. Keep the first-non-empty-line loop so multi-line content can never leak past line 1.
2. access.go:243-245 (`!facts.CanViewFull`) — use `previewFirstContentLine(result[i].Content, facts.PreviewContentRunes, facts.PreviewContentPct)`. This matches the docs tier "已登录未认证 → 评课正文有限制" (docs/product-specs/course-review.md:133) and restores the pre-58243689 semantics without reintroducing multi-line exposure.
3. access.go:240-242 (`!facts.Authenticated`) — decide explicitly with product. Safest default: keep `facts.PreviewTitleRunes` here (guest teaser stays tight at 24) and document that `review_preview_title_chars` governs the guest teaser; otherwise use the content budget here too, but then note that guest exposure widens from 24 to 120 runes by default.
4. Fix the title knob's documented meaning: either re-apply `result[i].Title = previewText(result[i].Title, facts.PreviewTitleRunes, 100)` in the `!facts.CanViewFull` branch, or update the seeded description of `review_preview_title_chars` in a new migration so it says "游客正文预览最大字符数" and no longer claims to trun

#### P2-18. Sensitive-word admin mutations never invalidate the moderation filter, and Filter.Refresh is dead code, so new block rules take up to 5 minutes to apply

- **位置**：`server/internal/modules/course/review/handler_sensitive_word_admin.go:60`
- **区域**：评课与内容　**类别**：cache-invalidation　**验证票数**：1/1

**证据**

```
handler_sensitive_word_admin.go:60-74 (create), :103-114 (update), :125-136 (delete) all mutate the rule set and then only log — no filter refresh and no cache invalidation:
	w, err := h.service.CreateSensitiveWord(c.Request.Context(), req.Word, req.Category, req.Level)
	...
	h.logAdminOp(c, "create_sensitive_word", "sensitive_word", w.ID, ...)
	response.Created(c, w)

filter.go:52-55 fixes a purely time-based TTL:
	return &Filter{ repo: repo, refreshTTL: 5 * time.Minute }
and filter.go:110-118 only reloads when `time.Since(f.lastRefresh) > f.refreshTTL`. The exported escape hatch is unused: `grep -rn "filter.Refresh|\.Refresh(ctx)"` (excluding tests) returns nothing, i.e. Filter.Refresh (filter.go:72) has no production caller. Contrast the module's established invalidation pattern for every other admin mutation: handler.go:266-277 invalidateCachePrefixes / cache.InvalidateByVersion.
```

**失败场景**

A moderator responding to a live brigading incident adds a `block`-level sensitive word via POST /api/v1/course/review/admin/sensitive-words. For up to 5 minutes each API replica keeps serving from its in-process matcher set, so PostReview and CreateReply continue to accept and publish content containing the newly blocked term. The same window applies in reverse: deactivating a false-positive word keeps rejecting legitimate reviews for 5 minutes. There is no mechanism at all to shorten it — no version key, no pub/sub, no call to Refresh.

**修复方案**

Do it in two layers, keeping invalidation in the Service (which owns `filter`), not the Handler.

1. Local (serving-replica) consistency — the high-value, low-risk half:
   - server/internal/modules/course/review/filter.go: add `func (f *Filter) Invalidate()` next to `Refresh` (filter.go:72) that takes `f.mu.Lock()` and sets `f.lastRefresh = time.Time{}`, so the next `ensureFresh` (filter.go:110) reloads. Cheap, no I/O, cannot fail, and cannot make a request error out.
   - server/internal/modules/course/review/service.go: call `s.filter.Invalidate()` after the successful repo call in `CreateSensitiveWord` (after line 693), `UpdateSensitiveWord` (after line 723) and `DeleteSensitiveWord` (after line 730). Handlers at handler_sensitive_word_admin.go:60/103/125 need no change.

2. Cross-replica consistency:
   - Give `Filter` an optional version source. Add a `NewFilterWithVersion(repo, *cache.Helper)` (or a functional option on `NewService`, wired from server/internal/app/modules_course_metrics.go:40 where the Redis client / `cache.Helper` is already available). On mutation, bump `review:sensitive_words` via the existing `cache.Helper.InvalidateByVersion` used at handler.go:230/239/270; in `ensureFresh`, read that version and force a reload when it differs from the version captured at last load.
   - Guard the hot path: memoize the version probe for a few seconds (e.g. 2-5s) so a burst of posts is not one Redis GET each, and on any Redis error log-and-fall-back to the existing

#### P2-19. Review write/moderation errors return generic error codes, so users see the wrong localized message

- **位置**：`server/internal/modules/course/review/http_errors.go:17`
- **区域**：评课与内容　**类别**：correctness　**验证票数**：1/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
reviewModerationErrorMappings = []response.ErrorMapping{
	response.MatchError(ErrTitleEmpty, 400, "title cannot be empty"),
	response.MatchError(ErrTitleTooLong, 400, "title is too long", errs.ErrParamOutOfRange),
	response.MatchError(ErrDangerousContent, 400, "content contains potentially dangerous elements"),   // <- no code arg
	response.MatchError(ErrSensitiveContent, 400, "content contains sensitive words", errs.ErrSensitiveContent),
	...
	response.MatchError(ErrContentTooShort, 400, "content is too short", errs.ErrParamOutOfRange),
	response.MatchError(ErrContentTooLong, 400, "content is too long", errs.ErrParamOutOfRange),
}
reviewWriteValidationErrorMappings = []response.ErrorMapping{
	...
	response.MatchError(ErrRatingRequired, 400, "at least one rating dimension is required"),   // <- no code arg
	response.MatchError(ErrInvalidRating, 400, "rating must be between 1 and 5"),               // <- no code arg
}

// response/mapped_error.go:19 -> omitted code falls back to defaultErrorCodeForStatus(400) = errs.ErrBadRequest (A0000400)
// errs/codes.go:182-183,193,203-204 define ErrReviewContentTooShort A0110003, ErrReviewContentTooLong A0110004,
// ErrRatingInvalid A0110200, E
```

**失败场景**

User posts a review at POST /api/v1/reviews with a 5-character body. review/service.go:296 returns ErrContentTooShort; respondPostReviewError -> reviewModerationErrorMappings maps it to HTTP 400 with code errs.ErrParamOutOfRange = "A0000403". PostReviewPage.vue:1062 calls getErrorMessage(error, ...), which looks up errors.A0000403 and renders "参数超出范围" instead of the existing string errors.A0110003 = "测评内容过短". Same for over-long content, dangerous content (returns A0000400 "请求参数错误" instead of A0110300 "内容包含危险元素"), missing rating dimension and out-of-range rating. The correct localized strings are already in both zh-CN and en-US bundles and are unreachable.

**修复方案**

Apply the narrow, non-breaking subset only.

1. server/internal/modules/course/review/http_errors.go — add explicit codes ONLY where the sentinel is unambiguous:
   - line 17: `response.MatchError(ErrDangerousContent, 400, "content contains potentially dangerous elements", errs.ErrDangerousContent)` (field-agnostic message, safe for review/draft/reply/report).
   - line 21: `ErrContentTooShort` -> `errs.ErrReviewContentTooShort` (only produced by validateReviewTextLengths, service.go:296 — review create + admin edit).
   - line 56: `ErrRatingRequired` -> `errs.ErrRatingDimensionMissing`.
   - line 57: `ErrInvalidRating` -> `errs.ErrRatingInvalid` (both only produced by validateRatingValues, service.go:322/338/341/344).
   - DO NOT change line 22 (`ErrContentTooLong`) in the shared table. Either leave it at `errs.ErrParamOutOfRange`, or do it properly: add distinct sentinels `ErrReplyContentTooLong` (service_interaction.go:405) and `ErrReportDescriptionTooLong` (service_report.go:118), keep those on `errs.ErrParamOutOfRange`, and map the review/draft-body `ErrContentTooLong` to `errs.ErrReviewContentTooLong` in a review-only mapping group consumed by respondPostReviewError / respondUpdateReviewError / respondSaveDraftError / respondAdminEditReviewError but NOT by respondCreateReplyError / respondReportReviewError.

2. server/internal/modules/course/review/http_errors_test.go:26 — MUST update the pinned expectation from `code: errs.ErrBadRequest` to `code: errs.ErrRatingInvalid

#### P2-20. REFRESH MATERIALIZED VIEW CONCURRENTLY runs under the global 5s DB_QUERY_TIMEOUT, so the 000018 projection can permanently stop updating

- **位置**：`server/internal/modules/course/review/repository_teacher_public.go:106`
- **区域**：评课与内容　**类别**：correctness　**验证票数**：1/1

**证据**

```
func (r *Repository) RefreshTeacherPublicStats(ctx context.Context) error {
	ctx = withDBTable(ctx, "mv_teacher_public_stats")
	_, err := r.db.Exec(ctx, `REFRESH MATERIALIZED VIEW CONCURRENTLY mv_teacher_public_stats`)
// db.Exec (db.go:267) does: ctx, cancel := d.withTimeout(ctx)  == WithTimeout(ctx, DB_QUERY_TIMEOUT)
// DB_QUERY_TIMEOUT defaults to 5s (config.go:374) and validation.go:50 caps it at 60s
```

**失败场景**

mv_teacher_public_stats aggregates teachers LEFT JOIN departments LEFT JOIN reviews (000001_initial_schema.up.sql:973). Once that CONCURRENTLY refresh (which builds a full temp copy and diffs it) takes longer than DB_QUERY_TIMEOUT, every teacher_public_stats_refresh job claimed by runTeacherPublicStatsRefreshWorker fails with context deadline exceeded. ListPublicTeachers and ListHotTeachers read only the materialized view, so the public teacher list permanently serves stale avg_rating / review_count and omits newly created teachers, with no config escape hatch (raising DB_QUERY_TIMEOUT raises it for every query and is capped at 60s). It also amplifies: markRetrySQL (pkg/outbox/repository.go:137) resets attempt_count to 0 whenever locked_revision != revision, and the 000018 triggers bump revision on every reviews write, so the failing job never reaches dead_letter and the 2s-poll worker retries whole-view rebuilds indefinitely.

**修复方案**

Give the refresh its own budget, keep it cancellable, and stop the hot retry loop locally.

1. server/internal/pkg/db/db.go — add a long-operation entry point next to Exec (db.go:267), e.g.
   `func (d *DB) ExecWithTimeout(ctx context.Context, timeout time.Duration, sql string, args ...any) (pgconn.CommandTag, error)`
   that does `ctx, cancel := context.WithTimeout(ctxutil.Normalize(ctx), timeout)` (fall back to d.timeout when timeout <= 0) and reuses the exact same span + ObserveDBQueryDuration/ObserveDBQueryTotal path as Exec, with no automatic retry. Do NOT use ctxutil.DetachedTimeout — parent cancellation must still abort the statement so graceful shutdown works. Add a unit test asserting the explicit timeout wins over d.timeout and that parent cancel still propagates.

2. server/internal/pkg/config/config.go + validation.go — add `REVIEW_TEACHER_STATS_REFRESH_TIMEOUT` (seconds, default 60, validated e.g. 5..600, env-only per project law), and factor it into the shutdown budget in server/internal/app/server.go:52 (or document that the worker ctx cancel aborts it) so a mid-flight refresh cannot outlive shutdown.

3. Plumb the value into review.NewRepository (server/internal/app/modules_course*.go wiring) and change repository_teacher_public.go:104-110 to
   `_, err := r.db.ExecWithTimeout(ctx, r.statsRefreshTimeout, "REFRESH MATERIALIZED VIEW CONCURRENTLY mv_teacher_public_stats")`.
   SQL stays in the Repository; the timeout is config, not a hardcoded constant.

4. Add a

> **原报告完整性说明：**Claude 更新后的修复方案仍在第 4 点中途截断；新版
> `AUDIT-REPORT.md` 只多出“增加 cadence floor”的半句，没有给出算法、SLO 或回归边界。
> 下述 Codex 记录是基于当前代码与真实数据库重新完成的结论，不把该半句当成已验证方案。

#### Codex 对 P2-20 的最终复核与实施记录（2026-07-31）

**最终结论**

- 结构缺陷成立：旧 `RefreshTeacherPublicStats` 经普通 `DB.Exec` 使用全局
  `DB_QUERY_TIMEOUT`，默认 5 秒；`REFRESH MATERIALIZED VIEW CONCURRENTLY` 是全量维护
  statement，随数据增长可能合理地超过普通在线查询预算。唯一的配置逃生口曾是抬高所有查询
  的 timeout，边界不正确。
- “投影已经永久停止更新”没有现状证据。开发库只读盘点为 15 名教师、8 个院系、13 条评课、
  15 行投影；复核期间两组共 6 次真实刷新为 4.040–20.517 ms。对应统一 outbox 行当前为
  `completed`、`attempt_count=0`、`last_error=NULL`。这些小样本只否定“当前已坏”的断言，
  不能证明生产规模永远不会跨过 5 秒。
- 原报告建议 timeout 最多可到 600 秒不安全。该 worker 复用共享 outbox 的 2 分钟
  `LockStaleAfter`；维护 statement 若可超过 lease，另一 worker/实例可能把仍在执行的 job
  当成 stale 重领。因此本次把独立预算限制为 5–90 秒，给取消和最终化留出余量；若 90 秒仍
  不足，正确动作是先测锁等待/数据规模并拆分或重构投影，而不是继续放大 timeout。
- 原报告“每 2 秒无限重建”的表述也过强：未发生 revision supersession 时，既有 worker 会按
  5 秒起步指数退避并在第 5 次 dead-letter；只有刷新期间持续写入导致 revision 变化时，
  supersession fence 才会把最新修订立即重新排队。当前耗时与失败数据不足以发明新的 cadence
  状态机，故本次不修改共享 outbox 语义。

**已实施的最小修复**

1. `db.DB` 增加 `ExecWithTimeout`，与普通 `Exec` 共用同一个内部执行路径、span、
   `db_query_duration_seconds` / `db_queries_total` 指标和“非幂等命令不自动重试”语义。
   显式 context 始终从 caller 派生；进程停机、调用方取消或更短的 parent deadline 优先，
   没有使用 detached context。
2. 新增 `REVIEW_TEACHER_STATS_REFRESH_TIMEOUT_SECONDS`，默认 60、合法范围 5–90；运行时
   config、统一 validation、开发/生产 env 模板和 app wiring 同步。Repository option 仍有
   60 秒安全默认，零值 option 不会覆盖它。
3. 只有 `REFRESH MATERIALIZED VIEW CONCURRENTLY mv_teacher_public_stats` 使用专用预算；
   其余在线 SQL 继续受原 5 秒普通预算保护。配置文档明确专用预算必须小于 2 分钟 lease，
   以及超过 90 秒时的重新设计门槛。
4. 没有重复创建投影 duration/retry 指标：稳定 table hint 已使刷新进入现有
   `db_query_duration_seconds{operation="exec",table="mv_teacher_public_stats"}`；每次失败、
   重试与 terminal 状态沿统一 `outbox_job_failures_total`、结构化日志和既有 terminal alert
   观测。没有权威 freshness SLO 前不新增会立即产生任意阈值告警的 projection-age gauge。

**交叉验证**

- DB 单元测试锁定显式 timeout 覆盖普通预算、非正值回退，以及较短 parent deadline 优先。
- 真实 PostgreSQL 测试把普通预算设为 20 ms：250 ms 显式预算成功完成约 60 ms statement；
  25 ms 显式预算按 deadline 终止，预先取消的 parent 保持 `context.Canceled`。
- Repository 真实 PostgreSQL 回归把普通预算设为 1 ns、刷新专用预算设为 2 秒，物化视图仍
  成功刷新，直接证明 app 所依赖的 Repository 路径没有意外回落到全局预算。
- config 默认值和 `0/4/91` 拒绝路径、Repository 默认/自定义/零值 option、既有刷新/list
  行为均有回归；全服务端 `go test -race -p 1 ./...`、`golangci-lint`（0 issues）、生产
  二进制 build、文档卫生和全部 infra contracts 均通过。生产数据量、长事务/锁等待、刷新
  p95/p99 和多副本 stale-lease 行为仍须发布环境只读观测；本地小表结果不能替代这些验收。

#### P2-21. Caller context cancellation is recorded as an external-source failure and trips the circuit breaker

- **位置**：`server/internal/modules/externaldata/oracle_student_directory.go:218`
- **区域**：外部数据源　**类别**：resilience-correctness　**验证票数**：2/2
- **级别修正**：验证方将 P1 修正为 P2

**证据**

```
queryCtx, cancel := withOptionalTimeout(ctx, d.queryTimeout)
defer cancel()

rows, err := d.db.QueryContext(queryCtx, d.query, normalizedID)
if err != nil {
	d.breaker.RecordFailure()
	return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
}
```

**失败场景**

queryCtx is derived from the inbound Gin request context (admission/handler_user.go:370 passes c.Request.Context()). net/http cancels that context when the client disconnects. Five students on flaky mobile networks close the page / their app times out mid-request on POST /api/v1/admission/school-email/academic-match: QueryContext returns context.Canceled, so RecordFailure() runs 5 times. circuitbreaker.RecordFailure has no decay window (failures only reset on RecordSuccess, circuitbreaker.go:203-205), so with no successful lookup in between the breaker opens for EXTERNAL_STUDENT_SOURCE_ORACLE_BREAKER_OPEN_SECONDS=30. Every subsequent lookup for that school then fails fast with "circuit breaker open" -> ErrAcademicLookupUnavailable -> HTTP 503 "academic student lookup is temporarily unavailable" for all students, while the Oracle source is perfectly healthy. Half-open admits only one probe, so one more abandoned request re-opens it for another 30s.

**修复方案**

All in `server/internal/modules/externaldata/oracle_student_directory.go`:

1. Add a helper that checks the PARENT ctx (deliberately, so the directory's own `queryTimeout` expiry still counts as a source failure, since that leaves `ctx.Err() == nil`):

```go
// callerAborted reports whether err came from the inbound caller (client
// disconnect or the caller's own deadline) rather than from the Oracle source.
// The parent ctx is checked on purpose: d.queryTimeout expiring leaves
// ctx.Err() == nil and must keep feeding the breaker.
func callerAborted(ctx context.Context, err error) bool {
	if ctx.Err() == nil {
		return false
	}
	return errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded)
}
```

2. Funnel every failure path through one method instead of calling `RecordFailure` inline:

```go
func (d *OracleStudentDirectory) failSource(ctx context.Context, operation string, err error) error {
	if callerAborted(ctx, err) {
		return err // caller went away: says nothing about source health
	}
	d.breaker.RecordFailure()
	return wrapOracleStudentSourceFailure(operation, err)
}
```

Apply at line 219-222 (`return nil, d.failSource(ctx, "lookup oracle student record", err)`), line 239-242, line 243-246, and inside the deferred close at 231-237 (keeping the existing "only overwrite when err == nil" semantics). Do the same in `Probe` at 172-176 and 182-189. Leave all `RecordSuccess()` calls unchanged, and keep the `sql.ErrNoRows` branch at 183-186 as is.

3.

#### P2-22. Per-record data-integrity rejection is reported as source unavailability and opens the shared breaker

- **位置**：`server/internal/modules/externaldata/oracle_student_directory.go:238`
- **区域**：外部数据源　**类别**：resilience-correctness　**验证票数**：1/1

**证据**

```
record, err = scanOracleStudentRecords(rows, d.schoolCode, normalizedID)
if err != nil {
	d.breaker.RecordFailure()
	return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
}
// scanOracleStudentRecords (line 462):
// if id == "" || id != expectedStudentID || !schoolauth.IsValidStudentID(id) ||
//	!name.Valid || !schoolauth.IsValidAcademicName(studentName) {
//	return nil, ErrStudentSourceInvalidRecord
// }
```

**失败场景**

One row in USR_JWBIZ.T_XS_JBXX (a ~130k-row table per infra/ops/tests/provision-external-student-source-oracle-readonly-contract.sh:64) has XM NULL, or a name containing a zero-width/format character, or >80 runes. The student whose XH matches that row calls POST /api/v1/admission/school-email/academic-match: scanOracleStudentRecords returns ErrStudentSourceInvalidRecord, which is wrapped as ErrStudentSourceUnavailable -> 503 "temporarily unavailable", so the user retries. Each retry adds a breaker failure (a permanent per-row data defect can never produce a success to reset the counter), so after 5 retries the breaker opens and every other student of that school gets 503 for 30s. One malformed external row plus one retrying user takes the whole school's verification flow offline.

**修复方案**

Separate "source unhealthy" from "this row is unusable", and keep the API contract stable.

1. server/internal/modules/externaldata/oracle_student_directory.go — split the sentinel first, so a bind-parameter violation stays a source failure:
   - In scanOracleStudentRecords (line 462), split the `id != expectedStudentID` condition out into a new `ErrStudentSourceRecordIdentityMismatch` (the source ignored the bind variable — keep this as RecordFailure + ErrStudentSourceUnavailable, it is a real source-integrity defect).
   - Keep ErrStudentSourceInvalidRecord for name-level data problems (!name.Valid, !IsValidAcademicName, empty id) and ErrStudentSourceAmbiguousRecord for conflicting duplicates.
   - At LookupStudent line 238-242:
       record, err = scanOracleStudentRecords(rows, d.schoolCode, normalizedID)
       if err != nil {
           if errors.Is(err, ErrStudentSourceInvalidRecord) || errors.Is(err, ErrStudentSourceAmbiguousRecord) {
               d.breaker.RecordSuccess() // the round trip succeeded; only the row is unusable
               return nil, err           // do NOT wrap in ErrStudentSourceUnavailable
           }
           d.breaker.RecordFailure()
           return nil, wrapOracleStudentSourceFailure("lookup oracle student record", err)
       }
     (The existing deferred closeRows at line 231-237 stays inert because err != nil, so rows are still closed and no second RecordFailure fires.)
   - Add observability so a malformed row is not invisible: a de

#### P2-23. A panic in any job handler permanently kills its outbox worker for the process lifetime

- **位置**：`server/internal/pkg/outbox/worker.go:112`
- **区域**：后端公共包　**类别**：resilience　**验证票数**：1/1

**证据**

```
`ProcessBatch` invokes the job handler with no recover, and `RunPollingWorker` wraps nothing:

```go
		if err := process(ctx, job); err != nil {
```

The only recover is at the goroutine root, `server/internal/app/runtime.go:151-169`:

```go
	go func() {
		defer rt.bgWg.Done()
		defer func() {
			if r := recover(); r != nil { ... logger.L().Error("background task panicked", fields...) }
		}()
		...
		run(ctx)          // never re-invoked
	}()
```

A panic therefore unwinds the entire `for { ... }` poll loop inside `RunPollingWorker`, logs one line, and the goroutine exits for good. This governs all six workers started via `startBackgroundTask`: review notification (service_notification_outbox.go:100), review FGA sync (service_fga_sync.go:94), teacher projection (teacher_projection_worker.go:12), user external sync (external_sync.go:173), resource cleanup (background_cleanup.go:49), plus the two hand-rolled loops `runFreshmanExpiryWorker` / `runMemberBlacklistExpiryWorker` (service_expiry.go:25-49). outbox/worker_test.go has no panic-recovery test.
```

**失败场景**

`processExternalSyncJob` → `syncVerifiedStudentRole` (or the Casdoor/FGA client beneath it) nil-derefs on one malformed record — e.g. a role-sync payload for a user whose Casdoor subject was deleted, so a nil `*RoleSyncClient` field is dereferenced. The panic kills the "user external sync worker" goroutine. The process keeps serving HTTP and passes /health, but from that moment no Casdoor role sync, no FGA tuple projection and no admission-verification projection is ever processed again: `domain_event_outbox` grows unbounded and every newly verified student silently fails to receive the `verified_student` role until someone notices the single "background task panicked" log line and restarts the pod. The claimed job is stuck in `processing` and only becomes re-claimable after LockStaleAfter (2 min) — by a replica that will panic on it too.

**修复方案**

Two changes, both narrow, plus a test. The proposed fix in the finding is the right shape; I would apply it with a corrected rationale (poison-pill dead-lettering and restoring the metric signal, not the fictional nil-RoleSyncClient deref).

1. server/internal/pkg/outbox/worker.go - convert a handler panic into an ordinary job error so the EXISTING retry/dead-letter accounting applies. Add (importing "runtime/debug"):

    func safeProcess[T any](ctx context.Context, process ProcessFunc[T], job T) (err error) {
        defer func() {
            if r := recover(); r != nil {
                err = fmt.Errorf("job handler panicked: %v\n%s", r, debug.Stack())
            }
        }()
        return process(ctx, job)
    }

and change line 112 to `if err := safeProcess(ctx, process, job); err != nil {`. This is the load-bearing half: it routes the panic through markFailure -> reachedMaxAttempts (terminal after IAMWorkerMaxAttempts=5) so a poison-pill job dead-letters instead of killing the worker, AND it makes metrics.ObserveOutboxJobFailure fire so existing alerting sees the problem. It also stops the 2-minute LockStaleAfter thrash where each replica re-claims and re-panics on the same row.

2. server/internal/app/runtime.go:145-170 - make startBackgroundTask supervise instead of exiting. Restructure so the recover wraps a single invocation and the goroutine retries while the context is live: `for ctx.Err() == nil { runOnce(ctx); if ctx.Err() != nil { break }; sleep(backoff) }`


### P3（9 项）

#### P3-1. AdminContentLayout silently drops the `description` prop two pages pass to it

- **位置**：`clients/admin/apps/web-ele/src/views/shared/AdminContentLayout.vue:3`
- **区域**：Admin 前端　**类别**：dead-prop　**验证票数**：1/1

**证据**

```
AdminContentLayout.vue:2-5 → `defineProps<{ title: string; total?: number; }>();` — no `description` prop and no description slot anywhere in the template (lines 8-37).
views/users/admission-policy/index.vue:312 → `description="控制目标群入群处理方式、入群后等待时长和学生认证审核行为。"`
views/dashboard/workspace/index.vue:162 → `:description="$t('admin.dashboard.summary.title')"`
```

**失败场景**

Opening /users/admission-policy shows only the title "入群认证策略" — the explanatory sentence the author wrote never renders. Because the component does not declare the prop and does not set `inheritAttrs: false`, the value instead lands on the root `<section>` as a non-standard `description="控制目标群…"` HTML attribute, so the guidance text is present in the DOM but invisible to users and meaningless to assistive tech. Same on /workspace.

**修复方案**

1) clients/admin/apps/web-ele/src/views/shared/AdminContentLayout.vue — add the prop and render it as a real subtitle:
- Extend defineProps (lines 2-5) to `defineProps<{ title: string; description?: string; total?: number; }>();`.
- Do NOT drop the <p> straight into .admin-content-page__heading: that div is `display: flex; align-items: center` (lines 63-68) and would place the description inline next to the <h1> and total badge. Instead wrap the existing `<h1>` + total `<span>` (lines 12-18) in an inner row element (e.g. `<div class="admin-content-page__heading-row">`), make `.admin-content-page__heading` `flex-direction: column; align-items: flex-start;`, move `gap: 10px; align-items: center;` onto the new row class, and add `<p v-if="description" class="admin-content-page__description">{{ description }}</p>` after the row.
- Add CSS: `.admin-content-page__description { margin: 4px 0 0; font-size: 13px; line-height: 1.5; color: var(--el-text-color-secondary); }` and change `.admin-content-page__header`'s `align-items: center` (line 57) to `flex-start` so the #actions buttons stay top-aligned once the header is two lines tall.
- Optionally also expose a `#description` slot for rich copy, mirroring the existing $slots.actions pattern.

2) clients/admin/apps/web-ele/src/views/dashboard/workspace/index.vue:162 — do not just start rendering the current value. `:description="$t('admin.dashboard.summary.title')"` resolves to "统计概览", a section heading (it is used as an <h2>/<p> labe

#### P3-2. PersistentAdminTableColumn clobbers a caller-supplied min-width with the undefined defaultMinWidth

- **位置**：`clients/admin/apps/web-ele/src/views/shared/admin-table/PersistentAdminTableColumn.vue:64`
- **区域**：Admin 前端　**类别**：component-contract　**验证票数**：1/1

**证据**

```
PersistentAdminTableColumn.vue:61-67 →
```
<ElTableColumn
  v-bind="$attrs"
  :column-key="columnKey"
  :min-width="defaultMinWidth"
  resizable
  :width="width"
>
```
`:min-width` is merged AFTER `v-bind="$attrs"`, so the later (undefined) value wins.
views/open-platform/consents/index.vue:223,234,246,257,273,284,296 → columns declare raw `min-width="120"` … `min-width="280"` instead of `:default-min-width`.
```

**失败场景**

On /open-platform/consents every column passes `min-width="…"` through `$attrs`, but the component overwrites it with `defaultMinWidth === undefined`, so ElTableColumn falls back to its 80px floor. The 用户ID / 应用 / 授权范围 / 授权时间 / 最近使用 / 操作 columns collapse well below their intended widths on narrow viewports, truncating client IDs and scope tag lists that the page was laid out to show.

**修复方案**

Apply both halves; either alone leaves a trap.

1. clients/admin/apps/web-ele/src/views/shared/admin-table/PersistentAdminTableColumn.vue (lines 61-67) - move the min-width binding ABOVE the attrs spread so an explicitly passed attr overrides the default instead of being erased:

<ElTableColumn
  :min-width="defaultMinWidth"
  v-bind="$attrs"
  :column-key="columnKey"
  resizable
  :width="width"
>

`:column-key`, `resizable` and `:width` must stay AFTER `v-bind="$attrs"` - the persisted width from `table.columnWidth(columnKey)` has to win over any caller attr, and the column key is the persistence identity. Only `min-width` moves. `vue/attributes-order` is off, so this ordering will not fail lint. (The alternative, `:min-width="defaultMinWidth ?? ($attrs['min-width'] as number | string | undefined)"` via `useAttrs()`, works too but is more code for the same behavior.)

2. clients/admin/apps/web-ele/src/views/open-platform/consents/index.vue lines 223, 234, 246, 257, 275, 284, 296 - replace `min-width="120"` / `"220"` / `"110"` / `"280"` / `"170"` / `"170"` / `"260"` with `:default-min-width="120"` etc., matching the convention already used by the other 14 admin table pages. This is what actually restores the intended layout today; parseMinWidth accepts both the number and the string form.

3. clients/admin/apps/web-ele/src/views/shared/admin-table/PersistentAdminTable.test.ts - add `minWidth: { type: [Number, String], default: undefined }` to ElTableColumnStub, emit it as `d

#### P3-3. Freshman verification shows enabled 通过/带天数通过/驳回 buttons on already-reviewed rows; every click 409s

- **位置**：`clients/admin/apps/web-ele/src/views/users/freshman-verification/index.vue:310`
- **区域**：Admin 前端　**类别**：dead-control　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
index.vue:299-357 — the action cell is unconditional: `<div class="freshman-action-group">` with `<ElButton data-action="approve" :disabled="rowReviewing(row)" @click="approve(row)">`, `data-action="approveWithDays"`, and `data-action="reject"`. The only disable condition is `rowReviewing(row)` (an in-flight request), never `row.status`.
index.vue:45-49 — the toolbar filter offers `pending | approved | rejected`, so reviewed rows are routinely on screen.
server/internal/modules/admission/service_operator.go:104-107 → `if app.Status != FreshmanApplicationPending { return ErrAdmissionInvalidStatus }`
server/internal/modules/admission/handler_errors.go:55-56 → `case errors.Is(err, ErrAdmissionInvalidStatus): response.Conflict(c, "admission session status invalid")`
Compare identity-review/index.vue:429 (`v-if="isPending(row)"`) and student-verification/index.vue:350 (`v-if="row.verificationStatus === 'pending'"`), which do gate on status.
```

**失败场景**

An operator switches the status filter to 已通过 and clicks 通过 (or 驳回, after typing a reason) on any listed row. `PUT /api/v1/admin/freshman-verifications/{id}` reaches `reviewFreshmanApplication`, which sees `app.Status == approved`, aborts the transaction and returns HTTP 409 with the raw English body `admission session status invalid`. The admin sees a red toast plus a persistent `.admin-load-error` alert containing that untranslated string, with no indication that the action was never valid for that row.

**修复方案**

In /home/wztxy/Code/StuHelper/clients/admin/apps/web-ele/src/views/users/freshman-verification/index.vue (action cell, lines 299-358):
1. Keep the 材料预览 ElButton rendered for every row (read-only evidence viewing stays legitimate).
2. Wrap only the mutation controls - approve ElButton, the extension-days ElInputNumber, approveWithDays ElButton, the reason ElInput, and the reject ElButton - in a container with v-if="row.status === 'pending'", and render <span v-else class="admin-cell-muted">—</span>, mirroring identity-review/index.vue:429 and student-verification/index.vue:350. Reduce the column :default-width accordingly is optional; leaving 420 is fine.
3. Add defensive early returns in the script so a row that went stale between render and click cannot fire a doomed request: at the top of approve() and reject() (or inside handleReview(), alongside the existing rowReviewing check at line 120) add `if (row.status !== 'pending') return false;`. This mirrors identity-review/index.vue:136 and :187.
4. Extend tests/e2e/admin-user-actions.spec.ts with an approved-status fixture (e.g. { ...freshmanApplication, id: 'freshman-action-3', status: 'approved' }) and assert row.locator('[data-action="approve"]') has count 0 while [data-material-preview] is visible. All current fixtures are status 'pending', so existing e2e and index.test.ts assertions keep passing.

Do NOT apply the shared-result.ts part of the original proposal: ErrAdmissionInvalidStatus is emitted by response.Conflict w

#### P3-4. Two docs/design files are migration plans with pre-change narrative, one describing storage that no longer exists

- **位置**：`docs/design/member-blacklist-unification.md:19`
- **区域**：文档　**类别**：doc-accuracy　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
docs/design/member-blacklist-unification.md — status: current, `authoritative-source: server/api/openapi.yaml and server/migrations/000001_initial_schema.up.sql after implementation`.
  L12-21 "## 背景 / 当前项目存在多套成员黑名单真源" table lists "当前存储 `blacklist.json`" for three Koishi paths. `git grep blacklist.json` matches only this file and one archived exec-plan — the identifier exists nowhere in bots/, clients/ or server/.
  L226-229, L301 "`blacklist.json` 不再作为写入真源。" / "控制台黑名单页面改为调用后端 list/create/release API。" / "Koishi 不再直接修改 `blacklist.json`" — task-list wording for work already shipped (server/internal/modules/admission/repository_blacklist.go + member_blacklist_entries in migrations).

docs/design/iam-v2-casdoor.md — status: draft, `supersedes: 2026-05-01-casdoor-open-platform-iam-design.md`.
  L23 "> 本 spec 取代 `2026-05-01-…-design.md`（commit 8295a1e7）…已被本文从架构上修正。"
  L21 "**迁移性质**：绿地架构，不做兼容数据迁移；历史 Zitadel external subject、session、token 全部失效"
  L751-752 per-line change orders: "| `…/middleware/auth.go:74-76` | 修改注释 | \"Zitadel introspection\" 改为 \"Casdoor introspection\" |"
  L884 "项目已采纳 \"no compat shim\" 原则…不再保留 `external_id` 双列、rename SQL 或运行时兼容层。"
docs/design/documentation-governanc
```

**失败场景**

A new maintainer opens docs/design/member-blacklist-unification.md (linked from docs/README.md's design list, marked status: current) to learn how member blacklisting works. The 背景 table tells them Koishi writes bans to a `blacklist.json` file and that the console still owns a local JSON source of truth. They go looking for that file, find nothing, and cannot tell which half of the document is current architecture and which half is a completed 2026-05 work plan. iam-v2-casdoor.md compounds it: a reader following its "修改注释" change table would try to apply edits that were applied months ago.

**修复方案**

Two docs, plus one durable guard. Do not delete the rationale sections — fix the tense and remove the completed to-do lists.

A. /home/wztxy/Code/StuHelper/docs/design/member-blacklist-unification.md
1. L5: `authoritative-source: server/api/openapi.yaml + server/migrations/000001_initial_schema.up.sql` — drop " after implementation".
2. L12-23: keep the rationale but make it unambiguously historical. Retitle `## 背景` -> `## 为什么统一`, change L14 to past tense ("统一前项目存在多套成员黑名单真源"), and change the L16 table header from "| 来源 | 当前存储 | 当前用途 |" to "| 来源 | 统一前存储 | 统一前用途 |". This keeps the design rationale legal under documentation-governance.md:22 while removing the false present-tense claim about a file that does not exist.
3. L198-202: DELETE the entire "待移除旧路由" block. Verified gone: `git grep -n "admission/qq-users\|admission/blacklist" -- server/` returns nothing.
4. L226-231 and L301: rewrite the "改为 / 不再" task bullets as present-tense statements of shipped behavior, e.g. L227 "`blacklist.json` 不再作为写入真源。" -> delete (the file does not exist, so there is nothing to negate); L228 -> "控制台黑名单页面调用后端 list/create/release API。"; L226 -> "`event-handlers.ts` 的入群申请黑名单判断调用后端统一准入接口。"
5. L42: "不保留 Koishi `blacklist.json` 作为长期写入路径。" -> drop this 非目标 bullet (satisfied, and it is the last remaining reference to the identifier).
6. Bump `last-verified` to the re-verification date.

B. /home/wztxy/Code/StuHelper/docs/design/iam-v2-casdoor.md
1. DELETE §11 "迁移：从 Zitadel 到 Casdoor" (L738-901) and §17

#### P3-5. docs/guides/ mixes dated audit snapshots and "now changed to" narrative into status:current guides, plus two stale structure claims

- **位置**：`docs/guides/github-migration.md:170`
- **区域**：文档　**类别**：doc-accuracy　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
docs/guides/github-migration.md (type: guide, status: current)
  L168-170 "## 当前就绪状态 / 以下状态于 2026-07-30 通过 GitHub API 重新核验：" followed by an audit table of 已验证 / 部分验证 / 未就绪 / 未验证 rows.
  L36 "## 历史与秘密基线" … "曾提交的真实部署环境文件" … "已删除的内部工具缓存和内部安全审查导出"
  L146 "远端部署脚本当前仍执行 registry login。迁移 GHCR 时，需要…"
docs/guides/automation.md:165 "远端机器不再由 CI / Ansible 在每次发布时下发 `deploy.remote.env`。现在改为：" (also L142 "…把旧版由开发模板带入的 `localhost` / `http` … 重写回生产占位符")
docs/guides/release-runbook.md:25 "…已完成备份（注：`prod-deploy.sh` 现已自动在迁移前执行 `backup-postgres.sh`）。"
docs/guides/frontend-development.md:31,39 the workspace tree lists `clients/web/src/constants/` and `clients/web/src/types/`; neither directory exists (actual children: api, components, composables, design-system, directives, i18n, modules, router, stores, styles, utils).
docs/guides/frontend-development.md:47 "当前主要分为" lists 7 modules; clients/web/src/modules/ actually holds 10 (admission, open-platform and resource are missing).
docs/guides/backend-development.md:69 "`modules/rbac/` 仅保留 `middleware.go`（capability 中间件），不再是完整 RBAC 模块。" — server/internal/modules/rbac/ contains authorizer.go, middleware.go and subject.go.
docs/design/documentation-governance
```

**失败场景**

A maintainer picks up the repo after 2026-07-30 and reads github-migration.md's readiness table as current fact: it says GHCR has no published packages and both environments have empty secrets. Once someone publishes images or fills in DEPLOY_* secrets, the table is silently wrong but still labelled 当前 and status: current, so the maintainer either re-does completed setup or skips a step believing it is already done. Separately, a frontend dev following frontend-development.md:31 creates shared constants under clients/web/src/constants/ — a directory the guide invents — instead of the real home (clients/shared/src/constants/, which is the documented single source for capability constants), and never learns that admission, open-platform and resource modules exist.

**修复方案**

Apply only the verified parts; skip the backend/runbook/registry churn.

1. `docs/guides/frontend-development.md` (highest value): delete the `│   ├── constants/` line (L31) and `│   ├── types/` line (L39) from the workspace tree — neither directory exists and both were deliberately removed (see docs/internal/exec-plans/completed/2026-04/2026-04-17-audit-closed-items.md:276,366). Add one sentence under the tree: 共享常量与共享类型只放在 `clients/shared/src/constants/` 与 `clients/shared/src/types/`，不要在 `clients/web/src/` 下重建本地影子层. Then fix L47: either list all 10 modules (admission, auth, common, course, errors, home, open-platform, resource, review, user) or replace the enumeration with a pointer to `clients/web/src/router/` as the authoritative list. Bump `last-verified` to the date the fix lands.

2. `docs/guides/github-migration.md`: move the whole `## 当前就绪状态` section (L168 through the "因此，代码与仓库治理可进入 PR 审核…" paragraph) into `docs/internal/github-migration-2026-07-29.md` (or a new `docs/internal/` snapshot with `status: snapshot`), where staleness is explicitly permitted. In the guide, keep only the steady-state requirement already in `## 仓库治理验收` and add a single line pointing at the internal snapshot for the latest dated verification. Leave L36 `## 历史与秘密基线` in place — its requirements are present-tense and normative ("所有可达提交都必须持续满足…不得包含…"); at most reword the two past-tense bullets ("曾提交的真实部署环境文件", "已删除的内部工具缓存和内部安全审查导出") as categories rather than history. Leave L146 alone — it is fact

#### P3-6. School SSO admission endpoints are declared `security: []` (public) but are registered behind authMW and return an undeclared 401

- **位置**：`server/api/paths/admission.yaml:463`
- **区域**：OpenAPI 契约　**类别**：contract-mismatch　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
Spec (server/api/paths/admission.yaml:458-485) for `/api/v1/admission/school-sso/{schoolCode}/login`:
    operationId: startAdmissionSchoolSSO
    security: []
    responses:
      '302': ...
      '400': ...      # no 401
Same `security: []` at line 492 for `/callback`.

Code (server/internal/modules/admission/handler.go:84-85):
	admission.GET("/school-sso/:schoolCode/login", authMW, h.handleStartSchoolSSO)
	admission.GET("/school-sso/:schoolCode/callback", authMW, h.handleCompleteSchoolSSO)
Both handlers then call `h.resolveAdmissionUserAndSchool(c)` -> `middleware.ResolveRequiredInternalUserID`, which does `response.Unauthorized(c, "authentication required", errs.ErrLoginRequired)` (server/internal/pkg/middleware/internal_user.go:27-30).
```

**失败场景**

A user whose StuHelper session expired while at the school IdP is redirected back to `GET /api/v1/admission/school-sso/4111010006/callback?code=...&state=...`; the auth middleware returns 401 JSON. For `/login` the same 401 is possible but is not in the declared response set at all, so generated clients/error handling have no branch for it. Worse for integration: the contract states these two operations need no credentials, so any SDK, gateway or API-doc consumer generated from the spec will call them anonymously and always get 401 instead of the documented 302.

**修复方案**

Spec-only change, no Go/TS hand-edits. In server/api/paths/admission.yaml: (1) delete `security: []` at line 463 (startAdmissionSchoolSSO) and line 492 (completeAdmissionSchoolSSO) so both inherit the global `cookieAuth`/`bearerAuth` requirement (equivalently write the explicit two-item list, matching the style used by every other authenticated operation in the file). (2) Extend startAdmissionSchoolSSO's responses to match what respondAdmissionError/authMW actually return - add '401', '403', '404', '409' and '503' all as `$ref: '../components/responses/common.yaml#/ErrorResponse'` (401 = authMW/no subject; 403 = unprovisioned user; 404 = school not found; 409 = ErrAdmissionLinkedSessionRequired; 503 = ErrAdmissionSSONotConfigured / ErrAdmissionRedisUnavailable); for completeAdmissionSchoolSSO add the missing '403', '404' and '409' (401 and 503 are already declared). (3) Same pass, fix the inverse drift: add `security: []` to the three operations `GET /api/v1/admission/freshman/mobile-camera-handoffs/{token}`, `POST .../camera-capture` and `POST .../continue`, which server/internal/modules/admission/handler.go:78-80 registers with no authMW and are therefore anonymous, token-scoped endpoints. Then run `cd /home/wztxy/Code/StuHelper/server && make bundle-spec generate` (targets exist at server/Makefile:100 and :112) and commit the regenerated server/api/openapi.bundled.yaml, server/internal/api/gen/server.gen.go (embedded spec) and clients/shared/src/types/api.gen.ts so `make c

#### P3-7. Bulk admin review export writes no audit record, unlike every other sensitive admin read/write in the module

- **位置**：`server/internal/modules/course/review/admin_export.go:36`
- **区域**：评课与内容　**类别**：audit-coverage　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
admin_export.go:36-53 — full-table export with no audit.LogFromGin and no h.logAdminOp:
	func (h *Handler) ExportReviews(c *gin.Context) {
		format := c.DefaultQuery("format", "json")
		status := c.DefaultQuery("status", "all")
		...
		if format == "csv" { h.exportCSVStream(c, status); return }
		h.exportNDJSONStream(c, status)
	}

Every sibling admin route records one: admin.go:173 `h.logAdminOp(c, req.Action, "review", reviewID, ...)`, admin.go:89, handler_content_flag.go:50, handler_sensitive_word_admin.go:70/112/134. The audit package even defines the event type for this: pkg/audit/audit.go:33 `EventDataExport EventType = "data.export"` — `grep -rn EventDataExport` shows it is declared and never used anywhere. The module's own precedent for auditing a sensitive admin *read* is user/handler_admin.go:64 (audit.EventDataAccess on identity review material). The exported stream (repository_operation_log.go:130-149, status="all") includes hidden and deleted review titles/content plus moderation_reason.
```

**失败场景**

An admin with the global AdminReviewsManage capability calls GET /api/v1/course/review/admin/export?format=csv&status=all and downloads every review in the platform, including hidden and soft-deleted content and moderation reasons. Nothing lands in audit_events or admin_operation_logs, so a later incident review of "who exfiltrated the review corpus and when" has no record — while a single hide of one review is fully logged. No page/limit bound applies either, so the size of the exfiltration is also unrecorded.

**修复方案**

In server/internal/modules/course/review/admin_export.go, emit exactly ONE audit event from the export path — do not also call h.logAdminOp (that helper writes the same audit_events table via repository_operation_log.go:30-47 and would produce a duplicate row typed data.update, since eventTypeForOperationAction("export") hits the default branch).

Concretely:
1. Change exportNDJSONStream / exportCSVStream to return (rowCount int, err error), incrementing a counter inside the StreamExportReviews callback.
2. In ExportReviews (admin_export.go:36-53), after the chosen stream returns, call:
   audit.LogFromGin(c, audit.Event{
     Type: audit.EventDataExport,
     Category: "admin_operation",   // required so the row appears in GET /api/v1/course/review/admin/logs (ListAdminOperations filters category='admin_operation') and inherits the 90-day CleanupAdminOperations retention
     ActorType: "admin",
     Resource: "review", ResourceType: "review", ResourceID: "bulk",
     Action: "export",
     Result: "success" | "failure",
     Reason: streamErr.Error() on failure,
     Details: map[string]any{"format": format, "status": status, "row_count": n, "row_limit": 10000},
   })
   Emit the failure variant (with rows written so far) when StreamExportReviews errors, since the response body is already partially streamed and the client keeps a truncated file.
3. Add a handler test asserting the event fires for both format=csv and format=ndjson, success and stream-error paths (the existin

#### Codex 对 P3-7 的最终复核与实施记录（2026-07-31）

**最终结论**

缺少批量导出审计真实存在，并且应按 P2 安全审计缺口处理。导出由全局
`admin:reviews:manage` capability 保护且 SQL 已有 10,000 行硬上限，因此不是“任意用户无限
导出”；但 `status=all` 确实包含隐藏和软删除评课，NDJSON 还会序列化
`moderationReason`。CSV 包含标题/正文和状态，但列清单不含 moderation reason，Claude 把
这一字段套到 CSV 的描述不准确。权限与上限不能替代“谁在何时导出、成功还是截断”的留痕。

**已实施的最小修复**

1. NDJSON/CSV stream helper 返回 `(rowCount, error)`；数据库迭代、JSON/CSV 写入、CSV flush
   和完成标记失败都会传播到统一出口。`rowCount` 表示服务端已成功序列化/处理的记录，不伪装
   成 TCP 对端或客户端磁盘已确认。
2. `ExportReviews` 在单一出口调用一次 `audit.LogFromGin`，事件为
   `category=admin_operation`、`event_type=data.export`、`resource_type=review`、
   `resource_id=bulk`，记录规范化 format/status、row count 和既有 10,000 行上限。
   success/failure 共用同一 envelope，failure 保存错误原因。
3. 未调用 `h.logAdminOp`，避免同一导出产生 `data.update` 与 `data.export` 两条记录；未新增
   audit 表、消息队列、管理 UI 或第二套 retention。事件沿用管理员操作的现有查询与 90 天清理
   路径。

**交叉验证**

- 隔离 PostgreSQL 18 为 NDJSON/CSV 各执行一次成功导出，响应均有数据和完成标记，落库
  `row_count=1`、`result=success`。
- 同两种格式各用预取消请求 context 触发真实查询失败，响应无完成标记、落库
  `row_count=0`、`result=failure` 且 reason 非空；审计写使用既有 `WithoutCancel` 派生上下文，
  因此没有随客户端取消丢失。
- 四个唯一 request ID 均精确匹配一条事件，actor、resource、action、filter 和 row limit
  均从 `audit_events` 复读断言。当前验证不等同于生产 Loki/数据库留存、告警或管理员页面
  人工验收，发布后仍需用受控账号做一次导出并确认查询和 retention。

#### P3-8. Three DELETE operations declare 204 No Content but the handlers return 200 with a JSON envelope

- **位置**：`server/internal/modules/course/review/review_interaction.go:84`
- **区域**：评课与内容　**类别**：contract-mismatch　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
Spec declares only 204 (no content):
  server/api/paths/review-favorite.yaml:70  '204': description: 取消收藏成功   (removeFavorite)
  server/api/paths/review-admin.yaml:563    '204': description: 删除成功       (deleteTeacher)
  server/api/paths/review-admin.yaml:705    '204': description: 删除成功       (deleteSensitiveWord)

Handlers all return 200 + body:
  review_interaction.go:84            response.Success(c, gin.H{"message": "favorite removed successfully"})
  handler_teacher_admin.go:122        response.Success(c, gin.H{"message": "teacher deleted successfully"})
  handler_sensitive_word_admin.go:136 response.Success(c, gin.H{"message": "sensitive word deleted"})

Generated types encode the wrong shape (clients/shared/src/types/api.gen.ts:9251-9258): `responses: { 204: { content?: never }, 401: ..., 403: ..., 500: ... }` — 200 is not a declared outcome at all.
```

**失败场景**

`adminApi.deleteTeacher(7)` returns HTTP 200 with `{"success":true,"data":{"message":"teacher deleted successfully"}}`, but the generated TS type says the only success is 204 with `content?: never`, so `result.data` is typed as never/undefined and the message is unreadable without a cast. Any consumer that branches on `response.status === 204` (or an OpenAPI response validator / contract test, which the repo already has the machinery for via kin-openapi) classifies every successful delete as a contract violation. Current callers only survive because `unwrapVoid` (clients/admin/apps/web-ele/src/api/shared-result.ts:73-82) checks a 200-299 range.

**修复方案**

Fix the SPEC, not the handlers (opposite of the finding's stated preference).

1. server/api/paths/review-favorite.yaml:70 — replace the bare `'204': description: 取消收藏成功` with a 200 response identical in shape to the sibling `addFavorite` at lines 37-52:
     '200':
       description: 取消收藏成功
       content:
         application/json:
           schema:
             allOf:
               - $ref: '../components/schemas/common.yaml#/SuccessResponse'
               - type: object
                 required: [data]
                 properties:
                   data:
                     $ref: '../components/schemas/common.yaml#/MessageData'
2. server/api/paths/review-admin.yaml:563 (deleteTeacher) and :705 (deleteSensitiveWord) — same replacement, description 删除成功. Keep the existing 401/403/404/500 refs untouched.
3. Regenerate and commit all three generated artifacts: `make bundle-spec` in server/ (redocly -> server/api/openapi.bundled.yaml), then `go generate ./internal/api/gen` (oapi-codegen) and `pnpm api:generate` in clients/ (openapi-typescript -> clients/shared/src/types/api.gen.ts). Do not hand-edit api.gen.ts. `make` has a bundle-sync check at server/Makefile:124 and clients has `check:api-drift`, so both will verify the result.

Rationale: this aligns the three outliers with the other five DELETE operations (all 200 + MessageData), makes `result.data.message` correctly typed for future consumers, and changes no server behavior — so no risk to the shipped web/uniappx/ad

#### P3-9. Cache version falls back to "0" and caches it on Redis error, so invalidated payloads can be re-served

- **位置**：`server/internal/pkg/cache/cache.go:268`
- **区域**：后端公共包　**类别**：cache-correctness　**验证票数**：1/1
- **级别修正**：验证方将 P2 修正为 P3

**证据**

```
```go
		version, err := h.client.Get(ctx, vk).Result()
		if err != nil {
			if errors.Is(err, redis.Nil) {
				return "0", nil
			}
			// 非 key-not-found 错误，记录日志并返回默认值
			logger.L().Warn("failed to get cache version from redis", zap.String("key", vk), zap.Error(err))
			return "0", nil
		}
```

A transport error is treated identically to "version key absent", and the caller then *caches* that answer for `versionLocalTTL` (cache.go:280-288). `BuildVersionedKey` (cache.go:294-297) therefore builds `prefix:v0:key`, and callers both read and write at that key (e.g. course.go:125/151, review/cache_response.go:26/46). `InvalidateByVersion` (cache.go:332-359) only ever bumps the counter forward, so v0 entries are never invalidated — they persist for their full `cache.DefaultTTL` (5 min) with `JitteredTTL`. The singleflight wrapper amplifies it: the flight captures the first caller's `ctx` (cache.go:249-271), so one client disconnect makes `Get` return context.Canceled and every request sharing that flight gets version "0".
```

**失败场景**

Redis has intermittent timeouts under load. Request A hits a timeout, GetVersion returns "0", the handler loads from Postgres and writes the payload to `review:course:v0:<key>` with a 5-minute TTL. Seconds later an admin hides an abusive review; `InvalidateByVersion("review:course")` bumps the real version to 7, which does nothing to the v0 entry. A subsequent Redis blip makes GetVersion return "0" again, the handler reads `review:course:v0:<key>` and serves the pre-hide payload — the hidden review reappears to users for up to 5 minutes with only a Warn log to explain it. The same mechanism serves any invalidated course/review list from the orphaned v0 namespace.

**修复方案**

Apply in server/internal/pkg/cache/cache.go, cheapest-first:

1. Stop one caller from poisoning the shared flight. Inside the GetVersion singleflight loader (cache.go:249-271), do not use the captured caller ctx for the Redis GET. Use the existing helper: `loaderCtx, cancel := ctxutil.DetachedTimeout(ctx, 200*time.Millisecond); defer cancel()` (server/internal/pkg/ctxutil/context.go:26, already used in audit.go:165 and oidc/verifier.go:47) and call `h.client.Get(loaderCtx, vk)`. This removes the most likely trigger: a client disconnecting mid-flight no longer makes healthy waiters — and the 1s process-wide version cache — see version "0".

2. Distinguish absent from unavailable. In the loader, keep `return "0", nil` only for `errors.Is(err, redis.Nil)`; for any other error log the warning and `return "", err`. Then in GetVersion, when DoValue returns an error, return the unknown signal and do NOT execute the local-cache write at cache.go:280-288, so a transport blip is never memoized for 1s.

3. Make "unknown version" bypass the cache instead of aliasing the live v0 namespace. Have GetVersion expose an ok/err form (e.g. add `GetVersionOK(ctx, prefix) (string, bool)`; keep GetVersion as a thin wrapper for the existing tests) and have BuildVersionedKey return `""` when the version is unknown. Add an empty-key guard at the top of GetRaw (cache.go:129), GetAs (cache.go:145) and Set (cache.go:166) so an empty key is a miss / no-op. That makes all five production call sites — cours

**Codex 复核与处置（2026-07-31）**

核心问题确认存在，且因失效后的旧内容可能重新出现，优先级从原报告 P3 调整为 P2。最终实现
保持现有 API 和初始 `v0` 语义：只有 `redis.Nil` 表示版本 key 尚未创建；Redis transport
error 或 caller context 已取消时，`GetVersion` 返回空字符串且不写 1 秒本地版本缓存，
`BuildVersionedKey` 传播为空 key，`GetRaw`/`GetAs` 把空 key 当 miss，`Set` 把空 key 当
no-op。因此五个生产调用点都会在该请求中回源权威数据库，但不会读取或重新写入孤立的
`prefix:v0:*` namespace。

故障回归先把真实版本 `7` 写入 miniredis，再关闭 socket：版本读取与 key 构建均返回 unknown，
本地版本 map 没有该条目，空 key 的 raw/typed get 为 miss、set 在 Redis 已关闭时仍安全 no-op；
重启同一 miniredis 后立即恢复为 `reviews:v7:course:123`。独立用例还覆盖预取消 context，
证明它不会被 memoize 为 `v0`。未采用原方案第 1 步的 200ms detached loader：这只能改善
singleflight 中首 caller 取消时其他等待者的命中率，不是消除陈旧数据所必需；当前等待者共同
bypass 并回源数据库是安全降级，等有 Redis/DB 延迟指标证明需要后再单独优化。


## Claude 原审计列为“被证伪”的发现（保留原始记录）

> **重要更正：不能再将本节全部解释为“不应修改”。**Codex 独立复核后，18 条中只有 6 条
> 维持驳回；R-6、R-8、R-11、R-17 应重新进入待办，另有 8 条属于部分成立、降级、加固或
> 延期项。最终结论见上方“对原‘被证伪’18 项的再复核”表。

### ~~Both stuhelper-core and stuhelper-group-guard register `guild-member-request`, so every join request is decided and answered twice~~

- 原报级别：P1　位置：`bots/koishi/plugins/stuhelper-group-guard/src/events.ts:25`　票数：0/2

> **驳回理由**（high 置信度）：REFUTED. The finding's central premise -- that stuhelper-core also registers `guild-member-request` -- is false at runtime, so there is exactly one listener and the race cannot occur.

1) Core's listener is unreachable dead code. `bots/koishi/plugins/stuhelper-core/src/core/modules/event-handlers.ts:33` sits inside `registerEventListeners(host)`, which is called only from `EventModule.init()` (core/modules/event.module.ts:55). `EventModule` is only constructed by `eventRuntimeModule` (event.module.ts:70-78), which is only listed in `MODULE_REGISTRATIONS` (runtime/registry.ts:45), which is only read by `getRuntimeModules()` (registry.ts:52). Grepping core's src for `getRuntimeModules` returns ONLY its own definition plus `registry.test.ts` -- no production caller. The actual plugin entry (`src/index.ts:37`) calls `registerRuntimeModules`, which is a no-op stub: `bots/koishi/plugins/stuhel

> **驳回理由**（high 置信度）：The finding's central premise is false: stuhelper-core does NOT register a `guild-member-request` listener at runtime, so nothing is handled twice.

Trace of core's listener, end to end:
- `/home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-core/src/core/modules/event-handlers.ts:33` (`host.ctx.on('guild-member-request', ...)`) lives inside `registerEventListeners(host)`.
- The only non-test caller of `registerEventListeners` is `EventModule.init()` (`src/core/modules/event.module.ts:55`).
- `EventModule` is only constructed by `eventRuntimeModule` (`event.module.ts:70`), which is only referenced by `src/runtime/registry.ts:9,45`.
- `registry.ts`'s `getRuntimeModules()` has ZERO non-test callers (`grep -rn "getRuntimeModules"` → only registry.ts:52 definition + registry.test.ts). `StuhelperGroupCenterService.registerModule` / `initModules` (`src/core/services/stuhelper-group-center


### ~~Operation-logs pagination never binds page-size: page count is computed from 10 while requests use 20, and the size selector is dead~~

- 原报级别：P1　位置：`clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue:170`　票数：0/2

> **驳回理由**（high 置信度）：REFUTED — the finding's core claim is contradicted by the current code; its evidence quote is the real template block with the decisive line deleted.

1. The binding exists. clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue:170-179 reads:
   <ElPagination
     v-model:current-page="query.page"        (171)
     v-model:page-size="query.pageSize"       (172)  <-- PRESENT; omitted from the finding's quote
     background                               (173)
     layout="prev, pager, next, sizes, total" (174)
     :page-sizes="[10, 20, 50, 100]"          (175)
     :total="total"                           (176)
     @current-change="refreshPage"            (177)
     @size-change="refreshPage(1)"            (178)
   />                                         (179)
   The finding's 8-line quote is byte-identical to the real 9-line block minus line 172, i.e. the "missing" l

> **驳回理由**（high 置信度）：The finding's central evidence is factually false for the current code. All three sub-claims fail.

(1) MISQUOTED EVIDENCE — `v-model:page-size` IS bound. clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue:170-179 actually reads:
  <ElPagination
    v-model:current-page="query.page"
    v-model:page-size="query.pageSize"     <-- line 172
    background
    layout="prev, pager, next, sizes, total"
    :page-sizes="[10, 20, 50, 100]"
    :total="total"
    @current-change="refreshPage"
    @size-change="refreshPage(1)"
  />
Line 172 is silently absent from the finding's quoted block. This is committed at HEAD (`git show HEAD:clients/admin/apps/web-ele/src/views/content/operation-logs/index.vue`), `git status --porcelain` on the file is clean, and `find` confirms exactly one operation-logs/index.vue exists in the repo — so the auditor was not looking at a different worktr


### ~~Identity review approves on URL presence only — a deleted/never-uploaded or expired-signature photo still enables Approve~~

- 原报级别：P1　位置：`clients/admin/apps/web-ele/src/views/users/identity-review/index.vue:73`　票数：0/2

> **驳回理由**（high 置信度）：REFUTED. The finding's frontend observations are accurate as code descriptions, but the load-bearing backend claim ("`PresignGetURL` is a local signing operation and never touches the network... there is no HEAD/Stat existence check") is factually wrong for the production code path, and that breaks both failure cases.

What the audit missed: `s.photoStore` is NOT an `*objectstorage.Store`. Production wiring at /home/wztxy/Code/StuHelper/server/internal/app/modules.go:292 is `user.WithIdentityPhotoStore(newIdentityPhotoStorageAdapter(storageService, storage.DefaultMountKey))`. That adapter (/home/wztxy/Code/StuHelper/server/internal/app/identity_photo_storage_adapter.go:33-36) implements `PresignGetURL` as:
  `url, err := a.service.GetDownloadURLByMountKey(ctx, a.mountKey, key)`
and `storage.Service.GetDownloadURLByMountKey` (/home/wztxy/Code/StuHelper/server/internal/modules/storage/serv

> **驳回理由**（high 置信度）：REFUTED — the core evidence claim is wrong, and the scenario fail-closes end to end.

1) "resolveIdentityReviewPhoto ... there is no HEAD/Stat existence check" is false for the production wiring. `photoStore` is never a raw `*objectstorage.Store`; the sole non-test injection is server/internal/app/modules.go:292 → `newIdentityPhotoStorageAdapter(storageService, storage.DefaultMountKey)`. That adapter's PresignGetURL (server/internal/app/identity_photo_storage_adapter.go:34-37) calls `storage.Service.GetDownloadURLByMountKey`, which at server/internal/modules/storage/service.go:231-234 runs `ensureObjectExists(ctx, driver, mount, objectKey)` BEFORE presigning; that calls `driver.Stat` (storage/driver.go:92-105) → `objectstorage.Store.Stat` → S3 HeadObject (objectstorage/store.go:165-181). The audit's grep was scoped to internal/modules/user/*.go and missed the check one layer down.

2) Ca


### ~~Runtime payload-guard helpers are hand-copied into 19 web files while clients/shared already exports isRecord~~

- 原报级别：P2　位置：`clients/web/src/modules/course/coursePayload.ts:17`　票数：0/1

> **驳回理由**（high 置信度）：The raw duplication count checks out, but the two load-bearing claims in the finding -- "the bodies are already identical / this is a pure move" and the failure scenario -- are both factually false against the current code, so the finding as written should not be acted on.

WHAT IS TRUE (verified):
- 19 private `function isRecord` definitions exist in non-test clients/web files, exactly the list given (grep count = 19).
- 15 web files define read*/readOptional* families whose bodies are semantically identical modulo formatting (/home/wztxy/Code/StuHelper/clients/web/src/stores/auth.ts, stores/user.ts, modules/review/reviewListPayload.ts, modules/open-platform/pagePayload.ts all have byte-equivalent `readString`/`readInteger`/`readBoolean`/`readNumber` bodies; the stores use 4-space/double-quote/semicolon formatting, the modules use 2-space/single-quote).
- /home/wztxy/Code/StuHelper/clie


### ~~Error-code reference documents 49 codes the server never emits and omits the 6 admission codes it does emit~~

- 原报级别：P2　位置：`docs/reference/error-codes.md:136`　票数：0/1

> **驳回理由**（high 置信度）：The raw facts are largely accurate, but the finding fails on failure scenario, framing, and fix quality.

WHAT CHECKS OUT
- I scripted qualified-reference analysis over all 116 `ErrorCode` constants in `/home/wztxy/Code/StuHelper/server/internal/pkg/errs/codes.go`: 51 (not 49) have no non-test `errs.X` emission site. Close enough.
- `/home/wztxy/Code/StuHelper/server/internal/modules/admission/handler_errors.go:14-19` really does declare 6 dotted codes outside `errs/codes.go`, and `handler_errors.go:48` really returns HTTP 410 + `admission.token_expired`. They are absent from `docs/reference/error-codes.md`.

WHY THE FAILURE SCENARIO CANNOT OCCUR
Every real consumer already handles those 6 codes by exact string, with tests:
- `/home/wztxy/Code/StuHelper/clients/web/src/modules/admission/admissionToken.ts:11-22` defines TERMINAL/INVALID/EXPIRED code sets containing all of them; `mapAdmiss


### ~~429 is not declared on four endpoints that carry EndpointRateLimitMiddleware~~

- 原报级别：P2　位置：`server/api/paths/user-identity.yaml:90`　票数：0/1

> **驳回理由**（high 置信度）：The factual observation is accurate, but the stated failure scenario is impossible and the P2 severity rests entirely on a fabricated mechanism.

WHAT CHECKS OUT
- /home/wztxy/Code/StuHelper/server/api/paths/user-identity.yaml:90-95 — uploadIdentityPhoto declares only 201/400/401/500. Confirmed.
- The routes really do carry the limiter: server/internal/modules/user/handler.go:101-105, 113, 114 and server/internal/modules/auth/handler.go:201; EndpointRateLimitMiddleware returns 429 via response.RateLimitExceeded (server/internal/pkg/middleware/ratelimit.go:241-245 -> server/internal/pkg/response/response.go:136-142). Confirmed.
- The internal inconsistency is real: exchangeNative (auth.yaml:448) declares 429 on the same refreshLimiter; requestStudentEmailOTP (:252) and verifyStudentEmailOTP (:326) declare 429 on the same verifyLimiter; review write ops declare 429 (review-crud.yaml:269,31


### ~~`capturedAt` is part of the declared camera-capture request and is sent by the web client, but the Go DTO has no such field so it is silently discarded~~

- 原报级别：P2　位置：`server/internal/modules/admission/handler_user.go:28`　票数：0/1

> **驳回理由**（high 置信度）：The mechanical claim is accurate, but the failure scenario is factually wrong and the field is inert, so this is a P3-at-most spec-hygiene nit, not a defect worth changing.

WHAT I CONFIRMED AS TRUE:
- /home/wztxy/Code/StuHelper/server/api/components/schemas/admission.yaml:500-511 declares `capturedAt` (type: string, format: date-time) as an OPTIONAL property (`required: [contentType, imageBase64]`).
- /home/wztxy/Code/StuHelper/server/internal/modules/admission/handler_user.go:28-31 binds only ContentType/ImageBase64, and both handlers (handler_user.go:137-142 and 307-311) build `CameraCaptureInput` / `FreshmanCameraHandoffCaptureInput` without any timestamp. models.go:249-254 and 286-290 have no CapturedAt field. `capturedAt` reaches nothing below the handler; buildFreshmanMaterial (service_freshman.go:593-609) and newFreshmanMaterialRecord (service_freshman.go:642-659) never see a tim


### ~~Redis outage is reported to users as an OTP cooldown; the error branch is dead code~~

- 原报级别：P2　位置：`server/internal/modules/admission/service_student.go:320`　票数：0/1

> **驳回理由**（high 置信度）：The code-level observation is accurate, but the severity-justifying failure scenario cannot occur — it is already handled by two independent fail-closed guards on the same Redis client, so this does not deserve to be reported as a P2 error-handling finding.

WHAT IS TRUE. At /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_student.go:314-327 the ordering is inverted: `if errors.Is(err, redis.Nil) || result != "OK" { return ErrAdmissionOTPCooldown }` precedes `if err != nil`. go-redis `StatusCmd.Result()` is `return cmd.val, cmd.err` (/home/wztxy/go/pkg/mod/github.com/redis/go-redis/v9@v9.18.0/command.go:821-823) and `cmd.val` is never assigned on failure, so on any non-Nil error `result == ""` and line 320 wins; lines 323-325 are unreachable. The twin at /home/wztxy/Code/StuHelper/server/internal/modules/user/service_student_email_otp.go:399-411 is identical, and /hom


### ~~RefreshCourseRatingStatsTx and RefreshTeacherRatingStatsTx are ~90 verbatim-duplicated lines differing by four tokens~~

- 原报级别：P2　位置：`server/internal/modules/course/review/repository_rating.go:247`　票数：0/1

> **驳回理由**（high 置信度）：The duplication itself is factually accurate, but the finding does not survive as a P2 defect.

WHAT I CONFIRMED (the finding's factual core is right):
Cited location is correct. `diff` of `server/internal/modules/course/review/repository_rating.go:247-336` against `server/internal/modules/course/review/repository_rating_stats.go:103-192` yields exactly 90 lines each, differing only in the 4 spots claimed: `r.teacher_id`/`r.course_id`, the DELETE table, the INSERT table + column list, and the `fmt.Errorf` label. Same base/stats/dist CTE, same local `statRow`, same scan loop, same DELETE-then-multi-VALUES-INSERT with 7 placeholders per row. No test pins the two to equivalent behavior (`service_review_stats_test.go:31,36` only counts rows in each table).

WHY I REFUTE ANYWAY:

1. The stated failure scenario is provably impossible. `ReviewRatings` is `map[string]int` (`model.go:10`), and `v


### ~~QueryTimeout also bounds connection establishment, so the configured 5s Oracle connect timeout is unreachable~~

- 原报级别：P2　位置：`server/internal/modules/externaldata/oracle_student_directory.go:215`　票数：0/1

> **驳回理由**（high 置信度）：The mechanical claim is accurate, but it is not a defect and the proposed fix would be a regression.

What I confirmed (mechanism is real):
- server/internal/modules/externaldata/oracle_student_directory.go:215-218 wraps the lookup in `withOptionalTimeout(ctx, d.queryTimeout)` and then calls `d.db.QueryContext(queryCtx, ...)`. `withOptionalTimeout` (lines 484-489) always installs a deadline because `normalizeOracleStudentDirectoryConfig` clamps QueryTimeout to 1-60s with a 3s default (lines 24, 306-311).
- go-ora v2.9.0 really does turn the DSN option into the dialer timeout: configurations/connect_config.go:251-258 maps `CONNECTION TIMEOUT` -> `SessionInfo.ConnectTimeout`, and network/session.go:519-542 dials via `net.Dialer{Timeout: ConnectTimeout}.DialContext(ctx)`. connection.go:459-462 additionally arms `session.StartContext(ctx)`, whose watchdog breaks/disconnects the session on ct


### ~~Migration 000018 triggers funnel every reviews/teachers/departments write through one outbox row, serializing all review writes~~

- 原报级别：P2　位置：`server/migrations/000018_teacher_public_stats_projection.up.sql:32`　票数：0/1

> **驳回理由**（high 置信度）：The cited code is quoted accurately and the Postgres mechanism is real, but this is a deliberate, documented, test-locked design whose expensive work is already kept outside the lock, and both proposed fixes are regressions.

CONFIRMED MECHANICS: server/migrations/000018_teacher_public_stats_projection.up.sql:22 hardcodes dedupe_key='teacher_public_stats'; line 32 is ON CONFLICT (stream, dedupe_key); domain_event_outbox_stream_dedupe_idx is UNIQUE on (stream, dedupe_key) at 000001_initial_schema.up.sql:2786. So INSERT ... ON CONFLICT DO UPDATE does take a row-exclusive lock on one tuple, held to commit, and concurrent qualifying statements serialize. PostReview (service_review_write.go:170 CreateReturning -> 194 IncrementCourseReviewCount -> 197 refreshReviewTargetTx -> 200 enqueueReviewFGASyncTx) and BatchUpdateReviews (service_admin.go:376-394, maxBatchSize=100 at admin.go:20) do keep


### ~~Disabling both reminder delivery channels makes the bot report reminders as successfully delivered~~

- 原报级别：P3　位置：`bots/koishi/plugins/stuhelper-group-guard/src/admission-actions.ts:93`　票数：0/1

> **驳回理由**（high 置信度）：The mechanics the finding describes are real, but they are the deliberately chosen, documented, and tested product behavior — not a defect — and the proposed fix would reintroduce a constraint the team removed two days ago plus create new failure loops.

What I confirmed as mechanically accurate:
- /home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/admission-reminder-delivery.ts:84-87 returns `{}` (no `cancelled`) when both channels are off.
- /home/wztxy/Code/StuHelper/bots/koishi/plugins/stuhelper-group-guard/src/admission-actions.ts:93-96 therefore ACKs `{action:'remind', success:true}`, mark `'reminder'`.
- Backend on that ACK: /home/wztxy/Code/StuHelper/server/internal/modules/admission/service_session.go:890-898 -> `applySuccessfulBotEventTx` -> `MarkReminderSentTx` (/home/wztxy/Code/StuHelper/server/internal/modules/admission/repository_bot_queries.go:94-98:


### ~~Least-privilege gate never ties the runtime Oracle account to the provisioned read-only account~~

- 原报级别：P3　位置：`infra/ops/lib/common.sh:399`　票数：0/1

> **驳回理由**（high 置信度）：REFUTED. The code fact (no gate ties EXTERNAL_STUDENT_SOURCE_ORACLE_USERNAME to EXTERNAL_STUDENT_SOURCE_ORACLE_READONLY_USERNAME) is true, but both halves of the stated failure scenario fail verification, and the proposed fix is not a clear improvement.

(1) Cited line is off by one: the gate is infra/ops/lib/common.sh:400-401, not :399.

(2) The SYS/SYSTEM half is already handled, and specifically as a PRE-DEPLOY die — the opposite of what the finding claims. server/internal/modules/externaldata/oracle_student_directory.go:285-287 calls isDisallowedOracleRuntimeUsername (defined :396-403, rejecting SYS/SYSBACKUP/SYSDG/SYSKM/SYSRAC/SYSTEM) inside normalizeOracleStudentDirectoryConfig, which runs from NewOracleStudentDirectory — invoked by server/cmd/external-student-source-smoke/main.go:99. So infra/ops/external-student-source-smoke.sh, which infra/ops/admission-student-source-go-live.sh


### ~~renovate.json coexists with .github/dependabot.yml, giving two competing dependency bots for the same ecosystems~~

- 原报级别：P3　位置：`renovate.json:1`　票数：0/1

> **驳回理由**（high 置信度）：The surface observation is accurate, but every consequence claimed in the failure scenario is either impossible in the current state or factually wrong.

WHAT CHECKS OUT
- Both files exist: `renovate.json` (658 B) and `.github/dependabot.yml` (1461 B).
- `docs/guides/github-migration.md` really does name Dependabot the single authority ("依赖更新由 `.github/dependabot.yml` 每周检查 GitHub Actions、Go modules、三个 JavaScript workspace 和 Docker 基础镜像。"), and never mentions Renovate.
- `renovate.json` is an unreferenced pre-migration leftover: added 2026-04-09 in `4acafde8 chore: checkpoint current workspace changes`, whereas `dependabot.yml` arrived 2026-07-28 in `c63c428f ci(github): prepare public monorepo migration`.

WHY IT IS STILL NOT A DEFECT

1. Renovate is not installed, so `renovate.json` is inert dead config, not a "competing bot". Renovate only acts as an installed GitHub App or via a self-


### ~~`IdentityPhotoUploadResult` declares three fields (rejectionReason, createdAt, updatedAt) that the upload handler never returns~~

- 原报级别：P3　位置：`server/api/components/schemas/user-system.yaml:165`　票数：0/1

> **驳回理由**（high 置信度）：The finding's raw observation is factually accurate, but it is not a defect and the claimed failure scenario is impossible. Refuted on four independent grounds.

1) It is NOT a contract mismatch. The three fields sit outside `required` (server/api/components/schemas/user-system.yaml:167 declares only `required: [key]`), so they are optional. A response of `{"key":"identity/...jpg"}` fully validates against `IdentityPhotoUploadResult`. Nothing is violated — no schema validation fails, and the generated code faithfully mirrors the spec. The generated Go type even renders them as `*time.Time` / `*string` with `,omitempty` (server/internal/api/gen/server.gen.go:6905-6912), i.e. the contract explicitly anticipates their absence. The category label "contract-mismatch" is wrong; this is at most spec over-declaration.

2) The failure scenario cannot occur. The claim is that a client "reads `unde


### ~~academics.ReplaceSnapshot upserts one row per round trip inside a transaction hard-bounded to DB_QUERY_TIMEOUT x 3~~

- 原报级别：P3　位置：`server/internal/modules/academics/repository_import.go:119`　票数：0/1

> **驳回理由**（high 置信度）：The mechanical claims are factually accurate, but the failure scenario cannot occur in the current code, and the proposed fix is not a clear improvement. Refuting on both grounds.

WHAT I CONFIRMED AS TRUE
1. /home/wztxy/Code/StuHelper/server/internal/modules/academics/repository_import.go:117-131 is indeed a per-row `tx.QueryRow(... RETURNING id)` loop, and the same row-at-a-time shape repeats at lines 137, 157, 183, 237, 245 and 271.
2. /home/wztxy/Code/StuHelper/server/internal/pkg/db/db.go:387 `txTimeout := d.timeout * txTimeoutMultiplier` with `txTimeoutMultiplier = 3` (db.go:30), and `DB_QUERY_TIMEOUT` defaults to 5 (config.go:374, .env.example:39), so the whole `WithTx` body really is bounded at 15s by default. db.go:431-438 also rejects the commit if the ctx expired, so an overrun does roll the import back, exactly as claimed.

WHY IT IS STILL REFUTED
3. The failure scenario has


### ~~Review list responses return `page`/`pageSize` that the spec's data schema does not declare~~

- 原报级别：P3　位置：`server/internal/modules/course/review/response_contract.go:5`　票数：0/1

> **驳回理由**（high 置信度）：The literal observation is factually accurate but the defect and its failure scenario are not.

VERIFIED TRUE (facts): `/home/wztxy/Code/StuHelper/server/internal/modules/course/review/response_contract.go:5-10` does emit `page`/`pageSize`, used at `review_read.go:70,114,173` and `response_contract.go:34`; and `/home/wztxy/Code/StuHelper/server/api/paths/review-crud.yaml` declares only `list`/`total` for all four data objects (lines 39-51, 88-100, 145-157, and 216-226 for the batch group), so `clients/shared/src/types/api.gen.ts:8299-8303` types `data` as `{ list: Review[]; total: number }`.

WHY IT IS REFUTED:
1. Not a contract violation. None of those response `data` objects sets `additionalProperties: false`, so extra members are permitted by the spec; the spec is merely under-declared. The runtime OpenAPI middleware validates requests only (`server/internal/pkg/middleware/openapi_val


### ~~Six exported helpers in server/internal/pkg have zero call sites, and three of them invite context-losing audit writes~~

- 原报级别：P3　位置：`server/internal/pkg/audit/audit.go:105`　票数：0/1

> **驳回理由**（high 置信度）：The reference counts are factually accurate — I confirmed all seven functions (the title says "six" but the evidence lists seven) have zero call sites in Go code including _test.go, and I checked the one aliased import (`auditpkg` at server/internal/modules/course/review/repository_operation_log.go:9) so no alias hid a caller. But an accurate "unreferenced" count is not the same as a defect, and the finding's severity narrative is affirmatively contradicted by project law.

1. THE CENTRAL CLAIM IS CONTRADICTED BY THE PROJECT'S OWN WRITTEN GUARDRAIL. All the claimed weight of this finding comes from the assertion that audit.Log/LogSuccess/LogFailure are "an active trap" a developer could autocomplete into. docs/design/iam-implementation-guardrails.md, section "审计写入上下文", settles this in the opposite direction:
  - line 66: "请求链路内的审计事件必须使用 `audit.LogContext(ctx, event)`、`audit.LogFromGin(c,


## Claude 主对话追加发现（保留原始记录）

> Codex 复核结论：X-1 的攻击链被 admin Entry 前置 capability 检查截断，判定不成立；
> X-2 仅部分成立且原计数、分类与 CI 方案不准确。最终处置见上方复核表。

### X-1. 评课审核范围 fail-open：无授权的 school_admin 可见全部学校数据

- **位置**：`server/internal/modules/course/review/admin_scope.go:136`
- **级别**：P1　**类别**：authorization

**证据**

```
admin_scope.go:136-159  schoolIDs()
    if s.superAdmin { return nil }              // super_admin -> nil
    ids := make([]int64, 0, len(seen))          // 无授权 admin -> []int64{} (空但非 nil)
    return ids

repository_admin.go:53        if len(schoolIDs) > 0 { qb.WriteString(" WHERE r.school_id = ANY($1)") }
repository_content_flag.go:87 if len(schoolIDs) > 0 { ... }

admin_cache_key.go:16-22      moderationScopeCachePart()
    if schoolIDs == nil { return "all" }        // 这里明确区分了 nil 与空
    if len(schoolIDs) == 0 { return "none" }
```

**失败场景**

用户的 Casdoor roles claim 含 `school_admin`，但 OpenFGA 上没有任何 `effective_admin` school tuple
（授权尚未配置，或 tuple 被清理）。链路：

1. `RoleScopeResolver.resolveSchoolAdminScopes` 拿到空 `schoolIDs`，不写入 `scopes`；
2. `ResolveRoleScopes` 因 `len(scopes)==0` 返回 `nil, nil`；
3. `withResolvedRoleScopes` 因 `len(scopes)==0` 原样返回，`orgScopedRoles` 保持 nil；
4. `resolveModerationScope` 得到 `superAdmin=false`、`schoolAdmins=nil`；
5. `schoolIDs()` 返回空切片（非 nil）；
6. repository 的 `len(schoolIDs) > 0` 为假，**不加 school 过滤**。

`requireModerationRole()` 只检查角色名，不检查作用域，因此该用户通过入口检查后可列出
**全部学校**的 review、report 和 flagged 内容。影响 3 个入口：`admin.go:34`、`admin.go:118`、
`handler_content_flag.go:19`。

同文件的 cache key 代码把 `nil` 映射为 `"all"`、空切片映射为 `"none"`，证明预期语义是
「空 = 无权限」，只是 repository 层未遵守。

**修复方案**

让「不限范围」与「无范围」在类型上不可混淆：`schoolIDs()` 返回 `([]int64, bool)`，或引入显式
unrestricted 标记。super_admin 为 unrestricted；非 super_admin 且集合为空时 repository 必须返回
空结果集而不是省略过滤条件。补一条集成测试，断言无授权 moderation 角色返回 0 条而非全量。

### X-2. 21 个后端读取的环境变量未出现在任何 env 模板

- **位置**：`.env.example` / `.env.prod.example`
- **级别**：P2　**类别**：configuration-surface

`server/internal/pkg/config` 共读取 187 个环境变量，其中 21 个在两个模板里都没有定义：

```
AWS_CA_BUNDLE                          CASDOOR_ADMIN_SCOPES
CASDOOR_APP_PROVISIONING_CERTIFICATE   CASDOOR_BOOTSTRAP_MODE
CASDOOR_INTROSPECTION_ENDPOINT         CASDOOR_ROLE_SYNC_CERTIFICATE
CASDOOR_UNIAPP_SCOPES                  CASDOOR_USER_LOOKUP_CERTIFICATE
CASDOOR_USER_PROFILE_CERTIFICATE       CASDOOR_WEB_SCOPES
DB_SSL_CERT                            DB_SSL_KEY
EMAIL_PROVIDER_POLICY                  FGA_SKIP_SCHOOL_TUPLES
GIN_MODE                               LOG_ENVIRONMENT
LOG_SERVICE_NAME                       LOG_SERVICE_VERSION
REDIS_TLS_CERT                         REDIS_TLS_KEY
STUHELPER_REDIS_INTEGRATION
```

复核结论：`EXTERNAL_POSTGRES_ALLOW_PLAINTEXT` 与 `FGA_SKIP_SCHOOL_TUPLES` **已有正确守卫**
（前者在 `validation.go:248` 于 production 环境被拒绝；后者仅作用于 `cmd/fga-setup` 引导工具，
不进入运行时授权路径）。因此这是配置面完备性问题，不是安全漏洞。

**修复方案**

补入模板并注明用途与默认值；增加 CI 检查，比对 `config` 包读取的变量集合与模板定义集合，
出现差集即失败。

## Claude 原始修复状态与 Codex 验证

| 编号 | 问题 | 状态 |
|------|------|------|
| P0-1 | Admin 落地页 404 循环 | Codex 已完成实现：按 capability-filtered route/menu 推导首页，并对 redirect 做可访问路由校验。2026-07-31 累计回归另发现并收口两个实现文件的 5 个 ESLint 门禁错误；首次全量浏览器回归发现旧 E2E 仍期待受限路径落到 404，快照确认真实行为是安全回退 `/analytics` 且越权 API 请求为 0，测试已同步并双视口 2/2 通过。Admin 全量 lint 0 warning/error、两段 TypeScript 检查、production build 和最终 214/214 浏览器场景通过；unit 最终为 32 files / 154 tests（含独立 N-2 回归）。功能、lint 与浏览器契约 follow-up 均独立入库，尚未发布 |
| P1-1 | 私聊空 guild 绕过策略并跨群读取复核队列 | Codex 已完成实现：缺 Session fail-closed；无 guild 仍校验 authority policy 且在数据访问前返回上下文提示；显式 guild 保持目标群授权与过滤。定向 14 tests、Koishi 全量 593 tests、build 和 UI contracts 通过。随独立修复提交入库，尚未发布 |
| P1-2 | 迁移指南要求修改已执行的初始基线 | Codex 已完成修复：权威来源改为完整有序 migration 集合，后续 schema 变更必须新增递增 `.up/.down`；Make、CI 与数据库文档已同步。文档/契约检查通过，隔离 PostgreSQL 18 按 CI 最小权限实际完成 19 版 `up → down 1 → up`，最终无 dirty 状态。随独立修复提交入库，尚未发布 |
| P1-3 | 物理备份命令无法生成、半成品可被同步且 evidence 不覆盖物理备份 | Codex 已完成实现：改为外部 staging 的 `plain + stream`、临时 replication slot、`pg_verifybackup`、SHA256 与 `.partial` 原子发布；同步排除临时工件，evidence 覆盖本地/取回的逻辑与物理备份及新鲜度。真实 PostgreSQL 18.4 备份在无网络隔离实例启动并读回探针，75 个 infra contracts 全部通过。随独立修复提交入库，尚未发布；生产对象存储/WAL PITR 仍须单独验收 |
| P1-4 | 部署包丢失 env 模板 | Codex 已完成实现：部署包只取干净 Git `HEAD`，打包后断言两个根 env 模板存在且无其他根 env。临时干净仓库实测忽略 secret/依赖不入包、未跟踪文件阻断，相关部署/CI 契约通过。随独立修复提交入库，尚未发布；未执行真实远端部署/回滚 |
| P1-5 | 日历过期的 image review 阻断生产发布和旧版本回滚 | Codex 已完成窄范围实现：普通生产部署仍按当天硬校验；只有同环境成功发布记录、完全相同 digest、当时有效 policy 和完整审计上下文才能复用历史窗口，并写 0600 JSONL。GitHub 用当前 workflow 控制器兼容旧 release，每日提前 3 天告警。ShellCheck、actionlint、文档卫生与 76 个 infra contracts 通过；尚未执行真实远端回滚 |
| P1-6 | 活跃 Admission SSE 阻塞优雅停机 | Codex 已完成实现：两个 SSE 复用 runtime shutdown context，发送 `end/shutdown` 后退出，周期分支工作前再次检查停机。真实 HTTP/PostgreSQL 测试证明 bot 流可让 `http.Server.Shutdown` 在 2 秒内成功，camera 流也主动结束；定向 race、admission 全包、vet 与 lint 通过。随独立修复提交入库，尚未发布；真实进程 SIGTERM 待演练 |
| P1-7 | 回复列表刷新后丢失当前用户 ownership | Codex 已完成实现：GET replies 契约与真实路由接入可选认证，声明并 fail-closed 503，bundle/Go/TS 生成物同步。真实 PostgreSQL route-level 测试覆盖 owner、其他登录用户、匿名和认证后端故障；定向 race、review 全包、vet、lint、spec/drift 与文档检查通过。随独立修复提交入库，尚未发布 |
| P1-8 | 举报处理可把作者已删除的评课改回隐藏态并再次发布 | Codex 已完成实现：举报入口复用统一状态机，作者删除态只结案 report、不改 review；缺失 review 映射 404。真实 PostgreSQL 覆盖 hide/delete、重复非法转换、计数、时间戳和后续 restore，评课包全量、定向 race、vet、全服务端 lint 与文档卫生检查通过。随独立修复提交入库，尚未发布 |
| P1-9 | `phone.read` 解密到的仍是掩码手机号 | Codex 已完成实现：Open Platform 仅按内部 user ID 请求权威手机号，app gateway 在允许的边界解析 Casdoor subject 并调用既有 client；本地投影不再参与明文恢复。真实 PostgreSQL/Redis 覆盖 phone API、identity-token、granted/denied 审计和 provider 故障；adapter、race、全包、vet、Casdoor 边界门禁与 lint 通过。随独立修复提交入库，尚未发布；真实 Casdoor 待验收 |
| P1-10 | Resource 逐行加载 tags/bindings 并在外层 cursor 打开时嵌套取连接 | Codex 已完成实现：先 drain/关闭主结果，再用两条批量 SQL 回填关联，查询数由 `1 + 2N` 固定为 3。真实 PostgreSQL `MaxConns=1` 覆盖 1/2 条数据的固定 query count、详情、排序、空数组和 6 并发；定向 race、资源全包、vet 与全服务端 lint 通过。随独立修复提交入库，尚未发布；生产规模影响仍待指标观测 |
| P2-1 | Guard/toolchain 文件变化不触发消费它们的 CI 门禁 | Codex 已完成实现：新增 guards 分类，并把 docs hygiene、Vue UI contract、Semgrep 与 Node pin 精确接到实际消费 job；两个 Node 版本文件必须同步。CI wiring contract、Bash 语法、actionlint 与文档卫生通过。随独立修复提交入库，真实 GitHub paths-filter 调度待验收 |
| P2-2 | Dockerfile/Koishi 输入可绕过只在 infra job 中运行的供应链合约 | Codex 已完成实现：Dockerfile/CI wiring 两个静态合约改为 always-on Required 依赖，Koishi package contract 接入已有 Koishi build/test job；没有扩大为所有机器人变更跑全套 infra。三个定向合约、actionlint 与全部 76 个 infra contracts 通过。随独立修复提交入库，真实 GitHub runner 待验收 |
| P2-3 | 每条消息读取并排序该群全部消息账本，且账本无 retention | Codex 已完成热路径最小修复：排序/limit 下推数据库并增加 `(guildId, createdAt)` 索引，保持原窗口语义；隔离 SQLite 10,000 行执行计划与全 Koishi 回归通过。开发库账本为 0 行，原 300k/OOM 只是推演。自动 retention 会影响迟到撤回取证，缺少权威期限和生产规模时不实施，降为 P3 决策 |
| P2-20 | 教师公开统计刷新受普通 5 秒查询预算截断 | Codex 已完成最小修复：刷新使用独立、可配置且继承 caller/shutdown cancel 的 60 秒预算，配置仅允许 5–90 秒以保持低于 2 分钟 outbox lease；其余 SQL 仍为普通预算。真实 PostgreSQL 覆盖显式 deadline/cancel 与 Repository 专用预算；开发库 6 次刷新仅 4.040–20.517 ms、outbox 为 completed/0 次失败，因此原“已经永久陈旧”未获证实。未新增重复指标或猜测 cadence SLO |
| P2-6 | Console listeners 不随插件作用域释放，停用后仍可能调用特权动作 | Codex 已完成实现：Group Guard 改为 required console 子作用域；Core 与 Group Guard listeners 均由 `ctx.effect` 管理，并以 registration identity 保护服务重载后的新注册，authority 固定为 4。build、contracts、595 unit、startup 与 package contract 通过；WebUI E2E 首轮无关用例 45/46，立即复跑 46/46。随独立修复提交入库；真实 Console 热重载/插件卸载待验收 |
| P2-7 | 单个不可转发的新生材料阻塞后续队列 | Codex 已完成 Koishi delivery 阶段修复：单项 delivery failure 记录后继续，健康项 ACK，批末保持失败信号；ACK failure 分相并 fail-fast。poison→healthy、ACK failure 及既有多群语义测试通过，完整 Koishi build/contracts/597 unit/startup/46 E2E 与 package contract 通过。随独立修复提交入库；功能默认关闭，真实启用验收及服务端 URL 批构建边界仍待处理 |
| P2-10 | 普通学籍 importer 破坏身份证 enc/hash 成对写入约束 | Codex 已完成最小修复：普通 TSV 明确拒绝两列并从全部写入阶段移除 hash，重导不触碰已有 pair，新行保持 `NULL/NULL`；真实 PostgreSQL 18、定向契约、ShellCheck、76 个 infra contracts 和文档卫生通过。随独立修复提交入库；仓库当前没有完整 pair 导入入口，本次没有过度扩建 PII API/CLI |
| P2-13/P2-14/P2-15 | claimed bot action 批次被单行错误丢弃，且逐行重复查询上下文 | Codex 已完成主因合并修复：上下文查询从 `2N` 固定为每批 2 条；补偿可写时，批量查询失败/caller cancel 用 detached context 归还未公开 lease；确定性 poison 只消耗本行 attempt，stale/failure/abandon 使用 attempt fence。真实 PostgreSQL 覆盖固定查询数、故障/取消 cleanup、旧 lease fence、poison 隔离及第 5 次单独 dead-letter；Admission 全包 `-race`、lint/build 和文档检查通过。补偿写同时失败仍可能保留 attempt，Admission dead-letter replay API 尚不存在，已明确保留为残余边界。P2-15 与 P2-14 重复，不单独计数或提交。随独立修复提交入库，尚未发布 |
| P2-21/P2-22 | caller cancellation 与单条坏数据错误污染 Oracle 共享 breaker | Codex 已完成合并修复：parent cancellation/deadline 和 invalid/ambiguous row 统一为 neutral 并释放 half-open probe；内部 query timeout、backend failure 和 bind identity mismatch 保持 failure。未采用会重置既有失败的 `RecordSuccess` 方案。固定 integrity metric 与现有 503 adapter 契约有专项回归，全服务端 race/lint/build/docs 通过；真实 Oracle/生产指标仍待发布验收 |
| P2-23 | 任一 outbox handler panic 永久杀死 polling worker | Codex 已完成最小修复：只在共享 outbox per-job 边界 recover，panic 带 stack 进入原 retry/dead-letter/metric 路径并继续健康 job。真实 PostgreSQL 18 验证 poison dead-letter 与 healthy completed；原文所列真实共享调用面是 5 个，不包括两个 Admission expiry 手写循环。未增加会形成 panic loop 的 runtime supervisor；生产告警/replay 与外部副作用幂等仍待验收 |
| P3-9 | Redis 版本读取故障被本地缓存成 `v0`，使失效数据可重新出现 | Codex 已完成最小修复：只有 `redis.Nil` 使用初始 `v0`，transport error/caller cancel 返回 unknown；空版本 key 的 get 为 miss、set 为 no-op，且故障结果不进入本地版本缓存。真实 miniredis 关闭/重启覆盖故障绕过与 `v7` 恢复；未引入只改善命中率的 detached loader |
| P2-16 | 可空课程元数据扫描到非空 Go 字段导致课程读取 500 | Codex 已确认 NULL 是合法未知值并完成 nullable contract：四条课程读取与收藏列表使用指针扫描，未分类 group 保留 null，学分排序 NULLS LAST；OpenAPI/Go/TS 与 Web/UniAppX 同步，隔离 PostgreSQL 五路径回归通过。原 `COALESCE(0/'')` 建议会伪造事实，未采用 |
| P3-7 | 敏感批量评课导出没有审计事件 | Codex 已完成最小修复：NDJSON/CSV 每次请求只写一条 `data.export`，记录 success/failure、已处理行数、规范化筛选和既有上限；真实 PostgreSQL 验证成功及请求取消失败后审计仍落库。CSV 不含 moderation reason，原风险描述已纠正 |
| P2-12 | Admission 学校邮箱两条路由可持续调用 Oracle 且无用户级预算 | Codex 已完成最小修复：academic-match/request-otp 在鉴权后共享每用户 5/min Redis key，第 6 次 429 且不进 gateway，用户间隔离，Redis outage 503；不查 Oracle 的 verify-otp 不纳入。OpenAPI 429/503 与生成契约同步 |
| X-1 | 无 scope 的 school_admin 全量可见 | Codex 已证伪；现有 capability 展开和 admin Entry 在 Handler 前返回 403，不按 P1 修复 |
| X-2 | env 模板差集 | Codex 判定部分成立；改为分类治理，不执行 21 项全量入模板/严格集合相等方案 |
| 其余条目 | — | 不再用“其余 41 项待修”概括；按 Codex 逐项表和四批实施顺序处置 |
