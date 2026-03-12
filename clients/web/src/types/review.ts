/**
 * 测评相关类型定义
 * Re-export from shared to maintain single source of truth
 */
export type {
  ReviewRatings,
  Review,
  PostReviewParams,
} from '@stuhelper/shared'

export { isValidRatings } from '@stuhelper/shared'

import type { Review } from '@stuhelper/shared'

/**
 * Adapter: convert API-returned review objects to the business Review type.
 *
 * The OpenAPI codegen types ratings as `{ [key: string]: number }` while the
 * business layer uses `Record<string, RatingValue>` (RatingValue = 1|2|3|4|5).
 * The server guarantees values are always 1-5, so this cast is safe.
 *
 * This function centralizes the single unavoidable type assertion that bridges
 * the generated API types with the stricter business types, replacing scattered
 * `as Review[]` assertions across multiple view files.
 */
export function toReviews<T extends { ratings: Record<string, number> }>(
  apiReviews: T[]
): Review[] {
  return apiReviews as unknown as Review[]
}
