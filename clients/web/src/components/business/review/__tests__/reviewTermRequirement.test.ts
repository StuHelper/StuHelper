import { describe, expect, it } from 'vitest'
import { buildCreateReviewPayload } from '../reviewPayload'

describe('buildCreateReviewPayload term requirement', () => {
  it('throws when termID is missing', () => {
    expect(() => buildCreateReviewPayload({
      courseID: 1,
      title: 'Title',
      content: 'Content content',
      ratings: { content: 5 },
    } as never)).toThrow('termID is required')
  })

  it('keeps a real termID in the payload', () => {
    expect(buildCreateReviewPayload({
      courseID: 1,
      title: 'Title',
      content: 'Content content',
      ratings: { content: 5 },
      termID: '2025-1'
    })).toEqual({
      courseID: 1,
      title: 'Title',
      content: 'Content content',
      ratings: { content: 5 },
      termID: '2025-1'
    })
  })
})
