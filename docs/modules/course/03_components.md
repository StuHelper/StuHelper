# 前端组件设计

本文档定义评课社区模块的前端组件结构和设计规范。

## 技术栈

- **框架**: Uni-app (Vue3)
- **UI 库**: uView UI / uni-ui
- **图标**: Font Awesome 6
- **状态管理**: Pinia

## 组件目录结构

```
src/pages/course-review/
├── components/           # 通用组件
│   ├── CourseCard.vue
│   ├── ReviewCard.vue
│   ├── RatingGroup.vue
│   ├── SearchBar.vue
│   └── DepartmentList.vue
├── pages/               # 页面组件
│   ├── home/
│   ├── courses/
│   ├── course-detail/
│   ├── reviews-latest/
│   ├── post-review/
│   └── search/
└── stores/              # 状态管理
    └── courseReview.ts
```

## 核心组件

### 1. ReviewCard 测评卡片

显示单条测评信息。

**Props**：

| 属性 | 类型 | 说明 |
|------|------|------|
| review | Review | 测评数据 |
| showCourse | boolean | 是否显示课程名 |

### 2. RatingGroup 评分组

四维评分选择器，使用表情图标。

**Props**：

| 属性 | 类型 | 说明 |
|------|------|------|
| modelValue | number | 当前值 (-2~2) |
| label | string | 标签文字 |

### 3. DepartmentList 院系列表

可折叠的院系课程列表。

**Props**：

| 属性 | 类型 | 说明 |
|------|------|------|
| departments | Department[] | 院系数据 |
| expandAll | boolean | 是否全部展开 |

### 4. SearchBar 搜索栏

课程搜索组件，支持异步加载。

**Props**：

| 属性 | 类型 | 说明 |
|------|------|------|
| placeholder | string | 占位文字 |

**Events**：

| 事件 | 参数 | 说明 |
|------|------|------|
| select | Course | 选中课程 |
