# 教学中心模块设计文档

> 域名：course.stuhelper.com

## 模块定位

教学中心是 StuHelper 的核心功能模块，为学生提供一站式教学相关服务。

## 子模块规划

| 子模块 | 路由 | 状态 | 说明 |
|--------|------|------|------|
| 评课社区 | `/review` | 🟡 开发中 | 匿名课程评价平台 |
| 资料共享 | `/resource` | 🔴 规划中 | 课程资料、往年题共享 |
| SPOC | `/spoc` | 🔴 规划中 | 校内在线课程聚合 |

## 评课社区开发进度

### 已完成功能

- [x] SSO 单点登录（Casdoor OAuth2）
- [x] 评分维度配置 API
- [x] 课程/院系/测评基础 API
- [x] 动态评分系统（1-5星，JSON存储）
- [x] 雷达图评分统计 API
- [x] 前端评分输入组件
- [x] 前端雷达图组件（ECharts）
- [x] 课程详情页集成

### 待开发功能

- [ ] 数据库实际连接
- [ ] 课程数据导入
- [ ] 用户认证集成
- [ ] 测评发布流程完善
- [ ] 搜索功能优化

## 文档索引

| 文档 | 说明 |
|------|------|
| [01_data_model.md](./01_data_model.md) | 数据模型设计 |
| [02_api_design.md](./02_api_design.md) | API 接口设计 |
| [03_components.md](./03_components.md) | 前端组件设计 |
| [04_routes.md](./04_routes.md) | 页面路由设计 |
| [05_ui_spec.md](./05_ui_spec.md) | UI 设计规范 |
| [06_security.md](./06_security.md) | 安全与风控设计 |
| [07_rating_dimensions.md](./07_rating_dimensions.md) | 评分维度系统设计 |

## 技术栈

### 前端
- **框架**: Vue3 + Element Plus (H5)
- **构建**: Vite

### 后端
- **语言**: Go (Gin)
- **数据库**: PostgreSQL + Redis
- **架构**: 三层架构（Handler → Service → Repository）

## 后端架构

评课社区后端采用分层架构设计，详见 [分层架构设计](../../architecture/layered-architecture.md)。

### 代码结构

```
server/internal/modules/course/review/
├── handler.go      # HTTP 处理器
├── service.go      # 业务逻辑层
├── repository.go   # 数据访问层
├── model.go        # 数据模型
├── cache.go        # 缓存处理
├── rating.go       # 评分相关处理
└── utils.go        # 工具函数
```

### 各层职责

| 层级 | 文件 | 职责 |
|------|------|------|
| Handler | handler.go, review.go, rating.go | HTTP 请求处理、缓存、响应格式化 |
| Service | service.go | 业务逻辑、数据验证、事务管理 |
| Repository | repository.go | SQL 查询、数据库操作 |
