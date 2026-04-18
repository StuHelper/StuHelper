import { describe, expect, it } from 'vitest'

import { getRatingColor, getScaledRatingColor, toRatingBucket } from '../rating'

describe('rating design-system helpers', () => {
  it('normalizes fractional rating values to the nearest bucket', () => {
    expect(toRatingBucket(1.49)).toBe(1)
    expect(toRatingBucket(1.5)).toBe(2)
    expect(toRatingBucket(2.5)).toBe(3)
    expect(toRatingBucket(4.6)).toBe(5)
  })

  it('maps buckets to shared rating CSS variables', () => {
    expect(getRatingColor(2.5)).toBe('var(--color-rating-3)')
    expect(getRatingColor(4.2)).toBe('var(--color-rating-4)')
    expect(getRatingColor(5.1)).toBe('var(--color-rating-5)')
  })

  it('supports value/max scales for progress components', () => {
    expect(getScaledRatingColor(1, 5)).toBe('var(--color-rating-1)')
    expect(getScaledRatingColor(3, 5)).toBe('var(--color-rating-3)')
    expect(getScaledRatingColor(4, 5)).toBe('var(--color-rating-4)')
  })
})
