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
    expect(policySources.emojiRating).toContain('role="img"')
    expect(policySources.emojiRating).toContain('review.rating.face${normalizedValue}')
    expect(policySources.ratingDisplay).toContain('getRatingFacePath')
    expect(policySources.ratingDistribution).toContain('<EmojiRating')
    expect(policySources.reviewCard).toContain('<EmojiRating')
    expect(policySources.teacherHubPage).toContain('<EmojiRating')
    expect(policySources.teacherProfilePage).toContain('<EmojiRating')
  })

  it('keeps the rating trend surface non-numeric', () => {
    expect(policySources.teacherProfilePage).toContain('<EmojiRating :value="point.avgRating"')
    expect(policySources.teacherProfilePage).not.toContain('formatRating(')
  })

  it('gives live course rating bars qualitative accessible names', () => {
    expect(policySources.courseDetailPage).toContain('role="img"')
    expect(policySources.courseDetailPage).toContain(
      ':aria-label="dimensionRatingBarAriaLabel(dim)"',
    )
    expect(policySources.courseDetailPage).toContain('aria-hidden="true"')
  })
})
