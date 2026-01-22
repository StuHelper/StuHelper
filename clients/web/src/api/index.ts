import axios, { type AxiosInstance, type AxiosRequestConfig, type AxiosError } from 'axios'
import { clearAuth } from '@/utils/auth'

// API 响应类型（后端统一返回格式）
export interface ApiResponse<T> {
  data: T
  code?: number
  message?: string
}

// Token 刷新状态管理
class TokenRefreshManager {
  private isRefreshing = false
  private subscribers: ((success: boolean) => void)[] = []

  async handleUnauthorized<T>(
    error: AxiosError,
    axiosInstance: AxiosInstance,
    transformResponse?: (response: unknown) => T
  ): Promise<T> {
    const originalRequest = error.config
    if (!originalRequest) {
      return Promise.reject(error)
    }

    if (!this.isRefreshing) {
      this.isRefreshing = true
      const success = await this.refreshToken()
      this.isRefreshing = false
      this.notifySubscribers(success)

      if (success) {
        const response = await axiosInstance(originalRequest)
        return transformResponse ? transformResponse(response) : response as T
      } else {
        this.handleAuthFailure()
        return Promise.reject(error)
      }
    } else {
      return new Promise((resolve, reject) => {
        this.addSubscriber(async (success: boolean) => {
          if (success) {
            try {
              const response = await axiosInstance(originalRequest)
              resolve(transformResponse ? transformResponse(response) : response as T)
            } catch (e) {
              reject(e)
            }
          } else {
            reject(error)
          }
        })
      })
    }
  }

  private async refreshToken(): Promise<boolean> {
    try {
      await axios.post('/api/auth/refresh', {}, { withCredentials: true })
      return true
    } catch {
      return false
    }
  }

  private notifySubscribers(success: boolean): void {
    this.subscribers.forEach(callback => callback(success))
    this.subscribers = []
  }

  private addSubscriber(callback: (success: boolean) => void): void {
    this.subscribers.push(callback)
  }

  private handleAuthFailure(): void {
    clearAuth()
    // 使用 hash 路由跳转，兼容 Vue Router
    const loginPath = '/login'
    if (window.location.hash !== `#${loginPath}`) {
      window.location.hash = loginPath
    }
  }
}

const tokenManager = new TokenRefreshManager()

// 创建带 token 刷新的响应拦截器
function createAuthInterceptor<T>(
  axiosInstance: AxiosInstance,
  transformResponse?: (response: unknown) => T
) {
  return async (error: AxiosError): Promise<T> => {
    const originalRequest = error.config

    // 如果是 401 错误且不是刷新请求本身
    if (
      error.response?.status === 401 &&
      originalRequest &&
      !originalRequest.url?.includes('/auth/refresh')
    ) {
      return tokenManager.handleUnauthorized(error, axiosInstance, transformResponse)
    }
    return Promise.reject(error)
  }
}

// 创建通用 API 实例
export const request = axios.create({
  baseURL: '/api',
  timeout: 10000,
  withCredentials: true
})

// 创建课程评价 API 实例
const api = axios.create({
  baseURL: '/api/v1/course-review',
  timeout: 10000,
  withCredentials: true
})

// 响应拦截器 - 通用
request.interceptors.response.use(
  (response) => response,
  createAuthInterceptor(request)
)

// 响应拦截器 - 课程评价（返回 response.data）
api.interceptors.response.use(
  (response) => response.data,
  createAuthInterceptor(api, (response) => (response as { data: unknown }).data)
)

// 类型安全的 API 客户端封装
export const courseApi = {
  get<T>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return api.get(url, config) as Promise<ApiResponse<T>>
  },
  post<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return api.post(url, data, config) as Promise<ApiResponse<T>>
  },
  put<T>(url: string, data?: unknown, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return api.put(url, data, config) as Promise<ApiResponse<T>>
  },
  delete<T>(url: string, config?: AxiosRequestConfig): Promise<ApiResponse<T>> {
    return api.delete(url, config) as Promise<ApiResponse<T>>
  }
}

export default courseApi
