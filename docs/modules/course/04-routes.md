# 路由设计

本文档定义评课社区当前使用中的前端路由结构。

## 路由原则

1. 对象优先：先写课程或教师，再写该对象上的动作。
2. 详情与子资源分离：课程概览和课程测评列表共用一个对象根路径。
3. 发布动作保留独立页面：`/courses/:id/reviews/post` 明确表达“给这门课发测评”。
4. 旧路由只保留兼容重定向，不再作为主入口。

## 当前主路由

```typescript
const routes = [
	{
		path: "/review",
		name: "review",
		component: () => import("@/modules/review/views/ReviewPage.vue"),
	},
	{
		path: "/courses",
		name: "course-list",
		component: () => import("@/modules/course/views/CourseListPage.vue"),
	},
	{
		path: "/courses/:id",
		name: "course-detail",
		component: () => import("@/modules/review/views/CourseDetailPage.vue"),
	},
	{
		path: "/courses/:id/reviews",
		name: "course-reviews",
		component: () => import("@/modules/review/views/CourseDetailPage.vue"),
	},
	{
		path: "/courses/:id/reviews/post",
		name: "course-review-post",
		component: () => import("@/modules/review/views/PostReviewPage.vue"),
		meta: { title: "发布测评", requiresAuth: true },
	},
	{
		path: "/teachers/:id",
		name: "teacher-profile",
		component: () => import("@/modules/review/views/TeacherProfilePage.vue"),
	},
];
```

## 路由说明

| 路径                        | 名称                 | 认证 | 说明             |
| --------------------------- | -------------------- | ---- | ---------------- |
| `/review`                   | `review`             | 否   | 评课社区首页     |
| `/courses`                  | `course-list`        | 否   | 课程列表页       |
| `/courses/:id`              | `course-detail`      | 否   | 课程概览页       |
| `/courses/:id/reviews`      | `course-reviews`     | 否   | 课程测评列表     |
| `/courses/:id/reviews/post` | `course-review-post` | 是   | 专门的发布测评页 |
| `/teachers/:id`             | `teacher-profile`    | 否   | 教师详情页       |

## 兼容重定向

旧链接仍会被重定向到新地址：

- `/review/courses` → `/courses`
- `/review/courses/:id` → `/courses/:id/reviews`
- `/review/teachers/:id` → `/teachers/:id`
- `/courses/:id/review` → `/courses/:id/reviews`

## 为什么不再使用 `/review/post`

`/review/post` 缺少课程上下文，带来几个问题：

- 进入页面后还要补课程 ID
- 登录回跳和草稿恢复都需要额外参数
- 链接分享后语义不完整

因此当前规范路径是 `/courses/:id/reviews/post`。这既符合 REST 风格的对象层级，也和现有页面逻辑一致。
