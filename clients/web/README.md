# StuHelper Web 前端

基于 Vue 3 + Vite + TypeScript + Tailwind CSS v4 的评课社区前端应用。

## 技术栈

- **框架**: Vue 3.5, Vite 6.0, TypeScript 5.7
- **样式**: Tailwind CSS v4, Glassmorphism 设计系统
- **状态**: Pinia 2.2+, Vue Router 4.5+
- **动画**: @vueuse/motion, GSAP
- **UI**: Element Plus, Radix Vue

## 目录结构

```
src/
├── api/              # API 调用层
├── components/       # 组件库
│   ├── ui/          # 基础 UI 组件（Button, Card, Input）
│   ├── common/      # 通用功能组件（Toast, Modal, ErrorBoundary）
│   ├── business/    # 业务组件
│   │   └── review/  # 评课相关组件
│   ├── animated/    # 动画包装器
│   └── layout/      # 布局组件（AppShell, Navbar）
├── modules/         # 功能模块
│   ├── admin/       # 管理后台
│   ├── auth/        # 认证
│   ├── course/      # 课程
│   ├── home/        # 主页
│   ├── review/      # 评课
│   ├── teacher/     # 教师
│   └── user/        # 用户中心
├── stores/          # Pinia 状态管理
├── composables/     # 组合式函数
├── utils/           # 工具函数
├── design-system/   # 设计系统 tokens
├── i18n/            # 国际化
├── router/          # 路由配置
└── styles/          # 全局样式

```

## 组件分类原则

- **ui/**: 纯 UI 组件，无业务逻辑
- **common/**: 通用功能组件，可跨业务复用
- **business/**: 业务相关组件，按领域分组
- **layout/**: 页面布局组件

## 开发

```bash
pnpm install
pnpm dev      # http://localhost:3000
pnpm build
```
