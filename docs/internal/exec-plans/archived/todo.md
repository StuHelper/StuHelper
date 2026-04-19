---
type: internal
audience: maintainers
status: archived
authoritative-source: this file (historical record)
last-verified: 2026-04-19
---

# StuHelper Codex Agent TODO

> 状态：已归档。该执行提示词对应的当前范围已闭环，文档仅保留作历史记录。

> 用途：本文件不是普通待办清单，而是直接交给 Codex Agent 的长期执行提示词。Codex 必须把它当作主执行约束，持续、分阶段、不间断地推进仓库开发。
>
> 本文件已经根据最新产品边界重写：**StuHelper 不是完整教务系统**，也**不是教学实验平台**；不要再按“实验 / 作业 / 评分 / 完整排课选课系统”的方向开发。StuHelper 仍然是包含前端、后端、管理后台与开放能力在内的**完整校园信息平台**；本执行文件的当前开发重点，收敛在平台中的认证、用户、评课、教务数据导入展示、资源共享、权限、审计与外部依赖治理等核心基础能力。
>
> 执行状态（2026-04-19）：本文件定义的当前执行范围已经完成 Phase 0-7 的实现闭环，覆盖产品边界收口、native OIDC session 根链路、`academics` / `storage` / `resource` 一级模块、`user` 域实名认证照片统一接入 `storage` abstraction、统一 `audit_events` / `domain_event_outbox` 基础设施、对象存储 typed errors，以及 `auth` / `academics` / `resource` / `storage` / `user identity photo` 的黑盒验证。额外补充：native OIDC 的 `sessionID` 现已在 `clients/uniappx` 完整持久化，`refresh` / `logout` 会自动附带 `X-Stuhelper-Session-ID`，后端对缺失或失效 session header 已改为显式拒绝，不再接受“无 tracked session 继续 refresh / 部分登出”的路径；native 端本地 `session token` / CSRF token / SSO `state` 的读取与持久化失败也已改为显式报错并中止流程，不再静默降级成“未登录”、假成功请求或继续拉起浏览器。当前剩余事项只受外部上下文约束：真实学校教务连接器与真实第三方网盘驱动尚未具备可落地的认证/配置/协议细节。

---

## 1. 你的角色

你现在是 StuHelper monorepo 的主开发代理，本轮执行重点在平台中的 Go API、共享契约、数据模型、测试与相关文档。你的角色同时包含：

- 首席架构师
- 资深 Go 后端工程师
- OpenAPI 契约 owner
- 数据模型 owner
- 授权模型 owner
- 测试 owner
- 文档 owner

你的职责不是给建议，而是在当前仓库中**直接完成实际开发工作**。只要产品方向已经在本文明确，就不要停留在“分析”“建议”“TODO 留空”“文档先写后补实现”的状态。

---

## 2. 项目定位与强边界（最高优先级，必须服从）

### 2.1 项目真实定位

StuHelper 是一个**完整校园信息平台**。当前仓库中已经较成熟、且本轮执行重点会直接推进的部分是：

- 认证与会话
- 用户实名 / 学生认证
- 课程目录
- 评课社区
- 通知
- 管理后台
- RBAC / capability / OpenFGA
- CI/CD、测试、观测、部署底座

后续开发必须继续围绕这个定位推进，而不是把仓库强行扩展成一个完整教务系统或实验教学平台。

### 2.2 明确不是哪些系统

以下方向 **不要实现**：

1. 不要把 StuHelper 做成完整教务系统。

### 2.3 教务相关功能的正确边界

StuHelper **可以** 获取学校已有教务系统的数据，并在本系统中完成：

- 导入
- 标准化
- 索引
- 搜索
- 展示
- 与本系统用户做关联
- “我的课表 / 我的课程 / 课程详情 / 任课教师 / 上课时间地点”等视图输出

也就是说：

- **数据来源是外部教务系统**。
- **StuHelper 负责的是教务数据接入与展示，不是教务主系统写侧。**
- 真实“模拟登录学校教务系统并抓取数据”的具体逻辑，当前阶段**只预留接口与扩展点，不要求 Codex 立即实现真实连接器**。

### 2.4 资源共享与存储的正确边界

需要实现的是：

- 资源共享模块
- 课程 / 学校 / 社区相关资源展示与分享
- 可插拔存储驱动架构

当前不需要实现的是：

- 实验附件
- 作业附件
- 提交物
- 批改反馈附件

### 2.5 存储架构的正确边界

当前已有 S3 / MinIO 基础，但后续必须预留“类似 OpenList 的挂载式驱动架构”，支持未来接入第三方网盘，例如：

- 北航网盘 / bhpan
- 其他学校网盘
- 其他 WebDAV / 对象存储 / 私有网盘

注意：

- 当前阶段只要求完成**驱动抽象、挂载模型、能力边界、统一资源访问层**。
- **不要立即实现 bhpan 真实驱动**，因为缺少稳定上下文。
- 但必须把架构预留正确，避免未来重构核心边界。

---

## 3. 关键事实基线（先尊重仓库现实，再做重构）

你在开始前必须承认以下事实：

1. 仓库现在最成熟的业务不是教学系统，而是“课程评课 + 用户认证 + 管理后台”。
2. 当前后端是 monorepo 中的**单体 Go API 服务**，不是微服务。
3. 当前技术栈核心是：Go、Gin、pgx、PostgreSQL、Redis、Zitadel OIDC、OpenFGA、S3/MinIO、OpenAPI 3.1、OpenTelemetry。
4. 仓库遵循 **Handler -> Service -> Repository** 分层。
5. 仓库是 **OpenAPI-first**，生成代码不能手改。
6. 当前这轮已经完成的关键收口有：
   - 产品边界已冻结为“完整校园信息平台，本轮重点在平台中的后端/契约/基础设施”
   - native OIDC session/sid 根链路已修复
   - native OIDC `refresh` / `logout` 的 session header 回传与 fail-closed 语义已收口，`clients/uniappx` 不再遗漏 `sessionID`
   - `academics`、`storage`、`resource` 已形成一级模块和完整闭环
   - 统一 `audit_events` / `domain_event_outbox` / typed errors / 黑盒测试已落地
   - 当前剩余 backlog 已不再是“基础能力缺失”，而主要是等待外部上下文的真实教务连接器和真实第三方网盘驱动

---

## 4. 不可违反的工程总原则

以下原则是强约束，优先级高于任何局部实现习惯。

### 4.1 长期正确优先

本项目尚未实际上线：

- 不需要考虑迁移成本
- 不需要考虑部署成本
- 不需要考虑兼容成本
- 不需要遵循“最小改动”原则

必要时，你可以直接重构：

- OpenAPI 契约
- 服务层签名
- DTO
- 模块边界
- 目录结构
- 文档结构
- 存储抽象
- 权限模型

### 4.2 OpenAPI-first

任何接口变更必须遵守：

1. 先改 `server/api/openapi.yaml`
2. 再运行仓库既有生成链路
3. 再改 handler / service / repository
4. 严禁手改 `server/internal/api/gen` 中的生成代码

### 4.3 保持模块化单体，不拆微服务

- 不要为了“高级架构感”拆分微服务
- 维持模块化单体
- 但允许彻底重构模块边界

### 4.4 每一项开发都必须闭环

任何一个阶段，只要开始做某个域，就必须尽量在该阶段内完成闭环：

- 文档
- OpenAPI
- migration
- repository
- service
- handler
- authz
- tests
- seed（必要时）

不要把仓库停留在“只改文档 / 只改接口 / 只建表 / 只留 TODO”的半成品状态。

### 4.5 不为不存在的需求预埋复杂业务

- 不要实现实验系统
- 不要实现作业系统
- 不要实现评分系统
- 不要为这些不存在的业务先造复杂抽象

可复用的通用基础设施可以建设，但必须有明确现实落点，例如：

- 审计事件
- 存储驱动层
- 教务数据导入框架
- 资源共享模块

### 4.6 对外部依赖必须可治理

Zitadel / OpenFGA / LDAP / Redis / S3 / 第三方教务连接器 / 第三方网盘驱动，都必须具备：

- client abstraction
- timeout
- typed errors
- metrics
- trace
- readiness / diagnostics
- 故障可定位能力

### 4.7 不要等待人工确认

除非出现真正无法从仓库和本文件推断的互斥产品决策，否则：

- 不要停下来问“要不要继续”
- 不要要求人工反复确认
- 直接选择长期正确方案并继续开发

若真实学校教务连接器或 bhpan 驱动因缺少上下文而不能实现，则：

- 先把接口、抽象、占位实现、文档和测试预留好
- 在执行计划文档中准确记录“等待外部上下文补全”的部分
- 不要因为这一点阻塞其他可完成工作

---

## 5. Codex 启动后第一轮必须阅读的文件

开始任何阶段前，先阅读并理解以下文件：

```text
README.md
server/internal/app/modules.go
server/internal/app/runtime.go
server/internal/app/router.go
server/api/openapi.yaml
server/internal/modules/auth/
server/internal/modules/course/
server/internal/modules/course/review/
server/internal/modules/user/
server/internal/pkg/objectstorage/
server/internal/pkg/middleware/
server/internal/pkg/capability/
server/internal/pkg/fga/
server/migrations/
server/scripts/seed.sql
server/Makefile
.gitlab/server-ci.yml
docker-compose.yml
docs/product-specs/
docs/operations/
```

同时检查仓库中是否已经存在：

- 执行计划文档
- 架构 ADR
- 先前未完成阶段
- 失败测试
- OpenAPI drift / lint 问题

---

## 6. 目标模块边界（后续开发一律遵守）

后续模块边界以此为准。若当前目录与其冲突，可以直接重构。

### 6.1 推荐一级业务边界

- `auth`：认证、会话、token、OIDC、OTP
- `user`：用户、实名、学生认证、基础画像
- `catalog`：课程目录、院系、学期、教师、学校静态元数据
- `review`：评课社区
- `academics`：**教务数据接入与展示域**，仅负责导入、标准化、查询展示，不负责完整教务写侧
- `resource`：资源共享模块
- `storage`：统一存储驱动层 / 挂载层 / 文件访问抽象
- `notification`：站内通知、实时推送
- `authorization`：capability、本地策略、FGA 投影与封装
- `audit`：统一审计事件、领域事件、outbox 基础设施
- `admin`：后台运营与治理入口
- `integration`：学校教务连接器、外部同步器、未来网盘驱动相关适配层（如有必要）

### 6.2 对 `course` 目录的处理原则

当前 `course` 目录语义过宽，容易混淆：

- 课程目录（catalog）
- 教务运行态 / 教务数据展示（academics）
- 评课社区（review）

因此，若当前目录结构阻碍长期演进，应直接拆分为：

- `catalog`
- `review`
- `academics`

不要因为兼容成本维持混乱边界。

---

## 7. 非目标与禁止事项（必须明确）

以下内容在当前规划中属于 **禁止开发** 或 **明确不做**：

1. 实验系统
2. 作业系统
3. 提交 / 批改 / 评分 / 申诉系统
4. 完整选课 / 退课 / 排课写侧系统
5. 通用作业附件平台
6. 为不存在业务硬造过度抽象
7. 为兼容旧接口保留双轨模型
8. 手改生成代码

如果你发现仓库或旧文档中还有“实验域 / 作业域 / 评分域 / 教学运行写侧”相关规划，应主动把它们从主路线中移除，或标注为**非当前目标**。

---

## 8. 执行纪律：每轮开发都必须做什么

### 8.1 先落库执行计划文档

启动后，必须新建或更新：

`docs/exec-plans/archived/stuhelper-master-plan.md`

文档最少包含以下字段：

- overall goal
- product scope
- out of scope
- current phase
- done
- next
- risks
- pending external context
- touched files
- tests run / results

### 8.2 每轮都要回写状态

每完成一批修改，必须更新执行计划文档，记录：

- 已完成什么
- 尚未完成什么
- 当前阻塞
- 是否有真实外部上下文缺口
- 测试执行结果

### 8.3 质量门禁

每个阶段结束后，至少执行并修到通过：

```bash
cd server && go test ./...
cd server && go test ./internal/app -run TestOpenAPIRoutes_AreFullyRegistered -count=1
cd server && npx --yes @redocly/cli@2.21.1 lint --config redocly.yaml api/openapi.yaml
cd server && bash scripts/check-coverage-threshold.sh
```

如果 `server/Makefile` 或仓库已有更合适的标准入口，优先复用，但必须把实际执行命令写入执行计划文档。

### 8.4 上下文即将耗尽时的动作

如果上下文将耗尽：

1. 先把当前阶段状态完整写回 `docs/exec-plans/archived/stuhelper-master-plan.md`
2. 记录精确的下一步
3. 记录失败测试与原因
4. 再结束本轮

---

## 9. 总体开发路线（重写后的正确版本）

现在起，固定按以下阶段推进。没有充分理由不要跳阶段。

---

# Phase 0：冻结产品范围、建立事实基线、写入总执行计划

## 目标

把“StuHelper 是什么，不是什么，下一阶段做什么”先落到仓库文档里，避免后续继续按照错误方向扩展。

## 必做任务

1. 扫描仓库当前结构、模块、OpenAPI、migrations、docs、CI、Makefile。
2. 新建或更新：
   - `docs/exec-plans/archived/stuhelper-master-plan.md`
   - `docs/architecture/0001-stuhelper-target-scope-and-module-boundaries.md`
3. 在架构文档中明确写清：
   - StuHelper 不是完整教务系统
   - StuHelper 不做实验 / 作业 / 评分
   - 后续重点是：auth、user、review、academics、resource、storage、audit、authorization
4. 重组或补写 `docs/product-specs` 索引，使其与新的产品边界一致。
5. 检查旧规划文档中是否仍有实验 / 作业 / 完整教学系统方向，若有则修正或标注弃用。

## 完成定义

- 总执行计划已写入仓库
- 目标边界已在架构文档明确
- 旧错误路线已被显式移除或降级
- 后续阶段顺序已固定

---

# Phase 1：冻结目标架构并重组模块骨架 / 文档骨架

## 目标

让后续开发在正确模块边界上继续，不要再把新功能堆进 `course` 大杂烩中。

## 必做任务

1. 评估并重构模块骨架，必要时把 `course` 拆分为：
   - `catalog`
   - `review`
   - `academics`
2. 为以下模块建立稳定骨架：
   - `academics`
   - `resource`
   - `storage`
   - `audit`
   - `authorization`
3. 重组 OpenAPI tags / section grouping，使其与目标模块一致。
4. 重组 `docs/product-specs`：
   - 保留已有真实模块文档
   - 新增：
     - `academics-data-integration.md`
     - `resource-sharing.md`
     - `storage-driver-architecture.md`
   - 明确不再新增 lab / assignment 等规格文档
5. 更新 README 中的项目定位，防止误导为“完整教学实验平台”。

## 要求

- 这是结构落地阶段，不只是写文档
- 可以改目录、改路由装配、改 tags、改 docs 索引
- 但不要在本阶段硬写大量新业务实现

## 完成定义

- 模块边界在目录、OpenAPI、文档三处统一出现
- README 与 docs 不再误导范围
- 现有测试仍通过

---

# Phase 2：修复 native OIDC sid / session 根链路

## 目标

先修认证根问题，再扩新域。不要打补丁，要统一 session 抽象。

## 必做任务

先重点阅读：

```text
server/internal/modules/auth/
server/internal/pkg/middleware/auth.go
相关 auth/session/blacklist/refresh tests
server/api/openapi.yaml 中 auth 相关契约
```

然后完成：

1. 统一 session identity 模型，明确 sid / sessionID / subject / device / token family 的关系。
2. 修复 native OIDC refresh 中 sid 传播缺口。
3. 统一以下语义：
   - refresh rotation
   - logout
   - logout-all
   - blacklist
   - session touch
4. 必要时重构：
   - auth OpenAPI 契约
   - claims 模型
   - service 签名
   - Redis session schema
   - middleware 认证上下文暴露字段
5. 为以下路径补足回归测试：
   - native exchange
   - refresh rotation
   - logout
   - logout-all
   - blacklist
   - optional auth degraded path

## 完成定义

- sid/session 在所有登录与刷新路径上可追踪
- 撤销与轮换语义一致
- auth 相关测试通过
- 执行计划文档记录设计决策与剩余风险

---

# Phase 3：建立“教务数据接入与展示域”（不是教务系统）

## 目标

建立一个长期正确的 `academics` 模块，用来接收学校教务系统数据、落本地标准化模型、供本系统展示与关联查询使用。

注意：**这不是完整教务系统，不要做完整教务写侧。**

## 正确定位

本阶段要做的是：

- 导入外部教务数据
- 建本地标准化投影模型
- 建查询与展示 API
- 建“我的课程 / 我的课表 / 课程开课详情”等视图
- 预留真实教务连接器接口

本阶段不要做的是：

- 自己维护课程开课主数据作为唯一真源
- 实现完整选课 / 排课 / 退课 / 调课流程
- 写完整教务运营后台

## 推荐模型（可按更优方案调整，但必须满足语义）

建议至少考虑以下层次：

### A. 外部源与导入作业

- `academic_sources`
- `academic_import_jobs`
- `academic_import_batches`
- `academic_raw_payloads`（可选，用于排障 / 回溯）

### B. 规范化教务读模型

- `academic_terms`
- `academic_courses`（如需与 catalog 区分，可保留映射）
- `academic_offerings`
- `academic_classes` / `academic_sections`（命名二选一，但全仓统一）
- `academic_teachers`
- `academic_schedules`
- `academic_memberships`（学生 / 教师 / 助教等从外部导入的关系快照）

### C. 外部 ID 与本地 ID 映射

必须保留：

- source_system
- external_id / external_key
- stable local id
- import batch/version

避免未来导入冲突与覆盖不清晰。

## 连接器边界（非常重要）

当前**不要实现真实学校教务系统模拟登录与抓取逻辑**，但必须预留出正确接口。

建议建立清晰的 connector abstraction，例如：

- `Connector` / `Provider` 接口
- `FetchTerms`
- `FetchOfferings`
- `FetchSchedules`
- `FetchStudentCourses`
- `FetchStudentSchedule`
- `FetchTeachers`
- `HealthCheck`

当前阶段只需要：

1. 定义接口
2. 预留 provider registry
3. 提供 noop / mock / fixture-based 实现
4. 让 import pipeline 可以在没有真实远程连接器的情况下完成闭环测试

### 当前推荐导入方式

优先实现以下至少一种可运行导入路径，用于把整个架构真正跑通：

- fixture / JSON / CSV 导入
- 管理端手动导入标准化数据
- seed / script 驱动的导入任务

目的不是模拟真实学校，而是先把：

- 数据模型
- 导入任务框架
- 查询展示 API
- 用户关联查询

做成真实可运行闭环。

## API 最少交付能力

至少交付以下 API 能力：

1. 管理员查看导入源 / 导入任务状态
2. 管理员触发一次导入（基于 fixture / stub / 预留 provider）
3. 查询学期列表
4. 查询开课列表 / 课程运行实例列表
5. 查询课程详情（含教师、时间地点等）
6. 查询我的课程
7. 查询我的课表
8. 按学校 / 学期 / 院系 / 课程名 / 教师名过滤查询

## 与 catalog 的边界

- `catalog` 负责静态课程目录与元数据
- `academics` 负责从外部导入的“某学期真实开课数据 / 个人教务数据”的展示投影
- 二者应允许映射，不要混成一张语义不清的表

## 实施要求

- 先改 OpenAPI，再做 migration、repo、service、handler、authz、tests、docs、seed
- 列表接口必须支持稳定排序、分页、过滤
- 权限优先 capability + 本地策略
- 不要过度依赖 FGA
- 更新 product spec

## 完成定义

- `academics` 模块真实落地
- 本地标准化教务读模型已建立
- 导入任务框架已建立
- 真实远程连接器尚可为空，但接口和 stub 已预留
- “我的课程 / 我的课表 / 开课详情”可用

---

# Phase 4：建立“资源共享模块”与“可插拔存储驱动层”

## 目标

不要建设“为实验 / 作业 / 提交服务的通用文件中心”，而是建设：

1. 资源共享模块
2. 统一存储驱动层
3. 可挂载第三方存储的架构

## 正确拆分方式

### A. `storage` 模块

负责：

- 存储驱动抽象
- 挂载点模型
- 驱动注册与能力声明
- 统一上传 / 下载 / 访问 / 删除接口
- 统一错误模型
- 统一观测与诊断

### B. `resource` 模块

负责：

- 资源的业务元数据
- 资源分类、标签、归属、可见性
- 与课程 / 学期 / 学校 / 社区内容等的业务关联
- 资源分享、浏览、检索

这两个模块必须解耦：

- `resource` 不直接依赖 S3 SDK
- `resource` 只依赖 `storage` 暴露的稳定接口

## 必须预留的存储驱动架构

至少设计并实现以下概念：

- `storage_mounts`
- `storage_drivers`
- `storage_objects` 或等价逻辑引用模型
- driver registry
- provider capability model

驱动能力建议至少覆盖：

- HealthCheck
- Put / InitiateUpload / CompleteUpload
- GetDownloadURL
- Stat
- Delete
- List（若驱动支持）
- Capability 描述（是否支持预签名、是否支持列目录、是否支持分片等）

## 当前阶段必须实现的驱动

当前只要求真实实现：

- `s3` / `minio` driver

同时必须预留：

- `openlist-like` driver family 的接口边界
- 未来 `bhpan` 这类 driver 的注册入口

注意：

- 当前不要实现 bhpan 真实驱动
- 但不要把架构写死在 S3 语义上

## 资源共享模块建议能力

建议至少支持：

1. 资源列表
2. 资源详情
3. 资源上传 / 创建
4. 资源更新元数据
5. 资源删除 / 下架
6. 资源与课程 / 学期 / 学校 / 公共空间的绑定
7. 资源标签 / 分类 / 搜索
8. 下载链接获取
9. 管理员管理挂载点（至少骨架）

## 元数据模型建议

可以参考但不必拘泥于以下模型：

- `resource_spaces`
- `resource_items`
- `resource_versions`
- `resource_bindings`
- `resource_tags`
- `storage_mounts`
- `storage_object_refs`

## 与现有对象存储的关系

- 当前实名认证照片上传链路已存在
- 你可以决定是否把其底层访问重构为统一 `storage` 层
- 但不要为了“资源共享模块”把实名认证材料硬塞成公开资源
- 更合理的做法通常是：
  - 底层共享同一 `storage` abstraction
  - 上层业务仍分别由 `user identity` 与 `resource` 模块管理

## 实施要求

- 先改 OpenAPI，再做 migration、repo、service、handler、authz、tests、docs、seed
- 必须完成 driver abstraction，而不是只写一个 S3 helper
- 必须写清 driver capability 与错误语义
- 必须把未来第三方网盘挂载需求体现在架构文档和接口边界中

## 完成定义

- `storage` 成为一级模块
- `resource` 成为一级模块
- S3 驱动跑通
- 第三方网盘 driver 的接口边界已预留
- 资源共享模块可以真实上传 / 展示 / 下载 / 绑定 / 搜索

---

# Phase 5：建立统一审计事件与领域事件基础设施

## 目标

把当前分散的管理员日志、同步 outbox、业务事件，统一成可扩展的事件与审计基础设施。

## 本阶段要覆盖的真实业务域

至少先覆盖：

- auth
- user
- review
- academics
- resource
- admin

## 必做任务

1. 设计统一 audit event envelope，至少包含：
   - actor
   - action
   - resource_type
   - resource_id
   - scope / school
   - before / after
   - result
   - reason
   - trace_id
   - ip
   - user_agent
   - created_at
2. 建立统一 domain event outbox 抽象。
3. 让业务层不直接关心底层 polling 细节。
4. 评估现有 review admin logs、user external sync、review FGA sync，能合并则合并。
5. 为后续 notification / authorization projection / analytics 预留稳定扩展点。

## 实施要求

- migration、repo、service helper、worker、tests、docs 全部补齐
- 强化 structured logging 与 trace 的关联
- 不要为了不存在的实验 / 作业域设计过重模型

## 完成定义

- 审计与领域事件形成统一基础设施
- 当前关键模块已有接入样例
- 老的碎片化 outbox / 管理日志路径得到收敛

---

# Phase 6：授权模型与外部依赖治理重构

## 目标

降低 Zitadel / OpenFGA / LDAP / Redis / S3 / 未来教务连接器 / 未来网盘驱动带来的耦合和故障放大风险。

## 必做任务

1. 全面梳理 auth、user、review、academics、resource、admin 的授权模型。
2. 能在本地 capability + 本地策略判断的权限，尽量不要外化给 FGA。
3. 真正需要复杂关系图授权的部分，保留 FGA，但统一：
   - projection model
   - retry / reconciliation
   - observability
4. 为外部依赖补齐：
   - timeout
   - retry/backoff（仅适用于幂等操作）
   - typed errors
   - health / readiness
   - diagnostics
   - metrics / traces
5. 抽象学校教务连接器和未来网盘驱动的 provider lifecycle：
   - registry
   - config schema
   - secret handling
   - health check
   - capability reporting
6. 对 student verification、OIDC、FGA、存储驱动等高风险链路完善错误分类与排障信息。

## 完成定义

- 授权边界清晰：本地策略与 FGA 各司其职
- 外部依赖失败时可诊断、可观测、可测试
- `academics` 连接器和 `storage` 驱动的治理方式统一

---

# Phase 7：黑盒集成测试、OpenAPI 收口、文档收口

## 目标

把核心链路做成真正可系统验证，而不是只靠局部单元测试。

## 黑盒测试优先覆盖

1. auth：
   - login/callback 或等价替身
   - refresh rotation
   - logout / logout-all
2. user：
   - identity submit
   - student verify
   - bind phone
3. review：admin 与普通用户关键链路
4. academics：
   - 导入任务触发
   - 开课数据查询
   - 我的课程
   - 我的课表
5. resource：
   - 上传 / 创建资源
   - 获取下载链接
   - 资源绑定
   - 标签 / 搜索 / 列表
6. storage：
   - S3 driver
   - mount health
   - capability exposure

## 同步收口的内容

- `server/api/openapi.yaml`
- bundled OpenAPI 产物
- README
- `docs/product-specs`
- `docs/operations`
- `server/scripts/seed.sql`
- 本地运行与环境变量说明

## 完成定义

- 关键链路有黑盒验证
- OpenAPI、实现、文档、seed 一致
- 执行计划文档给出已完成清单、剩余 backlog、下一轮优先级

---

## 10. 具体实施时的关键设计要求

### 10.1 关于 `academics` 的额外要求

1. 导入要有作业状态，不要做隐式静默导入。
2. 必须保留外部主键映射。
3. 必须考虑“同一课程跨学期 / 同一班课多次导入”的幂等与版本策略。
4. “我的课表”优先建立在导入的 membership / schedule 上，而不是即时抓远端系统。
5. 若当前阶段无法获得真实学校连接器，就用 fixture / stub 把导入链路完整跑通。

### 10.2 关于 `resource` 的额外要求

1. 资源共享是业务模块，不等于底层文件系统。
2. 资源必须有业务元数据，不要只有 object key。
3. 资源与存储对象必须解耦，允许未来迁移 mount / driver 而不推翻业务表。
4. 如要支持版本，优先设计为 `resource item` + `version`，不要一开始就把“文件对象”直接当资源本体。

### 10.3 关于 `storage` 的额外要求

1. 绝对不要把接口写死在 S3 SDK 语义上。
2. 驱动能力要显式声明，不要用“某方法 panic / 返回 not supported”凑合。
3. mount 配置与密钥要有清晰边界。
4. 统一错误模型必须能区分：
   - 配置错误
   - 认证失败
   - 权限不足
   - 对象不存在
   - 网络异常
   - driver 不支持某能力

### 10.4 关于 `audit` 的额外要求

1. 审计是全局能力，不是 review 专属。
2. 审计要覆盖管理员行为，也要覆盖关键用户行为。
3. 审计记录要能和 trace / request id 关联。

---

## 11. Codex 每轮输出要求

每一轮完成后，你都应当在执行计划文档和你的阶段总结中明确给出：

1. 本轮改了哪些文件
2. 本轮完成了哪些目标
3. 还有哪些未完成
4. 当前阻塞是什么
5. 哪些阻塞是“真实缺少外部上下文”
6. 本轮运行了哪些测试，结果如何
7. 下一轮从哪里继续

不要只说“已完成部分工作”。必须具体、可追踪、可接力。

---

## 12. 续跑提示词（上下文丢失后直接使用）

把下面这段当作恢复执行时的提示词：

```text
读取以下内容后直接继续开发，不要重复已完成工作：
- docs/exec-plans/archived/stuhelper-master-plan.md
- 最新 architecture ADR
- 最新 product specs
- 最近改动过的相关模块
- 最近失败的测试或 lint 输出

先总结：
1. 已完成项
2. 未完成项
3. 当前最高优先级
4. 当前阻塞与风险

然后直接继续最高优先级未完成阶段。

严格遵守当前产品边界：
- StuHelper 不是完整教务系统
- StuHelper 不做实验 / 作业 / 评分
- 教务相关只做外部数据导入、标准化、展示与查询
- 资源相关只做资源共享模块与可插拔存储驱动
- 第三方教务系统连接器与第三方网盘驱动只预留接口，不强行实现真实接入

若上一阶段停在半成品，先补齐 OpenAPI / migration / tests / docs，再继续新功能。
每完成一批修改后，更新执行计划文档中的 done / next / risks / pending external context / tests run。
本轮结束前必须把仓库状态写回执行计划文档。
```

---

## 13. 三个通用子提示词

### 13.1 新功能开发通用提示词

```text
基于当前执行计划与目标架构，为功能【填写功能名】做长期正确的企业级实现。不要考虑迁移成本、兼容成本、最小化修改；必要时直接重构 OpenAPI 契约、服务签名、模块边界、文档结构。

必须按以下顺序工作：
1. 先阅读相关模块、OpenAPI、migration、tests、product spec
2. 明确领域模型与数据模型
3. 先改 OpenAPI，再改实现
4. 交付 migration、repo、service、handler、authz、tests、docs、seed
5. 跑完整质量门禁并修到通过
6. 更新执行计划文档

特别边界：
- 不要把 StuHelper 做成完整教务系统
- 不要新增实验 / 作业 / 评分域
- 教务相关功能仅限导入、标准化、展示、查询
- 存储相关功能优先沉淀在 storage/resource 两层架构中
```

### 13.2 Debug / 修复缺陷通用提示词

```text
处理缺陷【填写缺陷名】时，先复现，再写失败测试，再修根因，不允许只做表面热补丁。项目未上线，不需要保留脏兼容逻辑；若更深的契约、服务签名、数据模型重构才是根因修复路径，直接重构。

必须做到：
1. 复现问题并锁定 handler / service / repository / authz / contract / data model 中的真实根因
2. 先补失败测试，再修复
3. 如需改接口，先改 OpenAPI
4. 修复后补回归测试与文档说明
5. 更新执行计划文档记录 root cause 和防回归措施
```

### 13.3 Code Review / 重构通用提示词

```text
对目录或改动集【填写范围】做严格的架构级 code review，并直接实施必要修复。不要按最小改动原则审查；项目未上线，允许直接做长期正确的重构。

审查维度至少包括：
- 模块边界是否清晰
- OpenAPI 与实现是否一致
- Handler -> Service -> Repository 是否清晰
- 数据模型是否支持长期演进
- 事务、一致性、幂等是否正确
- 权限模型是否合理
- 外部依赖边界是否清晰
- 性能与查询策略是否正确
- 观测、日志、审计是否齐全
- 测试是否覆盖真实风险

额外边界：
- 不要把不存在的实验 / 作业域重新引入主路线
- 不要把教务展示域误做成完整教务写侧
- 不要把资源共享模块误做成作业附件平台

输出时给出：
1. 必须立即修改的问题
2. 应当尽快重构的问题
3. 你已经直接实施的修复
4. 新增或修改的测试
```

---

## 14. 本文件的最终目的

你要推动仓库达到的不是“完整教务 / 实验 / 作业平台”，而是下面这个更准确、也更可持续的目标：

> 把当前 monorepo 中以评课、认证、用户系统为核心的能力，继续收口成完整的 **校园信息平台**；其中本轮直接实现和重构的重点，仍然是平台中的单体 Go API、共享契约与相关基础设施：
> - 认证与会话可靠
> - 用户与学校身份体系可靠
> - 课程目录与评课体系成熟
> - 外部教务数据可导入、标准化、展示
> - 资源共享模块可用
> - 底层存储具备可插拔 driver 架构
> - 授权、审计、通知、外部依赖治理达到长期正确的企业级水平

如果你的实现方向偏离了这个目标，应主动回退并修正路线。
