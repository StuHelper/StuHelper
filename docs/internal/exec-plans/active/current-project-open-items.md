---
type: internal
audience: maintainers
status: current
authoritative-source: active goal + current repository state
last-verified: 2026-06-03
---

# 当前项目待办

本文件只记录当前仍然活跃的工作项。历史 identity 入口方案、历史公网 smoke 证据和已废弃迁移路径不再作为当前待办或验收来源。

## 当前架构事实

- `sso.stuhelper.com` 是唯一公开登录认证系统和 OIDC issuer。
- `stuhelper.com` 承载账号中心、学生认证、QQ 绑定、开放平台、授权应用、开发者应用和业务 API。
- `join.stuhelper.com` 只承载加群验证业务闭环，公开验证链接固定为 `https://join.stuhelper.com/verify/<token>?qq=<qq>`。
- 仓库、Nginx 模板、smoke、runbook 和文档不再保留独立 StuHelper identity host。
- 学校对外识别使用教育部 10 位学校代码；北京航空航天大学为 `4111010006`。

## 活跃任务

| 任务 | 当前状态 | 完成标准 |
|------|----------|----------|
| 清理独立 identity 入口 | 独立 identity issuer 代码、bootstrap、smoke、prod parity 和长期 404 契约已删除 | 全仓库不再包含独立 identity host 域名；不再有旧迁移指南、旧 ingress diagnostic、旧 disabled-host smoke 或长期 404 契约 |
| 移除五位学校 ID 业务事实 | admission bootstrap、school directory migration、baseline seed 和部分 runbook 已按 `school_code=4111010006` 收敛 | API、前端、运维脚本、测试 fixture 和 seed/migration 中不再把旧五位学校 ID 作为学校事实；学校选择和配置以 `schoolCode` 为主 |
| 北航老生学号姓名即时匹配 | Join 老生流程已接入学号 + 姓名匹配，匹配后固定邮箱为学号邮箱并放开发送验证码按钮 | 主站和 Join 的学生认证流程均复用同一后端规则；北航只允许学号邮箱；学籍源缺失或不匹配时前端明确禁用验证码动作 |
| 外部学籍数据源模块 | 已新增外部数据源模块并支持 Oracle 只读查询配置；admission readiness 可识别 `external_oracle` 与本地 fallback | 生产真实 Oracle 学籍源完成只读连通 smoke 和抽样校验；真实 secret 只在 secret backend 中配置 |
| 学校目录和白名单 | 管理后台学校配置页已显示学校代码、启用统计和启停开关 | 所有前端学校选择器和后端公开可选学校接口都只返回 enabled 学校；默认只启用北京航空航天大学 |
| Admission 真实 QQ E2E | 已能生成 canonical join 链接并进入认证流程 | 真实 QQ 小号完成 QQ 绑定和有效学生认证后，Koishi 自动解除禁言并回写 release evidence |
| 邮件 provider 平台化 | 腾讯云 SES 与 Resend smoke 已有基础脚本，邮件内容需要保持一致 | 后端邮件适配器支持 provider 优先级、权重、故障转移；管理后台可维护 provider 策略；真实发送不泄漏 secret |

## 验收原则

- 任何生产修复必须回写为本地代码、脚本、模板或文档。
- 不把生产手工修改、历史 evidence 或窄 smoke 当作完成证明。
- 搜索残留必须覆盖代码、配置、脚本、测试和文档。
- 涉及 OpenAPI 的接口变更必须先改 `server/api/openapi.yaml`，再生成 Go 和 TypeScript 类型。
- 完成前必须提供当前仓库状态、测试命令和结果。
