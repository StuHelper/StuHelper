# API Responses JSON Schema

本文档描述了 `api_responses.json` 文件的数据结构。

## 顶层结构

```json
[
  {
    "url": "string",      // API 请求的 URL
    "status": "number",   // HTTP 状态码 (200, 204 等)
    "body": "object|null" // 响应体，可能为 null (如 204 响应)
  }
]
```

---

## API 端点架构

### 1. `/api/courses/list` - 课程列表

```typescript
{
  status: "success",
  cDatas: CourseListItem[]
}

interface CourseListItem {
  id: number;                // 课程 ID
  credits: number;           // 学分
  department: string;        // 开课院系
  elect_type: string;        // 选课类型
  elect_type_new: string;    // 新选课类型
  name: string;              // 课程名称
  name_pinyin: string;       // 课程名称拼音
  review_count: number;      // 评价数量
  system: string;            // 课程体系 (专业必修/专业任选等)
}
```

---

### 2. `/api/courses/view/{id}` - 课程详情

```typescript
{
  status: "success",
  cData: CourseDetail,
  tDatas: TermMapping,
  rDatas: PaginatedReviews,
  tsDatas: any[],
  filters: FilterOptions
}

interface CourseDetail {
  id: number;
  dean_internal_id: string;      // 教务内部 ID
  name: string;                  // 课程名称
  name_en: string;               // 英文名称
  name_pinyin: string;           // 拼音
  department: string;            // 开课院系
  credits: number;               // 学分
  created_at: string | null;     // 创建时间
  updated_at: string;            // 更新时间
  review_count: number;          // 评价数量
  level: string;                 // 课程级别 (BK=本科)
  system: string;                // 课程体系
  elect_type: string;
  elect_type_new: string;
  setup: string;                 // 开设对象
  averages: {                    // 按学期的评分统计
    [termId: string]: TermAverage
  };
  requested: number;
}

interface TermAverage {
  recommended: number;       // 推荐指数 (-2 到 2)
  rating_content: number;    // 内容评分
  rating_workload: number;   // 工作量评分
  rating_exam: number;       // 考试评分
  count: number;             // 评价数量
  teachers?: string[];       // 授课教师列表 (仅在 termId=0 时存在)
}

// 学期 ID 到学期名称的映射
interface TermMapping {
  [termId: string]: string;  // 例: "31": "25-26-1"
}

interface PaginatedReviews {
  current_page: number;
  data: Review[];
  first_page_url: string;
  from: number;
  last_page: number;
  last_page_url: string;
  links: PaginationLink[];
  next_page_url: string | null;
  path: string;
  per_page: number;
  prev_page_url: string | null;
  to: number;
  total: number;
}

interface Review {
  id: number;
  title: string;              // 评价标题
  teacher_name: string;       // 授课教师
  recommended: number;        // 推荐指数 (-2 到 2)
  rating_content: number;     // 内容评分
  rating_workload: number;    // 工作量评分
  rating_exam: number;        // 考试评分
  result: string | null;      // 成绩
  content: string;            // 评价内容 (Base64 编码)
  created_at: string;
  updated_at: string;
  vote_score: number;         // 投票分数
  term_id: number;            // 学期 ID
}

interface PaginationLink {
  url: string | null;
  label: string;
  active: boolean;
}

interface FilterOptions {
  teacher: string;
  term: number;
}
```

---

### 3. `/api/reviews/latest` - 最新评价

```typescript
{
  status: "success",
  tDatas: TermMapping,
  rDatas: ReviewWithCourse[]
}

interface ReviewWithCourse extends Review {
  course_id: number;
  course: {
    id: number;
    name: string;
  }
}
```

---

### 4. `/api/reviews/post` - 发布评价 (元数据)

```typescript
{
  status: "success",
  pMeta: {
    tDatas: TermMapping,
    cDatasF: CourseOption[]
  }
}

interface CourseOption {
  value: number;    // 课程 ID
  label: string;    // 格式: "课程名 (学分) (评价数)|拼音"
}
```

---

## 通用数据结构

### 学期映射 (TermMapping)

学期 ID 与学期名称的对应关系：

| ID | 学期名称 | 说明 |
|----|----------|------|
| 31 | 25-26-1  | 2025-2026 第一学期 |
| 30 | 24-25-3  | 2024-2025 第三学期 |
| 29 | 24-25-2  | 2024-2025 第二学期 |
| ... | ... | ... |
| 1  | 15-16-1  | 2015-2016 第一学期 |

### 评分范围

所有评分字段的取值范围：
- `-2`: 非常不推荐
- `-1`: 不推荐
- `0`: 中立
- `1`: 推荐
- `2`: 非常推荐

---

## 文件统计

- 文件大小: ~7.2MB
- 主要包含课程列表和课程详情的 API 响应
- 评价内容使用 Base64 编码存储
