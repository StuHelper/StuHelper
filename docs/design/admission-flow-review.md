---
type: design
audience: product, backend-dev, frontend-dev, ops, qa, maintainers
status: current
authoritative-source: server/api/openapi.yaml + server/internal/modules/admission + clients/web/src/modules/admission + bots/koishi + infra/ops
last-verified: 2026-06-02
---

# Admission Flow Review And Decision Plan

## 结论

入群认证 MVP 的主线决策保持不变：

- `sso.stuhelper.com` 是唯一公开登录认证入口和 OIDC issuer。
- `stuhelper.com` 承载账号中心、学生认证、QQ 绑定、开放平台和业务 API。
- `join.stuhelper.com` 只承载入群认证业务闭环。
- `id.stuhelper.com` 是 disabled legacy host，公网入口应返回 404。
- 群内公开链接只使用 `https://join.stuhelper.com/verify/<token>?qq=<qq>`。
- Koishi / NapCat 是 QQ 执行器；后端 admission 表里的 `platform=qq` 表示被验证账号平台，不等同于 Koishi runtime adapter 名称。

当前代码已经具备“入群后禁言、创建认证链接、登录回到 join、绑定 QQ、完成学生认证或新生材料流程、Koishi 轮询并执行后续动作”的主体闭环。上线验收不能只看主站 smoke 或 token 页面加载，最终必须跑真实 QQ 小号 `bot-released` 证据：同一次 session 有 active student verification credential，Koishi 已解除禁言并把 release 结果回写后端。

## 已修复的问题

| 问题 | 影响 | 收敛方式 |
|------|------|----------|
| 登录回调后 SPA 不刷新 admission 状态 | 用户登录完成回到 join 后仍看到未登录态，手动刷新才进入认证 | 前端 admission 页面在登录回跳后主动刷新认证投影 |
| Koishi admission 首次扫描过早提醒 | 重启后可能快速重复发送认证提醒 | 延迟第一次 admission reminder scan，避免刚创建 session 立即重复提醒 |
| admission 默认优先展示新生材料 | 老生实际更常见，用户容易误走拍照材料流程 | 已登录且学校支持老生认证时默认进入老生认证，用户仍可手动切换新生路径 |
| pending 新生申请绑定旧 session | 用户重新生成链接后，旧 pending application 仍挂在上一条 session，导致当前入群流程不能正确完成 | 发现同用户同学校 pending application 时，重绑到当前 linked session |
| 并发创建新生申请导致唯一索引冲突 | 双击、刷新或网络重试可能让其中一次请求返回 409 / 500 | 捕获 pending 唯一索引冲突，重新读取 pending application 并复用或重绑 |
| SSO / admission smoke 对错误目标过宽 | 可能误把本机或非公网入口当成公网通过 | smoke 默认拒绝 local target，证据脚本写入更严格 |
| 邮件 smoke 没加载生产 env | 邮件 provider 检查可能与真实生产配置不一致 | 邮件 smoke 显式加载生产 env，证据不输出 secret |
| 管理后台缺少入群会话排障入口 | 运营无法按 QQ、群号或 bot selfID 定位“链接是否消费、是否认证、bot 是否报错” | 增加 admission session 列表、运行时过滤、状态/期限/bot 错误展示和当前链接复制 |
| Join 页面缺少进度和期限提示 | 用户不知道当前卡在绑定、学生认证、审核还是机器人解禁，也不清楚各阶段截止时间 | 增加稳定进度条，区分 link、submission、manual review deadline，并提示认证通过后才提前解除禁言 |
| 学生邮箱格式允许空 local-part | `@buaa.edu.cn` 这类异常地址可能绕过域名检查，后续 mask 时存在 panic 风险 | 在 `schoolauth` 增加统一邮箱规范化，admission 和主站学生认证都要求 local/domain 非空且无空白 |
| Join 失效/账号不匹配页只显示重生命令 | 用户不知道如何高效把恢复动作发给群管理员 | 在 Join 页增加“复制指令”，复制 `重新生成认证链接 <QQ号>` 并给出 toast 反馈 |
| 管理后台只能复制恢复命令 | 运营仍需回到 QQ 群执行命令，无法在后台直接排队重发、重生或取消会话 | 增加 `admission:session:manage` 能力和 Admin session resend/regenerate/cancel API；重发/重生通过 `next_reminder_at` 进入 Koishi pending action 队列，不暴露 bot token |
| linked/material 阶段可能继承入群初始提醒时间 | 如果扩展 pending reminder 查询但不清理旧 `next_reminder_at`，用户开始认证后仍可能收到重复提醒 | link、材料提交、验证、取消和重生取消旧会话时清空旧 reminder；只有管理员显式重发才把 linked/material session 放回提醒队列 |

以上修复均已按提交收敛到本地仓库，并通过对应测试验证；是否已进入生产必须以当次部署记录、镜像/源码 sha 和生产 evidence 为准。生产 smoke 只能证明公共入口、SSO、DB readiness 和 Koishi 配置健康，不能替代真实 QQ 端到端验收。

## 验收分层

### 已有本地覆盖

- `make check-admission-mvp`：聚合后端 admission/auth/user 测试、Web admission/auth/user Vitest、admission/auth/user Playwright、Web build、Koishi group-guard 测试与 build、infra contracts。
- `clients/web/tests/e2e/auth-callback-and-admission.spec.ts`：覆盖匿名 admission 登录/注册跳转、登录回调回到 verify 后无需手动刷新、已消费 token 由原账号续办、QQ mismatch/expired 阻断提交、BUAA 学号邮箱 OTP、手机 handoff SSE、桌面/手机材料提交。
- `infra/ops/admission-prod-sim-e2e.sh`：在本机 production-parity 栈模拟完整 admission MVP，从 bot 创建 session、浏览器 SSO 登录、Join 绑定、BUAA 学号邮箱 OTP，到 bot 轮询 release action 并回写 release event。
- `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.test.ts`：覆盖 admission reminder 去重、后台 pending action 执行、release/kick/blacklist 结果回写、后端不可用时本地兜底禁言。

### 必跑生产 smoke

- `admission-public-smoke.sh`：验证 join canonical 路由、错误域 404、metrics beacon、camera handoff SSE ingress。
- `public-sso-smoke.sh`：验证 Casdoor discovery、JWKS、authorize 路由和账号密码/注册入口。
- `admission-production-readiness.sh`：验证学校、群 policy、bot service credential。
- `koishi-admission-production-evidence.sh`：验证 Koishi admission 插件配置、环境变量、bot API 和最近 admission 日志。
- `admission-mvp-production-evidence.sh`：聚合公网 smoke、浏览器证据和 readiness。

### 最终上线门禁

最终上线必须额外完成真实 QQ 小号 E2E：

1. `join-created`：真实 QQ 入群事件创建 session，群内链接是 canonical join URL。
2. `flow-completed`：用户登录、绑定 QQ，并完成正式学生认证或新生材料流程。
3. `bot-released`：后端存在 active student verification credential，Koishi 执行解除禁言并回写 release 结果。

只完成 QQ 绑定不应解除禁言；没有 active student verification credential 时，Koishi 不应 release。这个约束用于防止“绑定了 QQ 但不是学生”的用户绕过准入。

## 决策矩阵

| 议题 | 决策 | 原因 |
|------|------|------|
| 登录体系 | 采用 SSO-only：`sso.stuhelper.com` 是登录和 OIDC issuer，StuHelper 主站只做业务账号中心 | 避免 `id` 与 `sso` 双登录态；后续开放平台只接 StuHelper disclosure API，不直接暴露 Casdoor 用户表 |
| 业务流程域 | 入群流程只在 `join.stuhelper.com` 闭环 | 群内链接语义清晰，且不把业务步骤塞进 SSO 登录页 |
| token 消费 | token 只在已登录用户 link admission 时消费；消费后只能由同一已绑定用户继续当前 session | 防止链接转发给别人，同时允许用户刷新、返回和继续未完成的认证 |
| 学校识别 | 对外和前端以学校代码为主，数据库外键 `school_id` 只做内部实现 | 学校名称会变，代码更适合作为开放平台和导入数据的稳定 key |
| BUAA 老生认证 | BUAA 先校验学校代码、学号和姓名，再固定邮箱为 `学号@buaa.edu.cn` 进行 OTP | 避免别名邮箱绕过学籍身份校验；该逻辑必须做成学校专属配置，不硬编码到通用流程 |
| 新生材料 | 允许桌面直接请求摄像头，也允许手机扫码免复杂登录上传；桌面和手机用 handoff 状态锁定继续端 | 兼顾电脑摄像头质量和移动端拍照体验，避免重复提交 |
| handoff 实时性 | SSE 优先，短轮询仅作为兼容 fallback | 减少延迟和无意义请求，同时保留老浏览器/代理异常时的可用性 |
| 邮件发送 | `EMAIL_DRIVER=multi`，腾讯云 SES 优先，Resend 兜底；两者复用同一 HTML/TXT 内容；管理后台系统配置已支持 provider 启用、优先级、权重和 `priority`/`weighted` 策略 | 保障学校邮箱 OTP 可用性，避免 provider 单点故障；Resend 不使用腾讯云模板 ID，但输出视觉保持一致 |
| Koishi 命令 | 生产关闭公开群管命令和消息风控监听，保留 admission 管理命令 | 新插件不抢旧“举报 / 骰子 / 抽禁言”等命令，但管理员仍可重发/重生认证链接 |
| 生产部署 | 本地仓库、镜像、脚本、模板和 runbook 是事实来源 | 禁止把生产手工修改当成最终状态；任何生产变更都要能从仓库复现 |

## 下一步优先级

### P0: 上线前必须闭环

- 完成真实 QQ 小号 `bot-released` E2E，并留存 main + Koishi final evidence。
- 使用 `make prod-admission-mvp-final-evidence` 和 Koishi 节点 `make prod-admission-mvp-final-koishi-evidence` 作为最终上线证据；普通 production smoke 只能证明环境健康，不能替代真实 QQ release。
- 复核真实 QQ final evidence 的三段证据：`join-created`、`flow-completed`、`bot-released`。其中 `flow-completed` 必须包含 QQ 绑定和有效学生认证凭据或材料审核状态；只绑定 QQ 不可放行。
- 后端发布时减少 Koishi 轮询瞬间 502：优先做滚动或短维护窗口；不能把重启期间的短暂失败误判为 token 或 credential 问题。

### P1: 管理和恢复能力

- 管理后台 admission session 搜索、当前链接复制、Koishi 重生命令复制、直接请求重发、重新生成和取消会话已完成；后续继续补查看 bot release 记录。
- 管理后台增加 school config 页面：学校代码、校区、可用认证方式、邮箱策略、学校专属字段校验、开通状态。
- BUAA 学号姓名校验已通过学校配置 `academic_db_table=academic.buaa_students` 接入可导入学籍表；上线前仍需导入并校验真实北航学籍数据，且只有 `4111010006` 先启用，不把学校对照表误当白名单。
- QQ 昵称不作为权威业务字段；如运行时临时展示，只能作为非持久化 UI 辅助。

### P2: 体验优化

- join 页面稳定 stepper 和阶段 deadline 已完成；后续继续按真实 E2E 观察文案是否需要收敛。
- 摄像头权限前置检测：说明浏览器/系统权限状态，并提供手机扫码接力入口。
- 手机接力状态做实时 UI：已扫码、拍照中、已上传、选择继续端、另一端锁定。
- 所有失败状态给出管理员可执行的恢复动作名称，例如“重发认证链接”或“重新生成认证链接”；Join 失效和账号不匹配页已经支持复制重生命令，后续可继续补“请求管理员处理”的标准文案。

### P3: 扩展能力

- Open Platform disclosure API 按 scope 暴露 `verified_student`、学校代码、校区代码等业务数据，不把详细业务数据塞进 Casdoor token。
- 管理后台已支持邮件 provider 的启用、优先级、权重和发送策略模式；后续补超时、熔断窗口和 smoke 收件箱。
- admission 风险策略增加设备/IP/频率维度，但不影响第一版“先入群禁言再认证”的主路径。
- 对新生材料审核增加水印、只读下载代理和短期签名 URL，再考虑开启原图转发到 QQ 管理群。

## 当前仍不能宣称完成的事项

- 缺一次真实 QQ 小号完成学生认证后的 `bot-released` final evidence。
- BUAA 专属学籍校验接口已接入 `academic.buaa_students`，但生产仍需导入并抽样校验真实北航学籍数据后才能宣称该校老生认证完整可用。
- 邮件 provider 的基础管理后台配置面已完成；超时、熔断窗口和 smoke 收件箱仍需后续补齐，当前仍主要通过 env 和 smoke 脚本验收真实凭据与模板状态。
- 后端 app 重建仍可能让 Koishi 在短窗口内看到 502；长期应做滚动发布或让 Koishi 对短暂 5xx 做更清晰的退避和观测。
- ChatLuna、非 admission 群管能力或其他机器人插件错误不进入 admission 上线范围，除非它们影响入群认证事件处理。
