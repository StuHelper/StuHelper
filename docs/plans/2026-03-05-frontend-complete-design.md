# StuHelper 前端完整实现设计方案

> **状态更新（2026-03-06）**: 这份文档保留为当时的设计方案。当前已经落地的真实实现请优先查阅 `docs/architecture/frontend.md` 和 `docs/guides/frontend-development.md`。代码现状已经包含 `clients/web + clients/shared + clients/uniappx`、`openapi-fetch`、Storybook、Vitest、Playwright、`/courses/:id/reviews/post` 发布页，以及基于 Refresh Token 的会话续期。

> **设计日期**: 2026-03-05
> **设计目标**: 基于现代化技术栈，完整实现 StuHelper 评课社区前端（方案文档，当前代码已部分超出该范围）

---

## 一、设计概述

### 核心需求

- **功能范围**: 全部功能（评课核心、用户中心、通知系统、管理后台）
- **平台范围**: Web 为主，当前代码已补充 `clients/uniappx`
- **设计风格**: Glassmorphism（玻璃态）现代化设计
- **动画效果**: 流畅的交互动画和页面切换
- **实现方式**: 参考现有实现，完全重写

### 设计原则

1. **现代化优先** - 使用最新技术栈和设计理念
2. **动画优先** - 每个交互都有流畅动画
3. **组件化** - 构建完整的设计系统和组件库
4. **可扩展** - 为未来模块预留扩展空间
5. **类型安全** - 完整的 TypeScript 类型定义

---

## 二、技术架构

### 核心技术栈（最新版本）

**框架与构建**：

- Vue 3.5.13（Composition API + `<script setup>`）
- Vite 6.0（最新版）
- TypeScript 5.7（严格模式）
- pnpm 9.x（workspace 管理）

**UI 与样式**：

- Tailwind CSS v4.0（最新版）
- Glassmorphism 自定义设计系统
- CSS Variables（动态主题）

**动画库**：

- @vueuse/motion 2.2+（页面切换、组件动画、滚动触发）
- GSAP 3.12+（复杂动画序列、数字滚动、SVG 动画）

**状态与数据**：

- Pinia 2.2+（最小化使用）
- Vue Router 4.5+
- `openapi-fetch` + 浏览器适配层
- @vueuse/core 11.x

**开发工具**：

- Storybook 8.4+（组件开发与文档）
- Vitest 2.x（单元测试）
- Playwright 1.49+（E2E 测试）
- ESLint 9.x + Prettier 3.x

**额外增强**：

- Radix Vue（无样式组件库，提供可访问性基础）
- unplugin-auto-import（自动导入 API）
- unplugin-vue-components（自动导入组件）

### 项目结构

```
clients/web/
├── src/
│   ├── components/          # 组件库
│   │   ├── ui/             # 基础 UI 组件（Glassmorphism）
│   │   ├── animated/       # 动画组件
│   │   └── business/       # 业务组件
│   ├── modules/            # 功能模块
│   │   ├── home/           # 主页
│   │   ├── course/         # 课程模块
│   │   ├── review/         # 评课模块
│   │   ├── teacher/        # 教师模块
│   │   ├── user/           # 用户中心
│   │   ├── notification/   # 通知
│   │   └── admin/          # 管理后台（Vue-vben-admin）
│   ├── design-system/      # 设计系统
│   ├── composables/        # 组合式函数
│   ├── stores/             # Pinia stores
│   ├── api/                # API 调用
│   ├── router/             # 路由配置
│   └── styles/             # 全局样式
└── .storybook/             # Storybook 配置
```

---

## 三、Glassmorphism 设计系统

### 玻璃效果配置

```typescript
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
};
```

### 色彩系统

- **主色**: #ec4899（Pink-500）
- **辅助色**: #3b82f6（Blue-500）
- **评分色彩**: 渐变版本（优秀/良好/一般/较差）

### 动画配置

```typescript
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
};
```

### 响应式设计

**全屏幕响应式策略**：

- 基于屏幕宽度 + 横竖比组合查询
- 实时监听方向变化，立即调整布局
- 竖屏（手机）：单列布局
- 横屏（手机）：双列布局
- 平板/电脑：多列布局

---

## 四、组件库设计

### 基础 UI 组件（`components/ui/`）

**布局**: Container, Stack, Grid, Divider

**数据展示**: Card（玻璃态核心组件）, Badge, Avatar, Rating, Progress, Chart, Empty

**表单**: Input, Textarea, Select, Radio, Checkbox, Switch, Slider, DatePicker

**反馈**: Button（多种变体）, Toast, Modal, Drawer, Popover, Loading, Skeleton

**导航**: Navbar（玻璃态固定）, Sidebar, Tabs, Breadcrumb, Pagination

### 动画组件（`components/animated/`）

基于 @vueuse/motion 和 GSAP：

- FadeIn, SlideIn, ScaleIn
- StaggerList（列表交错动画）
- ParallaxSection（视差滚动）
- CountUp（数字滚动）
- RevealOnScroll（滚动触发）

### 业务组件（`components/business/`）

**课程相关**:

- CourseCard, CourseHomePage, CourseInfoCard
- CourseScheduleTable（开课情况表格）

**评价相关**:

- ReviewPage, ReviewCard, ReviewEditor
- RatingInput, ReplyList

**课程资源**:

- ResourcePage, ResourceCard, ResourceUploader

**教师相关**:

- TeacherProfilePage（参考北航设计）
  - 左侧：头像 + 点赞按钮
  - 右侧：详细信息 + 标签页
- TeacherCard, TeacherCourseList

**用户相关**:

- UserSettingsPage（参考主流社区设计）
  - 侧边导航 + 内容区域
  - 个人资料、账号安全、隐私设置、通知设置等

### 管理后台

**使用 Vue-vben-admin 框架**：

- 完整的权限系统（RBAC）
- 动态菜单管理（后台配置）
- 多标签页、主题切换
- 模块化架构，为未来模块预留空间

---

## 五、页面结构和路由

### 前台路由

```
/                           # 主页
/courses                    # 课程列表
/courses/:id                # 课程主页
/courses/:id/reviews        # 评课详情
/courses/:id/resources      # 课程资源
/teachers                   # 教师列表
/teachers/:id               # 教师主页
/user/reviews               # 我的评价
/user/favorites             # 我的收藏
/user/votes                 # 我的投票
/user/settings              # 个人设置
```

### 管理后台路由

```
/admin/dashboard            # 数据概览
/admin/course/*             # 课程中心模块
/admin/tools/*              # 工具箱模块（预留）
/admin/*                    # 管理后台（当前不包含社群模块）
/admin/notification/*       # 通知模块（预留）
/admin/system/*             # 系统管理
```

### 路由守卫

- 认证检查（requiresAuth）
- 管理员权限检查（requiresAdmin）
- Token 过期自动刷新（当前已落地）

---

## 六、数据流和状态管理

### 数据流设计

```
组件 → Composable → API 层 → `openapi-fetch` 适配层 → 后端
  ↓
本地状态（ref/reactive）
  ↓
UI 更新
```

### Pinia 最小化使用

**仅在以下场景使用**：

- 认证状态（stores/auth.ts）
- 收藏状态（stores/favorites.ts）
- 通知状态（stores/notifications.ts）

**不使用 Pinia**：

- 页面级数据 → Composable
- 组件通信 → props/emits
- 临时状态 → 本地 ref

### 数据缓存

使用 VueUse 的 useStorage 实现本地缓存。

---

## 七、开发流程

### 实施阶段

**第一阶段：设计系统基础**（1周）

- Glassmorphism 设计 tokens
- 基础动画组件封装
- 布局系统

**第二阶段：核心组件库**（1.5周）

- 基础 UI 组件
- 动画组件
- Storybook 文档

**第三阶段：页面组装**（2周）

- 课程模块页面
- 教师模块页面
- 用户中心页面
- 集成 Vue-vben-admin

**第四阶段：功能完善**（1周）

- API 对接
- 状态管理
- 错误处理
- 性能优化

### 开发规范

- 组件文件：PascalCase.vue
- Props：camelCase
- Events：on + PascalCase
- 使用 Storybook 开发组件
- 编写单元测试（Vitest）

---

## 八、技术亮点

1. **最新技术栈** - Vue 3.5 + Vite 6 + TypeScript 5.7 + Tailwind v4
2. **现代化设计** - Glassmorphism 玻璃态风格
3. **流畅动画** - @vueuse/motion + GSAP 混合使用
4. **完整设计系统** - 可复用的组件库
5. **企业级后台** - Vue-vben-admin 框架
6. **全屏幕响应式** - 横竖屏自适应
7. **类型安全** - 完整的 TypeScript 支持
8. **可扩展架构** - 模块化设计，易于扩展

---

## 九、后续扩展

### 预留模块

- 工具箱模块
- 社群模块
- 通知中心模块

### uni-app 实现

当前仓库已经存在 `clients/uniappx`，后续工作重点不再是“从零创建”，而是继续补齐页面与跨端适配。

---

**设计完成日期**: 2026-03-05
