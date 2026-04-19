import { describe, expect, it } from 'vitest'

import { resolveProtectedRouteAuthFailure } from '../auth-guard-decision'

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
