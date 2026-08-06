import {
  createAuthApi,
  createCourseApi,
  createDraftApi,
  createIdentityApi,
  createNotificationApi,
  createRatingApi,
  createReplyApi,
  createReviewApi,
  createStudentVerificationApi,
  createUserApi,
} from '@stuhelper/shared/api'
import { apiClient } from './shared-client'

export const api = {
  auth: createAuthApi(apiClient),
  course: createCourseApi(apiClient),
  draft: createDraftApi(apiClient),
  identity: createIdentityApi(apiClient),
  notification: createNotificationApi(apiClient),
  rating: createRatingApi(apiClient),
  reply: createReplyApi(apiClient),
  review: createReviewApi(apiClient),
  studentVerification: createStudentVerificationApi(apiClient),
  user: createUserApi(apiClient),
}

export type { components, operations } from '@stuhelper/shared/types'
