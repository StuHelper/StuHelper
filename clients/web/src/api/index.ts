import { apiClient } from './client'
import {
  createAuthApi,
  createCourseApi,
  createReviewApi,
  createUserApi,
  createIdentityApi,
  createUserAdminApi,
  createRbacApi,
  createNotificationApi,
  createDraftApi,
  createAdminApi,
  createRatingApi,
  createReplyApi
} from '@stuhelper/shared/api'

// 导出统一的 API 对象
export const api = {
  auth: createAuthApi(apiClient),
  course: createCourseApi(apiClient),
  review: createReviewApi(apiClient),
  user: createUserApi(apiClient),
  identity: createIdentityApi(apiClient),
  userAdmin: createUserAdminApi(apiClient),
  rbac: createRbacApi(apiClient),
  notification: createNotificationApi(apiClient),
  draft: createDraftApi(apiClient),
  admin: createAdminApi(apiClient),
  rating: createRatingApi(apiClient),
  reply: createReplyApi(apiClient)
}

// 导出类型
export type { components } from '@stuhelper/shared'
