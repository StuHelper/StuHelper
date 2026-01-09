# 05 高级搜索页面分析文档

## 页面概述

该页面是"高级搜索"功能模块，属于"**非官方课程测评@北京大学**"网站的核心搜索功能。页面提供多维度搜索条件组合，支持按课程条件和测评条件进行精确搜索。

- **页面标题**: 高级搜索 - 非官方课程测评@北京大学
- **技术栈**: React + Bootstrap + Font Awesome 6
- **版本信息**: 前端 2.5.1201.endor_hotfix, 后端 2512.31.main

## UI 设计分析

### 1. 页面结构

```
┌──────────────────────────────────────────────────────────┐
│  顶部导航栏 (navbar navbar-dark bg-dark sticky-top)       │
│  [Logo] 课程测评  |  [搜索框]  |  [课程] [最新] [发测评]  │
├──────────────────────────────────────────────────────────┤
│  [←] 高级搜索                                             │
│  说明文字: 自由组合以下搜索条件以搜索测评...              │
│                                                          │
│  ┌─────────────────────┐  ┌─────────────────────┐       │
│  │ 按课程条件 (蓝色头)  │  │ 按测评条件 (蓝色头)  │       │
│  │ ├─ 课程名称         │  │ ├─ 教师姓名         │       │
│  │ └─ 或 课程编号      │  │ └─ 学期选择         │       │
│  └─────────────────────┘  └─────────────────────┘       │
│                                                          │
│  [搜索] 按钮 (btn-primary btn-block)                     │
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
| 发测评按钮 | `btn-info` | 图标 `fa-pencil`，链接 `/reviews/post` |

### 3. 页面标题区域

```html
<h3>
  <a href="/"><svg class="fa-chevron-left">...</svg></a>
  高级搜索
</h3>
<p>自由组合以下搜索条件以搜索测评, 或者试试顶栏中较为简单的按课程搜索.</p>
```

- 左箭头图标 (`fa-chevron-left`) 点击返回首页
- 说明文字引导用户使用搜索功能

### 4. 搜索条件卡片

页面采用两列布局 (`row`)，左右各一个搜索条件卡片：

#### 4.1 按课程条件卡片 (左侧)

```html
<div class="card mb-2">
  <div class="card-header p-2 bg-primary text-white">按课程条件</div>
  <div class="card-body p-2">...</div>
</div>
```

| 字段 | 类型 | placeholder | 说明 |
|------|------|-------------|------|
| 课程名称 | `input[type="text"]` | 西方音乐欣赏 | 支持中英文及拼音模糊搜索 |
| 课程编号 | `input[type="text"]` | 00432140 | 学校内部课程编号，一般为八位数 |

**特点**: 两个字段为"或"关系，用 `<strong>或</strong>` 标签强调

#### 4.2 按测评条件卡片 (右侧)

| 字段 | 类型 | placeholder | 说明 |
|------|------|-------------|------|
| 教师姓名 | `input[type="text"]` + 后缀"老师" | 孟策 | 支持模糊搜索 |
| 学期 | `select` 下拉框 | - | 可选择任意学期或特定学期 |

**学期选项** (使用 `<optgroup>` 分组，追溯至 2015-2016年)

### 5. 搜索按钮

```html
<button type="submit" class="btn btn-primary btn-block">搜索</button>
```

- 使用 `btn-block` 实现全宽按钮
- 蓝色主题色 (`btn-primary`)

## 功能实现分析

### 1. 搜索条件组合逻辑

搜索采用 **AND** 逻辑组合各条件：

```
(课程名称 OR 课程编号) AND 教师姓名 AND 学期
```

### 2. 数据结构设计

```typescript
interface SearchParams {
  course_name?: string;        // 课程名称
  course_internal_id?: string; // 课程编号
  teacher_name?: string;       // 教师姓名
  term_id?: string;            // 学期ID (如: "25-26-1")
}
```

### 3. API 接口设计

```typescript
// 高级搜索
GET /api/reviews/search
Query Parameters:
  - course_name?: string
  - course_internal_id?: string
  - teacher_name?: string
  - term_id?: string
  - page?: number
  - pageSize?: number
```

### 4. 组件结构

```
AdvancedSearchPage/
├── index.tsx                    // 页面主组件
├── components/
│   ├── CourseConditionCard.tsx  // 课程条件卡片
│   ├── ReviewConditionCard.tsx  // 测评条件卡片
│   └── TermSelect.tsx           // 学期选择器
└── hooks/
    └── useSearch.ts             // 搜索逻辑 Hook
```

## 总结

高级搜索页面是课程测评系统的核心搜索入口，提供比顶栏简单搜索更强大的多条件组合功能。

**核心特点**:
1. **双卡片布局** - 课程条件与测评条件分离，逻辑清晰
2. **多维度搜索** - 支持课程名称、编号、教师、学期组合
3. **模糊搜索** - 课程名称支持中英文及拼音模糊匹配
4. **学期分组** - 使用 `<optgroup>` 按年份分组，便于选择
5. **响应式设计** - 两列布局在移动端自动堆叠
