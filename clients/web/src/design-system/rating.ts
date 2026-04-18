import { ratingColors } from './tokens'

export type RatingBucket = 1 | 2 | 3 | 4 | 5

function clampNumber(value: number, min: number, max: number): number {
  return Math.max(min, Math.min(max, value))
}

export function toRatingBucket(value: number): RatingBucket {
  if (!Number.isFinite(value)) {
    return 1
  }

  return clampNumber(Math.round(value), 1, 5) as RatingBucket
}

export function getRatingColor(value: number): string {
  return ratingColors[toRatingBucket(value)]
}

export function getScaledRatingColor(value: number, max = 5): string {
  if (!Number.isFinite(value) || !Number.isFinite(max) || max <= 0) {
    return ratingColors[1]
  }

  return getRatingColor((value / max) * 5)
}

