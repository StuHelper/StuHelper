# 航小伴授权模型

这份文档回答：航小伴应该如何实现你提出的那种复杂、细粒度、企业级授权。

## 一句话结论

航小伴不应该继续停留在“简单 RBAC 表 + `isAdmin` 后台门禁”。

推荐的最终模型是：

- **RBAC**：应用级和模块级管理员
- **ReBAC**：课程、分类、内容等资源关系
- **ABAC**：学校、身份类型、学生认证、实名认证等属性规则
- **内容策略**：完整显示、部分显示、带标签显示

## 1. 角色层（RBAC）

角色只负责粗粒度能力，不负责高基数资源关系。

建议保留的角色：

| 角色 | 作用 |
| --- | --- |
| `hangxiaoban_admin` | 航小伴全局管理员 |
| `review_admin` | 评课社区全局管理员 |
| `resource_admin` | 资源共享全局管理员 |

如果多个管理员角色完全等价，不要重复建多个名字不同但权限一样的角色。

## 2. 关系层（ReBAC）

关系层负责高基数、资源级的授权。

建议建模的关系包括：

| 关系 | 含义 |
| --- | --- |
| `course#manager` | 某门课程全内容管理员 |
| `course#intro_editor` | 某门课程课程简介维护者 |
| `course#resource_editor` | 某门课程资料维护者 |
| `course#review_moderator` | 某门课程评价维护者 |
| `resource_category#manager` | 某分类资源管理员 |
| `review#owner` | 某条评课的发布者 |
| `resource#owner` | 某条资源的发布者 |
| `course#teacher` | 某门课程的任课教师 |

## 3. 属性层（ABAC）

属性层负责业务条件。

建议参与授权判断的属性包括：

| 属性 | 用途 |
| --- | --- |
| `schoolID` | 学校范围判断 |
| `actorType` | 学生 / 老师判断 |
| `studentVerified` | 学生认证状态 |
| `identityVerified` | 实名认证状态 |

## 4. 典型业务场景如何实现

### 4.1 StuHelper 全生态管理员

这是平台级概念，不属于航小伴业务角色。  
应在 Casdoor 平台侧定义，不应复用成航小伴业务管理员。

### 4.2 航小伴应用管理员

使用 `hangxiaoban_admin`。

### 4.3 评课社区整个模块管理员

使用 `review_admin`。

### 4.4 某门课程的全内容管理员

使用 `course#manager`。

一个人可以关联多门课；一门课也可以有多个人。

### 4.5 某门课程的指定内容管理员

使用更细的关系：

- `course#intro_editor`
- `course#resource_editor`
- `course#review_moderator`

### 4.6 资源共享模块管理员

使用 `resource_admin`。

### 4.7 资源共享部分内容管理员

按业务对象建关系：

- `resource_category#manager`
- 或 `course#resource_editor`

### 4.8 内容发布者维护自己的内容

使用 owner 关系：

- `review#owner`
- `resource#owner`

### 4.9 按学校、认证状态决定查看和发布

这类规则不能只靠角色解决。

推荐按属性判断：

- 未登录：不允许查看完整信息
- 已登录但未学生认证 `10006`：只能查看部分信息
- 已学生认证 `10006` 且已实名认证：允许完整查看和发布

如果产品策略改为“允许未认证用户发布，但要打标签”，则策略输出应变成：

- allow with warning label

而不是简单 `allow / deny`。

### 4.10 任课教师默认维护自己课程资料和简介

从课程表数据库推导 `course#teacher` 关系，  
再把该关系映射为：

- 课程简介编辑权
- 课程资料编辑权

## 5. 为什么推荐 OpenFGA / SpiceDB

当你需要同时支持这些能力时，简单 RBAC 表会很快失控：

- 多门课程、多分类、多内容的 many-to-many 委派
- owner、teacher、course-manager 多种关系叠加
- 资源继承
- 审计与回溯

这就是 Zanzibar 风格授权系统最擅长的场景。

建议长期路线：

- Casdoor 负责身份
- 航小伴负责业务事实
- OpenFGA / SpiceDB 负责关系授权
- 航小伴后端再叠加属性规则和内容整形

## 6. 反模式

### 不要这样做

- 给每门课程在 Casdoor 里建一个全局角色
- 把课程管理员、分类管理员都塞进 SSO 平台组
- 用 `isAdmin` 控制航小伴后台
- 用纯前端逻辑做“部分可见”

### 应该这样做

- 角色负责粗粒度能力
- 关系负责资源级授权
- 属性负责条件判断
- 后端统一做最终授权和内容输出决策
