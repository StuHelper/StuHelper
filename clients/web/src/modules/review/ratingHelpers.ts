import type { RatingDimension, TermRatingStats } from '@stuhelper/shared/course'
import type { Review } from '@stuhelper/shared/review'
import { getRatingColor } from '@/design-system/rating'
import { normalizeRatingLevel } from '@/modules/review/ratingFaces'

type Translate = (
  key: string,
  params?: Record<string, number | string>,
) => string

interface DimensionTextOptions {
  key: string
  t: Translate
  fallback?: string | null
}

interface RatingBarTextOptions extends DimensionTextOptions {
  avgRating: number
}

const DIMENSION_LABEL_KEYS: Readonly<Record<string, string>> = {
  difficulty: 'review.ratingEmoji.difficulty',
  usefulness: 'review.ratingEmoji.usefulness',
  teaching: 'review.ratingEmoji.teaching',
  grading: 'review.ratingEmoji.grading',
  recommendation: 'review.detail.recommend',
  content_quality: 'review.detail.contentQuality',
  workload: 'review.detail.workload',
  assessment: 'review.detail.exam',
}

const DIMENSION_DESCRIPTION_KEYS: Readonly<Record<string, string>> = {
  difficulty: 'review.ratingEmojiDescription.difficulty',
  usefulness: 'review.ratingEmojiDescription.usefulness',
  teaching: 'review.ratingEmojiDescription.teaching',
  grading: 'review.ratingEmojiDescription.grading',
  recommendation: 'review.ratingEmojiDescription.recommendation',
  content_quality: 'review.ratingEmojiDescription.content_quality',
  workload: 'review.ratingEmojiDescription.workload',
  assessment: 'review.ratingEmojiDescription.assessment',
}

function translatedText(
  key: string,
  keyMap: Readonly<Record<string, string>>,
  t: Translate,
): string | null {
  const messageKey = keyMap[key]
  if (!messageKey) return null
  const translated = t(messageKey)
  return translated === messageKey ? null : translated
}

export function ratingDimensionLabel(options: DimensionTextOptions): string {
  const translated = translatedText(options.key, DIMENSION_LABEL_KEYS, options.t)
  if (translated) return translated

  const fallback = options.fallback?.trim()
  return fallback || options.t('review.ratingEmoji.fallback')
}

export function ratingDimensionDescription(
  options: DimensionTextOptions,
): string | undefined {
  const translated = translatedText(
    options.key,
    DIMENSION_DESCRIPTION_KEYS,
    options.t,
  )
  if (translated) return translated

  const fallback = options.fallback?.trim()
  return fallback || undefined
}

export function localizeRatingDimension(
  dimension: RatingDimension,
  t: Translate,
): RatingDimension {
  return {
    ...dimension,
    name: ratingDimensionLabel({
      key: dimension.key,
      fallback: dimension.name,
      t,
    }),
    description: ratingDimensionDescription({
      key: dimension.key,
      fallback: dimension.description,
      t,
    }),
  }
}

export function ratingBarColor(avg: number): string {
  return getRatingColor(Math.round(avg))
}

export function ratingBarAriaLabel(options: RatingBarTextOptions): string {
  const dimension = ratingDimensionLabel(options)
  const level = options.t(
    `review.rating.face${normalizeRatingLevel(options.avgRating)}`,
  )
  return options.t('review.detail.ratingBarAria', { dimension, level })
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
