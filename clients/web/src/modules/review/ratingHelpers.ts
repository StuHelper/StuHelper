import type { TermRatingStats } from '@stuhelper/shared/course'
import type { Review } from '@stuhelper/shared/review'
import { getRatingColor } from '@/design-system/rating'

const DIMENSION_LABELS: Readonly<Record<string, string>> = {
  recommendation: 'review.detail.recommend',
  content_quality: 'review.detail.contentQuality',
  workload: 'review.detail.workload',
  assessment: 'review.detail.exam',
}

export function dimensionLabel(key: string, t: (k: string) => string): string {
  const labelKey = DIMENSION_LABELS[key]
  if (labelKey) return t(labelKey)

  const translated = t(`review.ratingEmoji.${key}`)
  return translated === `review.ratingEmoji.${key}` ? key : translated
}

export function ratingBarColor(avg: number): string {
  return getRatingColor(Math.round(avg))
}

export function termReviewCount(term: TermRatingStats): number {
  if (!term.dimensions || term.dimensions.length === 0) return 0
  return term.dimensions[0].ratingCount ?? 0
}

export function avgRatingForReview(r: Review): number {
  const vals = Object.values(r.ratings || {})
  if (vals.length === 0) return 3
  return vals.reduce((a, b) => a + b, 0) / vals.length
}

export function reviewCardBorderClass(r: Review): string {
  const avg = avgRatingForReview(r)
  if (avg <= 1) return 'border-l-4 !border-l-danger shadow-[0_0_12px_var(--color-danger)/40]'
  if (avg <= 2) return 'border-l-4 !border-l-warning shadow-[0_0_12px_var(--color-warning)/40]'
  return ''
}
