import { describe, expect, it } from 'vitest'

import { normalizeReviewList } from '../review'

describe('normalizeReviewList', () => {
  it('normalizes a valid paginated review response', () => {
    expect(
      normalizeReviewList({
        list: [{ id: 'review-1', content: 'hello' }],
        total: 1,
      } as never),
    ).toEqual({
      list: [{ id: 'review-1', content: 'hello' }],
      total: 1,
    })
  })

  it('fails closed when page data is missing or malformed', () => {
    expect(() => normalizeReviewList(null)).toThrow(
      'Invalid review list response',
    )
    expect(() => normalizeReviewList({ total: 0 } as never)).toThrow(
      'Invalid review list response',
    )
    expect(() => normalizeReviewList({ list: [], total: -1 } as never)).toThrow(
      'Invalid review list response',
    )
  })
})
