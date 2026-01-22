import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import * as authApi from '@/api/auth'
import { userManager, clearAuth } from '@/utils/auth'

export const useAuthStore = defineStore('auth', () => {
  // 状态
  const user = ref<authApi.UserInfo | null>(userManager.getUser())
  const loading = ref(false)

  // 计算属性 - 基于 user.value 响应式更新
  const isAuthenticated = computed(() => !!user.value)

  // 登录 - 获取登录 URL 并跳转
  const login = async () => {
    try {
      loading.value = true
      const { data } = await authApi.getLoginURL()
      // 保存 state 用于验证
      sessionStorage.setItem('oauth_state', data.state)
      // 跳转到 SSO 登录页面
      window.location.href = data.url
    } catch (error) {
      console.error('Failed to get login URL:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  // 注册 - 获取注册 URL 并跳转
  const signup = async () => {
    try {
      loading.value = true
      const { data } = await authApi.getSignupURL()
      sessionStorage.setItem('oauth_state', data.state)
      window.location.href = data.url
    } catch (error) {
      console.error('Failed to get signup URL:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  // 处理 OAuth 回调
  const handleCallback = async (code: string, state: string) => {
    try {
      loading.value = true
      // 验证 state
      const savedState = sessionStorage.getItem('oauth_state')
      if (savedState !== state) {
        throw new Error('Invalid state parameter')
      }

      // 获取用户信息（token 已通过 HttpOnly Cookie 设置）
      const { data } = await authApi.handleCallback(code, state)

      // 保存用户信息
      userManager.setUser(data.user)
      user.value = data.user

      // 清除 state
      sessionStorage.removeItem('oauth_state')

      return data
    } catch (error) {
      console.error('Failed to handle callback:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  // 获取当前用户信息
  const fetchUser = async () => {
    try {
      loading.value = true
      const { data } = await authApi.getCurrentUser()
      user.value = data
      userManager.setUser(data)
      return data
    } catch (error) {
      console.error('Failed to fetch user:', error)
      throw error
    } finally {
      loading.value = false
    }
  }

  // 登出
  const logout = async () => {
    try {
      loading.value = true
      await authApi.logout()
    } catch (error) {
      console.error('Failed to logout:', error)
    } finally {
      // 无论是否成功，都清除本地认证信息
      clearAuth()
      user.value = null
      loading.value = false
    }
  }

  return {
    user,
    loading,
    isAuthenticated,
    login,
    signup,
    handleCallback,
    fetchUser,
    logout
  }
})
