import type { ApiClient } from './client'
import type { components, operations } from '../types/api.gen'

type AdminUpdateReviewRequest = components['schemas']['AdminUpdateReviewRequest']
type ProcessReportRequest = components['schemas']['ProcessReportRequest']
type BatchUpdateReviewsRequest = components['schemas']['BatchUpdateReviewsRequest']
type AdminEditReviewRequest = components['schemas']['AdminEditReviewRequest']
type CreateTeacherRequest = components['schemas']['CreateTeacherRequest']
type UpdateTeacherRequest = components['schemas']['UpdateTeacherRequest']
type CreateSensitiveWordRequest = components['schemas']['CreateSensitiveWordRequest']
type UpdateSensitiveWordRequest = components['schemas']['UpdateSensitiveWordRequest']
type OpenPlatformApproveScopeRequest = components['schemas']['OpenPlatformApproveScopeRequest']
type OpenPlatformRejectScopeRequest = components['schemas']['OpenPlatformRejectScopeRequest']
type OpenPlatformImportCasdoorAppRequest = components['schemas']['OpenPlatformImportCasdoorAppRequest']
type OpenPlatformLifecycleActionRequest = components['schemas']['OpenPlatformLifecycleActionRequest']
type OpenPlatformRedirectURIReviewRequest = components['schemas']['OpenPlatformRedirectURIReviewRequest']
type OpenPlatformSecretRotationRequest = components['schemas']['OpenPlatformSecretRotationRequest']
type OpenPlatformAdminConsentRevokeRequest = components['schemas']['OpenPlatformAdminConsentRevokeRequest']
type OpenPlatformScope = components['schemas']['OpenPlatformScope']
type OpenPlatformAppStatus = components['schemas']['OpenPlatformApp']['status'] | 'all'
type OpenPlatformTokenProbeResult = components['schemas']['OpenPlatformTokenProbeEvidence']['result']
type OpenPlatformResourceType = components['schemas']['OpenPlatformResourceType']
type OpenPlatformResourceGrantRequest = components['schemas']['OpenPlatformResourceGrantRequest']
type OpenPlatformResourceGrantResult =
  operations['listAdminOpenPlatformResourceGrants']['responses'][200]['content']['application/json']['data']

export const createAdminApi = (client: ApiClient) => ({
  getStats: () =>
    client.GET('/api/v1/course/review/admin/stats'),

  getReviews: (params?: { status?: 'published' | 'pending_review' | 'hidden' | 'deleted' | 'all'; page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/reviews', { params: { query: params } }),

  updateReview: (id: string, data: AdminUpdateReviewRequest) =>
    client.PUT('/api/v1/course/review/admin/reviews/{reviewID}', { params: { path: { reviewID: id } }, body: data }),

  editReview: (id: string, data: AdminEditReviewRequest) =>
    client.POST('/api/v1/course/review/admin/reviews/{reviewID}/edit', { params: { path: { reviewID: id } }, body: data }),

  batchUpdateReviews: (data: BatchUpdateReviewsRequest) =>
    client.PATCH('/api/v1/course/review/admin/reviews/batch', { body: data }),

  getReports: (params?: { status?: 'pending' | 'resolved' | 'rejected' | 'all'; page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/reports', { params: { query: params } }),

  processReport: (id: string, data: ProcessReportRequest) =>
    client.PUT('/api/v1/course/review/admin/reports/{reportID}', { params: { path: { reportID: id } }, body: data }),

  getLogs: (params?: { page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/logs', { params: { query: params } }),

  exportReviews: (params?: { format?: 'json' | 'ndjson' | 'csv'; status?: 'all' | 'published' | 'pending_review' | 'hidden' | 'deleted' }) =>
    client.GET('/api/v1/course/review/admin/export', { params: { query: params } }),

  getTeachers: (params?: { page?: number; pageSize?: number; search?: string; departmentID?: number }) =>
    client.GET('/api/v1/course/review/admin/teachers', { params: { query: params } }),

  createTeacher: (data: CreateTeacherRequest) =>
    client.POST('/api/v1/course/review/admin/teachers', { body: data }),

  updateTeacher: (id: number, data: UpdateTeacherRequest) =>
    client.PUT('/api/v1/course/review/admin/teachers/{teacherID}', { params: { path: { teacherID: id } }, body: data }),

  deleteTeacher: (id: number) =>
    client.DELETE('/api/v1/course/review/admin/teachers/{teacherID}', { params: { path: { teacherID: id } } }),

  getSensitiveWords: (params?: { page?: number; pageSize?: number; category?: string; level?: 'block' | 'warn' | 'review' }) =>
    client.GET('/api/v1/course/review/admin/sensitive-words', { params: { query: params } }),

  getContentFlags: (params?: { page?: number; pageSize?: number }) =>
    client.GET('/api/v1/course/review/admin/content-flags', { params: { query: params } }),

  clearContentFlag: (id: string) =>
    client.PUT('/api/v1/course/review/admin/content-flags/{reviewID}/clear', { params: { path: { reviewID: id } } }),

  createSensitiveWord: (data: CreateSensitiveWordRequest) =>
    client.POST('/api/v1/course/review/admin/sensitive-words', { body: data }),

  updateSensitiveWord: (id: string, data: UpdateSensitiveWordRequest) =>
    client.PUT('/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}', { params: { path: { sensitiveWordID: id } }, body: data }),

  deleteSensitiveWord: (id: string) =>
    client.DELETE('/api/v1/course/review/admin/sensitive-words/{sensitiveWordID}', { params: { path: { sensitiveWordID: id } } }),

  getOpenPlatformApps: (params?: { page?: number; pageSize?: number; status?: OpenPlatformAppStatus }) =>
    client.GET('/api/v1/admin/open-platform/apps', { params: { query: params } }),

  getOpenPlatformAuditEvents: (params?: {
    appID?: number
    eventType?: string
    page?: number
    pageSize?: number
    scope?: OpenPlatformScope
    userID?: number
  }) =>
    client.GET('/api/v1/admin/open-platform/audit-events', { params: { query: params } }),

  getOpenPlatformConsents: (params?: {
    appID?: number
    page?: number
    pageSize?: number
    userID?: number
  }) =>
    client.GET('/api/v1/admin/open-platform/consents', { params: { query: params } }),

  getOpenPlatformTokenProbeEvidence: (params?: {
    appID?: number
    clientID?: string
    page?: number
    pageSize?: number
    result?: OpenPlatformTokenProbeResult
    reviewerUserID?: number
  }) =>
    client.GET('/api/v1/admin/open-platform/token-probe-evidence', { params: { query: params } }),

  getOpenPlatformDisclosureReport: (params?: { windowHours?: number }) =>
    client.GET('/api/v1/admin/open-platform/disclosure-report', { params: { query: params } }),

  getOpenPlatformResourceGrants: (appID: number, resourceType: OpenPlatformResourceType) =>
    client.GET('/api/v1/admin/open-platform/apps/{appID}/resource-grants', {
      params: { path: { appID }, query: { resourceType } },
    }),

  grantOpenPlatformResourceAccess: (appID: number, data: OpenPlatformResourceGrantRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/resource-grants', {
      body: data,
      params: { path: { appID } },
    }),

  revokeOpenPlatformResourceAccess: (appID: number, data: OpenPlatformResourceGrantRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/resource-grants/revoke', {
      body: data,
      params: { path: { appID } },
    }),

  revokeOpenPlatformConsent: (appID: number, data: OpenPlatformAdminConsentRevokeRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/consents/revoke', {
      body: data,
      params: { path: { appID } },
    }),

  approveOpenPlatformScope: (appID: number, scope: OpenPlatformScope, data?: OpenPlatformApproveScopeRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/approve', {
      body: data,
      params: { path: { appID, scope } },
    }),

  rejectOpenPlatformScope: (appID: number, scope: OpenPlatformScope, data?: OpenPlatformRejectScopeRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/scopes/{scope}/reject', {
      body: data,
      params: { path: { appID, scope } },
    }),

  approveOpenPlatformApp: (appID: number) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/approve', {
      params: { path: { appID } },
    }),

  approveOpenPlatformRedirectURIRequest: (appID: number, requestID: number, data?: OpenPlatformRedirectURIReviewRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/approve', {
      body: data,
      params: { path: { appID, requestID } },
    }),

  rejectOpenPlatformRedirectURIRequest: (appID: number, requestID: number, data?: OpenPlatformRedirectURIReviewRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/redirect-uri-requests/{requestID}/reject', {
      body: data,
      params: { path: { appID, requestID } },
    }),

  rotateOpenPlatformAppSecret: (appID: number, data?: OpenPlatformSecretRotationRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/secret/rotate', {
      body: data,
      params: { path: { appID } },
    }),

  suspendOpenPlatformApp: (appID: number, data: OpenPlatformLifecycleActionRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/suspend', {
      body: data,
      params: { path: { appID } },
    }),

  resumeOpenPlatformApp: (appID: number, data: OpenPlatformLifecycleActionRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/resume', {
      body: data,
      params: { path: { appID } },
    }),

  revokeOpenPlatformApp: (appID: number, data: OpenPlatformLifecycleActionRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/{appID}/revoke', {
      body: data,
      params: { path: { appID } },
    }),

  importOpenPlatformCasdoorApp: (data: OpenPlatformImportCasdoorAppRequest) =>
    client.POST('/api/v1/admin/open-platform/apps/import-casdoor', { body: data })
})

export type {
  OpenPlatformResourceGrantRequest as AdminOpenPlatformResourceGrantRequest,
  OpenPlatformResourceGrantResult as AdminOpenPlatformResourceGrantResult,
  OpenPlatformResourceType as AdminOpenPlatformResourceType,
}
