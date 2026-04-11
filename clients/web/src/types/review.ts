/**
 * 测评相关类型定义
 * Re-export from shared to maintain single source of truth
 */
export type {
  PaginatedResult,
  ReviewRatings,
  Review,
  ReviewContentCheck,
  PostReviewParams,
} from '@stuhelper/shared'

export {
  isValidRatings,
  normalizeReview,
  normalizeReviews,
  normalizeReviewList,
  normalizeContentCheck,
} from '@stuhelper/shared'
