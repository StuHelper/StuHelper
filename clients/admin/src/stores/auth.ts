import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { api } from '@/api'

interface UserInfo {
  id: string
  name: string
  displayName?: string
  email?: string
  avatar?: string
  isAdmin: boolean
}

export const useAuthStore = defineStore('auth', () => {
  const user = ref<UserInfo | null>(null)
  const loading = ref(false)
  const initialized = ref(false)

  const isAuthenticated = computed(() => !!user.value)
  const isAdmin = computed(() => user.value?.isAdmin === true)

  async function bootstrap(): Promise<boolean> {
    if (initialized.value) return isAuthenticated.value
    loading.value = true
    try {
      const res = await api.auth.me()
      const data = (res.data as Record<string, unknown>)?.data as UserInfo | undefined
      if (data) {
        user.value = {
          id: data.id,
          name: data.name,
          displayName: data.displayName ?? data.name,
          email: data.email,
          avatar: data.avatar,
          isAdmin: data.isAdmin === true,
        }
        return true
      }
      return false
    } catch {
      user.value = null
      return false
    } finally {
      loading.value = false
      initialized.value = true
    }
  }

  function getSSOLoginURL(): string {
    const ssoURL = import.meta.env.VITE_SSO_URL
    if (!ssoURL) return '/login'
    return ssoURL
  }

  async function logout() {
    try {
      const res = await api.auth.logout()
      const data = (res.data as Record<string, unknown>)?.data as { ssoLogoutURL?: string } | undefined
      user.value = null
      if (data?.ssoLogoutURL) {
        window.location.href = data.ssoLogoutURL
      } else {
        window.location.href = '/'
      }
    } catch {
      user.value = null
      window.location.href = '/'
    }
  }

  return {
    user,
    loading,
    initialized,
    isAuthenticated,
    isAdmin,
    bootstrap,
    getSSOLoginURL,
    logout,
  }
})
