/**
 * 认证工具函数
 */

// 存储键名
const USER_KEY = 'stuhelper_user'
const TOKEN_EXPIRY_KEY = 'stuhelper_token_expiry'

// 用户信息类型（仅存储 UI 必需字段，最小化 localStorage 暴露面）
export interface StoredUser {
  id: string
  name: string
  displayName: string
  avatar?: string
  isAdmin?: boolean
}

// M-105: 校验 localStorage 中的用户数据结构，包含 ID 格式验证
function isValidStoredUser(data: unknown): data is StoredUser {
  if (typeof data !== 'object' || data === null) return false
  const obj = data as Record<string, unknown>
  return (
    typeof obj.id === 'string' && obj.id.length > 0 &&
    /^[a-zA-Z0-9_-]+$/.test(obj.id) &&
    typeof obj.name === 'string' && obj.name.length > 0 &&
    typeof obj.displayName === 'string' &&
    (obj.avatar === undefined || typeof obj.avatar === 'string') &&
    (obj.isAdmin === undefined || typeof obj.isAdmin === 'boolean')
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
    // H5: 仅存储 UI 必需字段，不持久化 email 等敏感信息
    const minimal: StoredUser = {
      id: user.id,
      name: user.name,
      displayName: user.displayName,
      ...(user.avatar !== undefined && { avatar: user.avatar }),
      ...(user.isAdmin !== undefined && { isAdmin: user.isAdmin }),
    }
    localStorage.setItem(USER_KEY, JSON.stringify(minimal))
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

// Token 过期预检缓冲时间（秒）
const TOKEN_EXPIRY_BUFFER_SECONDS = 60

/**
 * 判断 token 是否已过期（客户端预检，不替代服务端校验）
 * 提前 TOKEN_EXPIRY_BUFFER_SECONDS 秒判定过期，给刷新留余量
 */
export function isTokenExpired(): boolean {
  const expiresAt = tokenExpiry.get()
  if (expiresAt === null) return true
  return expiresAt < Date.now() + TOKEN_EXPIRY_BUFFER_SECONDS * 1000
}

// 清除所有认证信息（含草稿重定向状态）
export const clearAuth = (): void => {
  userManager.removeUser()
  tokenExpiry.remove()
  sessionStorage.removeItem('oauth_state')
  sessionStorage.removeItem('post_login_redirect')
  sessionStorage.removeItem('draft_redirect')
  sessionStorage.removeItem('draft_pending')
}
