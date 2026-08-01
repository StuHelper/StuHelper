import { describe, expect, it } from 'vitest'

import {
  readReviewPagePayload,
  readReviewPayload,
} from '../reviewListPayload'

const review = {
  id: 'review-1',
  courseID: 101,
  courseName: '数据结构与算法',
  teacherID: null,
  teacherName: '张老师',
  termID: '2026-spring',
  termName: '2026 春',
  title: '课堂体验',
  content: '课程内容扎实。',
  ratings: { recommendation: 4 },
  likeCount: 3,
  dislikeCount: 0,
  replyCount: 1,
  status: 'published',
  createdAt: '2026-04-01T10:00:00Z',
  updatedAt: '2026-04-01T10:00:00Z',
}

describe('review list payload readers', () => {
  it('reads a valid review page', () => {
    expect(
      readReviewPagePayload({
        list: [review],
        total: 1,
      }),
    ).toEqual({
      list: [review],
      total: 1,
    })
  })

  it('fails closed when a required review field is missing', () => {
    const malformedReview: Partial<typeof review> = { ...review }
    delete malformedReview.termID

    expect(() =>
      readReviewPagePayload({
        list: [malformedReview],
        total: 1,
      }),
    ).toThrow('Invalid review list response')
  })

  it('fails closed when review status is outside the contract', () => {
    expect(() =>
      readReviewPayload({
        ...review,
        status: 'approved',
      }),
    ).toThrow('Invalid review response')
  })

  it('fails closed when review ratings are outside the contract', () => {
    expect(() =>
      readReviewPayload({
        ...review,
        ratings: { recommendation: 6 },
      }),
    ).toThrow('Invalid review response')
  })

  it.each(['like', 'dislike'] as const)(
    'preserves the current user %s vote',
    userVote => {
      expect(
        readReviewPayload({
          ...review,
          userVote,
        }).userVote,
      ).toBe(userVote)
    },
  )

  it('keeps an omitted userVote unset', () => {
    expect(readReviewPayload(review).userVote).toBeUndefined()
  })

  it('fails closed when userVote is outside the contract', () => {
    expect(() =>
      readReviewPayload({
        ...review,
        userVote: 'upvote',
      }),
    ).toThrow('Invalid review response')
  })
})
