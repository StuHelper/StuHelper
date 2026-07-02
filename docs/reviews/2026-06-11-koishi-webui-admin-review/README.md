# 2026-06-11 Koishi 插件 / WebUI / StuHelper Admin 全面审查

四个并行深度探索的产出，作为同日优化执行计划（`docs/internal/exec-plans/active/2026-06-11-koishi-admin-optimization.md`）的事实输入。

| 报告 | 范围 |
|------|------|
| [koishi-server-plugins.md](koishi-server-plugins.md) | `bots/koishi` 服务端：stuhelper-core/src、group-guard、binding、admin、packages/shared、moderation-core |
| [koishi-webui.md](koishi-webui.md) | `bots/koishi/plugins/stuhelper-core/client` 全部视图、shell、数据通道、e2e 覆盖 |
| [stuhelper-admin.md](stuhelper-admin.md) | `clients/admin/apps/web-ele` 全部页面、API 层、认证、权限 |
| [cross-boundary-and-findings.md](cross-boundary-and-findings.md) | 三条对接链路契约分析 + 从 2026-06-10 全库 review 提取的 52 条相关 findings |

行号引用以 2026-06-11 develop 分支（`5240e842`）为准。
