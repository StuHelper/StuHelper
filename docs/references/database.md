# 数据库设计

## 数据面

| 存储 | 用途 |
|------|------|
| PostgreSQL | 业务数据 |
| Redis | 会话、黑名单、限流、缓存、通知广播 |
| 对象存储（MinIO/S3） | 证件照片、资源文件（统一经 `storage` abstraction 访问） |
| Zitadel | 身份平面 |
| OpenFGA | 资源关系授权 |

## 权威来源

1. `server/migrations/*.sql` — 唯一权威 schema
2. `server/internal/modules/**/repository*.go` — 查询和事务

## 身份平面

Zitadel 管理账号、OIDC 会话、粗粒度角色。

应用侧保留 `users` 表作为 shadow user：业务外键锚点 + 最小用户画像缓存。

## 业务平面

PostgreSQL 存储：
- 课程、教师、院系、学期、分类
- 评课、回复、投票、举报、草稿、收藏
- 实名认证、学生认证、学校配置、系统配置
- 通知、审计事件、领域 outbox

## 授权平面

能力由角色静态展开，不落本地 RBAC 表。资源级权限由 OpenFGA 增强。

## 表索引

### 用户系统

| 表 | 用途 |
|----|------|
| `users` | Shadow user、业务外键锚点 |
| `user_identities` | 实名认证（`doc_number_enc` 密文、`person_uid` HMAC、`doc_photo_*` 证件照片对象 key） |
| `user_profiles` | 学生认证档案 |
| `schools` | 学校主数据（`id BIGINT` / `code` / `name`），其他学校维度表统一引用它 |
| `school_configs` | 学校认证配置（`school_id BIGINT`，FK → `schools.id`） |
| `system_configs` | 全局配置 |
| `academic.buaa_students` | 学籍数据本地表（从教务同步，含学号、姓名、院系、年级等） |
| `domain_event_outbox` | 统一领域 outbox；当前 `stream` 已落地 `user_external_sync`、`review_fga_sync` |

### 课程与评课

| 表 | 用途 |
|----|------|
| `departments` | 院系 |
| `terms` | 学期 |
| `course_categories` | 课程分类 |
| `courses` | 课程 |
| `teachers` | 教师 |
| `rating_dimensions` | 评分维度 |
| `reviews` | 评课（`status` 支持 `published / hidden / deleted / pending_review`；`content_flag` 记录 `warn / review / cleared` 审核状态） |
| `review_votes` | 投票 |
| `review_reports` | 举报 |
| `review_replies` | 回复（`status` 同步支持 `pending_review`） |
| `course_favorites` | 收藏 |
| `review_drafts` | 草稿 |
| `course_rating_stats` | 课程评分统计（按课程+学期+维度聚合） |
| `teacher_rating_stats` | 教师评分统计（按教师+学期+维度聚合） |
| `sensitive_words` | 敏感词（分类 + 级别：block/warn/review） |

### 物化视图

| 名称 | 用途 |
|------|------|
| `mv_teacher_public_stats` | 教师公开列表聚合视图，缓存教师评分 / 课程数 / 院系名称，避免每次列表请求做全表聚合 |

### 通知与审计

| 表 | 用途 |
|----|------|
| `notifications` | 通知（`user_id` / `body` / `source_module` / `source_id` / `source_url` 结构，含 `payload` JSONB 扩展字段） |
| `notification_preferences` | 通知偏好（用户按通知类型开关，复合主键 `user_id` + `type`） |
| `audit_events` | 统一审计事件；管理员操作通过 `category = 'admin_operation'` 收口 |

## 关键 schema 备注

- `school_configs.school_id` 已从早期 `VARCHAR(10)` 统一为 `BIGINT`，并通过 `schools.id` 做权威外键。
- `user_profiles.school_id` 同步使用 `BIGINT`，不再直接挂在旧字符串主键上。
- `reviews.content_flag` / `content_flag_cleared_at` / `content_flag_cleared_by` 用于内容审核流水线。
- `reviews.status` 与 `review_replies.status` 已支持 `pending_review`，表示进入人工审核队列。
- `domain_event_outbox` 采用 `stream + dedupe_key` 唯一键，以及 `pending / processing / completed / failed` 状态机；后台 worker 只消费所属 stream，主事务不直连外部系统。

## 搜索索引

- `pg_trgm` 扩展已启用，`courses.name`、`courses.code`、`teachers.name` 上建有 GIN trigram 索引（见 migration `000006_search_trgm.up.sql`）

## 已知限制

- 证件照和资源文件都落在对象存储，数据库只保存对象 key / 业务引用

## 关联文档

- [product-specs/auth-sso.md](../product-specs/auth-sso.md)
- [product-specs/course-review.md](../product-specs/course-review.md)
- [product-specs/user-system.md](../product-specs/user-system.md)
- [product-specs/notification.md](../product-specs/notification.md)
