import { describe, expect, it } from 'vitest'

import {
  canListFullReviews,
  canManageReviews,
  canShowAdminEntry,
} from '../adminAccess'

describe('admin access helpers', () => {
  it('uses scoped capabilities for admin visibility even when global capabilities are empty', () => {
    const user = {
      capabilities: ['admin:reviews:manage'],
      globalCapabilities: [],
      canAccessAdmin: true,
    }

    expect(canShowAdminEntry(user)).toBe(true)
    expect(canManageReviews(user)).toBe(true)
  })

  it('detects full review list capability from scoped capabilities', () => {
    const user = {
      capabilities: ['review:list:full'],
      globalCapabilities: [],
      canAccessAdmin: false,
    }

    expect(canListFullReviews(user)).toBe(true)
  })

  it('denies admin visibility when neither admin flag nor scoped capability is present', () => {
    const user = {
      capabilities: [],
      globalCapabilities: ['admin:reviews:manage'],
      canAccessAdmin: false,
    }

    expect(canShowAdminEntry(user)).toBe(false)
    expect(canManageReviews(user)).toBe(false)
  })
})
