import { defineStore } from 'pinia'
import { ref, computed } from 'vue'

interface User {
  id: string
  name: string
  email?: string
  avatar?: string
}

export const useAuthStore = defineStore('auth', () => {
  const token = ref<string>('')
  const user = ref<User | null>(null)

  const isAuthenticated = computed(() => !!token.value)

  const setToken = (newToken: string) => {
    token.value = newToken
    uni.setStorageSync('token', newToken)
  }

  const setUser = (userData: User) => {
    user.value = userData
    uni.setStorageSync('user', JSON.stringify(userData))
  }

  const clearSession = () => {
    token.value = ''
    user.value = null
    uni.removeStorageSync('token')
    uni.removeStorageSync('user')
  }

  const initAuth = () => {
    const savedToken = uni.getStorageSync('token')
    const savedUser = uni.getStorageSync('user')
    if (savedToken) token.value = savedToken
    if (savedUser) {
      try {
        user.value = typeof savedUser === 'string' ? JSON.parse(savedUser) : savedUser
      } catch {
        user.value = null
      }
    }
  }

  return {
    token,
    user,
    isAuthenticated,
    setToken,
    setUser,
    clearSession,
    initAuth
  }
})
