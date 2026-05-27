import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearAuth,
  consumeOAuthState,
  isTokenExpired,
  tokenExpiry,
  userManager,
  type StoredUser,
} from '../auth'

class MemoryStorage implements Storage {
  private store = new Map<string, string>()

  get length() {
    return this.store.size
  }

  clear() {
    this.store.clear()
  }

  getItem(key: string) {
    return this.store.get(key) ?? null
  }

  key(index: number) {
    return Array.from(this.store.keys())[index] ?? null
  }

  removeItem(key: string) {
    this.store.delete(key)
  }

  setItem(key: string, value: string) {
    this.store.set(key, value)
  }
}

const localStorageMock = new MemoryStorage()
const sessionStorageMock = new MemoryStorage()

Object.defineProperty(globalThis, 'localStorage', {
  value: localStorageMock,
  configurable: true,
})

Object.defineProperty(globalThis, 'sessionStorage', {
  value: sessionStorageMock,
  configurable: true,
})

describe('auth utilities', () => {
  beforeEach(() => {
    localStorageMock.clear()
    sessionStorageMock.clear()
  })

  it('removes malformed user payloads from storage', () => {
    localStorage.setItem('stuhelper_user', JSON.stringify({ id: '', name: 1 }))

    expect(userManager.getUser()).toBeNull()
    expect(localStorage.getItem('stuhelper_user')).toBeNull()
  })

  it('persists and restores a valid user payload', () => {
    const user: StoredUser = {
      id: 'user_1',
      name: 'alice',
      displayName: 'Alice',
    }

    userManager.setUser(user)

    expect(userManager.getUser()).toEqual(user)
    expect(userManager.isAuthenticated()).toBe(true)
  })

  it('strips capability-like fields before persisting the cached user', () => {
    userManager.setUser({
      id: 'user_1',
      name: 'alice',
      displayName: 'Alice',
      avatar: 'https://example.com/avatar.png',
      globalCapabilities: ['admin:reviews:manage'],
      canAccessAdmin: true,
    } as StoredUser)

    expect(JSON.parse(localStorage.getItem('stuhelper_user') ?? 'null')).toEqual({
      id: 'user_1',
      name: 'alice',
      displayName: 'Alice',
      avatar: 'https://example.com/avatar.png',
    })
  })

  it('treats tokens inside the safety buffer as expired', () => {
    tokenExpiry.set(30)

    expect(isTokenExpired()).toBe(true)

    tokenExpiry.set(120)
    expect(isTokenExpired()).toBe(false)
  })

  it('does not proactively refresh when the token expiry is unknown', () => {
    expect(isTokenExpired()).toBe(false)
  })

  it('clears auth and redirect session state together', () => {
    localStorage.setItem('stuhelper_user', JSON.stringify({ id: 'user_2', name: 'bob', displayName: 'Bob' }))
    localStorage.setItem('stuhelper_token_expiry', String(Date.now() + 60_000))
    sessionStorage.setItem('oauth_state', 'oauth-state')
    sessionStorage.setItem('post_login_redirect', '/courses/reviews/post')

    clearAuth()

    expect(localStorage.getItem('stuhelper_user')).toBeNull()
    expect(localStorage.getItem('stuhelper_token_expiry')).toBeNull()
    expect(sessionStorage.getItem('oauth_state')).toBeNull()
    expect(sessionStorage.getItem('post_login_redirect')).toBeNull()
  })

  it('consumes a matching oauth state and clears it from session storage', () => {
    sessionStorage.setItem('oauth_state', 'oauth-state')

    expect(consumeOAuthState('oauth-state')).toBe(true)
    expect(sessionStorage.getItem('oauth_state')).toBeNull()
  })

  it('rejects a mismatched oauth state and still clears it from session storage', () => {
    sessionStorage.setItem('oauth_state', 'oauth-state')

    expect(consumeOAuthState('other-state')).toBe(false)
    expect(sessionStorage.getItem('oauth_state')).toBeNull()
  })
})
