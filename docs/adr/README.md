---
type: reference
audience: all
status: current
authoritative-source: this file
last-verified: 2026-04-25
---

# 架构决策记录

`adr/` 用于沉淀已经采纳、代价高、难回退的单项架构决策。

- 系统整体边界、主路线和长期设计解释，请看 [`docs/design/`](../design/)。
- `adr/` 只回答“为什么这里做了这个关键选择”，不承担全局架构说明，也不充当变更日志。

| ADR | 内容 | 状态 | 日期 |
|-----|------|------|------|
| [0001](0001-scrollreveal-over-gsap-scrolltrigger.md) | 选用自研 ScrollReveal，而不是 GSAP ScrollTrigger | 已采纳 | 2026-03-31 |
| [0002](0002-glass-card-css-utility-pattern.md) | 玻璃卡片效果采用全局 CSS utility，而不是组件包裹层 | 已采纳 | 2026-03-31 |
| [0003](0003-animation-composable-safety-pattern.md) | 动画 composable 统一使用三段式安全守卫模式 | 已采纳 | 2026-03-31 |
| [0004](0004-dark-mode-dual-selector-rule.md) | Scoped dark mode 使用双选择器规则 | 已采纳 | 2026-03-31 |
| [0005](0005-uniappx-shadow-file-cleanup.md) | 清理 UniApp X 的 `.js` 阴影文件并加 CI 守卫 | 已采纳 | 2026-04-16 |
| [0006](0006-koishi-core-ui-as-single-webui-entry.md) | 保留 stuhelper-core 作为唯一 Koishi WebUI 入口 | 已采纳 | 2026-04-25 |
| [0007](0007-casdoor-as-sole-identity-provider.md) | Casdoor 作为唯一身份提供方，不采用 Zitadel / Keycloak | 已采纳 | 2026-05-01 |
| [0008](0008-postgresql-authorization-control-plane.md) | PostgreSQL 授权控制面与 OpenFGA 运行时判定面 | 已采纳 | 2026-07-31 |
