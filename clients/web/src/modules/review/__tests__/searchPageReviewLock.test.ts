import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const searchPageSource = readFileSync(
  resolve(__dirname, '../views/SearchPage.vue'),
  'utf-8',
)
const courseDetailSource = readFileSync(
  resolve(__dirname, '../views/CourseDetailPage.vue'),
  'utf-8',
)

describe('SearchPage review locking', () => {
  it('renders review results through ReviewCard instead of direct content output', () => {
    expect(searchPageSource).toContain('<ReviewCard')
    expect(searchPageSource).toContain("import ReviewCard from '@/components/business/review/ReviewCard.vue'")
    expect(searchPageSource).not.toContain('v-text="review.content"')
    expect(searchPageSource).not.toContain('<EmojiRating')
  })

  it('keeps course detail review bodies behind the same lock state', () => {
    const lockIndex = courseDetailSource.indexOf('v-else-if="isReviewContentLocked"')
    const contentIndex = courseDetailSource.indexOf('v-text="r.content"')

    expect(courseDetailSource).toContain('<LockedReviewContent')
    expect(courseDetailSource).toContain('review.card.loginToView')
    expect(courseDetailSource).toContain('review.card.verifyToView')
    expect(courseDetailSource).toContain('canListFullReviews')
    expect(lockIndex).toBeGreaterThan(-1)
    expect(contentIndex).toBeGreaterThan(lockIndex)
  })
})
