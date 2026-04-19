# 架构决策记录

`adr/` 用于沉淀已经采纳的单项架构决策。它与 `architecture/` 的区别是：

- `architecture/` 关注系统整体边界与主路线。
- `adr/` 关注某一个具体架构选择为什么这样定。

| ADR | 内容 | 状态 | 日期 |
|-----|------|------|------|
| [0001](0001-scrollreveal-over-gsap-scrolltrigger.md) | 选用自研 ScrollReveal，而不是 GSAP ScrollTrigger | 已采纳 | 2026-03-31 |
| [0002](0002-glass-card-css-utility-pattern.md) | 玻璃卡片效果采用全局 CSS utility，而不是组件包裹层 | 已采纳 | 2026-03-31 |
| [0003](0003-animation-composable-safety-pattern.md) | 动画 composable 统一使用三段式安全守卫模式 | 已采纳 | 2026-03-31 |
| [0004](0004-dark-mode-dual-selector-rule.md) | Scoped dark mode 使用双选择器规则 | 已采纳 | 2026-03-31 |
| [0005](0005-uniappx-shadow-file-cleanup.md) | 清理 UniApp X 的 `.js` 阴影文件并加 CI 守卫 | 已采纳 | 2026-04-16 |
