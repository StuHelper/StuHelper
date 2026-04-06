/**
 * 课程评价状态管理
 * 带缓存机制和增强错误处理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import type { Department, Course } from '@/types/course'
import { api } from '@/api'
import { isApiError, isNetworkError } from '@/api/errors'
import i18n from '@/i18n'

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

// 缓存管理（访问时惰性清理，避免额外后台定时器）
function createCache<T>() {
  const cache = new Map<string, CacheItem<T>>()

  const pruneExpired = (now = Date.now()) => {
    for (const [key, item] of cache) {
      if (now - item.timestamp > CACHE_TTL) {
        cache.delete(key)
      }
    }
  }

  return {
    get(key: string): T | null {
      pruneExpired()
      const item = cache.get(key)
      return item ? item.data : null
    },
    set(key: string, data: T) {
      pruneExpired()
      cache.set(key, { data, timestamp: Date.now() })
    },
    // 支持按 key 前缀清除
    invalidate(keyPrefix?: string) {
      if (!keyPrefix) {
        cache.clear()
        return
      }
      for (const key of cache.keys()) {
        if (key.startsWith(keyPrefix)) {
          cache.delete(key)
        }
      }
    },
    clear() {
      cache.clear()
    },
    dispose() {
      cache.clear()
    }
  }
}

export const useCourseStore = defineStore('course', () => {
  // 缓存
  let deptCache = createCache<Department[]>()
  let courseCache = createCache<Course[]>()

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
      if (isNetworkError(err.code)) {
        return { type: 'network', message: i18n.global.t('errors.NETWORK_ERROR') }
      }
      return { type: 'server', message: err.getUserMessage() }
    }
    if (err instanceof Error) {
      return { type: 'unknown', message: err.message }
    }
    return { type: 'unknown', message: i18n.global.t('errors.UNKNOWN') }
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
      const res = await api.course.getDepartments(category ? { category } : undefined)
      // 在更新状态前检查 requestID
      if (requestID !== deptRequestID) return departments.value

      const data = res.data?.data || []
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
      const res = await api.course.getCourses({ page: 1, pageSize: 100, departmentID: deptID })
      // 在更新状态前检查 requestID，过期响应直接丢弃
      if (requestID !== courseRequestID) return courses.value

      const data = res.data?.data?.list || []
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

  // 使课程缓存失效（支持按院系 ID 精确清除）
  const invalidateCourseCache = (deptID?: number) => {
    if (deptID !== undefined) {
      courseCache.invalidate(String(deptID))
    } else {
      courseCache.clear()
    }
  }

  // 清除错误
  const clearErrors = () => {
    departmentsError.value = null
    coursesError.value = null
  }

  // 重置状态（包括请求计数器和缓存实例）
  const reset = () => {
    departments.value = []
    departmentsLoading.value = false
    departmentsError.value = null
    courses.value = []
    coursesLoading.value = false
    coursesError.value = null
    deptCache.dispose()
    courseCache.dispose()
    deptCache = createCache<Department[]>()
    courseCache = createCache<Course[]>()
    deptRequestID = 0
    courseRequestID = 0
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
    invalidateCourseCache,
    clearErrors,
    reset
  }
})
