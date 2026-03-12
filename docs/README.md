# StuHelper 文档中心

文档按“先跑起来，再按任务查，再看原理”的顺序组织。当前可作为事实来源的内容，优先级从高到低如下：

1. `docs/reference/` 与 `server/api/openapi.yaml`
2. `docs/guides/` 与 `docs/modules/`
3. `docs/architecture/`
4. `docs/plans/`（历史方案，保留设计背景，不作为当前实现的唯一事实来源）

## 先看哪里

| 你要做什么                   | 去哪里看                                                                   |
| ---------------------------- | -------------------------------------------------------------------------- |
| 第一次把项目跑起来           | [tutorials/quick-start.md](tutorials/quick-start.md)                       |
| 在现有前端上继续开发         | [guides/frontend-development.md](guides/frontend-development.md)           |
| 在后端新增接口               | [guides/backend-quickstart.md](guides/backend-quickstart.md)               |
| 走 OpenAPI 3 Spec-First 流程 | [guides/openapi-development-guide.md](guides/openapi-development-guide.md) |
| 了解前端整体结构             | [architecture/frontend.md](architecture/frontend.md)                       |
| 查询 API、数据库、错误码     | [reference/](reference/)                                                   |
| 查看模块设计和状态           | [modules/](modules/)                                                       |

## 目录说明

| 类型                  | 目录         | 用途                              |
| --------------------- | ------------ | --------------------------------- |
| [教程](tutorials/)    | 新人顺序阅读 | 从零搭环境、第一次启动项目        |
| [指南](guides/)       | 已上手开发者 | 完成某个具体任务的操作手册        |
| [架构](architecture/) | 设计决策     | 解释为什么这样组织系统            |
| [参考](reference/)    | 事实与约束   | OpenAPI、数据库、错误码等权威信息 |
| [模块](modules/)      | 业务域文档   | 认证、评课、通知、审核、日志等    |
| [计划](plans/)        | 历史方案     | 记录某次重构或设计提案            |

`docs/tutorials` 和 `docs/guides` 不重复：

- `tutorials/` 解决“我第一次进入这个仓库，应该按什么顺序做”。
- `guides/` 解决“环境已经好了，现在我要完成某件事，应该查哪里”。

这套边界是合理的，但过去的问题是索引不够清楚、部分内容长期未同步。当前文档已经按这个边界重新整理。

## 模块状态

| 模块       | 状态                                     | 文档                                           |
| ---------- | ---------------------------------------- | ---------------------------------------------- |
| 身份认证   | 🟢 Casdoor SSO 已实现，LDAP 仍在原型阶段 | [modules/auth/](modules/auth/)                 |
| 评课社区   | 🟢 已实现                                | [modules/course/](modules/course/)             |
| 通知中心   | 🟢 站内通知已实现，更多渠道待扩展        | [modules/notification/](modules/notification/) |
| 举报与审核 | 🟢 评课域已实现                          | [modules/moderation/](modules/moderation/)     |
| 日志系统   | 🟢 已实现并在后台使用                    | [modules/logging/](modules/logging/)           |
| 工具箱     | 🔴 待开发                                | [modules/tools/](modules/tools/)               |
| 策略配置   | 🔴 待开发                                | [modules/policy/](modules/policy/)             |

## 开发规范

- 项目级规则：`.project_rule/project_rules.md`
- 历史归档：`.project_rule/archiving.md`
- API 契约权威来源：`server/api/openapi.yaml`
