import { describe, expect, it } from 'vitest'
import { buildCreateReviewPayload } from '../reviewPayload'

describe('buildCreateReviewPayload', () => {
  it('throws when termID is missing', () => {
    expect(() => buildCreateReviewPayload({
      courseID: 1,
      title: 'Title',
      content: 'Content content',
      ratings: { content: 5 },
      termID: '   '
    })).toThrow('termID is required')
  })

  it('keeps termID when a real term is provided', () => {
    const payload = buildCreateReviewPayload({
      courseID: 1,
      title: 'Title',
      content: 'Content content',
      ratings: { content: 5 },
      termID: '2025-1'
    })

    expect(payload).toEqual({
      courseID: 1,
      title: 'Title',
      content: 'Content content',
      ratings: { content: 5 },
      termID: '2025-1'
    })
  })
})
