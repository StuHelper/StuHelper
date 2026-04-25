import { describe, expect, it } from 'vitest'

import {
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
