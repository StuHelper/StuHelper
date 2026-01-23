/**
 * 认证工具函数
 */

// 用户信息存储键名
const USER_KEY = 'stuhelper_user'

// 用户信息类型
export interface StoredUser {
  id: string
  name: string
  displayName: string
  email: string
  avatar?: string
}

// 用户信息管理
export const userManager = {
  getUser(): StoredUser | null {
    const userStr = localStorage.getItem(USER_KEY)
    if (!userStr) return null
    try {
      return JSON.parse(userStr) as StoredUser
    } catch {
      return null
    }
  },

  setUser(user: StoredUser): void {
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  },

  removeUser(): void {
    localStorage.removeItem(USER_KEY)
  },

  isAuthenticated(): boolean {
    return !!this.getUser()
  }
}

// 清除所有认证信息
export const clearAuth = (): void => {
  userManager.removeUser()
  sessionStorage.removeItem('oauth_state')
}
