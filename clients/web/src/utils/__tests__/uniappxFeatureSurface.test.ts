import { describe, expect, it } from 'vitest'
import {
  getHomeFeatures,
  getUniappxAppNotice,
  getUserMenuItems,
} from '../../../../uniappx/src/config/featureSurface'

describe('uniappx feature surface', () => {
  it('exposes the current user center routes', () => {
    expect(getUserMenuItems().map((item) => item.path)).toEqual([
      '/pages/user/reviews',
      '/pages/user/votes',
      '/pages/user/favorites',
      '/pages/user/notifications',
    ])
  })

  it('does not expose teacher profile without a required id route parameter', () => {
    expect(getHomeFeatures().some((item) => item.path === '/pages/teacher/profile')).toBe(false)
  })

  it('states that uniappx now uses the shared openapi contract', () => {
    expect(getUniappxAppNotice()).toContain('OpenAPI')
  })
})
