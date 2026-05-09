/**
 * 认证工具函数
 */

// 存储键名
const USER_KEY = 'stuhelper_user'
const TOKEN_EXPIRY_KEY = 'stuhelper_token_expiry'
const OAUTH_STATE_KEY = 'oauth_state'

// 用户信息类型（仅持久化最小展示字段，权限信息必须来自服务端会话）
export interface StoredUser {
  id: string
  name: string
  displayName: string
  avatar?: string
}

function isNonEmptyString(value: unknown): value is string {
  return typeof value === 'string' && value.length > 0
}

function constantTimeEqual(left: string, right: string): boolean {
  const maxLength = Math.max(left.length, right.length)
  let diff = left.length ^ right.length

  for (let index = 0; index < maxLength; index += 1) {
    const leftCode = left.charCodeAt(index) || 0
    const rightCode = right.charCodeAt(index) || 0
    diff |= leftCode ^ rightCode
  }

  return diff === 0
}

// 校验 localStorage 中的用户数据结构，包含 ID 格式验证
function isValidStoredUser(data: unknown): data is StoredUser {
  if (typeof data !== 'object' || data === null) return false
  const obj = data as Record<string, unknown>
  return (
    isNonEmptyString(obj.id) &&
    /^[a-zA-Z0-9_-]+$/.test(obj.id) &&
    isNonEmptyString(obj.name) &&
    typeof obj.displayName === 'string' &&
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
    } catch (_error) { void _error;
      localStorage.removeItem(USER_KEY)
      return null
    }
  },

  setUser(user: StoredUser): void {
    // 仅存储展示必需字段，不持久化角色/能力等可影响 UI 权限面的数据
    const minimal: StoredUser = {
      id: user.id,
      name: user.name,
      displayName: user.displayName,
      ...(user.avatar !== undefined && { avatar: user.avatar }),
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

export function consumeOAuthState(callbackState: string): boolean {
  const expectedState = sessionStorage.getItem(OAUTH_STATE_KEY) ?? ''
  sessionStorage.removeItem(OAUTH_STATE_KEY)

  if (!isNonEmptyString(expectedState) || !isNonEmptyString(callbackState)) {
    return false
  }

  return constantTimeEqual(expectedState, callbackState)
}

// 清除所有认证信息（含草稿重定向状态）
export const clearAuth = (): void => {
  userManager.removeUser()
  tokenExpiry.remove()
  sessionStorage.removeItem(OAUTH_STATE_KEY)
  sessionStorage.removeItem('post_login_redirect')
  sessionStorage.removeItem('draft_redirect')
  sessionStorage.removeItem('draft_pending')
}
