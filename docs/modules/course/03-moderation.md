# 内容审核

本文档描述评课社区当前已经落地的审核能力，同时保留后续要实现的审核目标，避免把规划内容写成已实现事实。

## 举报流程

### 举报类型

| 类型            | 说明     |
| --------------- | -------- |
| `spam`          | 垃圾广告 |
| `inappropriate` | 不当内容 |
| `harassment`    | 骚扰攻击 |
| `false_info`    | 虚假信息 |
| `other`         | 其他     |

### 创建举报

`ReportReview` 流程：

1. 校验举报原因非空
2. XSS 防护：对描述文本执行 `ContainsDangerousContent` 检测和 `SanitizeText` 清洗
3. 事务内操作：
   - 检查评论存在性（`ReviewExistsTx`）
   - 检查是否已举报（`ReportExistsTx`），同一用户对同一评论只能举报一次，重复举报返回 `ErrAlreadyReported`
   - 创建举报记录（`CreateReportTx`），初始状态为 `pending`

### 限流

举报操作受 Redis 限流器保护，默认限制为 10 次/分钟。

## 管理员处理举报

管理员通过 `ProcessReport` 处理举报，支持三种操作：

| 操作            | 举报状态变更            | 关联评论操作                                                         |
| --------------- | ----------------------- | -------------------------------------------------------------------- |
| `reject`        | `pending` -> `rejected` | 无                                                                   |
| `hide`          | `pending` -> `resolved` | 将评论状态改为 `hidden`，如果原状态为 `published` 则递减课程评论计数 |
| `delete`        | `pending` -> `resolved` | 软删除评论，如果原状态为 `published` 则递减课程评论计数              |

整个处理过程在事务内完成，使用 `GetReportByIDForUpdate` 加行锁防止并发处理。处理完成后记录操作日志，失效举报列表缓存和评论缓存。

## 评论管理

### 单条评论管理

管理员通过 `AdminUpdateReview` 对单条评论执行隐藏/恢复/删除操作。

操作流程：

1. 事务内读取评论当前状态和课程 ID（行锁）
2. 通过 `validTransitions` 白名单校验状态转移合法性
3. 执行状态变更和计数同步：
   - **隐藏**：状态设为 `hidden`，递减课程评论计数。如果提供了 AdminID，同时记录屏蔽原因（`moderation_reason`）和审核人（`moderated_by`）
   - **恢复**：状态设为 `published`，清除屏蔽信息，递增课程评论计数
   - **删除**：软删除。如果原状态为 `published`，递减课程评论计数
4. 返回旧状态用于操作日志

### 批量评论管理

管理员通过 `BatchUpdateReviews` 对多条评论执行批量操作：

- 支持 `hide`、`restore`、`delete` 三种操作
- 批量上限 100 条（handler 和 service 层双重校验）
- 所有 ID 必须为合法 UUID 格式
- 事务内执行：先锁定涉及的评论行（`LockReviewsTx`），再调整课程计数，最后批量更新状态
- 返回实际受影响的行数

### 管理员编辑评论内容

管理员通过 `AdminEditReviewContent` 直接编辑评论的标题和内容：

1. 事务内确认评论存在
2. 调用 `AdminEditReviewContentTx` 保存编辑：
   - 将当前标题和内容保存为 `original_title` / `original_content`（保留原始内容）
   - 更新标题和内容为管理员提供的新值
   - 记录编辑原因（`moderation_reason`）和审核人（`moderated_by`/`moderated_at`）
3. 记录操作日志
4. 失效缓存

## 敏感词管理

敏感词存储在 `sensitive_words` 表中，支持完整的 CRUD 管理。

### 当前实现

当前写链路只接入本地敏感词过滤器。

| 级别    | 当前行为                                                     |
| ------- | ------------------------------------------------------------ |
| `block` | 命中后直接阻止提交                                           |
| `warn`  | 当前只保留级别语义和匹配能力，后续要补特殊标记和二次审核联动 |

当前过滤器的执行方式：

- 敏感词列表从数据库加载并缓存在内存中，每 5 分钟自动刷新
- 使用 `singleflight` 去重并发刷新请求，避免多个 goroutine 同时查询数据库
- 纯 ASCII 英文词使用 `\b` 词边界正则匹配，避免误伤 `class` 这类单词
- 中文等非 ASCII 词使用子串匹配
- 检查时内容统一转小写后匹配

### 后续目标

后续审核链路按流水线扩展，不替换现有敏感词能力。

| 节点              | 目标行为                                                              |
| ----------------- | --------------------------------------------------------------------- |
| 白名单 / 免审     | 基于 RBAC 的角色、用户组、个人授权和额外白名单规则决定是否跳过审核    |
| 固定词库          | 先执行本地敏感词检查，命中 `block` 直接返回                           |
| 特殊标记          | 命中 `warn` 时给内容打审核标记，进入后续复核链路                      |
| 外部审核 Provider | 在本地词库通过后，再串联 AI 审查、腾讯云文本安全等 Provider           |
| 聚合决策          | 多个 Provider 返回统一 verdict，支持阻止、放行、标记待审              |
| 配置策略          | 支持按学校、模块、内容类型决定是否启用某个 Provider，支持并存和优先级 |

建议把审核能力抽象成统一接口：

- `ReviewProvider` 负责一次审核
- `ReviewPipeline` 负责节点编排
- `ReviewVerdict` 负责统一输出
- `ReviewBypassPolicy` 负责免审判断

这样后面接 AI 或云审核时，不会把写链路继续塞进单个过滤器文件里。

### CRUD 接口

| 操作 | 端点                                 | 说明                                         |
| ---- | ------------------------------------ | -------------------------------------------- |
| 列表 | `GET /admin/sensitive-words`         | 支持按 category 和 level 过滤，分页          |
| 创建 | `POST /admin/sensitive-words`        | word 必填 1-100 字符                         |
| 更新 | `PUT /admin/sensitive-words/{id}`    | 支持部分更新（word/category/level/isActive） |
| 删除 | `DELETE /admin/sensitive-words/{id}` | 硬删除                                       |

所有管理操作记录操作日志。

## 教师管理

教师存储在 `teachers` 表中，支持 CRUD 管理。

### CRUD 接口

| 操作 | 端点                          | 说明                                                       |
| ---- | ----------------------------- | ---------------------------------------------------------- |
| 列表 | `GET /admin/teachers`         | 支持按名称搜索和院系 ID 过滤，分页，返回含院系名称和评论数 |
| 创建 | `POST /admin/teachers`        | name 必填 1-100 字符，departmentID 可选                    |
| 更新 | `PUT /admin/teachers/{id}`    | name 必填，departmentID 可选                               |
| 删除 | `DELETE /admin/teachers/{id}` | 硬删除                                                     |

教师列表返回 `AdminTeacher` 类型，包含院系名称（通过 JOIN departments）和关联评论数。

所有管理操作记录操作日志。

## 数据导出

管理员通过 `ExportReviews` 导出评论数据，支持三种格式：

### NDJSON 导出

- Content-Type: `application/x-ndjson; charset=utf-8`
- 每行一个 JSON 对象，换行分隔
- 流式输出，逐行从数据库读取并写入 HTTP 响应
- 文件末尾追加 `# EXPORT_COMPLETE` 标记

### CSV 导出

- Content-Type: `text/csv; charset=utf-8`
- 写入 UTF-8 BOM 头（`0xEF 0xBB 0xBF`），确保 Excel 正确识别编码
- 表头：ID/课程ID/课程名称/教师ID/教师名称/学期/标题/内容/成绩/评分/点赞数/踩数/状态/创建时间
- 流式输出，逐行从数据库读取并写入
- CSV 公式注入防护：以 `= + - @ \t \r` 及其全角变体开头的单元格自动添加前缀 `'`
- 评分字段格式化为 `key1:value1;key2:value2` 形式
- 文件末尾追加 `# EXPORT_COMPLETE` 标记

### 导出过滤

支持按评论状态过滤（`status` 参数），可选值：`published`/`hidden`/`deleted`/`all`（默认）。

## 操作审计日志

### 日志记录

所有管理操作通过 `logAdminOp` 辅助函数记录日志，写入 `admin_operation_logs` 表：

| 字段             | 说明                                                                          |
| ---------------- | ----------------------------------------------------------------------------- |
| `admin_user_id`  | 操作人 ID                                                                     |
| `admin_username` | 操作人用户名                                                                  |
| `action`         | 操作类型（如 `hide`/`restore`/`delete`/`batch_hide`/`process_report_reject`） |
| `resource_type`  | 资源类型（`review`/`report`/`teacher`/`sensitive_word`）                      |
| `resource_id`    | 资源 ID                                                                       |
| `old_value`      | 变更前值（JSON）                                                              |
| `new_value`      | 变更后值（JSON）                                                              |
| `ip_address`     | 操作人 IP                                                                     |
| `user_agent`     | User-Agent（截断处理）                                                        |

### 日志查询

`GetOperationLogs` 返回操作日志列表，按时间降序排列，支持分页。

### 自动清理

后台定时任务（`StartBackgroundJobs`）每 24 小时执行一次日志清理：

- 清理 90 天以前的操作日志（`CleanupOldOperationLogs`）
- 服务启动时立即执行一次清理
- 通过可取消的 context 实现优雅关闭
- 清理完成后记录删除行数

## 管理后台统计仪表盘

`GetAdminStats` 返回管理后台的统计数据，使用版本化缓存：

| 指标               | 说明           |
| ------------------ | -------------- |
| `totalReviews`     | 评论总数       |
| `publishedReviews` | 已发布评论数   |
| `hiddenReviews`    | 已隐藏评论数   |
| `deletedReviews`   | 已删除评论数   |
| `pendingReports`   | 待处理举报数   |
| `totalReports`     | 举报总数       |
| `todayReviews`     | 今日新增评论数 |
| `weekReviews`      | 本周新增评论数 |
