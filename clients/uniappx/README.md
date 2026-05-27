# StuHelper UniApp X

StuHelper 移动端应用，基于 uni-app x 开发。

## 技术栈

- **框架**: uni-app x
- **语言**: TypeScript (uts)
- **UI**: Vue 3 + Composition API
- **状态管理**: Pinia
- **共享代码**: @stuhelper/shared

## 目录结构

```
src/
├── pages/           # 页面
│   ├── home/       # 首页
│   ├── course/     # 课程模块
│   ├── review/     # 评课模块
│   ├── teacher/    # 教师模块
│   ├── user/       # 用户中心
│   └── auth/       # 认证模块
├── stores/         # Pinia 状态管理
├── api/            # API 请求
├── composables/    # 组合式函数
├── utils/          # 工具函数
├── App.vue         # 应用入口
├── main.ts         # 主入口
├── pages.json      # 页面配置
└── manifest.json   # 应用配置
```

## 开发命令

```bash
# 开发 H5
pnpm dev:h5

# 开发微信小程序
pnpm dev:mp-weixin

# 构建 H5
pnpm build:h5

# 构建微信小程序
pnpm build:mp-weixin
```

## 当前状态

`clients/uniappx` 已从占位脚手架升级为接入共享契约的移动端实现：

- 统一复用 `@stuhelper/shared` 的 OpenAPI 类型与 API 工厂
- 已接入 Cookie Session / OIDC 登录、课程列表、课程详情、评课广场、发评课、教师详情、通知中心
- 已接入个人中心的「我的评课 / 我的投票 / 我的收藏 / 通知」页面
- 运行时请求通过 uni-app transport 适配层实现，但业务接口仍统一来自 `clients/shared`
- H5 本地开发默认把 `/api` 代理到 `VITE_DEV_PROXY_TARGET`，缺省为 `http://localhost:8080`；
  原生 / 小程序构建需要提供绝对 `VITE_API_URL`
- H5 启动只在存在浏览器会话提示（`csrf_token` cookie 或已持久化 CSRF token）时探测
  `/api/v1/auth/me`，游客态不得产生预期内的 401 控制台噪声；原生 App 仍以 secure-storage
  bridge 中的 token 作为会话提示

## 功能模块

- **首页**: 统计概览、快捷入口、热门课程
- **课程列表 / 详情**: 分页浏览、查看评分统计、收藏课程
- **评课广场**: 浏览最新评课、点赞/点踩
- **发布评课**: 学期/教师选择、评分维度填写、草稿保存与提交
- **教师主页**: 教师评分统计与课程列表
- **个人中心**: 认证概览、我的评课、我的投票、我的收藏、通知
- **登录**: 校园 SSO / OIDC 登录入口

## 共享代码

通过 `@stuhelper/shared` 包共享以下内容：

- 类型定义 (Course, Review, User 等)
- API 接口定义
- 常量配置
- 工具函数 (rating, color 等)
