# 前端组件设计

本文档记录评课社区当前已经落地的组件组织方式，而不是旧版目录结构。

## 当前技术栈

| 技术         | 版本线 |
| ------------ | ------ |
| Vue          | 3.5    |
| TypeScript   | 5.7+   |
| Vite         | 6      |
| Element Plus | 2.13+  |
| ECharts      | 6      |
| Pinia        | 2.2+   |
| Storybook    | 8      |
| Vitest       | 4      |

## 组件目录结构

```text
clients/web/src/
├── components/
│   ├── common/                    # 通用组件
│   ├── layout/                    # 顶层壳与导航
│   └── business/
│       └── review/                # 评课域业务组件
├── modules/
│   ├── course/views/              # 课程列表、教学门户
│   ├── review/views/              # 测评首页、课程详情、教师主页、发布页
│   ├── user/views/                # 用户中心、通知中心
│   └── admin/views/               # 举报、日志、敏感词、管理后台
└── stores/
    ├── courseReview.ts
    ├── draft.ts
    └── notification.ts
```

## 评课域核心组件

| 组件                    | 作用                                     |
| ----------------------- | ---------------------------------------- |
| `CourseCard.vue`        | 课程卡片，列表入口                       |
| `CourseListItem.vue`    | 课程列表项，适用于更紧凑的列表布局       |
| `ReviewCard.vue`        | 单条测评展示，支持投票、回复、举报等交互 |
| `ReviewForm.vue`        | 发布和编辑测评的表单主体                 |
| `ReviewDialog.vue`      | 全局快捷发布弹窗，仍用于非课程上下文入口 |
| `RatingGroup.vue`       | 动态评分维度录入                         |
| `CourseRatingChart.vue` | 课程评分统计图                           |
| `TeacherStatsCard.vue`  | 教师评分摘要                             |

## 页面组件

| 页面                     | 路径                                   | 说明                       |
| ------------------------ | -------------------------------------- | -------------------------- |
| `ReviewPage.vue`         | `/review`                              | 评课社区首页               |
| `CourseDetailPage.vue`   | `/courses/:id`、`/courses/:id/reviews` | 课程概览与测评列表共享页面 |
| `PostReviewPage.vue`     | `/courses/:id/reviews/post`            | 专门的发布测评页           |
| `TeacherProfilePage.vue` | `/teachers/:id`                        | 教师主页                   |

## 通用组件与文档化

- 通用组件位于 `clients/web/src/components/common/`
- Storybook 配置位于 `clients/web/.storybook/`
- 当前已提供组件示例：`clients/web/src/components/common/EmptyState.stories.ts`

新增通用组件时，优先同时补充：

1. Storybook story
2. 必要的 Vitest 单测
3. 在业务页面中的真实使用场景
