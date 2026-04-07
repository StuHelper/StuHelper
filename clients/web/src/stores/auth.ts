/**
 * 认证状态管理
 */
import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api'
import { userManager, clearAuth, tokenExpiry } from '@/utils/auth'
import { isApiError, isAuthError, isCsrfError, isNetworkError } from '@/api/errors'
import { useUserStore } from '@/stores/user'
import { useNotificationStore } from '@/stores/notification'
import { useCourseStore } from '@/stores/courseReview'
import { useDraftStore } from '@/stores/draft'
import { useVerificationStore } from '@/stores/verification'
import i18n from '@/i18n'
import type { components } from '@stuhelper/shared'

// 认证错误类型
export type AuthErrorType =
  | 'network'
  | 'invalid_state'
  | 'auth_failed'
  | 'unknown'

export interface AuthError {
  type: AuthErrorType
  message: string
}

type UserInfo = components['schemas']['UserInfo']
type LoginURLResponse = components['schemas']['LoginURLResponse']

// 登出结果类型
export type LogoutResult =
  | { ok: true }
  | { ok: false; reason: 'network' | 'server'; error: unknown }

function getSSOOrigin(): string {
  const ssoURL = import.meta.env.VITE_SSO_URL
  if (!ssoURL) {
    throw new Error('VITE_SSO_URL is not configured')
  }

  const parsed = new URL(ssoURL)
  if (parsed.protocol !== 'https:' && parsed.protocol !== 'http:') {
    throw new Error('Invalid SSO URL protocol')
  }

  return parsed.origin
}

function resolveLoginRedirectTarget(redirect?: string): string | undefined {
  if (typeof window === 'undefined') {
    return redirect
  }

  if (!redirect) {
    return window.location.href
  }

  if (redirect.startsWith('/') && !redirect.startsWith('//')) {
    return new URL(redirect, window.location.origin).toString()
  }

  try {
    const parsed = new URL(redirect, window.location.origin)
    if (
      parsed.origin === window.location.origin &&
      (parsed.protocol === 'https:' || parsed.protocol === 'http:')
    ) {
      return parsed.toString()
    }
  } catch {
    // ignore invalid redirect and fall back to current page
  }

  return window.location.href
}

function normalizeCurrentUser(
  data: UserInfo,
  currentUser: UserInfo | null,
): UserInfo {
  return {
    id: data.id,
    name: data.name,
    displayName: data.displayName ?? data.name,
    email: data.email ?? currentUser?.email,
    avatar: data.avatar ?? currentUser?.avatar,
    roles: data.roles ?? currentUser?.roles ?? [],
    isPlatformAdmin: data.isPlatformAdmin === true,
    capabilities: data.capabilities ?? [],
    globalCapabilities: data.globalCapabilities ?? [],
    capabilityGrants: data.capabilityGrants ?? [],
    canAccessAdmin: data.canAccessAdmin === true,
  }
}

function normalizeStoredUser(data: ReturnType<typeof userManager.getUser>): UserInfo | null {
  if (!data) {
    return null
  }

  return {
    id: data.id,
    name: data.name,
    displayName: data.displayName,
    avatar: data.avatar,
    roles: [],
    isPlatformAdmin: false,
    capabilities: [],
    globalCapabilities: [],
    capabilityGrants: [],
    canAccessAdmin: false,
  }
}

export const useAuthStore = defineStore('auth', () => {
  // localStorage 数据可能损坏，读取失败时降级为空
  let initialUser: UserInfo | null = null
  try {
    initialUser = normalizeStoredUser(userManager.getUser())
  } catch {
    initialUser = null
  }

  // 状态
  const user = ref<UserInfo | null>(initialUser)
  const loading = ref(false)
  const error = ref<AuthError | null>(null)

  // 计算属性
  const isAuthenticated = computed(() => !!user.value)
  const globalCapabilities = computed(() => user.value?.globalCapabilities ?? [])

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
      if (isNetworkError(err.code)) {
        return { type: 'network', message: t('errors.NETWORK_ERROR') }
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
    apiCall: () => Promise<{ data?: { data?: LoginURLResponse } }>,
    errorMsg: string,
  ) => {
    clearError()
    loading.value = true
    try {
      const res = await apiCall()
      const data = res.data?.data
      if (!data?.url || !data?.state) {
        throw new Error('Invalid OAuth response')
      }
      // 校验 OAuth URL 必须指向配置的 SSO Origin
      const oauthURL = new URL(data.url)
      if (
        oauthURL.protocol !== 'https:' &&
        oauthURL.protocol !== 'http:'
      ) {
        throw new Error('Invalid OAuth URL protocol')
      }
      if (oauthURL.origin !== getSSOOrigin()) {
        throw new Error(
          'OAuth URL must point to the configured SSO origin',
        )
      }
      sessionStorage.setItem('oauth_state', data.state)
      window.location.href = data.url
      // 跳转失败时 3 秒后清除 loading 状态
      setTimeout(() => {
        loading.value = false
      }, 3000)
    } catch (err) {
      loading.value = false
      const authErr = handleError(err, errorMsg)
      setError(authErr.type, authErr.message)
      throw err
    }
  }

  // 登录
  const login = (redirect?: string) =>
    startOAuthFlow(
      () => api.auth.login(resolveLoginRedirectTarget(redirect)),
      i18n.global.t('common.login.loginUrlFailed'),
    )

  // 注册
  const signup = (redirect?: string) =>
    startOAuthFlow(
      () => api.auth.signup(resolveLoginRedirectTarget(redirect)),
      i18n.global.t('common.login.signupUrlFailed'),
    )

  // 获取当前用户
  const fetchUser = async () => {
    clearError()
    loading.value = true
    try {
      const res = await api.auth.me()
      const data = res.data?.data
      if (!data) {
        throw new Error('Invalid user response')
      }
      const normalizedUser = normalizeCurrentUser(data, user.value)
      user.value = normalizedUser
      userManager.setUser(normalizedUser)
      return normalizedUser
    } catch (err) {
      // 区分网络错误和认证错误
      if (isApiError(err) && !isNetworkError(err.code)) {
        clearAuth()
        user.value = null
      }
      const authErr = handleError(
        err,
        i18n.global.t('common.login.fetchUserFailed'),
      )
      setError(authErr.type, authErr.message)
      throw err
    } finally {
      loading.value = false
    }
  }

  const bootstrapSession = async () => {
    clearError()

    try {
      const res = await api.auth.me()
      const data = res.data?.data
      if (!data) {
        if (user.value) {
          clearAuth()
          user.value = null
        }
        return false
      }
      const normalizedUser = normalizeCurrentUser(data, user.value)
      user.value = normalizedUser
      userManager.setUser(normalizedUser)
      return true
    } catch (err) {
      if (
        isApiError(err) &&
        !isNetworkError(err.code) &&
        !isCsrfError(err.code) &&
        (err.status === 401 || err.status === 403 || isAuthError(err.code))
      ) {
        clearAuth()
        user.value = null
        return false
      }

      const authErr = handleError(
        err,
        i18n.global.t('common.login.fetchUserFailed'),
      )
      setError(authErr.type, authErr.message)
      if (import.meta.env.DEV) {
        console.error('[Auth] bootstrapSession failed:', err)
      }
      return false
    }
  }

  // 刷新会话（依赖 HttpOnly refresh token Cookie）
  const refreshSession = async () => {
    try {
      const res = await api.auth.refresh()
      const data = res.data?.data
      if (typeof data?.expiresIn === 'number') {
        tokenExpiry.set(data.expiresIn)
      }
      return data
    } catch (err) {
      if (
        isApiError(err) &&
        !isCsrfError(err.code) &&
        (err.status === 401 || isAuthError(err.code))
      ) {
        clearSession()
      }
      throw err
    }
  }

  // 登出（返回结构化结果，调用方根据结果决定跳转或提示重试）
  const logout = async (): Promise<LogoutResult> => {
    clearError()
    loading.value = true
    try {
      await api.auth.logout()
      resetAllStores()
      return { ok: true }
    } catch (err) {
      // 区分认证失效 vs CSRF 拦截 vs 服务端错误 vs 网络错误
      if (isApiError(err)) {
        // CSRF 403: 请求被中间件拦截，但服务端会话仍有效，应提示重试
        if (isCsrfError(err.code)) {
          return { ok: false, reason: 'network', error: err }
        }
        // 401 或非 CSRF 的认证错误: 服务端会话已失效，本地直接清理即可
        if (err.status === 401 || isAuthError(err.code)) {
          resetAllStores()
          return { ok: true }
        }
        // 5xx 服务端错误：服务端会话可能仍有效，提示用户稍后重试
        if (err.status && err.status >= 500) {
          return { ok: false, reason: 'server', error: err }
        }
      }
      // 网络/超时/离线：无法确认服务端状态，提示检查网络
      return { ok: false, reason: 'network', error: err }
    } finally {
      loading.value = false
    }
  }

  // 提取 store 重置逻辑，logout 和 clearSession 共用
  function resetAllStores() {
    clearAuth()
    user.value = null
    useUserStore().reset()
    useCourseStore().reset()
    useDraftStore().reset()
    useVerificationStore().reset()
    const notificationStore = useNotificationStore()
    notificationStore.stopPolling()
    notificationStore.reset()
  }

  // 清除本地会话（不调用 API，用于 token 过期等场景）
  const clearSession = () => {
    resetAllStores()
  }

  // 手机验证码登录：请求验证码
  const requestPhoneOTP = async (phone: string) => {
    clearError()
    loading.value = true
    try {
      const res = await api.auth.requestPhoneOTP(phone)
      return res.data?.data
    } catch (err) {
      const authErr = handleError(
        err,
        i18n.global.t('common.login.otpSendFailed'),
      )
      setError(authErr.type, authErr.message)
      throw err
    } finally {
      loading.value = false
    }
  }

  // 手机验证码登录：验证并登录
  const verifyPhoneOTP = async (phone: string, code: string) => {
    clearError()
    loading.value = true
    try {
      const res = await api.auth.verifyPhoneOTP(phone, code)
      const data = res.data?.data
      if (data?.user) {
        const normalizedUser = normalizeCurrentUser(data.user, null)
        user.value = normalizedUser
        userManager.setUser(normalizedUser)
        if (typeof data.expiresIn === 'number') {
          tokenExpiry.set(data.expiresIn)
        }
      }
      return data
    } catch (err) {
      const authErr = handleError(
        err,
        i18n.global.t('common.login.otpVerifyFailed'),
      )
      setError(authErr.type, authErr.message)
      throw err
    } finally {
      loading.value = false
    }
  }

  return {
    user,
    loading,
    error,
    isAuthenticated,
    globalCapabilities,
    login,
    signup,
    fetchUser,
    bootstrapSession,
    refreshSession,
    logout,
    clearSession,
    clearError,
    requestPhoneOTP,
    verifyPhoneOTP,
  }
})
