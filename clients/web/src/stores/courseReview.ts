/**
 * 课程评价状态管理
 * 带缓存机制和增强错误处理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Department, Course } from '@/types/course'
import { getDepartments, getCourses } from '@/api/course'
import { isApiError } from '@/api/errors'

// 缓存配置
const CACHE_TTL = 5 * 60 * 1000 // 5 分钟

// 缓存项
interface CacheItem<T> {
  data: T
  timestamp: number
}

// 错误类型
export type StoreErrorType = 'network' | 'server' | 'unknown'

export interface StoreError {
  type: StoreErrorType
  message: string
}

// 缓存管理
function createCache<T>() {
  const cache = new Map<string, CacheItem<T>>()

  return {
    get(key: string): T | null {
      const item = cache.get(key)
      if (!item) return null
      if (Date.now() - item.timestamp > CACHE_TTL) {
        cache.delete(key)
        return null
      }
      return item.data
    },
    set(key: string, data: T) {
      cache.set(key, { data, timestamp: Date.now() })
    },
    clear() {
      cache.clear()
    }
  }
}

export const useCourseStore = defineStore('course', () => {
  // 缓存
  const deptCache = createCache<Department[]>()
  const courseCache = createCache<Course[]>()

  // 院系状态
  const departments = ref<Department[]>([])
  const departmentsLoading = ref(false)
  const departmentsError = ref<StoreError | null>(null)

  // 课程状态
  const courses = ref<Course[]>([])
  const coursesLoading = ref(false)
  const coursesError = ref<StoreError | null>(null)

  // 当前请求 ID（防止并发）
  let deptRequestID = 0
  let courseRequestID = 0

  // 统一加载状态
  const loading = computed(() =>
    departmentsLoading.value || coursesLoading.value
  )

  // 处理错误
  const handleError = (err: unknown): StoreError => {
    if (isApiError(err)) {
      if (err.isNetworkError()) {
        return { type: 'network', message: '网络连接失败' }
      }
      return { type: 'server', message: err.getUserMessage() }
    }
    if (err instanceof Error) {
      return { type: 'unknown', message: err.message }
    }
    return { type: 'unknown', message: '未知错误' }
  }

  // 获取院系列表
  const fetchDepartments = async (category?: string) => {
    const cacheKey = category || '_all'
    const cached = deptCache.get(cacheKey)
    if (cached) {
      departments.value = cached
      return cached
    }

    const requestID = ++deptRequestID
    departmentsLoading.value = true
    departmentsError.value = null

    try {
      const res = await getDepartments(category)
      // 检查是否为最新请求
      if (requestID !== deptRequestID) return departments.value

      const data = res.data || []
      departments.value = data
      deptCache.set(cacheKey, data)
      return data
    } catch (err) {
      if (requestID === deptRequestID) {
        departmentsError.value = handleError(err)
        departments.value = []
      }
      throw err
    } finally {
      if (requestID === deptRequestID) {
        departmentsLoading.value = false
      }
    }
  }

  // 获取课程列表
  const fetchCourses = async (deptID: number) => {
    const cacheKey = String(deptID)
    const cached = courseCache.get(cacheKey)
    if (cached) {
      courses.value = cached
      return cached
    }

    const requestID = ++courseRequestID
    coursesLoading.value = true
    coursesError.value = null

    try {
      const res = await getCourses(deptID)
      if (requestID !== courseRequestID) return courses.value

      const data = res.data || []
      courses.value = data
      courseCache.set(cacheKey, data)
      return data
    } catch (err) {
      if (requestID === courseRequestID) {
        coursesError.value = handleError(err)
        courses.value = []
      }
      throw err
    } finally {
      if (requestID === courseRequestID) {
        coursesLoading.value = false
      }
    }
  }

  // 清除缓存
  const clearCache = () => {
    deptCache.clear()
    courseCache.clear()
  }

  // 清除错误
  const clearErrors = () => {
    departmentsError.value = null
    coursesError.value = null
  }

  return {
    departments,
    departmentsLoading,
    departmentsError,
    courses,
    coursesLoading,
    coursesError,
    loading,
    fetchDepartments,
    fetchCourses,
    clearCache,
    clearErrors
  }
})
