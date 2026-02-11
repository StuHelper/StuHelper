# 前端组件设计

本文档定义评课社区模块的前端组件结构和设计规范。

## 技术栈

| 技术 | 版本 |
|------|------|
| Vue | 3.4 |
| TypeScript | 5.3 |
| Vite | 5.0 |
| Element Plus | 2.4 |
| ECharts | 5.5 |
| Pinia | 2.1 |

## 组件目录结构

```
clients/web/src/
├── components/
│   ├── common/                 # 通用组件
│   │   ├── EmptyState.vue
│   │   ├── SkeletonCard.vue
│   │   └── InfiniteScroll.vue
│   └── review/                 # 评课相关组件
│       ├── CourseCard.vue
│       ├── ReviewCard.vue
│       ├── RatingGroup.vue
│       ├── SearchBar.vue
│       ├── CourseRatingChart.vue
│       ├── PostReviewDialog.vue
│       ├── ReplyList.vue
│       ├── FavoriteButton.vue
│       ├── HotCourseCard.vue
│       └── TeacherStatsCard.vue
├── views/review/               # 评课页面
│   ├── IndexPage.vue
│   ├── CourseListPage.vue
│   ├── CourseDetailPage.vue
│   ├── PostReviewPage.vue
│   └── TeacherDetailPage.vue
└── stores/
    ├── courseReview.ts
    └── draft.ts
```

## 核心组件

### ReviewCard 测评卡片

显示单条测评信息，支持点赞、回复、举报等交互。

| Props | 类型 | 说明 |
|-------|------|------|
| review | Review | 测评数据 |
| showCourse | boolean | 是否显示课程名 |

### RatingGroup 评分组

动态评分维度选择器，支持 1-5 星评分。

| Props | 类型 | 说明 |
|-------|------|------|
| modelValue | number | 当前值 (1-5) |
| label | string | 维度标签 |

### CourseRatingChart 课程评分图表

使用 ECharts 展示课程评分雷达图。

| Props | 类型 | 说明 |
|-------|------|------|
| courseId | number | 课程 ID |

### FavoriteButton 收藏按钮

课程收藏按钮，带心跳动画效果。

| Props | 类型 | 说明 |
|-------|------|------|
| courseId | number | 课程 ID |

## 通用组件

### EmptyState 空状态

| Props | 类型 | 说明 |
|-------|------|------|
| icon | string | 图标类型 |
| title | string | 标题 |
| description | string | 描述文字 |

### SkeletonCard 骨架屏

| Props | 类型 | 说明 |
|-------|------|------|
| lines | number | 行数 |
| avatar | boolean | 是否显示头像 |

### InfiniteScroll 无限滚动

| Props | 类型 | 说明 |
|-------|------|------|
| loading | boolean | 加载状态 |
| hasMore | boolean | 是否有更多 |
