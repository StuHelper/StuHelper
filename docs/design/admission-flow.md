---
type: design
audience: product, backend-dev, frontend-dev, ops, qa, maintainers
status: current
authoritative-source: server/api/openapi.yaml + server/internal/modules/admission + clients/web/src/modules/admission + bots/koishi + infra/ops
last-verified: 2026-07-30
---

# 入群认证流程

## 1. 域划分

| 域 | 职责 |
|----|------|
| `sso.stuhelper.com` | 唯一公开登录认证入口和 OIDC issuer |
| `stuhelper.com` | 账号中心、学生认证、QQ 绑定、开放平台和业务 API |
| `join.stuhelper.com` | 入群认证业务闭环 |

群内公开链接只使用 `https://join.stuhelper.com/verify/<code>`。

Koishi / NapCat 是 QQ 执行器。后端 admission 表中的 `platform=qq` 表示被验证账号所属平台，与 Koishi runtime adapter 名称无关。

## 2. 主链路

```
QQ 入群事件
    ▼
Koishi 禁言新成员 + 创建 admission session + 群内发出 canonical join 链接
    ▼
用户打开 join 链接 → SSO 登录 → 回到 join
    ▼
link admission（消费 token，绑定当前已登录用户）
    ▼
老生：学号姓名校验 → 固定邮箱 OTP
新生：材料提交 → 人工审核
    ▼
后端产生 active student verification credential
    ▼
Koishi 轮询 pending action → 解除禁言 → 回写 release 结果
```

进入 linked 或 material 阶段后，join 页面的所有子流程（`/admission/me`、学校邮箱 OTP、学校 SSO、新生材料申请）必须继续携带当前 `admissionSessionID`。同一账号多群、多次重发或旧链接续办时，不能依赖「按 user_id 取最新 session」。后端在 `user_id + session_id` 精确校验后才写入。

## 3. 设计决策

| 议题 | 决策 | 原因 |
|------|------|------|
| 登录体系 | SSO-only：`sso.stuhelper.com` 是登录与 OIDC issuer，主站只做业务账号中心 | 避免双登录态；开放平台只接 StuHelper disclosure API，不暴露 Casdoor 用户表 |
| 业务流程域 | 入群流程只在 `join.stuhelper.com` 闭环 | 群内链接语义清晰，且不把业务步骤塞进 SSO 登录页 |
| token 消费 | token 只在已登录用户 link admission 时消费；消费后只能由同一已绑定用户继续当前 session | 防止链接转发，同时允许刷新、返回和继续未完成的认证 |
| token 错误分类 | `admission.token_consumed` 表示「已绑定账号，需续办或账号不匹配」，不是链接过期；只有 expired / not_found / session_not_found 进入失效处理 | 避免在可恢复的链接上显示「链接已失效」，也避免子流程把可排障的 consumed 错误吞成过期 |
| 子流程上下文 | 子流程必须携带 `admissionSessionID` | 多群、重发和刷新场景不能按 user_id 取最新 session |
| 学校识别 | 对外和前端以学校代码为主，`school_id` 外键只做内部实现 | 学校名称会变，代码更适合作为稳定 key |
| BUAA 老生认证 | 先校验学校代码、学号和姓名，再固定邮箱为 `学号@buaa.edu.cn` 发送 OTP | 避免别名邮箱绕过学籍身份校验；该逻辑是学校专属配置，不进通用流程 |
| 新生材料 | 桌面可直接请求摄像头，手机可扫码免登录上传；两端用 handoff 状态锁定继续端 | 兼顾摄像头质量与移动端拍照体验，避免重复提交 |
| handoff 实时性 | SSE 优先，短轮询仅作兼容 fallback | 减少延迟与无意义请求，同时保留代理异常时的可用性 |
| 邮件发送 | `EMAIL_DRIVER=multi`，腾讯云 SES 优先、Resend 兜底，共用同一 HTML / TXT 内容 | 保障学校邮箱 OTP 可用性；Resend 不使用腾讯云模板 ID，视觉保持一致 |
| Koishi 命令 | 生产关闭公开群管命令与消息风控监听，保留 admission 管理命令 | 不与既有群管命令冲突，管理员仍可重发 / 重生认证链接 |
| 生产部署 | 仓库、镜像、脚本、模板和 runbook 是事实来源 | 生产手工修改不构成最终状态；任何生产变更都要能从仓库复现 |

## 4. 放行约束

Koishi 只有在后端存在 **active student verification credential** 时才解除禁言。

以下状态都不构成放行依据：

- 仅完成 QQ 绑定；
- 仅创建新生申请但未提交材料；
- `verified` 但没有 active student verification credential —— 这是异常状态。

该约束防止「绑定了 QQ 但不是学生」的账号绕过准入。

## 5. 会话唯一性

`group_admission_sessions_active_qq_idx` 覆盖 `material_submitted`，保证同一 QQ / 群 / 平台下不存在并行的 in-progress session。Koishi 本地状态丢失或重复 create 时，已提交材料待审核的用户不会被创建出并行 session 导致 release / kick 动作互相干扰。

提醒时间的处理：link、材料提交、验证、取消以及重生取消旧会话时清空 `next_reminder_at`。只有管理员显式重发才把 linked / material session 放回提醒队列。

## 6. 验证分层

### 本地覆盖

- `make check-admission-mvp`：聚合后端 admission / auth / user 测试、Web admission / auth / user Vitest、Playwright、Web build、Koishi group-guard 测试与 build、infra contracts。
- `clients/web/tests/e2e/auth-callback-and-admission.spec.ts`：匿名 admission 登录跳转、登录回调免手动刷新、已消费 token 由原账号续办、QQ mismatch / expired 阻断、BUAA 学号邮箱 OTP、手机 handoff SSE、桌面与手机材料提交。
- `infra/ops/admission-prod-sim-e2e.sh`：在 production-parity 栈模拟完整链路，从 bot 创建 session、浏览器 SSO 登录、Join 绑定、邮箱 OTP，到 bot 轮询 release action 并回写 release event。
- `bots/koishi/plugins/stuhelper-group-guard/src/member-guard.test.ts`：admission reminder 去重、后台 pending action 执行、release / kick / blacklist 结果回写、后端不可用时本地兜底禁言。

### 生产 smoke

| 脚本 | 验证内容 |
|------|---------|
| `admission-public-smoke.sh` | join canonical 路由、错误域 404、metrics beacon、camera handoff SSE ingress |
| `public-sso-smoke.sh` | Casdoor discovery、JWKS、authorize 路由、账号密码与注册入口 |
| `admission-production-readiness.sh` | 学校、群 policy、bot service credential、学籍源可用性 |
| `koishi-admission-production-evidence.sh` | Koishi admission 插件配置、环境变量、bot API、近期 admission 日志 |
| `admission-mvp-production-evidence.sh` | 聚合公网 smoke、浏览器证据与 readiness |

生产 smoke 证明公共入口、SSO、DB readiness 和 Koishi 配置健康，不能替代真实 QQ 端到端验收。

### 上线门禁

真实 QQ 小号 E2E 的三段证据：

1. `join-created`：真实 QQ 入群事件创建 session，群内链接是 canonical join URL。
2. `flow-completed`：用户登录、绑定 QQ，并完成正式学生认证或新生材料流程。
3. `bot-released`：后端存在 active student verification credential，Koishi 执行解除禁言并回写 release 结果。

入口为 `make prod-admission-mvp-final-evidence`，Koishi 节点为 `make prod-admission-mvp-final-koishi-evidence`。

## 7. 学籍源

BUAA 学号姓名校验按 `schoolCode=4111010006` 路由到外部只读 Oracle 学籍源的 `USR_JWBIZ.T_XS_JBXX`，查询 `XH` / `XM`。未启用外部源时使用学校配置 `academic_db_table=academic.buaa_students` 作为本地 fallback。

生产 readiness 区分 `external_oracle` 与 `local_academic_table` 两种模式。学校对照表不作为白名单。

外部源的连接、权限与 TLS 约束见 [security-model.md](security-model.md)。

## 8. 边界

- QQ 昵称不是权威业务字段，只能作为非持久化 UI 辅助。
- Open Platform 按 scope 暴露 `verified_student`、学校代码、校区代码，业务数据不进 Casdoor token。
- 非 admission 的群管能力和其他机器人插件不属于本流程范围，除非影响入群认证事件处理。

## 9. 相关文档

- [Koishi 入群认证实现](koishi-admission-verification.md)
- [认证与会话](auth-and-session.md)
- [安全模型](security-model.md)
