import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { components } from '@/api'
import { api } from '@/api'
import { assertMutationSuccess, unwrapData, unwrapOptionalData } from '@/api/result'
import { translate } from '@/i18n'

type CurrentUser = components['schemas']['UserInfo']
type RequestPhoneOTPResult = { message: string; cooldown: number }
type VerifyPhoneOTPResult = { user: CurrentUser; expiresIn: number }
type ExchangeNativeResult = { accessToken: string; refreshToken?: string; expiresIn: number }

type UniPageLike = {
  options?: Record<string, string | undefined>
  route?: string
}

const BOOTSTRAP_STALE_MS = 60_000
const TOKEN_STORAGE_KEY = 'stuhelper:native-tokens'

/** 原生 App 本地存储的 token 结构 */
interface NativeTokens {
  accessToken: string
  refreshToken: string
  /** token 到期时间戳（毫秒） */
  expiresAt: number
}

/** 判断当前是否运行在原生 App 环境 */
function isNativeApp(): boolean {
  return typeof plus !== 'undefined'
}

/** 从本地存储读取原生 token */
function readNativeTokens(): NativeTokens | null {
  try {
    const raw = uni.getStorageSync(TOKEN_STORAGE_KEY)
    if (!raw || typeof raw !== 'string') return null
    const parsed = JSON.parse(raw) as NativeTokens
    if (!parsed.accessToken || !parsed.refreshToken) return null
    return parsed
  } catch (_error) { void _error;
    return null
  }
}

/** 持久化原生 token 到本地存储 */
function writeNativeTokens(tokens: NativeTokens): void {
  try {
    uni.setStorageSync(TOKEN_STORAGE_KEY, JSON.stringify(tokens))
  } catch (_error) { void _error;
    // 存储失败不阻断流程
  }
}

/** 清除本地存储的原生 token */
function clearNativeTokens(): void {
  try {
    uni.removeStorageSync(TOKEN_STORAGE_KEY)
  } catch (_error) { void _error;
    // ignore
  }
}

/** 检查原生 token 是否已过期（预留 30s 缓冲） */
function isTokenExpired(tokens: NativeTokens): boolean {
  return Date.now() >= tokens.expiresAt - 30_000
}

function buildCurrentRouteRedirect(): string {
  try {
    const pages = (typeof getCurrentPages === 'function' ? getCurrentPages() : []) as UniPageLike[]
    const currentPage = pages[pages.length - 1]
    if (!currentPage?.route) {
      return '/pages/user/index'
    }

    const query = currentPage.options
      ? Object.entries(currentPage.options)
          .filter(([, value]) => typeof value === 'string' && value.length > 0)
          .map(([key, value]) => `${encodeURIComponent(key)}=${encodeURIComponent(value as string)}`)
      : []

    return `/${currentPage.route}${query.length > 0 ? `?${query.join('&')}` : ''}`
  } catch (_error) { void _error;
    return '/pages/user/index'
  }
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<CurrentUser | null>(null)
  const loading = ref(false)
  const initialized = ref(false)
  const lastBootstrapAt = ref(0)

  const isAuthenticated = computed(() => !!user.value)
  const displayName = computed(
    () => user.value?.displayName || user.value?.name || translate('common.notLoggedIn')
  )

  function setUser(nextUser: CurrentUser | null) {
    user.value = nextUser
    lastBootstrapAt.value = nextUser ? Date.now() : 0
  }

  function clearSession() {
    user.value = null
    lastBootstrapAt.value = 0
    clearNativeTokens()
  }

  async function bootstrapSession(force = false) {
    const now = Date.now()
    if (!force && initialized.value && now - lastBootstrapAt.value < BOOTSTRAP_STALE_MS) {
      return
    }
    if (loading.value) return
    loading.value = true
    try {
      // 原生 App：检查本地 token 是否存在且未过期
      if (isNativeApp()) {
        const tokens = readNativeTokens()
        if (!tokens || isTokenExpired(tokens)) {
          // 无 token 或已过期——标记为未登录
          user.value = null
          lastBootstrapAt.value = Date.now()
          initialized.value = true
          return
        }
      }

      const result = await api.auth.me()
      user.value = unwrapOptionalData<CurrentUser>(result)
      lastBootstrapAt.value = Date.now()
      initialized.value = true
    } catch (error: unknown) {
      const status = (error as { status?: number })?.status
        ?? (error as { response?: { status?: number } })?.response?.status
      if (status === 401 || status === 403) {
        user.value = null
        lastBootstrapAt.value = Date.now()
        initialized.value = true
        // 原生 App 401 说明 token 失效，清除本地存储
        if (isNativeApp()) clearNativeTokens()
      }
      // 网络错误 / 超时 / 5xx：不更新 lastBootstrapAt 和 initialized，允许后续重试
    } finally {
      loading.value = false
    }
  }

  async function requestPhoneOTP(phone: string): Promise<RequestPhoneOTPResult> {
    const result = await api.auth.requestPhoneOTP(phone)
    return unwrapData<RequestPhoneOTPResult>(result)
  }

  async function verifyPhoneOTP(phone: string, code: string): Promise<VerifyPhoneOTPResult> {
    const result = await api.auth.verifyPhoneOTP(phone, code)
    const data = unwrapData<VerifyPhoneOTPResult>(result)
    setUser(data.user)
    initialized.value = true
    return data
  }

  async function logout() {
    assertMutationSuccess(await api.auth.logout())
    clearSession()
  }

  /** 原生 App SSO 回调：用授权码 + state 换取 token 并持久化 */
  async function exchangeNativeCode(code: string, state: string): Promise<void> {
    const result = await api.auth.exchangeNative(code, state)
    const data = unwrapData<ExchangeNativeResult>(result)

    // 持久化 token 到本地存储
    writeNativeTokens({
      accessToken: data.accessToken,
      refreshToken: data.refreshToken ?? '',
      expiresAt: Date.now() + data.expiresIn * 1000,
    })

    // 立即拉取用户信息
    await bootstrapSession(true)
  }

  /** 检查原生 App 是否持有有效 token */
  function hasNativeToken(): boolean {
    if (!isNativeApp()) return false
    const tokens = readNativeTokens()
    return tokens !== null && !isTokenExpired(tokens)
  }

  /** 获取原生 App 的 access token（供请求头注入） */
  function getNativeAccessToken(): string | null {
    if (!isNativeApp()) return null
    const tokens = readNativeTokens()
    if (!tokens || isTokenExpired(tokens)) return null
    return tokens.accessToken
  }

  async function requireAuth(message = translate('auth.requireLogin')) {
    if (isAuthenticated.value) return true
    await bootstrapSession()
    if (isAuthenticated.value) return true
    uni.showToast({ title: message, icon: 'none' })
    uni.navigateTo({ url: `/pages/auth/login?redirect=${encodeURIComponent(buildCurrentRouteRedirect())}` })
    return false
  }

  return {
    user,
    loading,
    initialized,
    isAuthenticated,
    displayName,
    setUser,
    clearSession,
    bootstrapSession,
    requestPhoneOTP,
    verifyPhoneOTP,
    exchangeNativeCode,
    hasNativeToken,
    getNativeAccessToken,
    logout,
    requireAuth,
  }
})
