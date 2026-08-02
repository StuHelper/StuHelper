import type { ApiClient } from '@stuhelper/shared/api'
import { createReviewApi } from '@stuhelper/shared/api'
import { readReviewPagePayload } from '@/modules/review/reviewListPayload'
import {
  normalizeContentCheck,
  type PaginatedResult,
  type Review,
  type ReviewContentCheck,
} from '@stuhelper/shared/review'

type ReviewWireApi = ReturnType<typeof createReviewApi>

export type ReviewAppApi = ReviewWireApi & {
  checkContentResult(data: Parameters<ReviewWireApi['checkContent']>[0]): Promise<ReviewContentCheck>
  getLatestReviewsPage(
    params?: Parameters<ReviewWireApi['getLatestReviews']>[0],
  ): Promise<PaginatedResult<Review>>
  getReviewsPage(
    courseId: Parameters<ReviewWireApi['getReviews']>[0],
    params?: Parameters<ReviewWireApi['getReviews']>[1],
  ): Promise<PaginatedResult<Review>>
  searchReviewsPage(
    params?: Parameters<ReviewWireApi['searchReviews']>[0],
    options?: Parameters<ReviewWireApi['searchReviews']>[1],
  ): Promise<PaginatedResult<Review>>
}

export function createReviewAppApi(client: ApiClient): ReviewAppApi {
  const wire = createReviewApi(client)

  return {
    ...wire,
    async checkContentResult(data) {
      const result = await wire.checkContent(data)
      return normalizeContentCheck(result.data?.data)
    },
    async getLatestReviewsPage(params) {
      const result = await wire.getLatestReviews(params)
      return readReviewPagePayload(result.data?.data)
    },
    async getReviewsPage(courseId, params) {
      const result = await wire.getReviews(courseId, params)
      return readReviewPagePayload(result.data?.data)
    },
    async searchReviewsPage(params, options) {
      const result = await wire.searchReviews(params, options)
      return readReviewPagePayload(result.data?.data)
    },
  }
}
