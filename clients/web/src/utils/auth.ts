// 用户信息存储键名
const USER_KEY = 'stuhelper_user'

// 用户信息类型
export interface StoredUser {
  id: string
  name: string
  display_name: string
  email: string
  avatar?: string
}

// 用户信息管理
export const userManager = {
  // 获取用户信息
  getUser(): StoredUser | null {
    const userStr = localStorage.getItem(USER_KEY)
    if (!userStr) return null
    try {
      return JSON.parse(userStr) as StoredUser
    } catch {
      return null
    }
  },

  // 设置用户信息
  setUser(user: StoredUser): void {
    localStorage.setItem(USER_KEY, JSON.stringify(user))
  },

  // 移除用户信息
  removeUser(): void {
    localStorage.removeItem(USER_KEY)
  },

  // 检查是否已登录（基于本地存储的用户信息）
  isAuthenticated(): boolean {
    return !!this.getUser()
  }
}

// 清除所有认证信息
export const clearAuth = (): void => {
  userManager.removeUser()
}
