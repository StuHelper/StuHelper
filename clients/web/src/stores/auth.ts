/**
 * 认证状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'
import { userManager, clearAuth } from '@/utils/auth'
import { ApiError, isApiError, ErrorCode } from '@/api/errors'

// 认证错误类型
export type AuthErrorType = 'network' | 'invalid_state' | 'auth_failed' | 'unknown'

export interface AuthError {
  type: AuthErrorType
  message: string
}

export const useAuthStore = defineStore('auth', () => {
  // 状态
  const user = ref<authApi.UserInfo | null>(userManager.getUser())
  const loading = ref(false)
  const error = ref<AuthError | null>(null)

  // 计算属性
  const isAuthenticated = computed(() => !!user.value)

  // 清除错误
  const clearError = () => {
    error.value = null
  }

  // 设置错误
  const setError = (type: AuthErrorType, message: string) => {
    error.value = { type, message }
  }

  // 处理 API 错误
  const handleError = (err: unknown, defaultMsg: string): AuthError => {
    if (isApiError(err)) {
      if (err.isNetworkError()) {
        return { type: 'network', message: '网络连接失败' }
      }
      return { type: 'auth_failed', message: err.getUserMessage() }
    }
    if (err instanceof Error) {
      return { type: 'unknown', message: err.message }
    }
    return { type: 'unknown', message: defaultMsg }
  }

  // 登录
  const login = async () => {
    clearError()
    loading.value = true
    try {
      const { data } = await authApi.getLoginURL()
      sessionStorage.setItem('oauth_state', data.state)
      window.location.href = data.url
    } catch (err) {
      const authErr = handleError(err, '获取登录链接失败')
      setError(authErr.type, authErr.message)
      throw err
    } finally {
      loading.value = false
    }
  }

  // 注册
  const signup = async () => {
    clearError()
    loading.value = true
    try {
      const { data } = await authApi.getSignupURL()
      sessionStorage.setItem('oauth_state', data.state)
      window.location.href = data.url
    } catch (err) {
      const authErr = handleError(err, '获取注册链接失败')
      setError(authErr.type, authErr.message)
      throw err
    } finally {
      loading.value = false
    }
  }

  // 处理 OAuth 回调
  const handleCallback = async (code: string, state: string) => {
    clearError()
    loading.value = true
    try {
      const savedState = sessionStorage.getItem('oauth_state')
      if (savedState !== state) {
        setError('invalid_state', '无效的认证状态')
        throw new Error('Invalid state parameter')
      }

      const { data } = await authApi.handleCallback(code, state)
      userManager.setUser(data.user)
      user.value = data.user
      sessionStorage.removeItem('oauth_state')
      return data
    } catch (err) {
      if (!error.value) {
        const authErr = handleError(err, '认证回调处理失败')
        setError(authErr.type, authErr.message)
      }
      throw err
    } finally {
      loading.value = false
    }
  }

  // 获取当前用户
  const fetchUser = async () => {
    clearError()
    loading.value = true
    try {
      const { data } = await authApi.getCurrentUser()
      user.value = data
      userManager.setUser(data)
      return data
    } catch (err) {
      // 区分网络错误和认证错误
      if (isApiError(err) && !err.isNetworkError()) {
        clearAuth()
        user.value = null
      }
      const authErr = handleError(err, '获取用户信息失败')
      setError(authErr.type, authErr.message)
      throw err
    } finally {
      loading.value = false
    }
  }

  // 登出
  const logout = async () => {
    clearError()
    loading.value = true
    try {
      await authApi.logout()
    } catch (err) {
      console.error('Logout API error:', err)
    } finally {
      clearAuth()
      user.value = null
      loading.value = false
    }
  }

  return {
    user,
    loading,
    error,
    isAuthenticated,
    login,
    signup,
    handleCallback,
    fetchUser,
    logout,
    clearError
  }
})
