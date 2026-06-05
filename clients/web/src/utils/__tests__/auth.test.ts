import { beforeEach, describe, expect, it } from 'vitest'
import {
  clearAuth,
  consumeIdentityCodeVerifier,
  consumeIdentityOAuthState,
  consumeOAuthState,
  isTokenExpired,
  storeIdentityCodeVerifier,
  storeIdentityOAuthState,
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

class ThrowingStorage implements Storage {
  get length() {
    return 0
  }

  clear() {
    throw new Error('storage unavailable')
  }

  getItem() {
    throw new Error('storage unavailable')
  }

  key() {
    throw new Error('storage unavailable')
  }

  removeItem() {
    throw new Error('storage unavailable')
  }

  setItem() {
    throw new Error('storage unavailable')
  }
}

const localStorageMock = new MemoryStorage()
const sessionStorageMock = new MemoryStorage()

describe('auth utilities', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'localStorage', {
      value: localStorageMock,
      configurable: true,
    })

    Object.defineProperty(globalThis, 'sessionStorage', {
      value: sessionStorageMock,
      configurable: true,
    })

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
    storeIdentityOAuthState('identity-state')
    storeIdentityCodeVerifier('identity-verifier')
    sessionStorage.setItem('post_login_redirect', '/courses/reviews/post')

    clearAuth()

    expect(localStorage.getItem('stuhelper_user')).toBeNull()
    expect(localStorage.getItem('stuhelper_token_expiry')).toBeNull()
    expect(sessionStorage.getItem('identity_oauth_state')).toBeNull()
    expect(sessionStorage.getItem('identity_code_verifier')).toBeNull()
    expect(sessionStorage.getItem('post_login_redirect')).toBeNull()
  })

  it('consumes a matching identity oauth state and clears it from session storage', () => {
    storeIdentityOAuthState('identity-state')

    expect(consumeIdentityOAuthState('identity-state')).toBe(true)
    expect(sessionStorage.getItem('identity_oauth_state')).toBeNull()
  })

  it('rejects a mismatched identity oauth state and still clears it from session storage', () => {
    storeIdentityOAuthState('identity-state')

    expect(consumeIdentityOAuthState('other-state')).toBe(false)
    expect(sessionStorage.getItem('identity_oauth_state')).toBeNull()
  })

  it('degrades when localStorage is unavailable', () => {
    const user: StoredUser = {
      id: 'user_1',
      name: 'alice',
      displayName: 'Alice',
    }

    Object.defineProperty(globalThis, 'localStorage', {
      value: new ThrowingStorage(),
      configurable: true,
    })

    expect(() => userManager.setUser(user)).not.toThrow()
    expect(userManager.getUser()).toBeNull()
    expect(userManager.isAuthenticated()).toBe(false)
    expect(() => userManager.removeUser()).not.toThrow()
    expect(() => tokenExpiry.set(60)).not.toThrow()
    expect(tokenExpiry.get()).toBeNull()
    expect(isTokenExpired()).toBe(false)
    expect(() => tokenExpiry.remove()).not.toThrow()
  })

  it('degrades when sessionStorage is unavailable', () => {
    Object.defineProperty(globalThis, 'sessionStorage', {
      value: new ThrowingStorage(),
      configurable: true,
    })

    expect(() => storeIdentityOAuthState('identity-state')).not.toThrow()
    expect(() => storeIdentityCodeVerifier('identity-verifier')).not.toThrow()
    expect(consumeIdentityOAuthState('identity-state')).toBe(false)
    expect(consumeOAuthState('oauth-state')).toBe(false)
    expect(consumeIdentityCodeVerifier()).toBe('')
    expect(() => clearAuth()).not.toThrow()
  })
})
