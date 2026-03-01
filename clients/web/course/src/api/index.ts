/**
 * API 客户端核心模块
 * 统一的请求处理、错误处理、Token 刷新
 */
import axios, {
  type AxiosInstance,
  type AxiosRequestConfig,
  type AxiosResponse,
  type AxiosError
} from 'axios'
import config from './config'
import { ApiError, ErrorCode, createErrorFromStatus } from './errors'
import { clearAuth, tokenExpiry } from '@/utils/auth'
import i18n from '@/i18n'

// H-02: 安全的 cookie 值解析，正确处理值中包含 '=' 的情况
function parseCookieValue(name: string): string | undefined {
  const prefix = name + '='
  const cookie = document.cookie
    .split('; ')
    .find(row => row.startsWith(prefix))
  if (!cookie) return undefined
  const value = decodeURIComponent(cookie.substring(prefix.length))
  return value.length > 0 ? value : undefined
}

// M-08/M-107: 类型守卫 - 校验错误响应结构
interface ErrorResponseBody {
  message?: string
  error?: string | {
    code?: string
    message?: string
    details?: Record<string, unknown>
  }
}

function isErrorResponseBody(data: unknown): data is ErrorResponseBody {
  if (typeof data !== 'object' || data === null) return false
  return true
}

// API 响应类型
export interface ApiResponse<T> {
  data: T
  code?: number
  message?: string
}

// 请求配置扩展
interface RequestConfigExtra {
  _retry?: boolean
  _retryCount?: number
}

type ExtendedAxiosRequestConfig = AxiosRequestConfig & RequestConfigExtra

// Token 刷新管理器
class TokenRefreshManager {
  private refreshPromise: Promise<boolean> | null = null
  private subscribers: Array<(success: boolean) => void> = []
  private maxSubscribers = 100

  async handleUnauthorized<T>(
    error: AxiosError,
    instance: AxiosInstance
  ): Promise<T> {
    const originalRequest = error.config as ExtendedAxiosRequestConfig
    if (!originalRequest || originalRequest._retry) {
      return Promise.reject(this.createAuthError())
    }
    originalRequest._retry = true

    if (!this.refreshPromise) {
      // H-01: 使用 Promise 队列替代 boolean 标志，避免竞态
      this.refreshPromise = this.refreshToken()
      try {
        const success = await this.refreshPromise
        this.notifySubscribers(success)

        if (success) {
          return await instance(originalRequest) as T
        }
        this.handleAuthFailure()
        return Promise.reject(this.createAuthError())
      } finally {
        this.refreshPromise = null
      }
    }

    // 防止订阅者队列过大
    if (this.subscribers.length >= this.maxSubscribers) {
      if (import.meta.env.DEV) {
        // L-59: 仅开发环境输出内部限制信息
        console.warn('[TokenRefresh] Subscriber queue full, rejecting request')
      }
      return Promise.reject(this.createAuthError())
    }

    return new Promise((resolve, reject) => {
      this.subscribers.push(async (success: boolean) => {
        if (success) {
          try {
            resolve(await instance(originalRequest) as T)
          } catch (e) {
            reject(e)
          }
        } else {
          reject(this.createAuthError())
        }
      })
    })
  }

  private async refreshToken(): Promise<boolean> {
    try {
      // H-02: 使用安全的 cookie 解析，校验 token 非空
      const csrfToken = parseCookieValue('csrf_token')

      const res = await axios.post(
        `${config.baseUrl}/auth/refresh`,
        {},
        {
          withCredentials: true,
          headers: csrfToken ? { 'X-CSRF-Token': csrfToken } : {}
        }
      )
      // 刷新成功后更新客户端 token 过期时间戳
      const expiresIn = res.data?.data?.expiresIn ?? res.data?.expiresIn
      if (typeof expiresIn === 'number' && expiresIn > 0) {
        tokenExpiry.set(expiresIn)
      }
      return true
    } catch {
      // L-30: 生产环境不泄露刷新失败细节
      if (import.meta.env.DEV) {
        console.error('[TokenRefresh] Failed to refresh token')
      }
      return false
    }
  }

  private notifySubscribers(success: boolean): void {
    this.subscribers.forEach(cb => cb(success))
    this.subscribers = []
  }

  private handleAuthFailure(): void {
    clearAuth()
    // Hash router 模式下使用 /#/login
    if (!window.location.hash.startsWith('#/login')) {
      window.location.href = '/#/login'
    }
  }

  private createAuthError(): ApiError {
    return new ApiError({
      message: i18n.global.t('errors.TOKEN_EXPIRED'),
      code: ErrorCode.TOKEN_EXPIRED,
      status: 401
    })
  }
}

const tokenManager = new TokenRefreshManager()

// 离线检测拦截器：请求前检查网络状态，避免离线时长时间等待超时
function addOfflineInterceptor(instance: AxiosInstance) {
  instance.interceptors.request.use((config) => {
    if (!navigator.onLine) {
      return Promise.reject(
        new ApiError({
          message: i18n.global.t('errors.OFFLINE'),
          code: ErrorCode.OFFLINE
        })
      )
    }
    return config
  })
}

// CSRF Token 请求拦截器：非 GET 请求自动附加 X-CSRF-Token
function addCSRFInterceptor(instance: AxiosInstance) {
  instance.interceptors.request.use((config) => {
    const method = config.method?.toUpperCase()
    if (method && method !== 'GET' && method !== 'HEAD' && method !== 'OPTIONS') {
      const csrfToken = parseCookieValue('csrf_token')
      if (csrfToken) {
        config.headers.set('X-CSRF-Token', csrfToken)
      }
    }
    return config
  })
}

// 错误转换函数
function transformError(error: AxiosError): ApiError {
  if (error.code === 'ECONNABORTED' || error.message.includes('timeout')) {
    return new ApiError({
      message: i18n.global.t('errors.TIMEOUT'),
      code: ErrorCode.TIMEOUT
    })
  }

  if (!error.response) {
    return new ApiError({
      message: i18n.global.t('errors.NETWORK_ERROR'),
      code: ErrorCode.NETWORK_ERROR
    })
  }

  const { status, data } = error.response
  // M-08/M-107: 使用类型守卫校验响应结构
  const responseData = isErrorResponseBody(data) ? data : undefined

  let message: string | undefined
  let details: Record<string, unknown> | undefined

  if (
    responseData?.error &&
    typeof responseData.error === 'object' &&
    responseData.error !== null &&
    'message' in responseData.error
  ) {
    message = responseData.error.message
    details = responseData.error.details
  } else {
    message = responseData?.message ||
      (typeof responseData?.error === 'string' ? responseData.error : undefined)
  }

  return createErrorFromStatus(status, message, details)
}

// 创建响应拦截器
function createResponseInterceptor<T>(
  instance: AxiosInstance,
  transform?: (res: unknown) => T
) {
  return {
    onFulfilled: (response: AxiosResponse): AxiosResponse => {
      // blob 响应不经过 transform，直接返回原始响应
      if (response.config.responseType === 'blob') {
        return response
      }
      return (transform ? transform(response) : response) as AxiosResponse
    },
    onRejected: async (error: AxiosError): Promise<T> => {
      const originalRequest = error.config

      // 401 错误且非刷新请求，登录页无需刷新 token
      if (
        error.response?.status === 401 &&
        originalRequest &&
        !originalRequest.url?.includes('/auth/refresh') &&
        !window.location.hash.startsWith('#/login')
      ) {
        return tokenManager.handleUnauthorized(error, instance)
      }

      return Promise.reject(transformError(error))
    }
  }
}

// 创建通用 API 实例
export const request = axios.create({
  baseURL: config.baseUrl,
  timeout: config.timeout,
  withCredentials: config.withCredentials
})
addOfflineInterceptor(request)
addCSRFInterceptor(request)

const requestInterceptor = createResponseInterceptor(
  request,
  (res) => (res as { data: unknown }).data
)
request.interceptors.response.use(
  requestInterceptor.onFulfilled,
  requestInterceptor.onRejected
)

// 创建课程评价 API 实例
const courseReviewApi = axios.create({
  baseURL: config.courseReviewBaseUrl,
  timeout: config.timeout,
  withCredentials: config.withCredentials
})
addOfflineInterceptor(courseReviewApi)
addCSRFInterceptor(courseReviewApi)

const courseInterceptor = createResponseInterceptor(
  courseReviewApi,
  (res) => (res as { data: unknown }).data
)
courseReviewApi.interceptors.response.use(
  courseInterceptor.onFulfilled,
  courseInterceptor.onRejected
)

// 创建课程实体 API 实例（院系、课程等非评价接口）
const courseBaseApi = axios.create({
  baseURL: config.courseBaseUrl,
  timeout: config.timeout,
  withCredentials: config.withCredentials
})
addOfflineInterceptor(courseBaseApi)
addCSRFInterceptor(courseBaseApi)

const courseBaseInterceptor = createResponseInterceptor(
  courseBaseApi,
  (res) => (res as { data: unknown }).data
)
courseBaseApi.interceptors.response.use(
  courseBaseInterceptor.onFulfilled,
  courseBaseInterceptor.onRejected
)

// 课程实体 API 客户端（/api/v1/course）
export const courseEntityApi = {
  get<T>(url: string, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseBaseApi.get(url, cfg) as Promise<ApiResponse<T>>
  },
  post<T>(url: string, data?: unknown, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseBaseApi.post(url, data, cfg) as Promise<ApiResponse<T>>
  },
  put<T>(url: string, data?: unknown, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseBaseApi.put(url, data, cfg) as Promise<ApiResponse<T>>
  },
  delete<T>(url: string, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseBaseApi.delete(url, cfg) as Promise<ApiResponse<T>>
  },
  // H-39: blob 响应独立方法，不经过 JSON transform
  downloadBlob(url: string, cfg?: AxiosRequestConfig): Promise<AxiosResponse<Blob>> {
    return courseBaseApi.get(url, { ...cfg, responseType: 'blob' })
  }
}

// 课程评价 API 客户端（/api/v1/course/review）
export const courseApi = {
  get<T>(url: string, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseReviewApi.get(url, cfg) as Promise<ApiResponse<T>>
  },
  post<T>(url: string, data?: unknown, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseReviewApi.post(url, data, cfg) as Promise<ApiResponse<T>>
  },
  put<T>(url: string, data?: unknown, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseReviewApi.put(url, data, cfg) as Promise<ApiResponse<T>>
  },
  delete<T>(url: string, cfg?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return courseReviewApi.delete(url, cfg) as Promise<ApiResponse<T>>
  },
  // H-39: blob 响应独立方法，不经过 JSON transform
  downloadBlob(url: string, cfg?: AxiosRequestConfig): Promise<AxiosResponse<Blob>> {
    return courseReviewApi.get(url, { ...cfg, responseType: 'blob' })
  }
}

export default courseApi
