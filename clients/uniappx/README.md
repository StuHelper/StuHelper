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

> 当前 `clients/uniappx` 为实验性脚手架，用于占位和后续多端实现收口。
> 目前已保留基础页面结构与导航骨架，但 **SSO 登录与部分业务能力尚未开放**。

## 功能模块

- **首页**: Hero 区域 + 实验性功能导航卡片
- **课程列表**: 占位页面
- **评课广场**: 占位页面
- **教师主页**: 占位页面（未在首页暴露）
- **个人中心**: 用户信息占位 + 已接线消息通知入口
- **消息通知**: 占位页面
- **SSO 登录**: 暂未开放，仅显示实验性提示

## 共享代码

通过 `@stuhelper/shared` 包共享以下内容：

- 类型定义 (Course, Review, User 等)
- API 接口定义
- 常量配置
- 工具函数 (rating, color 等)
