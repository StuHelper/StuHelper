# 互动系统

本文档描述评课社区自己的互动功能，包括投票、回复、收藏和用户中心。

回复相关的应用通知属于后续计划，但通知中心不属于本文档范围。当前评课模块只挂了通知读取和已读接口，长期应抽到航小版应用级通知模块。

## 投票

评论支持 `like`（赞）和 `dislike`（踩）两种投票类型。同一用户对同一评论只有一个有效投票，支持切换类型和取消。

### 投票逻辑

所有投票操作在单个数据库事务内完成，确保投票记录和评论计数的一致性：

| 当前状态   | 用户操作 | 执行动作                                                      |
| ---------- | -------- | ------------------------------------------------------------- |
| 无投票     | like     | 创建 like 投票，`like_count + 1`                              |
| 无投票     | dislike  | 创建 dislike 投票，`dislike_count + 1`                        |
| 已 like    | like     | 删除投票，`like_count - 1`（取消）                            |
| 已 like    | dislike  | 更新为 dislike，`like_count - 1`，`dislike_count + 1`（切换） |
| 已 dislike | dislike  | 删除投票，`dislike_count - 1`（取消）                         |
| 已 dislike | like     | 更新为 like，`dislike_count - 1`，`like_count + 1`（切换）    |

### 事务处理

1. 检查评论存在性（`ReviewExistsTx`）
2. 查询当前投票状态（`GetVoteType`）
3. 根据当前状态执行创建/更新/删除操作
4. 同步更新评论的 `like_count` / `dislike_count`

投票操作使用 `CreateVote` 的 `ON CONFLICT DO NOTHING` 策略，防止并发请求导致重复记录。`CreateVote` 返回 `inserted` 布尔值指示是否实际插入了新行，未插入时跳过计数更新。

### 限流

投票操作受 Redis 限流器保护，默认限制为 30 次/分钟。

## 回复

用户可以对评论创建回复，回复支持嵌套（通过 `parentID` 引用父回复）。

### 创建回复

`CreateReply` 流程：

1. 请求校验：`content` 必填，1-1000 字符；`parentID` 可选
2. XSS 防护：`ContainsDangerousContent` 检测
3. 内容清洗：`SanitizeText` 处理
4. 敏感词检查：通过 `Filter.CheckContent` 检测，block 级触发 `ErrSensitiveContent`
5. 事务内操作：
   - 检查评论存在性（`ReviewExistsTx`，事务内消除 TOCTOU 竞态）
   - 创建回复记录（`CreateReply`，RETURNING 返回时间戳）
   - 递增评论的 `reply_count`
6. 缓存失效

回复通知是后续计划。当前创建回复不会自动写入通知。

### 删除回复

`DeleteReply` 流程：

1. 事务内获取回复的所有者、关联评论 ID 和状态（行锁）
2. 已删除的回复返回 `ErrReplyNotFound`（防止并发双击导致计数双减）
3. 校验当前用户是回复所有者
4. 软删除回复
5. 递减评论的 `reply_count`

### 回复列表

`GetReplies` 返回指定评论的回复列表，支持分页。每条回复包含 `isOwner` 字段，根据当前用户的 userHash 计算。回复的 `userHash` 在 service 层返回前被手动清空（纵深防御，配合 `json:"-"` 标签）。

### 限流

回复操作受 Redis 限流器保护，默认限制为 10 次/分钟。

## 收藏

用户可以按课程维度收藏课程。

### 添加收藏

`AddFavorite` 流程：

1. 校验课程存在性
2. 创建收藏记录（`CreateFavorite`，使用 `ON CONFLICT DO NOTHING` 防止重复收藏）

### 取消收藏

`RemoveFavorite` 直接删除收藏记录，幂等操作。

### 收藏列表

`GetUserFavorites` 返回用户的收藏课程列表，支持分页。每条记录包含课程基本信息（名称、编码、学分、院系、评论数）和收藏时间。

## 用户中心

用户中心提供当前用户的评论、投票和收藏的聚合查询。

### 我的评论

`GetUserReviews` 返回当前用户发布的评论列表，按时间降序排列，支持分页。每条评论包含课程名称、教师名称和学期信息。

### 我的投票

`GetUserVotes` 返回当前用户的投票记录，支持按投票类型过滤：

- `voteType=like`（默认）：返回赞过的评论
- `voteType=dislike`：返回踩过的评论

支持分页。

### 我的收藏

复用上述收藏列表接口 `GetUserFavorites`。
