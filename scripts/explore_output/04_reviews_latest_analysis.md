# 04 最新评教页面分析文档

## 页面概述

该页面是"最新测评"功能模块（路由：`/reviews/latest`），属于"**非官方课程测评@北京大学**"网站的核心功能之一。页面采用卡片式布局，按时间倒序展示学生对课程的最新评价信息。

- **页面标题**: 最新测评 - 非官方课程测评@北京大学
- **技术栈**: React + Bootstrap + Font Awesome 6
- **版本信息**: 前端 2.5.1201.endor_hotfix, 后端 2512.31.main

## UI 设计分析

### 1. 页面结构

```
┌──────────────────────────────────────────────────────────┐
│  顶部导航栏 (navbar navbar-dark bg-dark sticky-top)       │
│  [Logo] 课程测评  |  [搜索框]  |  [课程] [最新✓] [发测评] │
├──────────────────────────────────────────────────────────┤
│  <h2>最新测评</h2>                                        │
│                                                          │
│  ┌────────────────────────────────────────────────────┐ │
│  │ 评教卡片 (card border-none / border-warning)       │ │
│  │ ├─ card-body: 标题 + 课程信息 + 评价内容 + 成绩    │ │
│  │ └─ card-footer: 四维评分图标 + 点赞/踩按钮         │ │
│  └────────────────────────────────────────────────────┘ │
│                                                          │
├──────────────────────────────────────────────────────────┤
│  页脚: © 2018-2025 非官方课程测评@PKU | FAQ | 关于       │
└──────────────────────────────────────────────────────────┘
```

### 2. 顶部导航栏 (实际实现)

基于 Bootstrap 的深色导航栏 (`navbar-dark bg-dark sticky-top`)：

| 元素 | 实现方式 | 说明 |
|------|----------|------|
| Logo + 品牌名 | `<a class="navbar-brand">` | 点击跳转首页 `/` |
| 搜索框 | `react-select` 组件 | placeholder: "搜索课程名称" |
| 课程按钮 | `btn-secondary` | 图标 `fa-list-ul`，链接 `/courses/list` |
| 最新按钮 | `btn-secondary active` | 图标 `fa-clock`，当前页面高亮 |
| 发测评按钮 | `btn-info` | 图标 `fa-pencil`，链接 `/reviews/post` |

### 3. 评教卡片设计 (实际实现)

每个评教卡片使用 Bootstrap Card 组件：

#### 3.1 卡片容器
- **普通卡片**: `card mt-2 border-none`
- **警告卡片**: `card mt-2 border-warning` (用于负面评价)

#### 3.2 Card Body 内容

| 元素 | CSS 类 | 说明 |
|------|--------|------|
| 评价标题 | `card-title h5` | 通过 CSS `::after` 伪元素注入内容 |
| 课程信息 | `card-subtitle h6 text-muted` | 格式: 课程链接 + 教师 + 学期 + 日期 |
| 评价内容 | `row > col` | font-size: 0.9em, line-height: 1.7 |
| 成绩 | `<strong>成绩:</strong>` | 显示具体分数或等级 |

### 4. 四维评分系统 (Card Footer)

**左侧 - 四维评分指标**:

| 维度 | 说明 |
|------|------|
| 是否推荐 | 整体推荐程度 |
| 课程内容 | 课程质量评价 |
| 工作量 | 作业负担评价 |
| 考核 | 考试难度评价 |

**表情图标对应**:

| 图标 | Font Awesome 类 | 颜色 | 含义 |
|------|-----------------|------|------|
| 😍 | `fa-face-grin-hearts` | text-success | 非常好 |
| 🙂 | `fa-face-smile` | text-success | 好 |
| 😐 | `fa-face-meh` | (默认) | 一般 |
| 🙁 | `fa-face-frown` | text-warning | 不好 |

**右侧 - 互动按钮**:
- 👍 点赞: `btn btn-sm btn-success`
- 👎 踩: `btn btn-sm btn-danger`

### 5. 视觉设计特点

- **配色方案**:
  - 主色调：蓝色系
  - 背景色：浅灰色 (#F5F5F5)
  - 卡片背景：白色
  - 强调色：黄色（星级评分）

- **卡片样式**:
  - 圆角设计 (border-radius: 8-12px)
  - 轻微阴影效果 (box-shadow)
  - 卡片间距：12-16px

- **字体层级**:
  - 课程名称：16-18px，粗体
  - 教师信息：14px，常规
  - 评价内容：14-15px，常规
  - 时间戳：12px，灰色

### 5. 底部导航栏

固定在底部的 Tab 导航，包含四个主要入口：
- 首页
- 课程
- 评教（当前高亮）
- 我的

## 功能实现分析

### 1. 核心功能

#### 1.1 评教列表展示
- 按时间倒序展示最新评教记录
- 支持下拉刷新获取最新数据
- 支持上拉加载更多（分页加载）

#### 1.2 筛选功能
- 按课程类型筛选
- 按评分区间筛选
- 按时间范围筛选

#### 1.3 评教详情
- 点击卡片可查看完整评价内容
- 查看评价的点赞/回复数

### 2. 数据结构设计 (基于实际 HTML 分析)

```typescript
interface ReviewItem {
  id: string;                    // 评教记录ID (如: tKpLN7dX33CYbhN9Z)
  title: string;                 // 评价标题 (通过 CSS ::after 注入)
  courseId: number;              // 课程ID (如: 1033)
  courseName: string;            // 课程名称 (如: 偏微分方程)
  teacherName: string;           // 教师姓名 (如: 周蜀林)
  semester: string;              // 学期 (如: 25-26-1 学期)
  date: string;                  // 发布日期 (如: 2026/01/09)
  grade: string;                 // 成绩 (如: 99, P, 80-85)
  ratings: {
    recommend: 'excellent' | 'good' | 'neutral' | 'bad';
    content: 'excellent' | 'good' | 'neutral' | 'bad';
    workload: 'excellent' | 'good' | 'neutral' | 'bad';
    exam: 'excellent' | 'good' | 'neutral' | 'bad';
  };
  likeCount: number;             // 点赞数
  dislikeCount: number;          // 踩数
  isWarning: boolean;            // 是否为警告卡片 (border-warning)
}
```

**实际数据示例** (从 HTML 提取):

| 课程 | 教师 | 标题 | 成绩 | 学期 |
|------|------|------|------|------|
| 偏微分方程 | 周蜀林 | 简单好学 | 99 | 25-26-1 |
| 可再生能源与低碳社会 | 萧立新 | pf摸鱼好课 | P | 25-26-1 |
| 普通化学实验（B） | 徐怡庄 | 相当麻烦的课... | 预估75+ | 25-26-1 |
| 羽毛球 | 张亚谦 | 没啥意思... | 除去体侧扣8分 | 25-26-1 |
| 量子力学 (A) | 全海涛 | 中规中矩... | 94 | 25-26-1 |

### 3. API 接口设计

```typescript
// 获取最新评教列表
GET /api/reviews/latest
Query Parameters:
  - page: number          // 页码
  - pageSize: number      // 每页数量
  - courseType?: string   // 课程类型筛选
  - minRating?: number    // 最低评分
  - maxRating?: number    // 最高评分
  - startDate?: string    // 开始日期
  - endDate?: string      // 结束日期

// 获取评教详情
GET /api/reviews/:id

// 点赞评教
POST /api/reviews/:id/like

// 取消点赞
DELETE /api/reviews/:id/like
```

### 4. 组件结构

```
ReviewsLatestPage/
├── index.tsx                    // 页面主组件
├── components/
│   ├── ReviewCard.tsx           // 评教卡片组件
│   ├── FilterModal.tsx          // 筛选弹窗组件
│   ├── RatingStars.tsx          // 星级评分组件
│   └── EmptyState.tsx           // 空状态组件
├── hooks/
│   └── useReviewsList.ts        // 评教列表数据 Hook
└── styles/
    └── index.module.scss        // 样式文件
```

### 5. 交互设计

| 交互行为 | 触发方式 | 响应效果 |
|----------|----------|----------|
| 下拉刷新 | 下拉页面 | 显示加载动画，刷新列表 |
| 上拉加载 | 滚动到底部 | 加载更多评教记录 |
| 点击卡片 | 单击 | 跳转到评教详情页 |
| 点击筛选 | 点击筛选图标 | 弹出筛选面板 |
| 点击返回 | 点击返回按钮 | 返回上一页 |
| 点赞 | 点击点赞按钮 | 点赞数+1，图标变色 |

### 6. 性能优化建议

1. **虚拟列表**: 当评教数量较多时，使用虚拟列表优化渲染性能
2. **图片懒加载**: 用户头像采用懒加载策略
3. **数据缓存**: 使用 SWR 或 React Query 进行数据缓存
4. **骨架屏**: 加载时显示骨架屏提升用户体验

## 技术实现要点 (基于 HTML 分析)

### 1. 技术栈
- **前端框架**: React (Create React App)
- **UI 框架**: Bootstrap 4/5
- **图标库**: Font Awesome 6
- **搜索组件**: react-select
- **构建工具**: Webpack (chunk splitting)

### 2. 特殊实现技巧

**CSS 伪元素注入内容**:
```css
/* 评价标题通过 ::after 伪元素注入，防止爬虫抓取 */
#tKpLN7dX33CYbhN9Z::after {
  content: "\7b80\5355\597d\5b66"; /* Unicode: 简单好学 */
}
```

### 3. 路由结构
| 路由 | 说明 |
|------|------|
| `/` | 首页 |
| `/courses/list` | 课程列表 |
| `/courses/view/:id` | 课程详情 |
| `/reviews/latest` | 最新测评 (当前页) |
| `/reviews/post` | 发布测评 |

## 总结

最新测评页面是"非官方课程测评@北京大学"网站的核心功能页面，采用 React + Bootstrap 技术栈实现。

**核心特点**:
1. **四维评分系统** - 从推荐度、内容、工作量、考核四个维度评价课程
2. **表情图标反馈** - 使用 Font Awesome 表情图标直观展示评分
3. **防爬虫设计** - 评价标题通过 CSS `::after` 伪元素注入
4. **警告卡片** - 负面评价使用 `border-warning` 样式突出显示
5. **互动功能** - 支持点赞/踩操作
