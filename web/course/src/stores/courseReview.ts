import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Department, Course } from '@/types/course'
import { getDepartments, getCourses } from '@/api/course'

export const useCourseStore = defineStore('course', () => {
  const departments = ref<Department[]>([])
  const courses = ref<Course[]>([])
  const loading = ref(false)

  const fetchDepartments = async (category?: string) => {
    loading.value = true
    try {
      const res = await getDepartments(category)
      departments.value = (res as any).data || []
    } finally {
      loading.value = false
    }
  }

  const fetchCourses = async (deptId: number) => {
    loading.value = true
    try {
      const res = await getCourses(deptId)
      courses.value = (res as any).data || []
    } finally {
      loading.value = false
    }
  }

  return { departments, courses, loading, fetchDepartments, fetchCourses }
})
