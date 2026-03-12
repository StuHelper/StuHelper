import { describe, expect, it } from 'vitest'
import { HOME_FEATURES, UNIAPPX_EXPERIMENTAL_NOTICE, USER_MENU_ITEMS } from '../../../../uniappx/src/config/featureSurface'

describe('uniappx feature surface', () => {
  it('does not expose missing user pages in the user menu', () => {
    expect(USER_MENU_ITEMS.map(item => item.path)).toEqual(['/pages/user/notifications'])
  })

  it('does not expose teacher profile without a required id route parameter', () => {
    expect(HOME_FEATURES.some(item => item.path === '/pages/teacher/profile')).toBe(false)
  })

  it('states that uniappx is currently experimental', () => {
    expect(UNIAPPX_EXPERIMENTAL_NOTICE).toContain('实验性')
  })
})
