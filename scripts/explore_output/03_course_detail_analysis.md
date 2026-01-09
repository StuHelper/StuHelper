# 课程详情页 (03_course_detail) UI 设计与功能分析

## 1. 页面概述

**页面名称**: 偏微分方程 - 非官方课程测评@北京大学
**URL**: https://courses.pinzhixiaoyuan.com/courses/view/1033
**页面定位**: 展示单个课程的详细信息、评分统计和用户测评

---

## 2. 页面结构

### 2.1 整体布局

页面采用**双栏布局**（桌面端），左侧为课程列表侧边栏，右侧为课程详情主内容区。

```
┌─────────────────────────────────────────────────────────────────────┐
│  顶部导航栏 (Sticky Navbar - 深色主题)                                │
├─────────────────────────────────────────────────────────────────────┤
│  移动端搜索栏 (仅移动端显示)                                          │
├─────────────────────────────────────────────────────────────────────┤
│  ┌──────────────────┬──────────────────────────────────────────┐   │
│  │  左侧边栏         │  右侧主内容区                              │   │
│  │  (col-xl-3)      │  (col-xl-9)                              │   │
│  │                  │                                          │   │
│  │  分类标签页       │  课程标题 + 院系/学分徽章                   │   │
│  │  院系卡片列表     │  评分卡片 (按学期统计)                      │   │
│  │  (当前院系高亮)   │  授课教师筛选                              │   │
│  │                  │  测评卡片列表                              │   │
│  │                  │  分页组件                                  │   │
│  └──────────────────┴──────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────────────┤
│  页脚 (Footer)                                                      │
└─────────────────────────────────────────────────────────────────────┘
```

### 2.2 响应式断点

| 断点 | 左侧边栏 | 右侧内容 |
|-----|---------|---------|
| xl (≥1200px) | `col-xl-3` | `col-xl-9` |
| md (≥768px) | `col-md-4` | `col-md-8` |
| < md | 隐藏 (`d-none d-md-block`) | `col-sm-12` |

---

## 3. UI 组件详解

### 3.1 顶部导航栏

与课程列表页相同，使用深色主题 (`navbar-dark bg-dark sticky-top`)。

**导航按钮组**:
```html
<div role="group" class="btn-group">
  <a href="/courses/list" class="btn btn-secondary">课程</a>
  <a href="/reviews/latest" class="btn btn-secondary">最新</a>
  <a href="/reviews/post" class="btn btn-info">发测评</a>
</div>
```

### 3.2 左侧边栏 - 课程列表

**特点**:
- 仅在桌面端显示 (`d-none d-md-block`)
- 包含分类标签页和院系卡片列表
- 当前课程所属院系卡片高亮显示

**当前院系高亮样式**:
```html
<!-- 普通院系卡片 -->
<div class="mt-2 card border-secondary">
  <div class="p-2 text-light bg-lightdark card-header">...</div>
</div>

<!-- 当前院系卡片 (高亮) -->
<div class="mt-2 card border-primary">
  <div class="p-2 text-light bg-primary card-header">
    <b>数学科学学院</b>
  </div>
  <div class="list-group-flush list-group">
    <!-- 课程列表展开显示 -->
  </div>
</div>
```

**当前课程高亮**:
```html
<a aria-current="page" class="p-2 list-group-item list-group-item-action active"
   href="/courses/view/1033">
  偏微分方程 <small class="text-muted">3 学分</small>
  <small class="float-right text-info">30 条测评</small>
</a>
```

### 3.3 课程标题区

```html
<div class="d-flex justify-content-between align-items-center">
  <div class="mr-auto">
    <h3 class="d-inline-block">偏微分方程</h3>
  </div>
  <h4>
    <span class="mr-2 d-none d-md-inline-block badge badge-primary">数学科学学院</span>
    <span class="mr-2 d-none d-md-inline-block badge badge-secondary">3 学分</span>
    <a href="/reviews/post/1033" class="d-none d-md-inline-block btn btn-success btn-sm">
      <svg class="fa-circle-plus">...</svg> 测评
    </a>
  </h4>
</div>

<!-- 移动端徽章 (单独显示) -->
<div class="d-block d-md-none">
  <span class="mr-2 badge badge-primary">数学科学学院</span>
  <span class="mr-2 badge badge-secondary">3 学分</span>
</div>
```

**元素说明**:
| 元素 | 样式 | 说明 |
|-----|------|------|
| 课程名称 | `h3` | 主标题 |
| 院系徽章 | `badge badge-primary` | 蓝色背景 |
| 学分徽章 | `badge badge-secondary` | 灰色背景 |
| 发测评按钮 | `btn btn-success btn-sm` | 绿色小按钮 |

### 3.4 评分卡片 (Rating Card)

**整体结构**:
```html
<div class="mt-2 card border-primary">
  <div class="p-1 card-body">
    <div class="px-2 py-1 text-primary mb-0 card-title h5"><b>评分 ></b></div>
    <!-- 评分内容 -->
  </div>
</div>
```

**评分维度**:
| 维度 | 说明 |
|-----|------|
| 推荐指数 | 是否推荐选修 |
| 内容质量 | 课程内容评价 |
| 工作量 | 作业/任务量 |
| 考核 | 考试难度/给分 |

**进度条颜色系统**:
```html
<!-- 高分 (>60%) -->
<div class="progress-bar bg-primary" style="width: 90%;"></div>

<!-- 中等 (30-60%) -->
<div class="progress-bar bg-warning" style="width: 50%;"></div>

<!-- 低分 (<30%) -->
<div class="progress-bar bg-danger" style="width: 3%;"></div>
```

**学期评分卡片**:
```html
<div class="py-2 col-md-4">
  <div class="ratingContainer">
    <h6>
      <b>全部学期</b>
      <span class="badge badge-info">30</span>  <!-- 测评数量 -->
    </h6>
    <div class="row">
      <div class="col-md-6">
        <small class="text-muted"><b>推荐指数</b></small>
        <div class="progress" style="height: 0.5em;">
          <div class="progress-bar bg-warning" style="width: 50%;"></div>
        </div>
      </div>
      <!-- 其他维度... -->
    </div>
  </div>
</div>
```

**数据不足提示**:
```html
<span class="badge badge-warning">数据较少</span>
<small class="text-muted text-center d-block">过少的数据没有统计代表性.</small>
```

### 3.5 授课教师筛选

```html
<div class="px-2 py-1 text-primary mb-1 card-title h5"><b>授课教师 ></b></div>
<div class="px-2 mb-2 courseViewTeacherLinks">
  <a href="/courses/view/1033">
    <span class="badge badge-pill badge-secondary">所有</span>
  </a>
  <a href="/courses/view/1033?teacher=3yZcM4HA25kk6">
    <span class="badge badge-pill badge-info">齐元伟</span>
  </a>
  <a href="/courses/view/1033?teacher=3x2yHihokPgpV">
    <span class="badge badge-pill badge-info">章志飞</span>
  </a>
  <!-- 更多教师... -->
</div>
```

**筛选机制**:
- 点击教师徽章可筛选该教师的测评
- URL 参数: `?teacher={teacherId}`
- "所有" 显示全部测评

### 3.6 测评卡片 (Review Card)

**卡片结构**:
```html
<div class="mt-2 card border-none">  <!-- 或 border-danger/border-warning -->
  <div class="p-1 card-body">
    <!-- 标题 (CSS 伪元素生成) -->
    <div class="px-2 pt-2 mb-1 card-title h5" id="t29oWw5EK"></div>
    <style>#t29oWw5EK::after { content: "简单好学"; }</style>

    <!-- 元信息 -->
    <div class="px-2 pt-2 mb-1 text-muted card-subtitle h6">
      <strong>周蜀林</strong>老师 (25-26-1 学期), 2026/01/09
    </div>

    <!-- 测评内容 -->
    <div class="px-2 pb-3">
      <div class="row">
        <div class="col" style="font-size: 0.9em; line-height: 1.7;">
          <!-- 测评正文 -->
          <br><strong>成绩:</strong> 99
        </div>
      </div>
    </div>
  </div>

  <!-- 卡片底部 -->
  <div class="p-2 card-footer">...</div>
</div>
```

**卡片边框颜色系统**:
| 边框类 | 含义 | 使用场景 |
|-------|------|---------|
| `border-none` | 无边框 | 正面/中性评价 |
| `border-warning` | 黄色边框 | 中等评价 |
| `border-danger` | 红色边框 | 负面评价 |

**评分表情图标系统**:
| 图标 | CSS类 | 含义 | 颜色 |
|------|------|------|------|
| 😍 `fa-face-grin-hearts` | `text-success` | 非常推荐 | 绿色 |
| 😊 `fa-face-smile` | `text-success` | 推荐 | 绿色 |
| 😐 `fa-face-meh` | 无 | 一般 | 默认 |
| 😟 `fa-face-frown` | `text-warning` | 不太推荐 | 黄色 |
| 😭 `fa-face-sad-cry` | `text-danger` | 不推荐 | 红色 |

**卡片底部 (Footer)**:
```html
<div class="p-2 card-footer" style="font-size: 0.9em;">
  <!-- 左侧: 评分指标 -->
  <div class="float-left">
    <b>是否推荐?</b> <svg class="fa-face-grin-hearts text-success">...</svg> ·
    <b>课程内容</b> <svg class="fa-face-smile text-success">...</svg> ·
    <b>工作量</b> <svg class="fa-face-smile text-success">...</svg> ·
    <b>考核</b> <svg class="fa-face-grin-hearts text-success">...</svg>
  </div>

  <!-- 右侧: 点赞/踩按钮 -->
  <div class="float-right">
    <button class="btn btn-sm btn-success">
      <svg class="fa-thumbs-up">...</svg>
    </button> 5
    <button class="btn btn-sm btn-danger">
      <svg class="fa-thumbs-down">...</svg>
    </button>
  </div>
</div>
```

### 3.7 分页组件 (Pagination)

```html
<ul class="pagination" role="navigation" aria-label="Pagination">
  <li class="page-item disabled">
    <a class="page-link" aria-label="Previous page" rel="prev">&lt;</a>
  </li>
  <li class="page-item active">
    <a class="page-link" aria-current="page">1</a>
  </li>
  <li class="page-item">
    <a class="page-link" aria-label="Page 2">2</a>
  </li>
  <li class="page-item">
    <a class="page-link" aria-label="Next page" rel="next">&gt;</a>
  </li>
</ul>
```

**特点**:
- Bootstrap 分页样式
- 无障碍支持 (ARIA 属性)
- 当前页高亮 (`active`)
- 首页时禁用上一页 (`disabled`)

---

## 4. 数据结构分析

### 4.1 课程数据

| 字段 | 示例值 | 说明 |
|-----|--------|------|
| courseId | 1033 | 课程唯一标识 |
| courseName | 偏微分方程 | 课程名称 |
| department | 数学科学学院 | 所属院系 |
| credits | 3 | 学分 |
| reviewCount | 30 | 测评总数 |

### 4.2 测评数据

| 字段 | 示例值 | 说明 |
|-----|--------|------|
| reviewId | t29oWw5EK | 测评唯一标识 |
| title | 简单好学 | 测评标题 |
| teacher | 周蜀林 | 授课教师 |
| semester | 25-26-1 | 学期 |
| date | 2026/01/09 | 发布日期 |
| grade | 99 | 成绩 |
| ratings | {recommend, content, workload, exam} | 四维评分 |
| upvotes | 1 | 点赞数 |

### 4.3 URL 结构

```
/courses/view/{courseId}              - 课程详情页
/courses/view/{courseId}?teacher={id} - 按教师筛选
/reviews/post/{courseId}              - 发布该课程测评
```

---

## 5. 技术实现要点

### 5.1 CSS 伪元素标题

**反爬虫技术**: 测评标题使用 CSS `::after` 伪元素生成，而非直接在 HTML 中显示。

```html
<div class="card-title h5" id="t29oWw5EK"></div>
<style>#t29oWw5EK::after { content: "简单好学"; }</style>
```

**目的**:
- 防止简单的 HTML 爬虫抓取标题内容
- 标题内容存储在 CSS 中，需要解析 CSS 才能获取

### 5.2 打印保护

```html
<style type="text/css">
  @media print { body { display: none } }
</style>
```

**目的**: 防止用户直接打印页面内容。

### 5.3 双栏响应式布局

```html
<div class="row-animate row">
  <!-- 左侧边栏 - 桌面端显示 -->
  <div class="d-none d-md-block col-xl-3 col-md-4">...</div>

  <!-- 右侧主内容 -->
  <div class="col-xl-9 col-md-8 col-sm-12">...</div>
</div>
```

---

## 6. 交互流程

### 6.1 查看课程测评

```
1. 从课程列表点击课程名称
   ↓
2. 进入课程详情页
   ↓
3. 查看评分统计 (按学期)
   ↓
4. 可选: 按教师筛选测评
   ↓
5. 浏览测评列表
   ↓
6. 可选: 点赞/踩测评
```

### 6.2 发布测评

```
1. 点击 "发测评" 按钮
   ↓
2. 跳转到 /reviews/post/{courseId}
```

---

## 7. 设计亮点

1. **双栏布局**: 左侧课程导航 + 右侧详情，便于快速切换课程
2. **当前课程高亮**: 左侧边栏中当前课程使用 `active` 样式突出显示
3. **多维度评分**: 四个维度全面评价课程质量
4. **学期分组统计**: 按学期展示评分趋势，便于了解课程变化
5. **教师筛选**: 支持按授课教师筛选测评
6. **表情图标系统**: 直观的五级评分表情，一目了然
7. **点赞机制**: 用户可对测评进行点赞/踩，提升内容质量
8. **反爬虫保护**: CSS 伪元素标题 + 打印保护

---

## 8. 与其他页面的对比

| 特性 | 首页 | 课程列表页 | 课程详情页 |
|-----|------|----------|-----------|
| 布局 | 单列居中 | 单列居中 | 双栏布局 |
| 导航栏 | 浅色透明 | 深色固定 | 深色固定 |
| 主要功能 | 引导入口 | 浏览筛选 | 详情展示 |
| 内容密度 | 低 | 中 | 高 |
| 侧边栏 | 无 | 无 | 有 (桌面端) |

---

## 9. 改进建议

1. **测评排序**: 支持按时间、点赞数、评分排序
2. **测评搜索**: 在测评列表中搜索关键词
3. **图表可视化**: 用图表展示评分趋势
4. **收藏功能**: 允许用户收藏课程
5. **分享功能**: 一键分享课程链接
6. **移动端侧边栏**: 添加抽屉式侧边栏
7. **锚点定位**: 支持直接跳转到特定测评
8. **测评举报**: 添加举报不当内容功能
