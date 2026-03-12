import { defineStore } from 'pinia'
import { ref } from 'vue'
import type { Course } from '@stuhelper/shared'

export const useCourseStore = defineStore('course', () => {
  const courses = ref<Course[]>([])
  const loading = ref(false)

  const fetchCourses = async () => {
    loading.value = true
    try {
      // TODO: 调用 API
      courses.value = []
    } finally {
      loading.value = false
    }
  }

  return {
    courses,
    loading,
    fetchCourses
  }
})
