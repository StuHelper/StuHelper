# OpenAPI 类型安全 API 客户端使用指南

## 生成类型

当后端 OpenAPI 规范更新后，运行：

```bash
pnpm api:generate
```

## 使用示例

### 1. 课程 API

```typescript
import { courseApi } from '@/api'

// 获取课程列表
const { data, error } = await courseApi.getCourses({ page: 1, limit: 20 })
if (data) {
  console.log(data) // 完整类型提示
}

// 搜索课程
const result = await courseApi.searchCourses('数据结构')

// 获取单个课程
const course = await courseApi.getCourse('course-id')
```

### 2. 评课 API

```typescript
import { reviewApi } from '@/api'

// 获取课程评价
const reviews = await reviewApi.getReviews('course-id', { page: 1 })

// 创建评价
const newReview = await reviewApi.createReview({
  courseId: 'course-id',
  rating: 5,
  content: '很好的课程'
})

// 更新评价
await reviewApi.updateReview('review-id', { rating: 4 })

// 删除评价
await reviewApi.deleteReview('review-id')
```

### 3. 认证 API

```typescript
import { authApi } from '@/api'

// 登录
const { data } = await authApi.login({
  email: 'user@example.com',
  password: 'password'
})

// 获取当前用户
const user = await authApi.me()

// 登出
await authApi.logout()
```

## 优势

✅ **类型安全**：所有请求和响应都有完整的 TypeScript 类型
✅ **自动补全**：IDE 自动提示所有可用的 API 端点和参数
✅ **编译时检查**：路径拼写错误在编译时就能发现
✅ **零维护**：API 变更时只需重新生成类型
✅ **统一管理**：所有 API 调用都通过类型安全的客户端
