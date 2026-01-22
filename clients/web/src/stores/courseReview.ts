import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Department, Course } from '@/types/course'
import { getDepartments, getCourses } from '@/api/course'

export const useCourseStore = defineStore('course', () => {
  // 院系数据状态
  const departments = ref<Department[]>([])
  const departmentsLoading = ref(false)
  const departmentsError = ref<string | null>(null)

  // 课程数据状态
  const courses = ref<Course[]>([])
  const coursesLoading = ref(false)
  const coursesError = ref<string | null>(null)

  const fetchDepartments = async (category?: string) => {
    departmentsLoading.value = true
    departmentsError.value = null
    try {
      const res = await getDepartments(category)
      departments.value = res.data || []
    } catch (e) {
      departmentsError.value = e instanceof Error ? e.message : '获取院系列表失败'
      departments.value = []
    } finally {
      departmentsLoading.value = false
    }
  }

  const fetchCourses = async (deptId: number) => {
    coursesLoading.value = true
    coursesError.value = null
    try {
      const res = await getCourses(deptId)
      courses.value = res.data || []
    } catch (e) {
      coursesError.value = e instanceof Error ? e.message : '获取课程列表失败'
      courses.value = []
    } finally {
      coursesLoading.value = false
    }
  }

  return {
    departments,
    departmentsLoading,
    departmentsError,
    courses,
    coursesLoading,
    coursesError,
    fetchDepartments,
    fetchCourses
  }
})
