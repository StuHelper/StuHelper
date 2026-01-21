import api from './index'
import type { Course, Department } from '@/types/course'

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

// 获取课程详情（别名）
export const getCourseDetail = getCourse

// 搜索课程
export const searchCourses = (q: string, limit = 10) => {
  return api.get<Course[]>('/courses/search', { params: { q, limit } })
}
