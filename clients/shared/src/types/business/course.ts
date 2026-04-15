/**
 * 课程相关类型定义
 *
 * 核心实体类型直接别名到 OpenAPI 生成类型（api.gen.ts 为唯一事实源）。
 * 仅保留 OpenAPI 未覆盖的前端展示 / 守卫 / 统计类型。
 */
import type { components } from '../api.gen'

// ---- OpenAPI 别名（唯一事实源） ----

export type Course = components['schemas']['Course']
export type Department = components['schemas']['Department']
export type Term = components['schemas']['Term']
export type CourseCategory = components['schemas']['CourseCategory']
export type FavoriteCourse = components['schemas']['FavoriteCourse']
export type RatingDimension = components['schemas']['RatingDimension']

// ---- 评分值（OpenAPI 中为 number，此处收窄为字面量联合以便前端守卫） ----

export type RatingValue = 1 | 2 | 3 | 4 | 5

export const RATING_VALUES: readonly RatingValue[] = [1, 2, 3, 4, 5] as const

export function isValidRating(value: unknown): value is RatingValue {
  return typeof value === 'number' && RATING_VALUES.includes(value as RatingValue)
}

// ---- 前端展示型（OpenAPI 未覆盖的纯 UI 模型） ----

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
