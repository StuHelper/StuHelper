# 路由设计

本文档定义评课社区模块的前端路由结构。

## 路由配置

```typescript
const routes = [
  {
    path: '/review',
    name: 'review',
    component: () => import('@/views/review/IndexPage.vue'),
    meta: { title: '评课社区' }
  },
  {
    path: '/review/courses',
    name: 'review-courses',
    component: () => import('@/views/review/CourseListPage.vue'),
    meta: { title: '课程列表' }
  },
  {
    path: '/review/courses/:id',
    name: 'review-course-detail',
    component: () => import('@/views/review/CourseDetailPage.vue'),
    meta: { title: '课程详情' }
  },
  {
    path: '/review/post',
    name: 'review-post',
    component: () => import('@/views/review/PostReviewPage.vue'),
    meta: { title: '发布测评', requiresAuth: true }
  },
  {
    path: '/review/teachers/:id',
    name: 'review-teacher-detail',
    component: () => import('@/views/review/TeacherDetailPage.vue'),
    meta: { title: '教师详情' }
  }
]
```

## 路由说明

| 路径 | 名称 | 认证 | 说明 |
|------|------|------|------|
| `/review` | review | 否 | 评课社区首页 |
| `/review/courses` | review-courses | 否 | 课程列表页 |
| `/review/courses/:id` | review-course-detail | 否 | 课程详情页 |
| `/review/post` | review-post | 是 | 发布测评页 |
| `/review/teachers/:id` | review-teacher-detail | 否 | 教师详情页 |
