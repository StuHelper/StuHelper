import api from './index'
import type { Course, Department, RatingDimension, CourseRatingStatsResponse } from '@/types/course'

// 获取院系列表
export const getDepartments = (category?: string) => {
  return api.get<Department[]>('/departments', { params: { category } })
}

// 获取课程列表
export const getCourses = (departmentId: number) => {
  return api.get<Course[]>('/courses', { params: { department_id: departmentId } })
}

// 获取课程详情
export const getCourse = (id: number) => {
  return api.get<Course>(`/courses/${id}`)
}

// 搜索课程
export const searchCourses = (q: string, limit = 10) => {
  return api.get<Course[]>('/courses/search', { params: { q, limit } })
}

// 统计数据类型
export interface Stats {
  courseCount: number
  reviewCount: number
  departmentCount: number
}

// 获取统计数据
export const getStats = () => {
  return api.get<Stats>('/stats')
}

// 获取评分维度配置
export const getRatingDimensions = () => {
  return api.get<RatingDimension[]>('/rating-dimensions')
}

// 获取课程评分统计（雷达图数据）
export const getCourseRatingStats = (courseId: number) => {
  return api.get<CourseRatingStatsResponse>(`/courses/${courseId}/rating-stats`)
}
