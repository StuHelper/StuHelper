# 安全设计

本文档定义评课社区模块的安全策略和防护措施。

## 用户隐私保护

### 匿名机制

所有测评采用匿名发布，用户身份通过 HMAC-SHA256 哈希处理：

```go
userHash = HMAC-SHA256(userID, secret)
```

**隐私保护要点**：
- 前端不展示任何用户标识
- 后端仅存储哈希值，API 响应中回复的 UserHash 在 service 层返回前被清空
- HMAC 密钥生产环境最低 32 字符，不足则拒绝启动

## 内容安全

### 敏感词过滤 + 质量评估

通过单一 `POST /content/check` 端点同时完成敏感词检测和内容质量评估：

```json
{
  "sensitive": {
    "hasSensitive": false,
    "matchCount": 0
  },
  "quality": {
    "score": 85,
    "suggestions": ["quality_too_short"]
  }
}
```

- 敏感词库存储在 `sensitive_words` 表，支持 block/warn/review 三级
- API 响应不返回具体匹配的敏感词（防探测），仅返回匹配数量
- 内容检查端点需要认证，防止敏感词列表被探测
- 内容提交前经过 HTML 清洗（解码实体 + 移除零宽字符 + 危险标签过滤）

### 数据库约束

- `reviews.title`: `CHECK (char_length(title) <= 200)`
- `review_replies.content`: `CHECK (char_length(content) <= 5000)`
- `reviews.status`: `CHECK (status IN ('published', 'hidden', 'deleted'))`
- `reviews.avg_rating`: `CHECK (avg_rating >= 0 AND avg_rating <= 5)`
- 各计数字段: `CHECK (xxx_count >= 0)`

### 举报机制

| 举报类型 | 说明 |
|----------|------|
| spam | 垃圾广告 |
| abuse | 辱骂攻击 |
| privacy | 隐私泄露 |
| false | 虚假信息 |
| other | 其他 |

## 防刷机制

### 频率限制（Redis 限流）

| 操作 | 限制 |
|------|------|
| 发布测评 | 5 次/分钟 |
| 投票 | 30 次/分钟 |
| 举报 | 10 次/分钟 |
| 回复 | 10 次/分钟 |
| 更新/删除 | 10 次/分钟 |

### 业务约束

- 同一用户对同一课程仅能发布 1 条未删除的测评（数据库唯一索引）
- 同一用户对同一测评仅能投票 1 次（支持切换类型和取消）
- 同一用户对同一测评仅能举报 1 次

### 限流故障策略

- Redis 故障时限流器 fail-closed（返回 503），不放行请求
- 熔断器开启时 fail-closed（返回 401）

## 安全中间件

### 安全响应头

通过 `security_headers.go` 中间件统一设置：
- `X-Content-Type-Options: nosniff`
- `X-Frame-Options: DENY`
- `X-XSS-Protection: 1; mode=block`
- `Referrer-Policy: strict-origin-when-cross-origin`
- `Content-Security-Policy`

### 请求体大小限制

`MaxBodySize` 中间件限制请求体大小，使用统一响应格式返回错误。

### 统一响应格式

所有中间件（auth/ratelimit/permission/security）均使用 `response` 包返回错误，格式统一为：

```json
{
  "success": false,
  "error": {
    "code": "ERROR_CODE",
    "message": "..."
  }
}
```

## 数据安全

### TOCTOU 防护

关键写操作（UpdateReview/DeleteReview/CreateReply/VoteReview/ProcessReport）的所有权/状态检查均在事务内完成，使用 `FOR UPDATE` 行锁防止竞态。

### 审计日志

记录所有管理操作：
- 删除/隐藏/恢复测评
- 处理举报
- 批量操作
- CSV 导出

### 数据备份

- 每日自动备份
- 保留 30 天历史
