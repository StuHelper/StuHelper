/**
 * 课程相关 API
 */
import api, { courseEntityApi } from './index'
import type {
  Course,
  CourseCategory,
  Department,
  RatingDimension,
  CourseRatingStatsResponse
} from '@/types/course'
import type { PaginatedResponse } from '@/types/api'

// 获取院系列表
export function getDepartments(category?: string) {
  return courseEntityApi.get<Department[]>('/departments', {
    params: category ? { category } : undefined
  })
}

// 获取课程分类列表
export function getCourseCategories() {
  return courseEntityApi.get<CourseCategory[]>('/categories')
}

// 获取课程列表
export function getCourses(departmentID?: number, category?: string) {
  const params: Record<string, unknown> = {}
  if (departmentID !== undefined && departmentID !== null) params.departmentID = departmentID
  if (category) params.category = category
  return courseEntityApi.get<PaginatedResponse<Course>>('/courses', { params })
}

// 获取课程详情
export function getCourse(id: number) {
  return courseEntityApi.get<Course>(`/courses/${id}`)
}

// 搜索课程
export function searchCourses(query: string, limit = 10) {
  return courseEntityApi.get<PaginatedResponse<Course>>('/courses/search', {
    params: { q: query, limit }
  })
}

// 统计数据
export interface Stats {
  courseCount: number
  reviewCount: number
  departmentCount: number
}

// 获取统计数据
export function getStats() {
  return api.get<Stats>('/stats')
}

// 获取评分维度配置
export function getRatingDimensions() {
  return api.get<RatingDimension[]>('/rating-dimensions')
}

// 获取课程评分统计
export function getCourseRatingStats(courseID: number) {
  return api.get<CourseRatingStatsResponse>(`/courses/${courseID}/rating-stats`)
}

// 评分趋势项（对齐后端 review.RatingTrendItem json tag）
export interface RatingTrendItem {
  termID: string
  termName: string
  avgRating: number
  count: number
}

// 获取课程评分趋势
export function getCourseRatingTrend(courseID: number) {
  return api.get<RatingTrendItem[]>(`/courses/${courseID}/rating-trend`)
}

// 获取课程的授课教师列表
export function getCourseTeachers(courseID: number) {
  return api.get<import('@/types/course').TeacherStats[]>(`/courses/${courseID}/teachers`)
}

// 教师评分统计
export interface TeacherRatingStats {
  teacherID: number
  teacherName: string
  departmentName: string
  avgRating: number | null
  courseCount: number
  reviewCount: number
  courses: Array<{
    id: number
    name: string
    avgRating: number | null
    reviewCount: number
  }>
  ratingTrend: Array<{
    termID: string
    termName: string
    avgRating: number
  }>
  radarChart: {
    labels: string[]
    datasets: Array<{
      label: string
      data: number[]
    }>
  }
}

// 获取教师评分统计
export function getTeacherStats(teacherID: number) {
  return api.get<TeacherRatingStats>(`/teachers/${teacherID}/stats`)
}
