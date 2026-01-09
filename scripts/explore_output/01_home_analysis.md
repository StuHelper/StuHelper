# 首页 (01_home) UI 设计与功能分析

## 1. 页面概述

**页面名称**: 非官方课程测评@北京大学 - 首页
**URL**: https://courses.pinzhixiaoyuan.com/
**页面定位**: 课程测评平台的入口页面，提供核心功能导航和快速搜索

---

## 2. 页面结构

### 2.1 整体布局

页面采用**垂直居中布局**，主要内容区域垂直居中显示（`min-height: 88vh`），使用 Bootstrap 的 Flexbox 布局系统。

```
┌─────────────────────────────────────────────┐
│  导航栏 (Navbar)                             │
├─────────────────────────────────────────────┤
│                                             │
│  ┌─────────────────────────────────────┐   │
│  │  Logo + 标题                         │   │
│  │  副标题/统计信息                      │   │
│  │  公告横幅                            │   │
│  │  搜索框                              │   │
│  │  操作按钮                            │   │
│  │  随机测评展示                         │   │
│  └─────────────────────────────────────┘   │
│                                             │
├─────────────────────────────────────────────┤
│  页脚 (Footer)                              │
└─────────────────────────────────────────────┘
```

---

## 3. UI 组件详解

### 3.1 导航栏 (Navbar)

**样式特点**:
- 使用 Bootstrap 的 `navbar navbar-expand navbar-light bg-none`
- 字体大小: `0.9em`
- 无边框设计

**导航项目**:
| 链接文本 | 目标路径 | 说明 |
|---------|---------|------|
| 课程列表 | `/courses/list` | 浏览所有课程 |
| 最新 | `/reviews/latest` | 查看最新测评 |
| 搜索 | `/search` | 搜索页面 |
| **发布测评** | `/reviews/post` | 发布新测评（加粗强调） |
| 关于 | `/about` | 关于页面 |

### 3.2 主标题区域 (Hero Section)

**Logo + 标题**:
```html
<h1>
  <img src="/static/media/logo.66622a8e.png" height="48px">
  <div class="pl-2">课程测评@PKU</div>
</h1>
```
- Logo 高度: 48px
- 使用 Flexbox 水平居中对齐

**副标题/统计信息**:
```html
<h2 class="text-muted" style="line-height: 1.6;">
  来自同学们的<b> 15024 条</b>课程测评,
  帮你找到讲得好、作业少、考试水、给分高的课.
</h2>
```
- 动态显示测评总数（当前: 15024 条）
- 灰色文字 (`text-muted`)
- 行高: 1.6
- 移动端换行 (`<br class="d-md-none">`)

### 3.3 公告横幅 (Banner)

```html
<div id="home-banner" class="mt-2 text-center row">
  <div class="text text-success">
    期末季&元旦快乐! 由于操作失误, 2025-12-30晚上的新测评发布时出现错误, 现在已经修复.
  </div>
</div>
```
- 绿色文字 (`text-success`)
- 居中显示
- 用于展示系统公告或通知

### 3.4 搜索框 (Search Box)

**技术实现**: 使用 `react-select` 组件

```html
<div id="home-search" class="mt-2 row">
  <div class="css-b62m3t-container">
    <!-- react-select 组件 -->
    <input
      id="react-select-2-input"
      type="text"
      role="combobox"
      aria-autocomplete="list"
      placeholder="搜索课程名称"
    >
    <!-- 搜索图标 (Font Awesome) -->
    <svg class="fa-magnifying-glass">...</svg>
  </div>
</div>
```

**功能特点**:
- 支持自动补全 (`aria-autocomplete="list"`)
- 下拉选择框样式
- 右侧显示搜索图标
- 无障碍支持 (ARIA 属性)

### 3.5 操作按钮 (Action Buttons)

```html
<div id="home-buttons" class="mt-4 row">
  <a href="/courses/list" class="mr-2 btn-lg btn btn-primary">
    <svg class="fa-list-ul">...</svg> 浏览课程
  </a>
  <a href="/reviews/post" class="btn-lg btn btn-info">
    <svg class="fa-pencil">...</svg> 发布测评
  </a>
</div>
```

| 按钮 | 样式 | 图标 | 链接 |
|-----|------|------|------|
| 浏览课程 | `btn-primary` (蓝色) | `fa-list-ul` | `/courses/list` |
| 发布测评 | `btn-info` (青色) | `fa-pencil` | `/reviews/post` |

### 3.6 随机测评展示 (Random Review)

**布局**: 使用 `blockquote` 引用样式

```html
<blockquote class="blockquote mx-auto text-left w-75 border-left border-info pl-3 mt-3">
  <!-- 测评内容摘要 -->
  <div class="mb-1" style="font-size: 0.8em; line-height: 1.7;">
    课程听感:做的实验比较接近高中教材...
  </div>

  <!-- 评分指标 -->
  <div style="font-size: 0.7em;">
    <b>是否推荐?</b> 😟 ·
    <b>课程内容</b> 😐 ·
    <b>工作量</b> 😟 ·
    <b>考核</b> 😐
  </div>

  <!-- 来源信息 -->
  <footer class="blockquote-footer text-right">
    <cite>"相当麻烦的课，应该不会有非必修但是选到了的吧"</cite>
    <a href="/courses/view/179">普通化学实验（B）</a> (徐怡庄老师)
  </footer>
</blockquote>
```

**评分图标系统**:
| 图标 | 含义 | CSS 类 |
|------|------|--------|
| 😟 `fa-face-frown` | 不推荐/差 | `text-warning` |
| 😐 `fa-face-meh` | 一般 | 无特殊颜色 |
| 😊 `fa-face-smile` | 推荐/好 | (推测) |

**评分维度**:
1. 是否推荐
2. 课程内容
3. 工作量
4. 考核难度

### 3.7 页脚 (Footer)

```html
<div class="mx-auto my-2 p-3 text-muted" style="font-size: 0.8em;">
  <hr>
  © 2018-2025 <b>非官方课程测评@PKU</b>
  <a href="/faq">FAQ (常见问答)</a> · <a href="/about">关于</a>
  <small>本网站为北京大学的课程提供学生之间的测评分享, 与学校官方机构无关.</small>
  <small>前端 2.5.1201.endor_hotfix.250915-0324, 后端 2512.31.main</small>
</div>
```

**包含信息**:
- 版权声明 (2018-2025)
- FAQ 和关于页面链接
- 免责声明
- 版本信息（前端/后端）

---

## 4. 技术栈分析

### 4.1 前端框架
- **React**: 基于 webpack 打包，使用 chunk 分割
- **Bootstrap**: 响应式布局和组件样式
- **Font Awesome 6**: 图标库
- **react-select**: 搜索下拉组件
- **Emotion CSS**: CSS-in-JS 方案

### 4.2 关键依赖
```javascript
// 从 HTML 中提取的资源
/static/css/main.54b383a4.chunk.css
/static/js/2.123bc1d6.chunk.js
/static/js/main.67b6f6bd.chunk.js
```

### 4.3 第三方服务
- **Google Analytics**: `G-CR5GMPGQ6G` 用于流量统计

---

## 5. 响应式设计

### 5.1 断点处理
- 使用 Bootstrap 的响应式类
- `d-md-none`: 中等屏幕以上隐藏
- `d-block d-sm-none`: 小屏幕显示

### 5.2 移动端适配
```html
<meta name="viewport" content="width=device-width,initial-scale=1,maximum-scale=1,user-scalable=no">
```
- 禁止用户缩放
- 初始缩放比例 1:1

---

## 6. 功能实现要点

### 6.1 核心功能
1. **课程搜索**: 首页搜索框支持课程名称搜索，带自动补全
2. **快速导航**: 两个主要 CTA 按钮引导用户浏览或发布
3. **随机展示**: 展示随机测评吸引用户参与
4. **统计展示**: 动态显示测评总数增加可信度

### 6.2 数据接口 (推测)
- 获取测评总数
- 获取随机测评
- 课程搜索自动补全

### 6.3 路由结构
```
/                    - 首页
/courses/list        - 课程列表
/courses/view/:id    - 课程详情
/reviews/latest      - 最新测评
/reviews/post        - 发布测评
/search              - 搜索页面
/about               - 关于页面
/faq                 - 常见问答
```

---

## 7. 设计亮点

1. **简洁明了**: 首页信息层次清晰，核心功能突出
2. **数据驱动**: 展示测评数量增加用户信任
3. **社区感**: 随机测评展示营造活跃社区氛围
4. **无障碍**: 使用 ARIA 属性支持屏幕阅读器
5. **PWA 支持**: 包含 manifest.json 和 apple-touch-icon

---

## 8. 改进建议

1. **搜索体验**: 可增加热门搜索词展示
2. **个性化**: 可根据用户院系推荐相关课程
3. **数据可视化**: 可增加课程评分分布图表
4. **深色模式**: 当前未见深色模式支持
