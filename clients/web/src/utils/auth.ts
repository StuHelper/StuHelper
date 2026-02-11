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

// 校验 localStorage 中的用户数据结构
function isValidStoredUser(data: unknown): data is StoredUser {
  if (typeof data !== 'object' || data === null) return false
  const obj = data as Record<string, unknown>
  return (
    typeof obj.id === 'string' && obj.id.length > 0 &&
    typeof obj.name === 'string' && obj.name.length > 0 &&
    typeof obj.displayName === 'string' &&
    typeof obj.email === 'string' &&
    (obj.avatar === undefined || typeof obj.avatar === 'string')
  )
}

// 用户信息管理
export const userManager = {
  getUser(): StoredUser | null {
    const userStr = localStorage.getItem(USER_KEY)
    if (!userStr) return null
    try {
      const parsed: unknown = JSON.parse(userStr)
      if (!isValidStoredUser(parsed)) {
        localStorage.removeItem(USER_KEY)
        return null
      }
      return parsed
    } catch {
      localStorage.removeItem(USER_KEY)
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
