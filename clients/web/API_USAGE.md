# 主站 API 使用指南

## 调用链

```
@/api/index.ts → @stuhelper/shared/api → openapi-fetch
```

## 入口

```ts
import { api } from "@/api";
```

按域拆分：`api.auth` / `api.course` / `api.review` / `api.user` / `api.identity` / `api.notification` / `api.admin` / `api.draft` / `api.reply` / `api.rating`

## 示例

### 登录

```ts
const { data } = await api.auth.login(redirect);
if (data?.url) window.location.href = data.url;
```

### 当前用户

```ts
const { data } = await api.auth.me();
```

### 搜索课程

```ts
const { data } = await api.course.searchCourses("数据结构", { pageSize: 20 });
```

### 评课列表

```ts
const { data } = await api.review.getReviews(1001, { page: 1, pageSize: 20, sort: "time" });
```

### 发评课

```ts
await api.review.createReview({
  courseID: 1001,
  termID: "2025-2",
  title: "课程体验",
  content: "整体节奏不错，作业量适中。",
  ratings: { teaching: 4, grading: 4, workload: 3 },
});
```

### 实名认证

```ts
await api.identity.submitIdentity({
  realName: "张三",
  docType: "MAINLAND_ID",
  docNumber: "110101199001011234",
});
```

### 学生认证

```ts
await api.identity.verifyStudent({
  schoolID: "buaa",
  verificationMethod: "manual",
  consentAccepted: true,
  manualFormData: { studentId: "23373333" },
});
```

## 错误处理

`authenticatedFetch` 已处理 Cookie 携带、CSRF 注入、401 自动 refresh、统一 `ApiError`。页面只需处理业务错误。

## 规则

1. 先改 OpenAPI，再改代码
2. 页面只从 `api` 对象调接口
3. 不手写重复类型
4. 默认不直接 `fetch`
   - 允许例外：OIDC callback 浏览器跳转、`sendBeacon` / `keepalive` 上报、浏览器/框架基础设施请求
   - 例外调用点必须写注释说明原因
