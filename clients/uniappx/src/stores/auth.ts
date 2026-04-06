import { defineStore } from 'pinia'
import { computed, ref } from 'vue'
import type { components } from '@/api'
import { api } from '@/api'
import { assertMutationSuccess, extractErrorMessage, unwrapData, unwrapOptionalData } from '@/api/result'
import { translate } from '@/i18n'

type CurrentUser = components['schemas']['UserInfo']
type RequestPhoneOTPResult = { message: string; cooldown: number }
type VerifyPhoneOTPResult = { user: CurrentUser; expiresIn: number }

type UniPageLike = {
  options?: Record<string, string | undefined>
  route?: string
}

const BOOTSTRAP_STALE_MS = 60_000

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
  } catch {
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
  }

  async function bootstrapSession(force = false) {
    const now = Date.now()
    if (!force && initialized.value && now - lastBootstrapAt.value < BOOTSTRAP_STALE_MS) {
      return
    }
    if (loading.value) return
    loading.value = true
    try {
      const result = await api.auth.me()
      user.value = unwrapOptionalData<CurrentUser>(result)
    } catch {
      user.value = null
    } finally {
      lastBootstrapAt.value = Date.now()
      initialized.value = true
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
    user.value = data.user
    initialized.value = true
    lastBootstrapAt.value = Date.now()
    return data
  }

  async function logout() {
    assertMutationSuccess(await api.auth.logout())
    clearSession()
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
    logout,
    requireAuth,
    errorMessage: extractErrorMessage,
  }
})
