# 06 发布测评页面分析文档

## 页面概述

该页面是"发布测评"功能模块（路由：`/reviews/post`），属于"**非官方课程测评@北京大学**"网站的核心用户交互功能。页面提供完整的测评发布表单，支持课程选择、多维度评分和详细评价内容输入。

- **页面标题**: 发布测评 - 非官方课程测评@北京大学
- **技术栈**: React + Bootstrap + Font Awesome 6
- **版本信息**: 前端 2.5.1201.endor_hotfix, 后端 2512.31.main

## UI 设计分析

### 1. 页面结构

```
┌──────────────────────────────────────────────────────────┐
│  顶部导航栏 (navbar navbar-dark bg-dark sticky-top)       │
│  [Logo] 课程测评  |  [搜索框]  |  [课程] [最新] [发测评✓] │
├──────────────────────────────────────────────────────────┤
│  [←] 发布测评                                             │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 课程名称: [react-select 搜索框]                     │ │
│  │ 授课老师: [输入框] 老师                             │ │
│  │ 学期:     [下拉选择框]                              │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
│  评分 ─────────────────────────────────────────────────  │
│  │ 总体评价: [😍] [🙂] [😐] [🙁] [😭]                  │ │
│  │ 内容质量: [😍] [🙂] [😐] [🙁] [😭]                  │ │
│  │ 工作量:   [😍] [🙂] [😐] [🙁] [😭]                  │ │
│  │ 考核/给分: [😍] [🙂] [😐] [🙁] [😭]                 │ │
│                                                          │
│  标题:     [输入框]                                      │
│  详细评价: [多行文本框 - 带默认模板]                     │
│  你的成绩: [输入框] (选填)                               │
│                                                          │
│  [提交测评] 按钮                                         │
│  提交说明文字                                            │
├──────────────────────────────────────────────────────────┤
│  页脚: © 2018-2025 非官方课程测评@PKU | FAQ | 关于       │
└──────────────────────────────────────────────────────────┘
```

### 2. 顶部导航栏

与其他页面保持一致的深色导航栏 (`navbar-dark bg-dark sticky-top`)：

| 元素 | 实现方式 | 说明 |
|------|----------|------|
| Logo + 品牌名 | `<a class="navbar-brand">` | 点击跳转首页 `/` |
| 搜索框 | `react-select` 组件 | placeholder: "搜索课程名称" |
| 课程按钮 | `btn-secondary` | 图标 `fa-list-ul`，链接 `/courses/list` |
| 最新按钮 | `btn-secondary` | 图标 `fa-clock`，链接 `/reviews/latest` |
| 发测评按钮 | `btn-info active` | 图标 `fa-pencil`，当前页面高亮 |

### 3. 表单字段设计

#### 3.1 课程信息区域

| 字段 | 类型 | name 属性 | placeholder | 说明 |
|------|------|-----------|-------------|------|
| 课程名称 | `react-select` | `course_id` (hidden) | 输入名称搜索 | 支持拼音搜索 |
| 授课老师 | `input[type="text"]` | `teacher_name` | 赵克常 | 必填，带"老师"后缀 |
| 学期 | `select` | `term_id` | - | `<optgroup>` 分组 |

#### 3.2 四维评分系统

使用 `btn-group-toggle` 实现单选按钮组：

| 维度 | name 属性 | 说明 |
|------|-----------|------|
| 总体评价 | `recommended` | 整体推荐程度 |
| 内容质量 | `rating_content` | 课程内容评价 |
| 工作量 | `rating_workload` | 作业负担评价 |
| 考核/给分 | `rating_exam` | 考试难度评价 |

**评分值与图标**:

| 值 | 图标 | Font Awesome 类 | 颜色 |
|----|------|-----------------|------|
| 2 | 😍 | `fa-face-grin-hearts` | text-success |
| 1 | 🙂 | `fa-face-smile` | text-success |
| 0 | 😐 | `fa-face-meh` | (默认) |
| -1 | 🙁 | `fa-face-frown` | text-warning |
| -2 | 😭 | `fa-face-sad-cry` | text-danger |

#### 3.3 评价内容区域

| 字段 | 类型 | name 属性 | 说明 |
|------|------|-----------|------|
| 标题 | `input[type="text"]` | `title` | 一句话总结观点 |
| 详细评价 | `textarea` | `content` | 必填，带默认模板 |
| 你的成绩 | `input[type="text"]` | `result` | 选填，退课填W |

**默认评价模板**:
```
课程听感:
作业/任务量:
关于考试:
```

## 功能实现分析

### 1. 数据结构设计

```typescript
interface PostReviewData {
  course_id: number;           // 课程ID
  teacher_name: string;        // 教师姓名
  term_id: string;             // 学期ID (如: "25-26-1")
  recommended: number;         // 总体评价 (-2 ~ 2)
  rating_content: number;      // 内容质量 (-2 ~ 2)
  rating_workload: number;     // 工作量 (-2 ~ 2)
  rating_exam: number;         // 考核/给分 (-2 ~ 2)
  title: string;               // 标题
  content: string;             // 详细评价
  result?: string;             // 成绩 (选填)
}
```

### 2. API 接口设计

```typescript
// 提交测评
POST /api/reviews/post
Body: PostReviewData

// 课程搜索 (react-select 异步加载)
GET /api/courses/search?q={keyword}
Response: { id: number, name: string }[]
```

### 3. 组件结构

```
PostReviewPage/
├── index.tsx                    // 页面主组件
├── components/
│   ├── CourseSelect.tsx         // 课程选择器
│   ├── RatingButtonGroup.tsx    // 评分按钮组
│   └── TermSelect.tsx           // 学期选择器
└── hooks/
    └── usePostReview.ts         // 提交逻辑 Hook
```

## 总结

发布测评页面是课程测评系统的核心用户交互入口，提供完整的测评发布功能。

**核心特点**:
1. **课程智能搜索** - 支持中文、拼音首字母模糊搜索
2. **四维评分系统** - 表情图标直观展示评分等级
3. **默认评价模板** - 引导用户从多角度评价课程
4. **表单验证** - 必填字段验证，成绩选填
5. **用户协议** - 提交即同意授权使用内容
