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
- 当前工作树在复核开始前已经有未提交修改。Codex 对 P0-1、P1-1、P1-2、P1-3、P1-4、
  P1-5、P1-6、P1-7、P1-8、P1-9、P1-10 与 P2-1 的修复已经按问题独立提交（提交见下方
  进度表），但均未合并或发布；原有其他工作树修改没有混入这些提交。
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
| X-2 | 部分确认，原计数错误 | P2 | 应改 | config 包运行时键为 181 个，主模板实际缺 17 个；另外 4 个分别属于 bootstrap/FGA 工具、`GIN_MODE` 和 Redis 集成测试，不能混入运行时模板。建立分类 allowlist，只要求 operator-facing 配置进入对应模板；识别并删除 `LOG_SERVICE_VERSION` 等死配置。要求所有 `getenv` 与两个模板严格集合相等会暴露危险开关并制造噪声，属于过度设计。 |

### P2 逐项复核

| 编号 | Codex 结论 | 调整级别 | 必要性 | 最小处置与过度设计判断 |
|------|------------|----------|--------|------------------------|
| P2-1 | 确认，已完成本地修复与静态契约验证 | P2 | 已修复，待 CI 验收 | 新增 `guards` 分类；文档、Vue UI、Semgrep 与 Node pin 分别触发实际消费它们的 job，Node 两个版本文件还必须一致。没有让所有脚本改动触发所有重型 job。原文“零 CI”已纠正为“零个相关门禁”，secret scan 原本仍会运行。 |
| P2-2 | 确认 | P2 | 必须 | supply-chain 契约检查 Dockerfile/Koishi 输入，但 infra filter 漏掉它们。便宜 shell contract 常跑，Koishi 特有契约放入 Koishi job；把所有机器人改动都触发整套 infra/E2E 过重。 |
| P2-3 | 确认机制，影响未量化 | 条件性 P2，否则 P3 | 先测并做低风险修复 | 将 repeat 检测的排序/limit 下推 DB，增加 `(guildId, createdAt)` 索引和基础 retention；记录表规模与 p95。立即建设 retention WebUI/完整配置面属于过度设计。 |
| P2-4 | 部分确认；原报告引用了一条 dead 路径 | P2 → P3 | 先测 | 当前 dashboard 仍有 recent events 与 active guards 的全扫，应下推过滤/limit 后测延迟。原文所称 admission console 全量扫描不成立，五步聚合与全面 API 重写过重。 |
| P2-5 | 确认 | P2 → P3 | 应改 | 修复空格分隔 ID 的 destructuring，并同步真实命令名和回归测试。低频、低成本 correctness，不需要新 parser 框架。 |
| P2-6 | 确认 | P2 | 必须 | 用 required console injection 处理服务重载，同时使用 identity-checked disposer 清掉 plugin unload 后仍可调用的 authority-4 listener。只做 injection 不能消除陈旧特权回调；无需全局 monkey patch。 |
| P2-7 | 确认 | P2 | 必须 | 每个 material-forward item 独立捕获失败、继续后项，并记录失败指标/日志；测 poison + healthy。立即加 server attempts/dead-letter/schema 不是解除队首阻塞的必要条件。 |
| P2-8 | 确认 | P2 → P4 | 可选 | 中英文增加 `school_email_otp`、`school_sso` 标签并可选保留 enum 原值 fallback。只是后台 i18n，不应占用 P2 修复预算。 |
| P2-9 | 强静态证据确认；运行验证待补 | P2 | 必须 | 用 `{{ playbook_dir }}` 构造 command/copy 绝对源路径，使用 `argv` 并加窄路径契约/语法检查。当前环境没有 `ansible-playbook`，因此仍需 CI/真实 controller 验证。通用扫描所有 Ansible command/shell 的检测器过重。 |
| P2-10 | 确认 | P2 | 必须 | 普通 importer 不应接受单独 hash；从普通 upsert 移除该字段并保留既有 enc/hash，若确需导入身份证则另做接受完整加密 pair 的受控入口。不能伪造空 `enc` 绕过约束。 |
| P2-11 | 确认潜在契约缺陷 | P2 → P3 | 应改 | 统一为 `explode: true`，Handler 用 `QueryArray` 并兼容旧逗号格式，重新生成；修正或删除未使用的 Web grouped adapter。OpenAPI 与 Handler 当前都按逗号语义，真正不兼容的是默认客户端，原文“三端各不兼容”不准确。 |
| P2-12 | 确认 | P2 | 必须 | 对会调用外部 Oracle 的 academic-match/request-otp 增加鉴权后的 Redis per-user 共享预算；测 429、用户隔离与 Redis 策略。现有全局/IP limiter 不等于完全无限流，但不足以保护昂贵 fan-out。 |
| P2-13 | 确认 | P2 | 必须 | claim 后批量加载上下文；批量查询失败时按 attempt fence 安全释放 lease，单行 stale/映射错误不能中断其余项，并用独立有界 cleanup context。原文只做 per-row continue 仍处理不了批量查询失败。 |
| P2-14 | 确认，属于 P2-13 的性能放大器 | 单独看 P3 | 合并修复 | claim 后一次批量加载 policy/failure contexts，并测 query count。默认批是 50，额外约 100 次查询，不是原文按 server 上限 200 推算的 400 次。 |
| P2-15 | 事实重复 | 与 P2-14 重复 | 不单独立项 | 与 P2-14 是同两个逐行 context query，不是第二个根因；从唯一问题计数和独立修复计划中删除。 |
| P2-16 | 确认 | P2 | 决策后必须 | 先盘点 NULL 数据和产品语义：非法则回填后 migration 加 NOT NULL；合法则 Go/OpenAPI 改 nullable 并生成。默认 `COALESCE` 为 0/空串会伪造数据，不能采用。 |
| P2-17 | 确认两个配置当前未生效，但行为可能是刻意安全收紧 | P2 → P3/P4 | 决策 | 优先决定是否废弃 content preview knobs，并把 title knob 说明改成锁定首行 teaser；若恢复，只对已认证非 full tier 接线并保持 guest 收紧。直接“恢复配置生效”可能削弱访问控制。 |
| P2-18 | 确认 | P2 | 必须 | 当前单 app 部署只需本地 `Filter.Invalidate()`：mutation 成功后标记过期，使下次检查 reload；测 create/update/delete 和 reload 失败。当前就引入 Redis version/pubsub 是过度设计，等真实多副本需求再做。 |
| P2-19 | 确认 | P2 → P3 | 应改 | 先为语义唯一的 dangerous/too-short/rating-required/invalid-rating 映射专用 code；共享的 content-too-long 需先拆 sentinel。不要把所有共享错误一刀切成 review code。 |
| P2-20 | 确认结构风险，生产是否超过 5 秒未验证 | 条件性 P2 | 先测后改 | 给 materialized-view refresh 独立可配置 timeout，继承 parent cancel；增加 duration、projection age、retry 指标。不要抬高全局 DB timeout，也无需先重写所有 shutdown/retry。 |
| P2-21 | 确认 | P2 | 必须 | 对外部源结果做分类：caller cancellation 为 neutral，内部 source timeout 仍为 failure；half-open 取消必须释放 in-flight probe。只跳过 `RecordFailure` 会让 half-open 永久卡住。 |
| P2-22 | 核心确认；503 是现行文档契约 | P2 | 必须 | transport 成功但 bad row 不应污染共享 breaker；保留 typed integrity error 和当前 adapter 的 503 映射，增加 integrity metric。与 P2-21 使用同一 outcome classifier。 |
| P2-23 | 确认 | P2 | 必须 | 在同一 worker goroutine 内把单 job panic 转成带 stack 的普通失败，复用现有 retry/dead-letter/指标；测试 poison 后下一 job 继续。无限 root supervisor 会形成 panic loop且绕开死信，属于过度设计。 |

P2 唯一根因应按以下修复簇合并，避免重复设计：

1. P2-1 + P2-2：CI 路径与静态契约触发。
2. P2-13 + P2-14 + P2-15：claim 后批量上下文、逐项隔离和 lease 安全释放。
3. P2-21 + P2-22：Oracle outcome 分类与 circuit-breaker neutral 语义。

### P3 逐项复核

| 编号 | Codex 结论 | 调整级别 | 必要性 | 最小处置与过度设计判断 |
|------|------------|----------|--------|------------------------|
| P3-1 | 确认 | P3 | 可选 | `AdminContentLayout` 增加可选 description/subtitle 渲染，或删掉无效调用；逐页判断重复文案。rich slot 不是必要修复。 |
| P3-2 | 确认 | P3 | 应改 | 修正 attrs/default min-width 合并顺序或统一 prop，并加 caller 提供 `min-width` 的测试。 |
| P3-3 | 确认 | P3 | 应改 | 已审核行隐藏/禁用通过与驳回按钮，点击前再检查 stale state；不需要全局改 409 语义。 |
| P3-4 | 部分确认 | P3 | 可选 | `member-blacklist-unification.md` 仍需把已落地内容改成当前时态；IAM 旧文在当前脏工作树已 rename/rewrite 为 `iam-architecture.md`，不要重复修改或覆盖现有改动。 |
| P3-5 | 部分确认 | P3 | 可选 | 只修 `frontend-development.md` 的缺失/不存在目录，以及 `backend-development.md` 过窄的 RBAC 入口。带明确日期的 GitHub 迁移快照和当前仍准确的 automation 叙事不应批量重写。 |
| P3-6 | 确认 | P3 | 应改 | 以 OpenAPI 为真源修 school SSO login/callback 和 mobile handoff 的 security/401/403，generate 后加路由契约测试；不要机械枚举未实现状态。 |
| P3-7 | 确认 | P3 → P2 | 必须 | 每次敏感批量导出写一个 success/failure 审计事件及 row count，让 stream 返回 `(count, error)`。CSV 不含 moderation_reason，风险描述应保持准确。 |
| P3-8 | 确认 | P3 | 应改 | 三个 Handler 改为真正的 204 No Content，保持 OpenAPI 真源；历史提交也表明 204 是有意契约，不应反向把 spec 改成 200。 |
| P3-9 | 确认 | P3 → P2 | 必须 | Redis `Nil` 才映射 v0；transport unavailable 必须 bypass cache，不能本地 memoize。unknown key 的 get 为 miss、set 为 no-op并补故障测试；200ms detached loader/兼容 wrapper 可后续再做。 |

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
| R-8 | Redis outage 被误报 cooldown | **确认，应移回待办（P2/P3）** | 两处代码先判断空 result，导致 transport error 永远落成 cooldown/429，后续错误分支不可达。先判 err 并映射现有 unavailable/503，补 transport error 测试。 |
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
4. P2-10 身份证 enc/hash 导入约束。
5. P3-7 敏感导出审计。
6. P1-9 `phone.read` 已按既有安全模型完成实时 Casdoor 读取与 fail-closed 修复；
   仍需带真实 Casdoor 凭据执行一次发布环境验收。

#### 第二批：发布、运行前进性与核心契约

1. P0-1、P1-4 与 P1-5 已完成修复、验证和独立提交；P1-5 仍需在受保护
   GitHub environment 与真实目标机执行一次带审计记录的回滚演练。
2. P1-2 migration 指南与 P1-6 SSE shutdown 已完成修复、真实回归验证和独立提交。
3. P2-1 CI guard 路径分类已完成修复、契约验证和独立提交；继续 P2-2 always-on
   静态供应链合约与 Koishi package contract，随后处理 P2-6 privileged listener 生命周期。
4. P2-7 转发 poison、P2-13/14/15 claimed batch、P2-23 outbox panic。
5. P2-21/P2-22 breaker 分类，R-8 Redis 错误分类，P3-9 cache version unavailable。
6. P2-9 Ansible 路径、P2-16 NULL 语义决策、P2-18 filter invalidation。

#### 第三批：可测量的性能与一致性

1. P2-3 ledger query/retention、P2-4 dashboard 查询。
2. P1-10 resource N+1 已完成固定三查询修复、单连接池并发验证和独立提交；继续
   P2-20 materialized-view refresh timeout。
3. R-11 outbox lock contention：只有指标或压测达到门槛后再改事件键/合并策略。
4. R-16 academics bulk import：真实 connector 上线前处理，不提前建设。

#### 第四批：低风险 UX、文档和清理

P1-7 已完成修复、真实路由/数据库回归和独立提交。继续 P2-5、P2-8、P2-11、P2-19、
P3-1 至 P3-6、P3-8、R-5 至 R-7、R-14、R-15、R-17，以及 X-2 的配置分类治理。

### 本轮执行的验证

- Admin：P0-1 的 capability-filtered 首页、可访问 redirect 和 6 类危险/无效 redirect
  定向 Vitest 10/10 通过；Admin 全量 Vitest 31 files / 153 tests 通过；`vue-tsc --noEmit
  --skipLibCheck`、Node 侧 `tsc`、`oxfmt --check`、`oxlint` 和 production build 均通过。
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
- image policy：2026-07-30 通过，2026-08-06 因 `review_by=2026-08-05` 失败，确认日历门禁。
- 授权：capability/RBAC/review 定向 Go 测试通过，确认 X-1 在 Handler 前 fail-closed。
- P2：3 个 infra/import contract 通过；Koishi 定向 29 tests 通过；outbox、externaldata、
  review、admission 定向 Go 测试通过。绿灯只证明现有 happy-path，不覆盖本轮新增的 poison、
  cancellation、panic-continuation、真实 DB pair 与大表场景。
- P3/驳回项：Admin 相关 Vitest 9/9、Koishi reminder 7/7、externaldata、OTP、cache、
  review projection 定向 Go 测试通过；仍缺 Redis transport error、cache v0 故障、
  min-width override、reviewed-row action 和 lock-wait 测试。
- P2-11 对 `openapi-fetch` 的实测确认数组被序列化为重复 `courseIDs` 参数。
- P2-9 因当前环境未安装 `ansible-playbook`，只有静态确认，运行验收仍是 pending。

### Codex 修复进度

| 编号 | 状态 | 实现与验证 | 独立提交 |
|------|------|------------|----------|
| P0-1 | 已修复，未发布 | 按已过滤路由推导首页；redirect 必须命中可访问路由，否则回退。定向 Vitest 10/10、Admin 全量 153 tests、两段 TypeScript 检查、格式、静态检查和 production build 通过 | `fix(admin): route users to an accessible home` |
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

### 明确不建议实施的“修复”

- 不为 X-1 做紧急授权模型/Repository 类型大改；攻击链在 admin Entry 前已被截断。
- 不要求所有 `getenv` 必须同时出现在两个 env 模板；先按运行时、工具、标准变量、测试变量分类。
- 不把 image review freshness 在所有生产部署和回滚中降成 report-only。
- 不为 P1-6 顺带强制 SSE 10 分钟断线。
- 不用 Repository 静默 no-op 替代 P1-8 的 Service 状态机。
- 不让 P1-9 的 Open Platform 业务模块持有 Casdoor subject，不落库/缓存完整手机号，也不把
  authoritative provider 空值降级成成功响应。
- 不为 P1-10 引入 ORM、DataLoader 或通用关联加载框架；两个本地批量查询已经消除该根因。
- 不用 `COALESCE('', 0)` 隐藏 P2-16 的 NULL 数据语义。
- 不在当前单副本部署为 P2-18 先建 Redis pub/sub/version 系统。
- 不为 P2-23 增加无限自动 supervisor；panic 必须进入现有 retry/dead-letter。
- 不在无真实 connector、无规模证据时为 P2-3/P2-4/R-16 建完整管理 UI、聚合平台或 bulk framework。
- 不把已明确有文档和测试的 R-12 产品行为改成失败。

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

3) Add config-driven retention. Add `ledgerRetentionDays` to GroupGuardModerationSettings in bots/koishi/packages/shared/src/guard/behavior-settings.ts (parse via the existing positiveIntegerOrDefault path at :217, add it to GROUP_GUARD_MODERATION_S

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
| P0-1 | Admin 落地页 404 循环 | Codex 已完成实现：按 capability-filtered route/menu 推导首页，并对 redirect 做可访问路由校验；定向测试 10/10、Admin 全量 153 tests、类型、格式、静态检查和 production build 通过。随独立修复提交入库，尚未发布 |
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
| X-1 | 无 scope 的 school_admin 全量可见 | Codex 已证伪；现有 capability 展开和 admin Entry 在 Handler 前返回 403，不按 P1 修复 |
| X-2 | env 模板差集 | Codex 判定部分成立；改为分类治理，不执行 21 项全量入模板/严格集合相等方案 |
| 其余条目 | — | 不再用“其余 41 项待修”概括；按 Codex 逐项表和四批实施顺序处置 |
