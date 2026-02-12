/**
 * 认证状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'
import { userManager, clearAuth, tokenExpiry } from '@/utils/auth'
import { isApiError } from '@/api/errors'
import i18n from '@/i18n'

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
    const t = i18n.global.t
    if (isApiError(err)) {
      if (err.isNetworkError()) {
        return { type: 'network', message: t('common.login.networkError') }
      }
      return { type: 'auth_failed', message: err.getUserMessage() }
    }
    if (err instanceof Error) {
      return { type: 'unknown', message: err.message }
    }
    return { type: 'unknown', message: defaultMsg }
  }

  // OAuth 跳转通用流程
  const startOAuthFlow = async (
    apiCall: () => Promise<{ data: { url: string; state: string } }>,
    errorMsg: string
  ) => {
    clearError()
    loading.value = true
    try {
      const { data } = await apiCall()
      // 校验后端返回的 URL 合法性，防止 Open Redirect
      const parsed = new URL(data.url)
      if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
        throw new Error('Invalid OAuth URL protocol')
      }
      if (parsed.host === window.location.host) {
        throw new Error('OAuth URL must not point to current site')
      }
      sessionStorage.setItem('oauth_state', data.state)
      window.location.href = data.url
    } catch (err) {
      loading.value = false
      const authErr = handleError(err, errorMsg)
      setError(authErr.type, authErr.message)
      throw err
    }
  }

  // 登录
  const login = () => startOAuthFlow(authApi.getLoginURL, i18n.global.t('common.login.loginUrlFailed'))

  // 注册
  const signup = () => startOAuthFlow(authApi.getSignupURL, i18n.global.t('common.login.signupUrlFailed'))

  // 处理 OAuth 回调
  const handleCallback = async (code: string, state: string) => {
    clearError()
    loading.value = true
    try {
      const savedState = sessionStorage.getItem('oauth_state')
      if (savedState !== state) {
        setError('invalid_state', i18n.global.t('common.login.invalidState'))
        throw new Error('Invalid state parameter')
      }

      const { data } = await authApi.handleCallback(code, state)
      userManager.setUser(data.user)
      user.value = data.user
      // 存储 token 过期时间，供客户端预检使用
      if (data.expiresIn) {
        tokenExpiry.set(data.expiresIn)
      }
      sessionStorage.removeItem('oauth_state')
      return data
    } catch (err) {
      if (!error.value) {
        const authErr = handleError(err, i18n.global.t('common.login.callbackFailed'))
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
      const authErr = handleError(err, i18n.global.t('common.login.fetchUserFailed'))
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
    } catch {
      // 登出 API 失败不影响本地清理
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
