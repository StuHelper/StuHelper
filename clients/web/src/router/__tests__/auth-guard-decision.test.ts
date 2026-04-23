import { describe, expect, it } from 'vitest'

import {
  hasRequiredRouteCapabilityAccess,
  resolveProtectedRouteAuthFailure,
} from '../auth-guard-decision'

describe('resolveProtectedRouteAuthFailure', () => {
  it('cancels protected navigation when session bootstrap failed but auth state is unresolved', () => {
    expect(
      resolveProtectedRouteAuthFailure({
        redirect: '/user/reviews',
        refreshFailed: false,
        requiresAuthRoute: true,
        sessionBootstrapFailed: true,
        stillAuthenticated: false,
      }),
    ).toBe(false)
  })

  it('cancels protected navigation when refresh failed but local session is still present', () => {
    expect(
      resolveProtectedRouteAuthFailure({
        redirect: '/user/reviews',
        refreshFailed: true,
        requiresAuthRoute: true,
        sessionBootstrapFailed: false,
        stillAuthenticated: true,
      }),
    ).toBe(false)
  })

  it('redirects to login only when refresh failed and local session is already cleared', () => {
    expect(
      resolveProtectedRouteAuthFailure({
        redirect: '/user/reviews',
        refreshFailed: true,
        requiresAuthRoute: true,
        sessionBootstrapFailed: false,
        stillAuthenticated: false,
      }),
    ).toEqual({
      name: 'login',
      query: { redirect: '/user/reviews' },
      replace: true,
    })
  })

  it('does nothing for public routes', () => {
    expect(
      resolveProtectedRouteAuthFailure({
        redirect: '/about',
        refreshFailed: true,
        requiresAuthRoute: false,
        sessionBootstrapFailed: true,
        stillAuthenticated: false,
      }),
    ).toBeNull()
  })
})

describe('hasRequiredRouteCapabilityAccess', () => {
  it('allows protected routes when the required capability only exists in full capabilities', () => {
    expect(
      hasRequiredRouteCapabilityAccess(
        ['review:create'],
        ['review:create'],
      ),
    ).toBe(true)
  })

  it('rejects protected routes when the required capability is missing', () => {
    expect(
      hasRequiredRouteCapabilityAccess(
        ['review:list:full'],
        ['review:create'],
      ),
    ).toBe(false)
  })
})
