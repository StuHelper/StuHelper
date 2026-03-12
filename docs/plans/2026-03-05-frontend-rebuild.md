# StuHelper 前端完整实施计划

> **状态更新（2026-03-06）**: 这份文档是重构实施计划，不再代表当前代码的精确状态。当前事实来源请以 `docs/architecture/frontend.md`、`docs/guides/frontend-development.md` 和 `clients/web/src/router/index.ts` 为准。

> **For Claude:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development to implement this plan task-by-task.

**目标**: 基于 Glassmorphism 设计系统，从零开始构建 StuHelper 评课社区前端（Web 版本）

**架构**: Vue 3.5 + Vite 6 + TypeScript 5.7 + Tailwind CSS v4 + Glassmorphism 设计系统。当前代码已落地为 `clients/web + clients/shared + clients/uniappx` 的 Monorepo 结构。

**技术栈**:

- 框架: Vue 3.5.13, Vite 6.0, TypeScript 5.7
- 样式: Tailwind CSS v4.0, Glassmorphism
- 动画: @vueuse/motion 2.2+, GSAP 3.12+
- 状态: Pinia 2.2+ (最小化), Vue Router 4.5+, `openapi-fetch` + 浏览器适配层
- 增强: Radix Vue, unplugin-auto-import, unplugin-vue-components
- 工具: Storybook 8.4+, Vitest 2.x, ESLint 9.x, Prettier 3.x

---

## 阶段一：项目初始化与设计系统

### 任务 1: 更新项目配置和依赖

**文件**:

- 修改: `clients/web/package.json`
- 修改: `clients/web/vite.config.ts`
- 修改: `clients/web/tsconfig.json`

**步骤 1: 更新 package.json**

在 `clients/web/package.json` 中添加最新依赖。

**步骤 2: 安装依赖**

运行: `cd clients/web && pnpm install`

**步骤 3: 更新 Vite 配置**

添加 Tailwind v4、自动导入插件。

**步骤 4: 验证**

运行: `pnpm type-check`

**步骤 5: 提交**

```bash
git add clients/web/package.json clients/web/vite.config.ts clients/web/tsconfig.json
git commit -m "chore(web): update to latest dependencies"
```

---

### 任务 2: 创建设计系统目录结构

**文件**:

- 创建: `clients/web/src/design-system/tokens.ts`
- 创建: `clients/web/src/design-system/index.ts`

**步骤 1: 创建 tokens.ts**

```typescript
// clients/web/src/design-system/tokens.ts
export const glassEffects = {
	card: {
		background: "rgba(255, 255, 255, 0.1)",
		backdropFilter: "blur(10px) saturate(180%)",
		border: "1px solid rgba(255, 255, 255, 0.2)",
		boxShadow: "0 8px 32px 0 rgba(31, 38, 135, 0.15)",
	},
	navbar: {
		background: "rgba(255, 255, 255, 0.8)",
		backdropFilter: "blur(20px) saturate(180%)",
	},
	modal: {
		background: "rgba(255, 255, 255, 0.95)",
		backdropFilter: "blur(30px) saturate(200%)",
	},
} as const;

export const colors = {
	primary: "#ec4899",
	secondary: "#3b82f6",
} as const;

export const animations = {
	easing: {
		smooth: "cubic-bezier(0.4, 0.0, 0.2, 1)",
		bounce: "cubic-bezier(0.68, -0.55, 0.265, 1.55)",
	},
	duration: {
		fast: 150,
		normal: 300,
		slow: 500,
		page: 600,
	},
} as const;
```

**步骤 2: 创建 index.ts**

```typescript
// clients/web/src/design-system/index.ts
export * from "./tokens";
```

**步骤 3: 提交**

```bash
git add clients/web/src/design-system/
git commit -m "feat(web): add Glassmorphism design tokens"
```

---

### 任务 3: 配置 Tailwind CSS v4

**文件**:

- 创建: `clients/web/src/styles/main.css`
- 修改: `clients/web/src/main.ts`

**步骤 1: 创建 main.css**

```css
/* clients/web/src/styles/main.css */
@import "tailwindcss";

@theme {
	--color-primary: #ec4899;
	--color-secondary: #3b82f6;

	--font-sans: system-ui, -apple-system, sans-serif;
}
```

**步骤 2: 在 main.ts 导入**

修改 `clients/web/src/main.ts`，添加 `import './styles/main.css'`

**步骤 3: 提交**

```bash
git add clients/web/src/styles/ clients/web/src/main.ts
git commit -m "feat(web): configure Tailwind CSS v4"
```

---

## 阶段二：基础 UI 组件库

### 任务 4: 创建 Card 组件（Glassmorphism 核心）

**文件**:

- 创建: `clients/web/src/components/ui/Card.vue`

**步骤 1: 创建 Card 组件**

```vue
<script setup lang="ts">
interface Props {
	variant?: "card" | "navbar" | "modal";
}

const props = withDefaults(defineProps<Props>(), {
	variant: "card",
});
</script>

<template>
	<div :class="['glass-card', `glass-${variant}`]">
		<slot />
	</div>
</template>

<style scoped>
.glass-card {
	border-radius: 1rem;
}

.glass-card {
	background: rgba(255, 255, 255, 0.1);
	backdrop-filter: blur(10px) saturate(180%);
	border: 1px solid rgba(255, 255, 255, 0.2);
	box-shadow: 0 8px 32px 0 rgba(31, 38, 135, 0.15);
}

.glass-navbar {
	background: rgba(255, 255, 255, 0.8);
	backdrop-filter: blur(20px) saturate(180%);
}

.glass-modal {
	background: rgba(255, 255, 255, 0.95);
	backdrop-filter: blur(30px) saturate(200%);
}
</style>
```

**步骤 2: 提交**

```bash
git add clients/web/src/components/ui/Card.vue
git commit -m "feat(web): add Glassmorphism Card component"
```

---

### 任务 5: 创建 Button 组件

**文件**:

- 创建: `clients/web/src/components/ui/Button.vue`

**步骤 1: 创建 Button 组件**

```vue
<script setup lang="ts">
interface Props {
	variant?: "primary" | "secondary" | "ghost";
	size?: "sm" | "md" | "lg";
}

withDefaults(defineProps<Props>(), {
	variant: "primary",
	size: "md",
});
</script>

<template>
	<button
		:class="[
			'btn',
			`btn-${variant}`,
			`btn-${size}`,
			'transition-all duration-300',
		]"
	>
		<slot />
	</button>
</template>

<style scoped>
.btn {
	border-radius: 0.5rem;
	font-weight: 500;
	cursor: pointer;
}

.btn-primary {
	background: var(--color-primary);
	color: white;
}

.btn-secondary {
	background: var(--color-secondary);
	color: white;
}

.btn-ghost {
	background: transparent;
	border: 1px solid currentColor;
}

.btn-sm {
	padding: 0.5rem 1rem;
	font-size: 0.875rem;
}
.btn-md {
	padding: 0.75rem 1.5rem;
	font-size: 1rem;
}
.btn-lg {
	padding: 1rem 2rem;
	font-size: 1.125rem;
}
</style>
```

**步骤 2: 提交**

```bash
git add clients/web/src/components/ui/Button.vue
git commit -m "feat(web): add Button component"
```

---

### 任务 6: 创建动画组件包装器

**文件**:

- 创建: `clients/web/src/components/animated/FadeIn.vue`

**步骤 1: 创建 FadeIn 组件**

```vue
<script setup lang="ts">
import { useMotion } from "@vueuse/motion";
import { ref } from "vue";

interface Props {
	delay?: number;
}

const props = withDefaults(defineProps<Props>(), {
	delay: 0,
});

const target = ref();

useMotion(target, {
	initial: { opacity: 0, y: 20 },
	enter: {
		opacity: 1,
		y: 0,
		transition: {
			delay: props.delay,
			duration: 300,
		},
	},
});
</script>

<template>
	<div ref="target">
		<slot />
	</div>
</template>
```

**步骤 2: 提交**

```bash
git add clients/web/src/components/animated/FadeIn.vue
git commit -m "feat(web): add FadeIn animation component"
```

---

## 阶段三：页面实现

### 任务 7: 创建课程列表页

**文件**:

- 创建: `clients/web/src/modules/course/views/CourseListPage.vue`
- 创建: `clients/web/src/components/business/CourseCard.vue`

**步骤 1: 创建 CourseCard 组件**

```vue
<script setup lang="ts">
import Card from "@/components/ui/Card.vue";
import type { Course } from "@stuhelper/shared/types";

interface Props {
	course: Course;
}

defineProps<Props>();
</script>

<template>
	<Card class="p-6 hover:scale-105 transition-transform cursor-pointer">
		<h3 class="text-xl font-bold mb-2">{{ course.name }}</h3>
		<p class="text-gray-600">{{ course.department }}</p>
	</Card>
</template>
```

**步骤 2: 创建 CourseListPage**

```vue
<script setup lang="ts">
import { ref, onMounted } from "vue";
import CourseCard from "@/components/business/CourseCard.vue";
import FadeIn from "@/components/animated/FadeIn.vue";

const courses = ref([]);
const loading = ref(true);

onMounted(async () => {
	// TODO: 从 API 加载课程数据
	loading.value = false;
});
</script>

<template>
	<div class="container mx-auto p-8">
		<h1 class="text-4xl font-bold mb-8">课程列表</h1>
		<div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-6">
			<FadeIn v-for="(course, i) in courses" :key="course.id" :delay="i * 50">
				<CourseCard :course="course" />
			</FadeIn>
		</div>
	</div>
</template>
```

**步骤 3: 添加路由**

在 `clients/web/src/router/index.ts` 添加路由。

**步骤 4: 提交**

```bash
git add clients/web/src/modules/course/ clients/web/src/components/business/
git commit -m "feat(web): add course list page"
```

---

### 任务 8: 创建用户中心页面

**文件**:

- 创建: `clients/web/src/modules/user/views/UserCenterPage.vue`

**步骤 1: 创建 UserCenterPage**

```vue
<script setup lang="ts">
import { useAuthStore } from "@/stores/auth";
import Card from "@/components/ui/Card.vue";

const authStore = useAuthStore();
</script>

<template>
	<div class="container mx-auto p-8">
		<Card class="p-8">
			<h1 class="text-3xl font-bold mb-4">个人中心</h1>
			<p v-if="authStore.isAuthenticated">已登录</p>
			<p v-else>未登录</p>
		</Card>
	</div>
</template>
```

**步骤 2: 添加路由**

在路由配置中添加 `/user` 路径。

**步骤 3: 提交**

```bash
git add clients/web/src/modules/user/
git commit -m "feat(web): add user center page"
```

---

## 阶段四：功能完善

### 任务 9: 集成 API 调用

**文件**:

- 创建: `clients/web/src/api/course.ts`

**步骤 1: 创建 course API**

```typescript
// clients/web/src/api/course.ts
import { httpClient } from "./client";
import type { Course, PaginatedResponse } from "@stuhelper/shared/types";

export const courseApi = {
	getCourses: (params: { page: number; pageSize: number }) =>
		httpClient.get<PaginatedResponse<Course>>("/api/v1/courses", params),

	getCourse: (id: number) => httpClient.get<Course>(`/api/v1/courses/${id}`),
};
```

**步骤 2: 在页面中使用**

更新 CourseListPage 使用 courseApi。

**步骤 3: 提交**

```bash
git add clients/web/src/api/
git commit -m "feat(web): integrate course API"
```

---

### 任务 10: 添加错误处理

**文件**:

- 修改: `clients/web/src/api/client.ts`

**步骤 1: 添加错误拦截器**

在浏览器 API 客户端中处理 401，并通过 Refresh Token 续期。

**步骤 2: 提交**

```bash
git add clients/web/src/api/client.ts
git commit -m "feat(web): add error handling"
```

---

## 验证和测试

### 任务 11: 运行开发服务器

**步骤 1: 启动服务器**

运行: `cd clients/web && pnpm dev`

预期: 服务器在 http://localhost:5173 启动

**步骤 2: 验证页面**

访问主页、课程列表、用户中心，确认页面正常显示。

**步骤 3: 验证构建**

运行: `pnpm build`

预期: 构建成功，无错误

---

## 后续开发

基础实现完成后，继续开发：

1. **更多 UI 组件**: Input, Modal, Drawer 等
2. **业务组件**: ReviewCard, TeacherCard 等
3. **完整页面**: 课程详情、教师详情、评课编辑等
4. **管理后台**: 集成 Vue-vben-admin
5. **测试**: Vitest 单元测试、Playwright E2E 测试
