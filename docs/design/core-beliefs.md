---
type: design
audience: all
status: current
authoritative-source: this file
last-verified: 2026-04-19
---

# 工程原则

## 1. 真实来源单一

每类事实有且只有一个权威来源：

| 事实 | 权威来源 | 不要从这里读 |
|------|----------|-------------|
| API 契约 | `server/api/openapi.yaml` | 代码注释、前端类型文件 |
| 数据库 schema | `server/migrations/` | 文档描述、历史快照文件 |
| 能力常量 | `server/internal/pkg/capability/` | 前端硬编码 |
| 运行时行为 | 源代码和测试 | 设计文档中的"应该" |

## 2. 渐进式披露

Agent 从 `AGENTS.md`（~120 行）开始，按需跳转深层文档。入口不堆细节。

## 3. 契约驱动

改接口：改 OpenAPI → `make generate` → 补充实现 → 执行校验。不反过来。

## 4. 分层不跨界

```
Handler    只管 HTTP
Service    只管业务
Repository 只管数据
```

不接受"临时"跨层。

## 5. 不可变优先

业务层数据流尽量创建新对象。数据库更新是存储层例外。

## 6. 小文件优于大文件

200–400 行为宜，800 行上限。模块变大时按子领域拆文件。

## 7. 机械化执行优于文档约定

能用 linter 检查的不写文档约定，能用 CI 拦截的不靠人工 review。

文档目录、frontmatter、链接和 retired 路径由 `make check-docs` 守卫。具体规则见 [documentation-governance.md](documentation-governance.md)。

## 8. 计划是版本化工件

- 活跃计划：`docs/internal/exec-plans/active/`（当前为空）
- 归档计划：`docs/internal/exec-plans/archived/`
- 执行计划索引：`docs/internal/exec-plans/README.md`

## 9. 快速合并 + 快速修正

PR 短小、变更可控、修正迅速。
