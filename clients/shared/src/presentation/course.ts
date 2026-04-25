/**
 * 课程 presentation 层
 *
 * 评分字面量联合、守卫函数、以及纯前端展示型接口（OpenAPI 未覆盖的 UI 模型）。
 * OpenAPI 别名仍在 types/business/course.ts。
 */

// ---- 评分值（字面量联合，收窄 OpenAPI 的 number） ----

export type RatingValue = 1 | 2 | 3 | 4 | 5

export const RATING_VALUES: readonly RatingValue[] = [1, 2, 3, 4, 5] as const

// ---- 类型守卫 ----

export function isValidRating(value: unknown): value is RatingValue {
  return typeof value === 'number' && RATING_VALUES.includes(value as RatingValue)
}

// ---- 纯前端展示型接口（OpenAPI 未覆盖的 UI 模型） ----

export interface DimensionStats {
  key: string
  name: string
  avgRating: number
  ratingCount: number
  distribution?: Record<string, number>
}

export interface TermRatingStats {
  termID?: string
  termName: string
  dimensions: DimensionStats[]
}

export interface RadarChartDataset {
  label: string
  data: number[]
  backgroundColor: string
  borderColor: string
}

export interface RadarChartData {
  labels: string[]
  datasets: RadarChartDataset[]
}

export interface TeacherStats {
  teacherID: number
  teacherName: string
  departmentName: string
  avgRating?: number | null
  courseCount: number
  reviewCount: number
  tags?: string[]
}

export interface CourseRatingStatsResponse {
  courseID: number
  overall: TermRatingStats
  byTerm: TermRatingStats[]
  allDimensionKeys: string[]
}
