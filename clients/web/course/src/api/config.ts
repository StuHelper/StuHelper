/**
 * API 配置
 * 集中管理所有 API 相关配置
 */

// 环境变量类型
interface ApiConfig {
  baseUrl: string
  courseBaseUrl: string
  courseReviewBaseUrl: string
  timeout: number
  withCredentials: boolean
  retryCount: number
  retryDelay: number
}

// 从环境变量读取配置
const config: ApiConfig = {
  baseUrl: import.meta.env.VITE_API_BASE_URL || '/api/v1',
  courseBaseUrl: import.meta.env.VITE_COURSE_API_URL || '/api/v1/course',
  courseReviewBaseUrl: import.meta.env.VITE_COURSE_REVIEW_API_URL || '/api/v1/course/review',
  timeout: (() => {
    const raw = import.meta.env.VITE_API_TIMEOUT
    if (raw === undefined || raw === '') return 15000
    const n = Number(raw)
    return Number.isFinite(n) && n > 0 ? n : 15000
  })(),
  withCredentials: true,
  retryCount: 3,
  retryDelay: 1000
}

export default config
