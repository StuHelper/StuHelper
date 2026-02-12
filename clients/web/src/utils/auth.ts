/**
 * 认证工具函数
 */

// 存储键名
const USER_KEY = 'stuhelper_user'
const TOKEN_EXPIRY_KEY = 'stuhelper_token_expiry'

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

// Token 过期时间管理
export const tokenExpiry = {
  // 设置过期时间戳（秒级 Unix 时间戳）
  set(expiresInSeconds: number): void {
    const expiresAt = Date.now() + expiresInSeconds * 1000
    localStorage.setItem(TOKEN_EXPIRY_KEY, String(expiresAt))
  },

  // 获取过期时间戳（毫秒）
  get(): number | null {
    const val = localStorage.getItem(TOKEN_EXPIRY_KEY)
    if (!val) return null
    const num = Number(val)
    return Number.isFinite(num) ? num : null
  },

  remove(): void {
    localStorage.removeItem(TOKEN_EXPIRY_KEY)
  }
}

/**
 * 判断 token 是否已过期（客户端预检，不替代服务端校验）
 * 提前 60 秒判定过期，给刷新留余量
 */
export function isTokenExpired(): boolean {
  const expiresAt = tokenExpiry.get()
  if (expiresAt === null) return true
  return expiresAt < Date.now() + 60_000
}

// 清除所有认证信息
export const clearAuth = (): void => {
  userManager.removeUser()
  tokenExpiry.remove()
  sessionStorage.removeItem('oauth_state')
}
