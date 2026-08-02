---
type: product-spec
audience: product, backend-dev
status: current
authoritative-source: server/api/openapi.yaml
last-verified: 2026-08-02
---

# 教务数据接入与展示

## 目标

定义 `academics` 域的产品边界与最小交付能力。该域负责把外部教务系统的数据导入到 StuHelper，并提供标准化后的查询与展示 API。

## 边界

### 包含

- 导入外部教务数据
- 标准化为本地读模型
- 学期、开课、教师、时间地点、成员关系的展示
- “我的课程” 与 “我的课表”
- 导入源与导入作业状态

### 不包含

- 选课、退课、排课、调课写侧
- 完整教务后台
- 真实学校连接器的模拟登录与抓取实现

## 核心对象

- `academic_sources`
- `academic_import_jobs`
- `academic_terms`
- `academic_courses`
- `academic_teachers`
- `academic_offerings`
- `academic_schedules`
- `academic_memberships`

## 连接器策略

当前阶段只要求：

- provider 接口
- provider 注册表
- 测试用 fixture / stub provider
- 健康检查

真实学校连接器后续补充，不阻塞当前读模型闭环。

连接器返回完整 snapshot，Repository 在一个事务内完成原子替换。terms、courses、teachers、
offerings、offering-teacher relations、schedules 与 memberships 均通过 PostgreSQL
`jsonb_to_recordset` 做 set-based upsert/insert；数据库往返次数不得随 snapshot 行数线性增长。
任何实体或关系写入失败都回滚整个 snapshot，成功后才裁剪该 source 的旧读模型行并把 import job
标记为成功。真实学校连接器上线前必须用接近生产规模的 snapshot 回归该事务预算，不能恢复逐行
SQL 循环，也不能通过无限放大 `DB_QUERY_TIMEOUT` 掩盖导入算法退化。

## 最小 API

- 管理员查看导入源
- 管理员查看导入作业
- 管理员触发一次导入
  仅允许对 `enabled = true` 的导入源执行
- 查询学期列表
- 查询开课列表
- 查询开课详情
- 查询我的课程
- 查询我的课表

## 查询约束

- 列表接口必须支持分页
- 列表接口必须支持稳定排序
- 支持学期、院系、课程名、教师名等过滤

## 权限

- 管理导入：后台 capability
- 我的课程 / 我的课表：登录用户本人
- 公共开课查询：按公开读模型开放
