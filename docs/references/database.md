# 数据库设计

## 数据面

| 存储 | 用途 |
|------|------|
| PostgreSQL | 业务数据 |
| Redis | 会话、黑名单、限流、缓存、通知广播 |
| 对象存储（MinIO/S3） | 证件照片 |
| Zitadel | 身份平面 |
| OpenFGA | 资源关系授权 |

## 权威来源

1. `server/migrations/*.sql` — 唯一权威 schema
2. `server/scripts/init.sql` — 可读快照
3. `server/internal/modules/**/repository*.go` — 查询和事务

## 身份平面

Zitadel 管理账号、OIDC 会话、粗粒度角色。

应用侧保留 `users` 表作为 shadow user：业务外键锚点 + 最小用户画像缓存。

## 业务平面

PostgreSQL 存储：
- 课程、教师、院系、学期、分类
- 评课、回复、投票、举报、草稿、收藏
- 实名认证、学生认证、学校配置、系统配置
- 通知、操作日志

## 授权平面

能力由角色静态展开，不落本地 RBAC 表。资源级权限由 OpenFGA 增强。

## 表索引

### 用户系统

| 表 | 用途 |
|----|------|
| `users` | Shadow user、业务外键锚点 |
| `user_identities` | 实名认证（`doc_number_enc` 密文、`person_uid` HMAC、`doc_photo_*` 对象存储 key） |
| `user_profiles` | 学生认证档案 |
| `school_configs` | 学校认证配置 |
| `system_configs` | 全局配置 |

### 课程与评课

| 表 | 用途 |
|----|------|
| `departments` | 院系 |
| `terms` | 学期 |
| `course_categories` | 课程分类 |
| `courses` | 课程 |
| `teachers` | 教师 |
| `rating_dimensions` | 评分维度 |
| `reviews` | 评课 |
| `review_votes` | 投票 |
| `review_reports` | 举报 |
| `review_replies` | 回复 |
| `course_favorites` | 收藏 |
| `review_drafts` | 草稿 |

### 通知与审计

| 表 | 用途 |
|----|------|
| `notifications` | 通知（已迁移到 `user_id` / `body` / `source_module` 结构） |
| `admin_operation_logs` | 操作日志 |

## 已知限制

- 搜索仍用 `LIKE` / `LOWER(...) LIKE`，缺少 `pg_trgm` 索引
- 证件照存储在对象存储，数据库只保存 key

## 关联文档

- [product-specs/auth-sso.md](../product-specs/auth-sso.md)
- [product-specs/course-review.md](../product-specs/course-review.md)
- [product-specs/user-system.md](../product-specs/user-system.md)
- [product-specs/notification.md](../product-specs/notification.md)
