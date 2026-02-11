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
import { clearAuth } from '@/utils/auth'

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
  private isRefreshing = false
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

    if (!this.isRefreshing) {
      this.isRefreshing = true
      const success = await this.refreshToken()
      // 先通知订阅者，再重置标志，防止竞态
      this.notifySubscribers(success)
      this.isRefreshing = false

      if (success) {
        // instance(originalRequest) 已经过响应拦截器的 transform，不再重复应用
        return await instance(originalRequest) as T
      }
      this.handleAuthFailure()
      return Promise.reject(this.createAuthError())
    }

    // 防止订阅者队列过大
    if (this.subscribers.length >= this.maxSubscribers) {
      return Promise.reject(this.createAuthError())
    }

    return new Promise((resolve, reject) => {
      this.subscribers.push(async (success: boolean) => {
        if (success) {
          try {
            // instance(originalRequest) 已经过响应拦截器的 transform，不再重复应用
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
      await axios.post(
        `${config.baseUrl}/auth/refresh`,
        {},
        { withCredentials: true }
      )
      return true
    } catch {
      return false
    }
  }

  private notifySubscribers(success: boolean): void {
    this.subscribers.forEach(cb => cb(success))
    this.subscribers = []
  }

  private handleAuthFailure(): void {
    clearAuth()
    const loginPath = '/login'
    if (window.location.hash !== `#${loginPath}`) {
      window.location.hash = loginPath
    }
  }

  private createAuthError(): ApiError {
    return new ApiError({
      message: '登录已过期',
      code: ErrorCode.TOKEN_EXPIRED,
      status: 401
    })
  }
}

const tokenManager = new TokenRefreshManager()

// 错误转换函数
function transformError(error: AxiosError): ApiError {
  if (error.code === 'ECONNABORTED' || error.message.includes('timeout')) {
    return new ApiError({
      message: '请求超时',
      code: ErrorCode.TIMEOUT
    })
  }

  if (!error.response) {
    return new ApiError({
      message: '网络连接失败',
      code: ErrorCode.NETWORK_ERROR
    })
  }

  const { status, data } = error.response
  const responseData = data as { message?: string; error?: string } | undefined
  const message = responseData?.message || responseData?.error

  return createErrorFromStatus(status, message)
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

      // 401 错误且非刷新请求
      if (
        error.response?.status === 401 &&
        originalRequest &&
        !originalRequest.url?.includes('/auth/refresh')
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

const requestInterceptor = createResponseInterceptor(request)
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
  }
}

export default courseApi
