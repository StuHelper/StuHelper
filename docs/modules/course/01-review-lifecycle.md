# 评论生命周期

本文档描述评论从创建到删除的完整生命周期，包括创建流程、更新机制、删除策略、草稿系统和状态流转规则。

## 评论创建

评论创建由 `PostReview` handler 和 `Service.PostReview` 协同完成，经过以下步骤：

### 1. 访问控制

handler 层通过 `resolveReviewAccessFactsForRequest` 计算访问事实。发布评论要求 `CanPostReview == true`，即用户同时通过学生认证和实名认证。

### 2. 请求校验

通过 gin binding 校验请求参数：

| 字段 | 规则 |
| --- | --- |
| `courseID` | 必填，大于 0 |
| `teacherID` | 可选，大于 0 |
| `termID` | 必填，最长 20 字符，格式 `YYYY-S`（如 `2024-1`） |
| `title` | 必填，最长 200 字符 |
| `content` | 必填，10-5000 字符 |
| `grade` | 可选，枚举值 `A+ A A- B+ B B- C+ C C- D F` |
| `ratings` | 必填，评分维度键值对 |

### 3. 评分维度校验

`validateAndSanitizeReview` 方法执行以下校验：

- 学期 ID 格式：正则 `^\d{4}-[12]$`
- 至少包含一个评分维度
- 评分维度 key 格式：正则 `^[a-z][a-z0-9_]{0,49}$`
- 评分维度 key 存在于 `rating_dimensions` 表中（通过缓存的维度名称白名单校验，缓存未命中时回退到数据库查询）
- 每项评分值在 1-5 之间

### 4. 内容清洗

标题和内容经过两层清洗：

- `sanitizer.SanitizeTitle` / `sanitizer.SanitizeText` 执行 HTML 清洗（解码实体、移除零宽字符、过滤危险标签）
- 清洗后检查标题和内容是否为空
- `sanitizer.ContainsDangerousContent` 检测危险内容（标题和内容分别检测）

### 5. 敏感词检查

通过 `Filter.CheckContent` 检查标题 + 内容的拼接文本。敏感词分两级：

- `block` 级：直接阻止提交，返回 `ErrSensitiveContent`
- `warn` 级：记录匹配数量，允许提交

### 6. 事务内创建

在数据库事务内完成以下操作：

1. 检查课程存在性（`CourseExistsTx`）
2. 如果提供了 teacherID，检查教师存在性（`TeacherExistsTx`）
3. 检查用户是否已评论该课程（`UserHasReviewedCourseTx`），同一用户对同一课程只能有一条未删除的评论
4. 使用 `CreateReturning` 创建评论记录并返回完整评论对象
5. 递增课程的 `review_count` 计数

### 7. 审计日志

创建完成后记录审计事件（`audit.EventDataCreate`），包含脱敏后的 userHash（前 12 字符）、IP 地址和 requestID。

### 8. 缓存失效

调用 `invalidateReviewCaches` 失效该课程的评论缓存和全局统计缓存。

## 评论更新

评论更新由 `UpdateReview` handler 和 `Service.UpdateReview` 完成：

1. **参数校验**：与创建流程相同的 binding 校验，content 必填 10-5000 字符，ratings 必填
2. **评分和内容校验**：调用 `validateAndSanitizeReview`，流程与创建一致（termID 传空字符串跳过学期校验）
3. **事务内更新**：
   - 使用 `GetReviewOwnerAndStatusTx` 获取评论所有者和状态（`SELECT ... FOR UPDATE` 行锁，防止 TOCTOU 竞态）
   - 校验评论状态为 `published`（已隐藏或已删除的评论返回 `ErrReviewNotFound`）
   - 校验当前用户是评论所有者（`userHash` 匹配）
   - 调用 `Update` 更新标题、内容、成绩和评分
4. **缓存失效**：失效全局评论缓存

## 评论删除

评论删除采用软删除策略，由 `DeleteReview` handler 和 `Service.DeleteReview` 完成：

1. **事务内删除**：
   - 使用 `GetReviewOwnerCourseIDAndStatusTx` 获取所有者、课程 ID 和状态（行锁）
   - 已删除的评论返回 `ErrReviewNotFound`
   - 校验当前用户是评论所有者
   - 调用 `SoftDeleteReview` 将状态设为 `deleted`
   - 如果原状态为 `published`，递减课程的 `review_count` 并刷新评分统计
2. **缓存失效**：失效全局评论缓存、统计缓存和投票缓存

软删除保留评论数据以支持恢复，前端仅展示未删除的评论。

## 草稿系统

草稿系统提供评论撰写过程中的自动保存和恢复能力。每个用户对每门课程只维护一份草稿。

### 保存草稿

`SaveDraft` 流程：

1. 校验请求参数（courseID 必填，content 最长 5000 字符）
2. 检查 content 非纯空白字符
3. 检查课程存在性
4. 校验 termID 格式（与发布链路一致）
5. XSS 防护：`ContainsDangerousContent` 检测 + `SanitizeTitle` / `SanitizeText` 清洗
6. 序列化评分数据
7. 通过 `UpsertDraft` 执行 UPSERT（按 userHash + courseID 唯一约束）

### 加载草稿

`GetDraft` 根据 userHash 和 courseID 查询草稿，草稿不存在返回 404。

### 删除草稿

`DeleteDraft` 根据 userHash 和 courseID 删除草稿。

## 状态流转

评论有三个状态：`published`、`hidden`、`deleted`。

```
                   用户发布
                     |
                     v
              +------------+
              | published  |
              +------------+
               |    |    ^
   管理员隐藏  |    |    |  管理员恢复
               v    |    |
            +--------+   |
            | hidden |---+
            +--------+
               |
   管理员删除  |    用户删除（从 published）
               v         |
            +---------+   |
            | deleted |<--+
            +---------+
```

### 状态转移规则（管理员）

| 操作 | 允许的源状态 | 目标状态 |
| --- | --- | --- |
| `hide` | `published` | `hidden` |
| `restore` | `hidden` | `published` |
| `delete` | `published`, `hidden` | `deleted` |

管理员操作通过 `validTransitions` 白名单严格校验，非法转换返回 `ErrInvalidTransition`。

隐藏操作可附带屏蔽原因（`moderation_reason`）和操作人（`moderated_by`），恢复操作自动清除屏蔽信息。

### 状态转移规则（用户）

| 操作 | 允许的源状态 | 目标状态 |
| --- | --- | --- |
| 用户删除 | `published`, `hidden` | `deleted` |

用户只能删除自己的评论，已删除的评论返回 404。

### 计数同步

状态变更时同步维护 `courses.review_count`：

- `published` -> `hidden`：递减计数
- `published` -> `deleted`：递减计数并刷新评分统计
- `hidden` -> `published`：递增计数
- `hidden` -> `deleted`：计数不变（隐藏时已递减）
