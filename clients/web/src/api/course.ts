/**
 * 课程相关 API
 */
import api from './index'
import type {
  Course,
  Department,
  RatingDimension,
  CourseRatingStatsResponse
} from '@/types/course'

// 获取院系列表
export function getDepartments(category?: string) {
  return api.get<Department[]>('/departments', {
    params: category ? { category } : undefined
  })
}

// 获取课程列表
export function getCourses(departmentId: number) {
  return api.get<Course[]>('/courses', {
    params: { departmentId }
  })
}

// 获取课程详情
export function getCourse(id: number) {
  return api.get<Course>(`/courses/${id}`)
}

// 搜索课程
export function searchCourses(query: string, limit = 10) {
  return api.get<Course[]>('/courses/search', {
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
export function getCourseRatingStats(courseId: number) {
  return api.get<CourseRatingStatsResponse>(`/courses/${courseId}/rating-stats`)
}
