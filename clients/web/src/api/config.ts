/**
 * API 配置
 * 集中管理所有 API 相关配置
 */

// 环境变量类型
interface ApiConfig {
  baseUrl: string
  courseReviewBaseUrl: string
  timeout: number
  withCredentials: boolean
  retryCount: number
  retryDelay: number
}

// 从环境变量读取配置
const config: ApiConfig = {
  baseUrl: import.meta.env.VITE_API_BASE_URL || '/api',
  courseReviewBaseUrl: import.meta.env.VITE_COURSE_REVIEW_API_URL || '/api/v1/course-review',
  timeout: Number(import.meta.env.VITE_API_TIMEOUT) || 15000,
  withCredentials: true,
  retryCount: 3,
  retryDelay: 1000
}

export default config
