# Casdoor SSO 集成与生态边界

> 状态：SSO 已实现；本文档定义的是 StuHelper 生态中的长期边界，不是只描述当前某一个应用的回调流程。

## 核心定位

`sso.stuhelper.com` 是 StuHelper 生态的统一身份平台。

它负责：

- 登录、注册、单点登录
- OAuth/OIDC 应用接入
- token、session、consent、scope
- 平台级管理员

它不负责：

- 航小伴的课程级管理员
- 资源共享分类管理员
- 内容 owner
- 基于 `schoolID`、学生/老师、学生认证、实名认证的业务授权决策

这些都属于应用业务域。

## 生态里的应用关系

StuHelper 生态中会存在多个实际应用：

- 航小伴
- 开发者平台（未来）
- 第三方接入应用

因此，Casdoor 中的 Application 应该按实际应用拆分，而不是假设存在一个“StuHelper 总应用”。

推荐命名示例：

- `hangxiaoban`
- `developer-portal`
- 未来第三方应用各自独立的 client

## 当前标准登录流程

```text
1. 应用前端调用 /api/v1/auth/login 或 /api/v1/auth/signup
2. 应用后端生成 Casdoor 授权地址和随机 state
3. 浏览器跳转到 https://sso.stuhelper.com
4. Casdoor 登录完成后回跳到应用自己的前端 callback 页面
5. 前端 callback 页面调用应用后端 /api/v1/auth/callback
6. 应用后端用授权码向 Casdoor 换取 access token / refresh token
7. 应用后端把 token 写入 HttpOnly Cookie，并返回当前应用可见的最小用户信息
8. 应用前端保存最小登录态，用于路由预检和 UI 渲染
```

关键点：

- 回调地址属于具体应用，而不是生态统一回调页。
- token 交换应在应用后端完成，不应把 client secret 暴露到前端。
- access token / refresh token 默认通过安全 Cookie 管理。

## 环境变量建议

### 航小伴本地开发

```bash
CASDOOR_ENDPOINT=https://sso.stuhelper.com
CASDOOR_CLIENT_ID=<hangxiaoban-client-id>
CASDOOR_CLIENT_SECRET=<hangxiaoban-client-secret>
CASDOOR_ORGANIZATION=stuhelper
CASDOOR_APPLICATION=hangxiaoban
CASDOOR_REDIRECT_URI=http://localhost:3000/auth/callback
CASDOOR_CERTIFICATE=<pem-content-or-empty>

TOKEN_ACCESS_TTL=900
TOKEN_REFRESH_TTL=604800
TOKEN_COOKIE_SECURE=false
```

### 其他应用

开发者平台或第三方应用必须使用自己的 Casdoor Application，不应复用 `hangxiaoban` 的 client。

## `isAdmin` 的使用限制

Casdoor 用户对象里的 `isAdmin` 只能表达：

- `sso.stuhelper.com` 平台管理能力
- 组织级或平台级管理员身份

它不应该直接决定：

- 能不能进入航小伴后台
- 能不能审核评课
- 能不能审核认证
- 能不能管理课程资料

也就是说：

- `isAdmin` 可以存在
- 但不能再作为航小伴业务后台的唯一门禁

## 航小伴应该如何消费 Casdoor

航小伴应该把 Casdoor 看成 **Identity Plane**：

- 负责证明“用户是谁”
- 提供基础 identity claims
- 提供 scope / consent 结果

然后由航小伴自己的后端做：

- 应用级能力判断
- 模块级管理员判断
- 资源级授权
- 认证状态、学校、身份类型等业务规则

## 对第三方应用开放什么

第三方应用可以通过 OAuth scope 获取最小必要身份事实，例如：

- `identityVerified`
- `studentVerified`
- `actorType`
- `schoolID`

默认不返回：

- 姓名
- 学号
- 手机号
- 身份证号

详情见 [05-open-platform-claims-and-scopes.md](05-open-platform-claims-and-scopes.md)。

## 前端集成约束

### 需要登录的页面

仍然使用：

```typescript
meta: {
  requiresAuth: true
}
```

### 不能再这样做

```typescript
meta: {
  requiresAdmin: true
}
```

如果 `requiresAdmin` 的含义是“只有 `isAdmin=true` 才能进应用后台”，这是错误边界。
后台菜单和路由应基于应用能力（capabilities / effective permissions），而不是平台级管理员标记。

## 官方文档

- [Casdoor Permission Overview](https://casdoor.org/docs/permission/overview)
- [Casdoor Permission Configuration](https://casdoor.org/docs/permission/permission-configuration)
- [Casdoor Token Overview](https://casdoor.org/docs/token/overview)
- [Casdoor User Roles](https://casdoor.org/docs/user/roles)
- [Casdoor User Permissions](https://casdoor.org/docs/user/permissions)

## 常见误区

### 误区 1：把 Casdoor 当成航小伴业务权限数据库

Casdoor 很适合做身份平台和应用接入中心，但不适合直接承载高基数的课程、分类、内容资源授权关系。

### 误区 2：把 `isAdmin` 当作应用后台总开关

这会把平台身份和业务权限混为一谈。

### 误区 3：所有应用共用一个 Casdoor Application

每个实际应用都应有自己的 Application、回调地址和 scope 策略。
