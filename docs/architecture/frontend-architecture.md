# 前端架构

## 设计背景

StuHelper 是一个生态，航小伴是其中一个 first-party 应用。本文档描述的是航小伴前端的整体架构，以及它如何接入 `sso.stuhelper.com`。

**核心需求：**
- 多模块：评课社区、资源共享、对象主页、用户中心等
- 跨平台：H5、微信小程序、App（iOS/Android）
- 统一身份：通过 `sso.stuhelper.com`（Casdoor）进行登录
- 应用授权：由航小伴后端返回 capabilities / effective permissions
- 统一 API：所有模块通过应用后端访问服务

**架构决策：**
- 代码组织：Monorepo 单一代码库
- 跨平台方案：uni-app 一套代码多端输出
- 认证：Cookie 会话，不在前端保存 access token
- 后台门禁：不再使用 `isAdmin` 作为应用业务后台总开关

---

## 整体架构

### 系统架构图

```
┌─────────────────────────────────────────────────────────┐
│                    stuhelper.com                        │
│                   (主页/导航页)                          │
└─────────────────────────────────────────────────────────┘
                            │
        ┌───────────────────┼───────────────────┐
        │                   │                   │
┌───────▼────────┐  ┌──────▼──────┐  ┌────────▼────────┐
│  /course/*     │  │  /tools/*   │  │ /community/*    │
│  评课模块       │  │  工具箱      │  │  社群模块        │
└────────────────┘  └─────────────┘  └─────────────────┘
        │                   │                   │
        └───────────────────┼───────────────────┘
                            │
                ┌───────────▼───────────┐
                │  api.stuhelper.com    │
                │    (统一 API)          │
                └───────────────────────┘
                            │
                ┌───────────▼───────────┐
                │  sso.stuhelper.com    │
                │  (Casdoor SSO)        │
                └───────────────────────┘
```

### 技术栈

| 层级 | 技术选型 | 说明 |
|------|----------|------|
| 框架 | uni-app (Vue 3) | 跨平台框架，一套代码多端输出 |
| 构建工具 | Vite | 快速构建和热更新 |
| 语言 | TypeScript | 类型安全 |
| 状态管理 | Pinia | 仅用于跨模块共享状态 |
| UI 框架 | uni-ui + Tailwind CSS v4 | 跨平台组件库 + 原子化 CSS |
| HTTP 客户端 | uni.request 封装 | 统一请求处理 |
| 认证 | Casdoor OAuth2/OIDC | 单点登录 |

### 核心设计原则

1. **模块化但不分离** - Monorepo 单一代码库，按功能模块组织
2. **路由驱动** - 通过路由区分模块，懒加载优化性能
3. **最小共享** - 只共享真正需要跨模块的状态（如用户信息、认证状态）
4. **平台适配层** - 抽象平台差异，业务代码平台无关
5. **类型安全** - OpenAPI 生成类型，端到端类型安全

---

## 目录结构

### Monorepo 结构

```
clients/web/uni-app/
├── src/
│   ├── pages/                # 页面（对应路由）
│   │   ├── index/            # 主页
│   │   ├── course/           # 评课模块
│   │   │   ├── list.vue
│   │   │   ├── detail.vue
│   │   │   └── review.vue
│   │   ├── teacher/          # 教师模块
│   │   ├── tools/            # 工具箱模块
│   │   ├── community/        # 社群模块
│   │   └── user/             # 用户中心
│   │
│   ├── components/           # 组件
│   │   ├── layout/           # 布局组件
│   │   ├── common/           # 通用组件
│   │   └── business/         # 业务组件
│   │
│   ├── api/                  # API 封装
│   │   ├── request.ts        # uni.request 封装
│   │   ├── course.ts
│   │   ├── teacher.ts
│   │   └── auth.ts
│   │
│   ├── stores/               # Pinia 状态
│   │   ├── auth.ts           # 认证状态
│   │   ├── user.ts           # 用户信息
│   │   ├── app.ts            # 应用状态
│   │   └── favorites.ts      # 收藏状态
│   │
│   ├── composables/          # 组合式函数
│   │   ├── useAsyncData.ts
│   │   └── usePagination.ts
│   │
│   ├── utils/                # 工具函数
│   ├── types/                # 类型定义
│   │   ├── api.gen.ts        # OpenAPI 生成
│   │   └── common.ts
│   ├── constants/            # 常量定义
│   ├── static/               # 静态资源
│   └── App.vue
│
├── pages.json                # 页面路由配置
├── manifest.json             # 应用配置
├── vite.config.ts
├── tsconfig.json
└── package.json
```

### 模块组织原则

1. **按功能模块划分** - 每个模块有独立的 pages 目录
2. **共享代码集中** - components/、api/、stores/ 等跨模块共享
3. **类型安全** - OpenAPI 生成的类型统一管理
4. **平台无关** - 业务代码尽量不依赖平台特性

---

## 路由设计

### 路由结构

**核心原则：对象主页用顶级路由，详情页用嵌套路由**

```
# 主页和导航
/pages/index/index           # 全站主页/导航页

# 课程模块
/pages/course/list           # 课程列表
/pages/course/detail         # 课程主页（query: id）
/pages/course/review         # 评课详情（query: courseId, reviewId）

# 教师模块
/pages/teacher/list          # 教师列表
/pages/teacher/detail        # 教师主页（query: id）

# 院系模块
/pages/department/list       # 院系列表
/pages/department/detail     # 院系主页（query: id）

# 工具箱模块
/pages/tools/index           # 工具箱首页
/pages/tools/classroom       # 空教室查询
/pages/tools/gpa             # GPA 计算器

# 用户中心
/pages/user/index            # 用户中心首页
/pages/user/reviews          # 我的评课
/pages/user/favorites        # 我的收藏
/pages/user/settings         # 设置

# 管理后台
/pages/admin/index           # 管理后台首页
/pages/admin/reviews         # 评课管理
/pages/admin/reports         # 举报管理

# 认证
/pages/auth/callback         # SSO 回调页面
```

### pages.json 配置示例

```json
{
  "pages": [
    {
      "path": "pages/index/index",
      "style": { "navigationBarTitleText": "StuHelper" }
    },
    {
      "path": "pages/course/list",
      "style": { "navigationBarTitleText": "课程列表" }
    },
    {
      "path": "pages/course/detail",
      "style": { "navigationBarTitleText": "课程详情" }
    }
  ],
  "tabBar": {
    "list": [
      { "pagePath": "pages/index/index", "text": "首页" },
      { "pagePath": "pages/course/list", "text": "评课" },
      { "pagePath": "pages/tools/index", "text": "工具" },
      { "pagePath": "pages/user/index", "text": "我的" }
    ]
  }
}
```

### 路由命名规范

- **H5 URL**：`stuhelper.com/course/detail?id=123`
- **小程序路径**：`/pages/course/detail?id=123`
- **参数传递**：使用 query 参数，不使用路径参数（uni-app 限制）

---

## API 层设计

### HTTP 请求封装

```typescript
// api/request.ts
interface RequestConfig {
  url: string
  method?: 'GET' | 'POST' | 'PUT' | 'DELETE'
  data?: unknown
  header?: Record<string, string>
}

export function request<T>(config: RequestConfig): Promise<T> {
  return new Promise((resolve, reject) => {
    uni.request({
      url: `${import.meta.env.VITE_API_URL}${config.url}`,
      method: config.method || 'GET',
      data: config.data,
      withCredentials: true,
      header: {
        'Content-Type': 'application/json',
        ...config.header
      },
      success: (res) => {
        if (res.statusCode === 200) {
          resolve(res.data as T)
        } else if (res.statusCode === 401) {
          // Token 过期，跳转登录
          uni.navigateTo({ url: '/pages/auth/login' })
          reject(new Error('Unauthorized'))
        } else {
          reject(new Error(res.data?.message || 'Request failed'))
        }
      },
      fail: reject
    })
  })
}
```

### API 模块化

```typescript
// api/course.ts
import { request } from './request'
import type { Course, Review, PaginatedResponse } from '@/types/api.gen'

export const courseApi = {
  getCourses: (params: { page: number; pageSize: number }) =>
    request<PaginatedResponse<Course>>({
      url: '/api/v1/courses',
      method: 'GET',
      data: params
    }),

  getCourse: (id: number) =>
    request<Course>({ url: `/api/v1/courses/${id}` }),

  getCourseReviews: (courseId: number, params: { page: number; pageSize: number }) =>
    request<PaginatedResponse<Review>>({
      url: `/api/v1/courses/${courseId}/reviews`,
      data: params
    })
}
```

### 类型安全

- 所有 API 类型从 OpenAPI 规范自动生成（`types/api.gen.ts`）
- 前后端类型定义保持同步
- TypeScript 编译时检查类型错误

---

## 认证流程

### Casdoor OAuth2/OIDC 集成

**认证流程：**

```
1. 用户访问需要登录的页面
   ↓
2. 前端调用应用后端登录入口
   ↓
3. 后端返回 Casdoor 授权地址和随机 state
   ↓
4. 浏览器跳转到 https://sso.stuhelper.com
   ↓
5. Casdoor 登录完成后回跳到应用前端 callback 页面
   ↓
6. 前端 callback 页面调用应用后端 /api/v1/auth/callback
   ↓
7. 后端使用 code + client_secret 换取 token，并写入 HttpOnly Cookie
   ↓
8. 前端只保存最小会话信息和回跳目标
```

### 认证 Store

```typescript
// stores/auth.ts
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { authApi } from '@/api/auth'

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserSession | null>(null)
  const capabilities = ref<string[]>([])
  const redirectUrl = ref<string>('/')

  const isAuthenticated = computed(() => user.value !== null)

  async function login() {
    const { data } = await authApi.getLoginURL()
    window.location.href = data.loginURL
  }

  async function handleCallback(code: string, state: string) {
    const { data } = await authApi.handleCallback({ code, state })
    user.value = data.user
    capabilities.value = data.capabilities ?? []
    return redirectUrl.value
  }

  return { user, capabilities, isAuthenticated, login, handleCallback }
})
```

---

## 状态管理

### Pinia 使用原则

**只在以下场景使用 Pinia：**

1. **跨模块共享且多处写入** - 如收藏状态（多个页面都可以收藏/取消收藏）
2. **需要持久化的用户状态** - 如认证信息、用户偏好设置
3. **全局 UI 状态** - 如主题、语言切换

**不使用 Pinia 的场景：**

- 单页面消费的数据 → 使用组件本地状态 + `useAsyncData`
- 父子组件通信 → 使用 props/emits
- 跨层级组件通信 → 使用 provide/inject

### 全局 Store 结构

```
stores/
├── auth.ts          # 认证状态（会话、用户信息、能力）
├── user.ts          # 用户信息（profile、偏好设置）
├── app.ts           # 应用状态（主题、语言）
└── favorites.ts     # 收藏状态（跨页面共享）
```

### 组件本地状态示例

```typescript
// pages/course/list.vue
<script setup lang="ts">
import { ref } from 'vue'
import { courseApi } from '@/api/course'

const courses = ref([])
const loading = ref(false)

async function loadCourses() {
  loading.value = true
  try {
    const res = await courseApi.getCourses({ page: 1, pageSize: 20 })
    courses.value = res.data
  } finally {
    loading.value = false
  }
}

onMounted(() => loadCourses())
</script>
```

---

## 跨平台适配

### 平台支持

**uni-app 一套代码，多端输出：**

- **H5**：部署到 stuhelper.com
- **微信小程序**：提交到微信平台
- **App**：iOS/Android 原生应用

### 平台差异处理

**条件编译：**

```vue
<template>
  <view>
    <!-- H5 显示 -->
    <!-- #ifdef H5 -->
    <web-header />
    <!-- #endif -->

    <!-- 小程序显示 -->
    <!-- #ifdef MP-WEIXIN -->
    <mp-header />
    <!-- #endif -->

    <!-- 通用内容 -->
    <course-list />
  </view>
</template>

<script setup lang="ts">
// #ifdef H5
console.log('H5 平台')
// #endif

// #ifdef MP-WEIXIN
console.log('微信小程序')
// #endif
</script>
```

### 小程序配置

**域名白名单（manifest.json）：**

```json
{
  "mp-weixin": {
    "setting": {
      "urlCheck": true
    },
    "permission": {
      "scope.userLocation": {
        "desc": "用于查询空教室位置"
      }
    }
  },
  "h5": {
    "domain": "stuhelper.com"
  }
}
```

**需要配置的域名：**
- request 合法域名：`https://api.stuhelper.com`
- request 合法域名：`https://sso.stuhelper.com`

---

## 构建和部署

### 构建命令

```json
{
  "scripts": {
    "dev:h5": "uni -p h5",
    "dev:mp-weixin": "uni -p mp-weixin",
    "dev:app": "uni -p app",
    "build:h5": "uni build -p h5",
    "build:mp-weixin": "uni build -p mp-weixin",
    "build:app": "uni build -p app",
    "type-check": "vue-tsc --noEmit",
    "lint": "eslint src --ext .vue,.ts"
  }
}
```

### 部署策略

**H5 部署：**
- 构建产物：`dist/build/h5/`
- 部署到：CDN 或 Nginx
- 域名：`stuhelper.com`
- 自动部署：GitLab CI/CD

**小程序部署：**
- 构建产物：`dist/build/mp-weixin/`
- 上传到：微信开发者工具 → 提交审核
- 手动部署（小程序审核流程）

**App 部署：**
- 构建产物：`dist/build/app/`
- 打包：HBuilderX 云打包
- 发布到：App Store / Google Play

### 环境变量

```bash
# .env.development
VITE_API_URL=http://localhost:8080
VITE_SSO_URL=http://localhost:9000
VITE_CASDOOR_CLIENT_ID=dev_client_id
VITE_APP_URL=http://localhost:5173

# .env.production
VITE_API_URL=https://api.stuhelper.com
VITE_SSO_URL=https://sso.stuhelper.com
VITE_CASDOOR_CLIENT_ID=prod_client_id
VITE_APP_URL=https://stuhelper.com
```

---

## 开发规范

### 组件规范

- 文件命名：PascalCase `CourseCard.vue`
- 组件名必须多单词
- 使用 `<script setup lang="ts">`
- Props 使用 TypeScript 类型定义

### 代码风格

- 遵循项目 ESLint 配置
- 使用 Prettier 格式化
- TypeScript 严格模式
- 禁止使用 `any` 类型

### 性能优化

- 路由懒加载
- 图片懒加载
- 列表虚拟滚动（长列表）
- 合理使用缓存

---

## 迁移计划

### 从现有前端迁移

1. **创建 uni-app 项目** - 初始化新的 uni-app 项目结构
2. **迁移共享代码** - API 层、类型定义、工具函数
3. **重构页面组件** - 按新的目录结构重构
4. **适配平台差异** - 添加条件编译
5. **测试验证** - H5、小程序、App 全平台测试
6. **灰度发布** - 先发布 H5，再发布小程序

### 注意事项

- OpenAPI 生成的类型定义保持不变
- 后端 API 无需修改
- 认证流程保持兼容
- 逐步迁移，降低风险
