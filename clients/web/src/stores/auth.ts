/**
 * 认证状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'
import { userManager, clearAuth, tokenExpiry } from '@/utils/auth'
import { isApiError } from '@/api/errors'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { useCourseStore } from '@/stores/courseReview'
import { useDraftStore } from '@/stores/draft'
import i18n from '@/i18n'

// 认证错误类型
export type AuthErrorType = 'network' | 'invalid_state' | 'auth_failed' | 'unknown'

export interface AuthError {
  type: AuthErrorType
  message: string
}

export const useAuthStore = defineStore('auth', () => {
  // M-91: 用 try-catch 包裹 localStorage 读取，防止数据损坏导致 store 初始化失败
  let initialUser: authApi.UserInfo | null = null
  try {
    initialUser = userManager.getUser()
  } catch {
    initialUser = null
  }

  // 状态
  const user = ref<authApi.UserInfo | null>(initialUser)
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
      // H-03: 严格同源校验，防止子域名攻击和端口不匹配
      const parsed = new URL(data.url)
      if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
        throw new Error('Invalid OAuth URL protocol')
      }
      const currentOrigin = window.location.origin
      if (new URL(data.url, currentOrigin).origin === currentOrigin) {
        throw new Error('OAuth URL must not point to current origin')
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

      // 重置其他 store 状态（M-39: 包含 draft store）
      useUserStore().reset()
      useCourseStore().reset()
      useDraftStore().reset()
      const notificationStore = useNotificationStore()
      notificationStore.stopPolling()
      notificationStore.reset()
    }
  }

  // 清除本地会话（不调用 API，用于 token 过期等场景）
  const clearSession = () => {
    clearAuth()
    user.value = null
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
    clearSession,
    clearError
  }
})
