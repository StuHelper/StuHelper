# 页面路由设计

本文档定义评课社区模块的页面路由结构。

## 路由表

| 路由 | 页面 | 说明 |
|------|------|------|
| `/course-review` | 首页 | 模块入口 |
| `/course-review/courses` | 课程列表 | 按院系浏览 |
| `/course-review/courses/:id` | 课程详情 | 评分和测评 |
| `/course-review/reviews/latest` | 最新测评 | 时间线 |
| `/course-review/reviews/post` | 发布测评 | 表单页 |
| `/course-review/search` | 高级搜索 | 多条件 |
| `/course-review/about` | 关于 | 说明页 |
| `/course-review/faq` | FAQ | 常见问答 |

## 页面详情

### 1. 首页

- **路由**: `/course-review`
- **功能**: 搜索入口、统计展示、随机测评

### 2. 课程列表

- **路由**: `/course-review/courses`
- **参数**: `?category=elective`
- **功能**: 分类标签、院系折叠列表

### 3. 课程详情

- **路由**: `/course-review/courses/:id`
- **参数**: `?teacher=xxx`
- **功能**: 评分统计、教师筛选、测评列表

### 4. 发布测评

- **路由**: `/course-review/reviews/post`
- **参数**: `?course_id=xxx`
- **功能**: 课程选择、四维评分、内容填写
