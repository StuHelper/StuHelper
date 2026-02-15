import { describe, it, expect } from 'vitest'
import { isTokenExpired } from '../auth'

// TOKEN_EXPIRY_BUFFER_SECONDS 在 auth.ts 中为 60 秒（未导出），
// isTokenExpired 会提前 60 秒判定过期

describe('isTokenExpired', () => {
  it('当 localStorage 无过期时间时应返回 true', () => {
    localStorage.removeItem('stuhelper_token_expiry')
    expect(isTokenExpired()).toBe(true)
  })

  it('当 token 远未过期时应返回 false', () => {
    // 设置一个 10 分钟后过期的时间戳
    const futureMs = Date.now() + 10 * 60 * 1000
    localStorage.setItem('stuhelper_token_expiry', String(futureMs))
    expect(isTokenExpired()).toBe(false)
  })

  it('当 token 即将在缓冲期内过期时应返回 true', () => {
    // 设置一个 30 秒后过期的时间戳（小于 60 秒缓冲）
    const soonMs = Date.now() + 30 * 1000
    localStorage.setItem('stuhelper_token_expiry', String(soonMs))
    expect(isTokenExpired()).toBe(true)
  })
})
