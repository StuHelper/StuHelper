// @vitest-environment jsdom

import { afterEach, beforeEach, describe, expect, it, vi } from 'vitest'
import { resolveProfileCompletionActionURL } from '../profileCompletionAction'

describe('profile completion action URLs', () => {
  beforeEach(() => {
    vi.stubEnv('VITE_WEB_URL', 'https://stuhelper.com')
  })

  afterEach(() => {
    vi.unstubAllEnvs()
  })

  it.each([
    'profile.username',
    'profile.email',
    'profile.avatar',
  ])('sends identity-provider field %s to the upstream account settings page', (key) => {
    expect(
      resolveProfileCompletionActionURL(
        { key, actionURL: '/account/profile' },
        'https://sso.stuhelper.com/account',
      ),
    ).toBe('https://sso.stuhelper.com/account')
  })

  it.each([
    ['profile.phone', '/user/phone-binding'],
    ['profile.identity', '/user/student-verification?method=real_name_identity_check'],
    ['profile.student', '/user/student-verification'],
    ['profile.school', '/user/student-verification'],
  ])('keeps StuHelper-owned field %s on its local writable flow', (key, actionURL) => {
    expect(
      resolveProfileCompletionActionURL(
        { key, actionURL },
        'https://sso.stuhelper.com/account',
      ),
    ).toBe(`https://stuhelper.com${actionURL}`)
  })

  it.each([
    'profile.username',
    'profile.email',
    'profile.avatar',
  ])('falls back to the declared action for %s when account settings are unavailable', (key) => {
    expect(
      resolveProfileCompletionActionURL({
        key,
        actionURL: '/account/profile',
      }),
    ).toBe('https://stuhelper.com/account/profile')
  })
})
