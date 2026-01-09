# 课程列表页 (02_courses_list) UI 设计与功能分析

## 1. 页面概述

**页面名称**: 课程列表 - 非官方课程测评@北京大学
**URL**: https://courses.pinzhixiaoyuan.com/courses/list
**页面定位**: 按院系分类浏览所有课程，支持分类筛选和快速定位

---

## 2. 页面结构

### 2.1 整体布局

页面采用**单列居中布局**，最大宽度 800px，使用 Bootstrap 的容器系统。

```
┌─────────────────────────────────────────────────────────────┐
│  顶部导航栏 (Sticky Navbar)                                  │
├─────────────────────────────────────────────────────────────┤
│  移动端搜索栏 (仅移动端显示)                                   │
├─────────────────────────────────────────────────────────────┤
│  ┌─────────────────────────────────────────────────────┐   │
│  │  分类标签页 (全校/通选/体育/英语/思政)                  │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  展开/收起按钮组                                      │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  院系卡片 1 (可折叠)                                  │   │
│  │    └─ 课程列表                                       │   │
│  ├─────────────────────────────────────────────────────┤   │
│  │  院系卡片 2 (可折叠)                                  │   │
│  │    └─ 课程列表                                       │   │
│  │  ...                                                │   │
│  └─────────────────────────────────────────────────────┘   │
├─────────────────────────────────────────────────────────────┤
│  页脚 (Footer)                                              │
└─────────────────────────────────────────────────────────────┘
```

---

## 3. UI 组件详解

### 3.1 顶部导航栏 (Sticky Navbar)

**与首页导航栏的区别**:
- 使用深色主题 (`navbar-dark bg-dark`)
- 固定在顶部 (`sticky-top`)
- 包含 Logo 和品牌名称
- 集成搜索框（桌面端）

```html
<nav class="navbar navbar-expand navbar-dark bg-dark sticky-top">
  <a href="/" class="active navbar-brand">
    <img src="/static/media/logo.66622a8e.png" width="24" height="24">
    <span>课程测评</span>
  </a>

  <!-- 桌面端搜索框 -->
  <form id="top-search" class="d-none d-md-block form-inline">
    <!-- react-select 搜索组件 -->
  </form>

  <!-- 导航按钮组 -->
  <form class="ml-2 form-inline">
    <div class="btn-group">
      <a href="/courses/list" class="active btn btn-secondary">课程</a>
      <a href="/reviews/latest" class="btn btn-secondary">最新</a>
      <a href="/reviews/post" class="btn btn-info">发测评</a>
    </div>
  </form>
</nav>
```

**导航按钮**:
| 按钮 | 样式 | 图标 | 状态 |
|-----|------|------|------|
| 课程 | `btn-secondary` | `fa-list-ul` | 当前激活 |
| 最新 | `btn-secondary` | `fa-clock` | 普通 |
| 发测评 | `btn-info` | `fa-pencil` | 强调 |

### 3.2 移动端搜索栏

```html
<div class="mx-auto mt-2 d-block d-md-none" id="mobileTopSearchBar">
  <div class="container-fluid">
    <!-- react-select 搜索组件 -->
  </div>
</div>
```

**响应式策略**:
- `d-block d-md-none`: 仅在小于 md 断点时显示
- 桌面端搜索框在导航栏内
- 移动端搜索框独立显示在导航栏下方

### 3.3 分类标签页 (Tab Navigation)

```html
<div class="mb-2 nav nav-tabs nav-fill">
  <div class="nav-item">
    <a class="nav-link active" href="/courses/list">全校</a>
  </div>
  <div class="nav-item">
    <a class="nav-link" href="/courses/list/elective">通选</a>
  </div>
  <div class="nav-item">
    <a class="nav-link" href="/courses/list/pe">体育</a>
  </div>
  <div class="nav-item">
    <a class="nav-link" href="/courses/list/english">英语</a>
  </div>
  <div class="nav-item">
    <a class="nav-link" href="/courses/list/pols">思政</a>
  </div>
</div>
```

**分类路由**:
| 标签 | 路径 | 说明 |
|-----|------|------|
| 全校 | `/courses/list` | 所有院系课程 |
| 通选 | `/courses/list/elective` | 通识选修课 |
| 体育 | `/courses/list/pe` | 体育课程 |
| 英语 | `/courses/list/english` | 英语课程 |
| 思政 | `/courses/list/pols` | 思想政治课程 |

**样式特点**:
- `nav-fill`: 标签等宽填充
- `nav-tabs`: 标签页样式
- 当前激活项使用 `active` 类

### 3.4 展开/收起按钮组

```html
<div role="group" class="mb-2 d-flex btn-group">
  <button type="button" class="btn btn-secondary">
    <svg class="fa-chevron-down">...</svg> 展开全部
  </button>
  <button type="button" class="btn btn-secondary">
    <svg class="fa-chevron-up">...</svg> 收起全部
  </button>
</div>
```

**功能**:
- 一键展开所有院系卡片
- 一键收起所有院系卡片
- 使用 Font Awesome 箭头图标指示方向

### 3.5 院系卡片 (Department Card)

```html
<div class="mt-2 card border-secondary">
  <!-- 卡片头部 - 可点击折叠 -->
  <div class="p-2 text-light bg-lightdark card-header" style="cursor: pointer;">
    <b>马克思主义学院</b>
    <div class="float-right my-auto">
      <svg class="fa-angle-up">...</svg>  <!-- 展开/收起指示器 -->
    </div>
  </div>

  <!-- 课程列表 -->
  <div class="list-group-flush list-group">
    <!-- 课程项目 -->
  </div>
</div>
```

**交互特点**:
- 卡片头部可点击，触发折叠/展开
- `cursor: pointer` 指示可点击
- 右侧箭头图标指示当前状态（`fa-angle-up` 表示已展开）
- 使用深色背景 (`bg-lightdark`) 区分头部

### 3.6 课程列表项 (Course Item)

```html
<a class="p-2 list-group-item list-group-item-action" href="/courses/view/2907">
  马克思主义理论导论
  <small class="text-muted">3 学分</small>
  <small class="float-right text-info">5 条测评</small>
</a>
```

**信息展示**:
| 元素 | 位置 | 样式 | 说明 |
|-----|------|------|------|
| 课程名称 | 左侧 | 默认 | 主要信息 |
| 学分 | 课程名后 | `text-muted` 灰色 | 次要信息 |
| 测评数 | 右侧浮动 | `text-info` 青色 | 吸引点击 |

**交互**:
- `list-group-item-action`: 悬停效果
- 整行可点击，链接到课程详情页

### 3.7 虚拟滚动占位符

```html
<div><div class="mt-2" style="height: 46px;"></div></div>
<!-- 重复多个 -->
```

**技术说明**:
- 页面使用虚拟滚动优化性能
- 未进入视口的院系卡片使用固定高度占位符
- 高度 46px 对应折叠状态的卡片头部高度
- 滚动到视口时才渲染实际内容

### 3.8 底部提示

```html
<div class="text-muted text-center my-3">
  找不到想要的课程?
  <a href="/search">高级搜索</a> 或者
  <a href="/reviews/post">发布测评</a>
</div>
```

**功能**:
- 引导用户使用高级搜索
- 鼓励用户发布新课程测评

---

## 4. 数据结构分析

### 4.1 课程数据示例 (马克思主义学院)

从 HTML 中提取的课程数据：

| 课程名称 | 学分 | 测评数 | 课程ID |
|---------|------|--------|--------|
| 马克思主义理论导论 | 3 | 5 | 2907 |
| 政治经济学 | 4 | 5 | 2193 |
| 科学社会主义 | 2 | 1 | 4089 |
| 中国化马克思主义 | 3 | 1 | 4088 |
| 习近平新时代中国特色社会主义重要思想概论 | 3 | 48 | 4053 |
| 思想道德修养与法律基础 | 2 | 58 | 34 |
| 中国近现代史纲要 | 2 | 65 | 44 |
| 中国近现代史纲要 | 3 | 169 | 2395 |
| 毛泽东思想和中国特色社会主义理论体系概论 | 3 | 85 | 3646 |
| 马克思主义基本原理概论 | 3 | 122 | 24 |

**观察**:
- 同名课程可能有不同学分版本（如"中国近现代史纲要"有2学分和3学分版本）
- 测评数差异大（1条到169条不等）
- 课程ID为数字，用于构建详情页URL

### 4.2 URL 结构

```
/courses/view/{courseId}  - 课程详情页
/courses/list             - 全校课程列表
/courses/list/elective    - 通选课列表
/courses/list/pe          - 体育课列表
/courses/list/english     - 英语课列表
/courses/list/pols        - 思政课列表
```

---

## 5. 技术实现要点

### 5.1 虚拟滚动 (Virtual Scrolling)

**实现原理**:
- 只渲染视口内的院系卡片
- 视口外使用固定高度占位符
- 滚动时动态替换内容
- 大幅减少 DOM 节点数量

**优势**:
- 支持大量院系和课程数据
- 保持页面流畅滚动
- 减少内存占用

### 5.2 折叠组件

**状态管理**:
- 每个院系卡片独立管理展开/收起状态
- "展开全部"/"收起全部"批量操作
- 使用 React 状态控制

**动画**:
- 箭头图标旋转指示状态
- 可能使用 CSS transition 或 React 动画库

### 5.3 搜索功能

**双搜索框设计**:
```
桌面端: 导航栏内嵌搜索框 (d-none d-md-block)
移动端: 独立搜索栏 (d-block d-md-none)
```

**react-select 配置**:
- 自动补全功能
- 异步加载搜索结果
- 自定义下拉指示器（搜索图标）

---

## 6. 响应式设计

### 6.1 断点策略

| 断点 | 搜索框位置 | 布局调整 |
|-----|----------|---------|
| < md (768px) | 独立搜索栏 | 全宽显示 |
| ≥ md | 导航栏内 | 居中容器 |

### 6.2 容器宽度

```css
.container {
  max-width: 800px;
}
```

- 内容区域最大宽度 800px
- 保持阅读舒适度
- 移动端自适应全宽

---

## 7. 交互流程

### 7.1 用户浏览流程

```
1. 进入课程列表页
   ↓
2. 选择分类标签 (全校/通选/体育/英语/思政)
   ↓
3. 滚动查找目标院系
   ↓
4. 点击院系卡片展开课程列表
   ↓
5. 点击课程进入详情页
```

### 7.2 搜索流程

```
1. 在搜索框输入课程名称
   ↓
2. 自动补全显示匹配结果
   ↓
3. 选择目标课程
   ↓
4. 跳转到课程详情页
```

---

## 8. 设计亮点

1. **虚拟滚动优化**: 处理大量院系数据时保持性能
2. **双搜索框设计**: 桌面端和移动端各有最佳体验
3. **折叠卡片**: 有效组织大量课程信息
4. **分类标签**: 快速筛选特定类型课程
5. **测评数展示**: 帮助用户判断课程热度
6. **Sticky 导航**: 随时可用的搜索和导航

---

## 9. 与首页的对比

| 特性 | 首页 | 课程列表页 |
|-----|------|----------|
| 导航栏样式 | 浅色透明 | 深色固定 |
| 搜索框位置 | 页面中央 | 导航栏/独立栏 |
| 主要功能 | 引导入口 | 内容浏览 |
| 布局 | 垂直居中 | 顶部对齐 |
| 内容密度 | 低 | 高 |

---

## 10. 改进建议

1. **搜索增强**: 支持按院系、学分、测评数筛选
2. **排序功能**: 按测评数、学分等排序课程
3. **收藏功能**: 允许用户收藏感兴趣的课程
4. **加载状态**: 虚拟滚动时显示加载指示器
5. **键盘导航**: 支持键盘快捷键浏览
6. **面包屑导航**: 显示当前分类路径
