import { describe, expect, it } from 'vitest'
import { readFileSync } from 'node:fs'
import { resolve } from 'node:path'

const repoRoot = resolve(__dirname, '../../../..')

function source(path: string): string {
  return readFileSync(resolve(repoRoot, path), 'utf-8')
}

const policySources: Record<string, string> = {
  reviewCard: source('src/components/business/review/ReviewCard.vue'),
  emojiRating: source('src/components/business/review/EmojiRating.vue'),
  ratingDisplay: source('src/components/business/review/RatingDisplay.vue'),
  ratingDistribution: source('src/components/business/review/RatingDistribution.vue'),
  semesterStatsGrid: source('src/components/business/review/SemesterStatsGrid.vue'),
  teacherStatsCard: source('src/components/business/review/TeacherStatsCard.vue'),
  courseDetailPage: source('src/modules/review/views/CourseDetailPage.vue'),
  teacherProfilePage: source('src/modules/review/views/TeacherProfilePage.vue'),
  teacherHubPage: source('src/modules/course/views/TeacherHubPage.vue'),
  ratingBar: source('src/components/common/RatingBar.vue'),
  ratingCircle: source('src/components/common/RatingCircle.vue'),
}

describe('review community rating display policy', () => {
  it('does not render exact rating numbers in review-community surfaces', () => {
    Object.entries(policySources).forEach(([name, text]) => {
      expect(text, name).not.toContain('toFixed(')
      expect(text, name).not.toContain('show-value')
      expect(text, name).not.toContain('showValue')
    })
  })

  it('uses the published-review face paths for visible emoji ratings', () => {
    expect(policySources.emojiRating).toContain('getRatingFacePath')
    expect(policySources.ratingDisplay).toContain('getRatingFacePath')
    expect(policySources.ratingDistribution).toContain('<EmojiRating')
    expect(policySources.reviewCard).toContain('<EmojiRating')
    expect(policySources.teacherHubPage).toContain('<EmojiRating')
    expect(policySources.teacherProfilePage).toContain('<EmojiRating')
  })

  it('keeps chart tooltips non-numeric for rating values', () => {
    expect(policySources.teacherProfilePage).toContain("return `${p.name}<br/>${t('teaching.profile.ratingLabel')}`")
  })
})
