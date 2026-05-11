import { describe, expect, it } from 'vitest'

import { resolveAdminConsoleURL } from '../adminUrl'

describe('resolveAdminConsoleURL', () => {
  it('falls back to the same-origin admin path when no URL is configured', () => {
    expect(resolveAdminConsoleURL()).toBe('/admin/')
    expect(resolveAdminConsoleURL('   ')).toBe('/admin/')
  })

  it('adds the admin base path to a host-only public admin URL', () => {
    expect(resolveAdminConsoleURL('http://localhost:3001')).toBe('http://localhost:3001/admin/')
  })

  it('preserves an explicit admin base path', () => {
    expect(resolveAdminConsoleURL('http://localhost:3001/admin')).toBe('http://localhost:3001/admin/')
    expect(resolveAdminConsoleURL('/admin')).toBe('/admin/')
  })
})
