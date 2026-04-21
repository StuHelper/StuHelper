# StuHelper Core Legacy Capabilities UI Integration Design

**Status:** approved-by-user

## 1. Goal

在当前 `stuhelper-core` 已经承载 grouphelper 风格 UI 与核心模块的基础上，把原来 StuHelper 插件中的额外业务能力正式并入当前插件，而不是继续停留在 `legacy-wrapper` 挂载态。

这次设计的目标不是“再加几个按钮”，而是：

- 保持当前 grouphelper 风格的模块化菜单和页面语言
- 把旧 StuHelper 的核心业务语义拉成清晰的 UI 边界
- 让首页、身份认证、处置中心、配置治理形成稳定的信息架构
- 为后续实现提供明确的前端页面边界、聚合 service 边界、联动规则和测试要求

## 2. User Decisions

以下决策已经由用户明确确认，本设计直接采用：

- 保留当前 grouphelper 风格的模块化菜单，不改成运营域侧栏
- 旧 StuHelper 能力采用“混合并入”策略
  - 高频、强业务能力独立成菜单
  - 低频、配置型能力并入现有页面
- 第一批范围同时覆盖：
  - 身份认证链路
  - 群审与准入链路
  - 策略与模板链路
- `Identity` 必须做成独立主菜单
- `Review` 必须做成完整处置中心，而不是只放人工复核
- `Dashboard` 必须升级为“总控首页”，保留图表，但同时承载待办与业务分发

## 3. Approaches Considered

### 3.1 方案 1：最小侵入扩展

- 保留现有顶栏菜单和页面结构
- 只新增少量入口
- 将大多数旧能力塞入既有页面内部

**优点：**

- 实现成本最低
- 风险最小

**问题：**

- 旧 StuHelper 业务语义会继续埋在现有页面里
- `Dashboard`、`Config`、`Roles` 会越来越重
- 最终 UI 会变成“功能堆放”，而不是清晰产品结构

### 3.2 方案 2：模块化增强

- 保留当前 grouphelper 的模块化菜单风格
- 明确重组页面职责
- 引入 `Identity`、`Review` 两个强业务主菜单
- 将策略治理能力并入 `Config`

**优点：**

- 最尊重当前 UI 风格
- 能把旧 StuHelper 的业务闭环正式抬升为一等页面
- 页面边界清晰，后续容易继续扩展

**问题：**

- 主导航数量增加
- `Dashboard` 和 `Config` 都需要升级，而不是简单追加字段

### 3.3 方案 3：双层导航增强

- 保留当前模块菜单
- 额外引入顶部运营域或二级导航

**优点：**

- 可扩展性最强

**问题：**

- 当前复杂度还不足以支撑双层导航
- 会让现有简洁的 grouphelper 风格被过早复杂化

**结论：**

采用 **方案 2：模块化增强**。

## 4. Information Architecture

顶层继续使用当前 [`client/pages/index.vue`](../../../bots/koishi/plugins/stuhelper-core/client/pages/index.vue) 这套顶栏切页模型，不改造成全新路由式控制台应用，也不增加第二层全局导航。

推荐固定主菜单顺序为：

`Dashboard` → `Config` → `Warns` → `Blacklist` → `Identity` → `Review` → `Roles` → `Logs` → `Chat` → `Subscription` → `Settings`

各页面的职责边界如下：

### 4.1 Dashboard

- 作为总控首页
- 保留现有图表、统计卡片和版本状态
- 新增待处理事项、快捷入口、重点告警、关键系统状态
- 不直接承载复杂编辑逻辑，只承载摘要与跳转

### 4.2 Config

- 从“群组配置”升级为“配置治理中心”
- 保持群配置为默认入口
- 在同一个 `Config` 顶层页面内拆分二级工作区：
  - 群配置
  - 模板库
  - 群绑定
  - 命令策略
- 不要求把所有治理对象都强行塞进单个群编辑表单
- `exemptUsers` 归属于模板编辑上下文，不作为独立顶级工作区
- 不新增单独 `Policy` 主菜单

这一条的作用域规则固定为：

- 群配置：以 `guildId` 为主键
- 模板库：全局模板集合，不带 `guildId`
- 群绑定：群到模板的绑定关系
- 命令策略：全局命令策略，不在本轮改造成群级策略

### 4.3 Warns / Blacklist / Logs / Chat / Subscription / Settings

- 保持当前 grouphelper 风格和既有页面主职责
- 允许补充必要联动入口
- 不在本轮设计中主动重构这些页面的主体信息架构

### 4.4 Identity

新增独立主菜单，统一承载：

- 绑定码记录
- 成员认证状态
- 未认证限制状态
- 自动解除记录
- 绑定与认证统计

### 4.5 Review

新增独立主菜单，作为完整处置中心统一承载：

- 人工复核
- 待准入成员
- 举报处置
- 关键事件联查

### 4.6 Roles

- 保持角色与权限管理主体
- 不再承载复核主流程
- 只作为认证、命令策略等功能的权限依赖页面

### 4.7 Top Navigation Overflow

主导航从 9 项扩展到 11 项后，必须明确桌面端的溢出策略：

- 宽屏桌面（`>= 1280px`）显示全部主菜单
- 中等桌面（`960px - 1279px`）保留高优先级菜单直接展示：
  - `Dashboard`
  - `Config`
  - `Warns`
  - `Blacklist`
  - `Identity`
  - `Review`
  - `Roles`
  - `Logs`
- 低优先级菜单折叠到 `More` 下拉：
  - `Chat`
  - `Subscription`
  - `Settings`
- 移动端继续使用当前抽屉式菜单

这里的“固定顺序”指信息架构顺序固定；在中等桌面下，低优先级项允许进入 `More`，但顺序仍按主菜单顺序显示。

## 5. Page Wireframes and Internal Sections

### 5.1 Dashboard

分区建议：

- 顶部：系统状态、版本、最后更新时间
- 中部：核心指标卡与趋势图表
- 下部左：待处理事项
- 下部右：快捷入口与最近告警

页面定位从纯图表看板升级为“看板 + 待办 + 分发台”。

### 5.2 Identity

分区建议：

- 顶部：认证概览、绑定成功率、待处理人数
- 顶部筛选：当前群、认证状态、关键词
- 中部左：当前群成员认证列表
- 中部右：成员详情、绑定记录、跨群状态摘要
- 底部：未认证限制记录、自动解除记录

页面重点是把“绑定 + 认证状态 + 限制与解除”统一到一页。

`Identity` 的默认作用域固定为“群优先”：

- 页面默认先进入当前选中群的成员认证工作区
- 选中成员后，右侧详情允许查看该成员的跨群认证/限制摘要
- 绑定记录仍然是用户级数据，但只在成员详情上下文中展示
- 不提供“全平台全用户”作为默认首页视角

从 `Dashboard` 跳入 `Identity` 时：

- 如果待办项只关联群，预选 `guildId`
- 如果待办项关联具体成员，预选 `guildId + memberId`

### 5.3 Review

分区建议：

- 顶部：筛选条、状态切换、快速搜索
- 主体左：统一处置列表
- 主体右：详情与处置面板
- 底部：相关事件链与处理记录

页面重点是统一列表视角与右侧处置面板，而不是拆出多个孤立小页面。

`Review` 的统一列表只承载“可操作工作项”，不直接把原始事件当成一类队列项。  
关键事件联查属于选中工作项的上下文信息，而不是与复核/准入/举报平级的第四种主队列对象。

### 5.4 Config

分区建议：

- 顶部：`群配置 / 模板库 / 群绑定 / 命令策略` 工作区切换
- `群配置` 工作区：群配置搜索、视图切换、群编辑表单
- `模板库` 工作区：模板列表、模板详情、模板启用状态、`exemptUsers`
- `群绑定` 工作区：群到模板的映射、绑定备注、有效策略预览
- `命令策略` 工作区：全局命令策略列表与编辑
- 底部或侧栏：有效策略来源与保存历史

页面重点是保留当前群配置工作方式，同时引入治理信息层级。

本页要明确展示两类不同的“来源解释”：

- 群管有效策略来源：
  - 群级配置
  - 绑定模板
  - 静态 fallback 配置
- 命令策略来源：
  - 全局 `CommandPolicy`

`CommandPolicy` 不参与群配置值覆盖优先级计算，它是并列的全局授权治理模型。

## 6. Data Flow and Backend Boundaries

本轮不新造一套并行 UI API，而是在当前 `stuhelper-core` 上把 legacy 挂载的旧 StuHelper 额外能力正式提升为 UI 可消费的一等数据源。

### 6.1 Dashboard Data

Dashboard 只消费摘要数据，不直接消费复杂编辑模型。  
它需要聚合以下四类摘要：

- 系统状态
- 身份认证待办
- 处置中心待办
- 配置治理异常

### 6.2 Identity Data

`Identity` 直接消费身份认证闭环数据：

- 绑定码记录
- 成员认证状态
- 未认证限制状态
- 自动解除记录
- 认证完成回流状态

这些数据来源于当前 legacy 挂载的 `binding` 和 `group-guard` 业务，但页面不能继续只依赖命令层副作用，必须存在可读查询接口。

### 6.3 Review Data

`Review` 统一消费复核与准入数据：

- 踢人 / 拉黑申请
- 待准入成员
- 举报处置项
- 相关事件链

前端主模型固定为判别联合 `ReviewWorkItem`，至少包含：

- `id`
- `kind`
  - `review`
  - `admission`
  - `report`
- `guildId`
- `memberId?`
- `subjectLabel`
- `status`
- `priority`
- `createdAt`
- `availableActions`
- `relatedEventIds`

动作矩阵固定为：

- `review`
  - `execute`
  - `reject`
- `admission`
  - `approve`
  - `deny`
  - `defer`
- `report`
  - `dismiss`
  - `escalate`
  - `create-review`

关键事件不是独立 `ReviewWorkItem.kind`，而是选中工作项后的只读上下文。

`Logs` 与 `Review` 的边界固定为：

- `Review`
  - 展示与当前工作项直接相关的事件摘录和处置上下文
- `Logs`
  - 负责全局检索、原始日志浏览、脱离工作项的历史查询

也就是说，`Review` 面向“处置”，`Logs` 面向“检索”。

### 6.4 Config Governance Data

`Config` 继续以群为主键，但在编辑上下文中拉取治理子模型：

- `GuardTemplate`
- `GuardBinding`
- `CommandPolicy`
- `exemptUsers`

这些治理模型共享同一 `Config` 顶层页面，但不共享同一个群编辑表单。

来源和优先级定义固定为：

- 群配置字段
  - 来源于群级配置记录
- Guard 生效策略
  - 优先使用群绑定对应的模板
  - 若不存在群绑定，则退回静态 fallback guard 配置
- `exemptUsers`
  - 来源于模板，不在群级单独覆写
- `CommandPolicy`
  - 独立全局模型，不参与 guard 值优先级链

### 6.5 Service Shape

后端设计上，`legacy-wrapper` 只是过渡装配层，不应成为最终 UI 聚合层。

建议在 `stuhelper-core/src/core/services/` 内建立面向页面域的聚合 service：

- `dashboard`
- `identity`
- `review`
- `config-governance`

前端 `client/api.ts` 也应逐步从当前的 stats/config/logs/chat 分散接口，升级为按页面域分组的读取与动作接口。

### 6.6 Navigation State

为满足 `Dashboard` 精确跳转与刷新后保持上下文，本轮需要引入统一导航状态模型。

顶层状态建议使用 query-synced 的 `ConsoleNavigationState`，至少包含：

- `view`
- `workspace?`
- `guildId?`
- `memberId?`
- `itemId?`
- `tab?`
- `keyword?`

约束如下：

- `Dashboard` 待办跳转必须写入该状态模型
- `Identity`、`Review`、`Config` 进入页面时都从该状态恢复上下文
- 刷新后保持当前上下文
- 浏览器后退/前进应能恢复最近一次页面级上下文

## 7. Interaction Rules and Error Exposure

本轮遵循现有工程硬规则：

- 不做静默 fallback
- 不伪造成功路径
- 契约缺失或状态非法必须显式暴露

### 7.1 Dashboard Rules

- 每个待办卡片必须能跳到准确的上下文页面
- 不允许只跳到目标页面首页

### 7.2 Identity Rules

成员详情必须展示：

- 当前认证状态
- 最近绑定记录
- 当前限制状态
- 最近解除动作

如果后端查不到认证数据，必须显示空态或错误态，不能默默当作“未认证”。

### 7.3 Review Rules

统一列表必须支持按以下维度过滤：

- 类型
- 状态
- 群组
- 成员
- 关键词

右侧处置面板动作提交后，要么刷新当前上下文，要么准确跳到下一条，不允许 silently 回到全列表。

### 7.4 Config Rules

配置编辑必须显式区分值来源：

- 群级覆盖
- 模板默认
- 绑定来源
- 静态 fallback

并单独显示以下治理来源：

- 命令策略
- 豁免名单

用户必须能看出“当前值来自哪里”，不能只展示最终结果。  
其中 `CommandPolicy` 与 `exemptUsers` 不再被描述成和群配置字段完全同类的覆盖来源。

### 7.5 Error Surfaces

统一采用三层错误暴露：

- 加载失败：页面内错误态 + 重试按钮
- 动作失败：显式通知，保留上下文，不自动清空状态
- 数据缺失 / 契约漂移：明确显示“记录不存在 / 数据未返回 / 状态非法”

## 8. Testing and Verification

本轮设计把测试范围一起锁死：

### 8.1 Frontend Model and Utility Tests

至少覆盖：

- Dashboard 待办聚合
- Identity 状态映射
- Review 统一处置项映射
- Config 治理标签与来源解析

### 8.2 Frontend Interaction Tests

至少覆盖：

- 顶栏导航切换
- 顶栏 `More` 溢出菜单展示与切换
- Dashboard 待办跳转
- Review 筛选与处置上下文保持
- Config 页内标签切换

### 8.3 Backend Service Tests

新增聚合 service 必须覆盖：

- 空数据
- 异常数据
- 跨模块拼装失败

### 8.4 Startup and Runtime Verification

必须保持：

- 5140 固定监听
- `stuhelper-core` 的控制台 WebUI entry 仍然可用
- 构建与启动烟雾测试不因新页面聚合而失效

## 9. Non-Goals

本轮设计不做以下事情：

- 不改造成运营域侧栏应用
- 不再新增 `Policy` 顶层菜单
- 不把认证状态揉进 `Roles` 成为同一主页面
- 不继续依赖 `legacy-wrapper` 直接向 UI 暴露拼装结果
- 不通过兼容 fallback 掩盖旧能力与新 UI 之间的契约缺口

## 10. Implementation Direction

后续实现应按以下顺序展开：

1. 先补面向页面域的聚合数据 service
2. 再改顶层菜单与页面骨架
3. 再分别实现 `Dashboard`、`Identity`、`Review`、`Config` 的页面升级
4. 最后补交互测试、聚合 service 测试和启动验证

本 spec 只定义设计和边界，不包含具体任务拆分。任务拆分将在下一步 implementation plan 中完成。
