/**
 * 课程相关类型定义
 */

// 评分值 (1-5 五级制)
export type RatingValue = 1 | 2 | 3 | 4 | 5

// 评分值数组（用于验证）
export const RATING_VALUES: readonly RatingValue[] = [1, 2, 3, 4, 5] as const

// 类型守卫：检查是否为有效评分值
export function isValidRating(value: unknown): value is RatingValue {
  return typeof value === 'number' && RATING_VALUES.includes(value as RatingValue)
}

// 评分维度配置
export interface RatingDimension {
  id: number
  key: string
  name: string
  description?: string
  sortOrder: number
  isActive: boolean
}

// 院系
export interface Department {
  id: number
  name: string
  shortName?: string
  category: string
}

// 课程
export interface Course {
  id: number
  name: string
  code?: string
  credits: number
  departmentID: number
  departmentName?: string
  category: string
  reviewCount: number
}

// 课程分类（后台可配置）
export interface CourseCategory {
  id: number
  schoolID: number
  name: string
  sortOrder: number
}

// 维度评分统计
export interface DimensionStats {
  key: string
  name: string
  avgRating: number
  ratingCount: number
  distribution: Record<string, number>
}

// 学期评分统计
export interface TermRatingStats {
  termID?: string
  termName: string
  dimensions: DimensionStats[]
}

// 雷达图数据集
export interface RadarChartDataset {
  label: string
  data: number[]
  backgroundColor: string
  borderColor: string
}

// 雷达图数据
export interface RadarChartData {
  labels: string[]
  datasets: RadarChartDataset[]
}

// 教师统计卡片数据
export interface TeacherStats {
  teacherID: number
  teacherName: string
  departmentName: string
  avgRating: number | null
  courseCount: number
  reviewCount: number
  tags?: string[]
}

// 课程评分统计响应
export interface CourseRatingStatsResponse {
  courseID: number
  overall: TermRatingStats
  byTerm: TermRatingStats[]
  allDimensionKeys: string[]
  radarChart: RadarChartData
}
