import { describe, expect, it } from 'vitest'

import {
  localizeRatingDimension,
  ratingBarAriaLabel,
  ratingDimensionLabel,
} from '../ratingHelpers'
import {
  getRatingFacePath,
  normalizeRatingLevel,
  ratingFacePaths,
} from '../ratingFaces'

const zhMessages: Record<string, string> = {
  'review.ratingEmoji.difficulty': '课程难度',
  'review.ratingEmoji.usefulness': '实用性',
  'review.ratingEmoji.teaching': '教学质量',
  'review.ratingEmoji.grading': '给分',
  'review.ratingEmoji.fallback': '评分',
  'review.ratingEmojiDescription.difficulty': '课程内容的难易程度',
}

function t(key: string): string {
  return zhMessages[key] ?? key
}

function interpolatingT(
  key: string,
  params?: Record<string, number | string>,
): string {
  const messages: Record<string, string> = {
    ...zhMessages,
    'review.detail.ratingBarAria': '{dimension}：{level}',
    'review.rating.face1': '很差',
    'review.rating.face3': '一般',
    'review.rating.face5': '超赞',
  }
  return (messages[key] ?? key).replace(
    /\{(\w+)\}/g,
    (_, name: string) => String(params?.[name] ?? `{${name}}`),
  )
}

describe('rating dimension helpers', () => {
  it('translates backend rating keys instead of exposing raw identifiers', () => {
    expect(ratingDimensionLabel({ key: 'difficulty', t })).toBe('课程难度')
    expect(ratingDimensionLabel({ key: 'usefulness', t })).toBe('实用性')
    expect(ratingDimensionLabel({ key: 'teaching', t })).toBe('教学质量')
    expect(ratingDimensionLabel({ key: 'grading', t })).toBe('给分')
  })

  it('keeps custom dimension names when no translation exists', () => {
    expect(
      ratingDimensionLabel({
        key: 'custom_dimension',
        fallback: '自定义维度',
        t,
      }),
    ).toBe('自定义维度')

    expect(ratingDimensionLabel({ key: 'custom_dimension', t })).toBe('评分')
  })

  it('localizes API-provided rating dimensions without mutating input', () => {
    const dimension = {
      id: 'dim-1',
      schoolID: 4111010006,
      key: 'difficulty',
      name: 'Difficulty',
      description: 'Raw difficulty description',
      sortOrder: 1,
      isActive: true,
      createdAt: '2026-01-01T00:00:00Z',
      updatedAt: '2026-01-01T00:00:00Z',
    }

    const localized = localizeRatingDimension(dimension, t)

    expect(localized.name).toBe('课程难度')
    expect(localized.description).toBe('课程内容的难易程度')
    expect(dimension.name).toBe('Difficulty')
  })

  it('describes visual rating bars with qualitative text only', () => {
    const label = ratingBarAriaLabel({
      key: 'teaching',
      avgRating: 4.6,
      t: interpolatingT,
    })

    expect(label).toBe('教学质量：超赞')
    expect(label).not.toContain('4.6')
  })
})

describe('rating face helpers', () => {
  it('normalizes arbitrary rating values to the published-review face levels', () => {
    expect(normalizeRatingLevel(0)).toBe(1)
    expect(normalizeRatingLevel(3.4)).toBe(3)
    expect(normalizeRatingLevel(4.6)).toBe(5)
    expect(normalizeRatingLevel(Number.NaN)).toBe(3)
  })

  it('uses the same face path set for rating display and input', () => {
    expect(getRatingFacePath(5)).toBe(ratingFacePaths[5])
    expect(getRatingFacePath(99)).toBe(ratingFacePaths[5])
  })
})
