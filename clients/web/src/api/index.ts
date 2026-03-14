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

// Raw fetch helper for endpoints not yet in generated types
const BASE = '/api/v1'

async function fetchJSON<T>(path: string, options?: RequestInit): Promise<T> {
  const url = `${BASE}${path}`
  const res = await fetch(url, {
    credentials: 'include',
    headers: { 'Content-Type': 'application/json', ...getCSRFHeader() },
    ...options,
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    const msg = body?.error?.message || res.statusText
    const err = new Error(msg) as Error & { status: number }
    err.status = res.status
    throw err
  }
  return res.json()
}

function getCSRFHeader(): Record<string, string> {
  const match = document.cookie.match(/(?:^|;\s*)csrf_token=([^;]*)/)
  if (match) return { 'X-CSRF-Token': decodeURIComponent(match[1]) }
  return {}
}

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
  reply: createReplyApi(apiClient),
  verification: {
    getIdentity: () =>
      fetchJSON<{ data: { identity: {
        userID: number; docType: string; realName: string; verified: boolean;
        verifyMethod: string | null; verifiedAt: string | null;
        rejectionReason: string | null; createdAt: string; updatedAt: string
      } } }>('/user/identity'),
    submitIdentity: (data: {
      docType: 'MAINLAND_ID' | 'HK_MACAU' | 'TW' | 'PASSPORT';
      docNumber: string; realName: string;
      docPhotoFront?: string; docPhotoBack?: string; docPhotoSelfie?: string
    }) =>
      fetchJSON<{ data: { identity: {
        userID: number; docType: string; realName: string; verified: boolean;
        verifyMethod: string | null; verifiedAt: string | null;
        rejectionReason: string | null; createdAt: string; updatedAt: string
      } } }>('/user/identity', { method: 'POST', body: JSON.stringify(data) }),
    getProfile: () =>
      fetchJSON<{ data: { profile: {
        userID: number; schoolID: string | null; studentIDs: string[] | null;
        activeStudentID: string | null;
        verificationStatus: 'unverified' | 'pending' | 'verified' | 'rejected';
        verificationMethod: string | null; phone: string | null;
        phoneVerified: boolean; consentGivenAt: string | null;
        verifiedAt: string | null; createdAt: string; updatedAt: string
      } } }>('/user/profile'),
    verifyStudent: (data: {
      schoolID: string; studentID: string; password: string; consent: boolean
    }) =>
      fetchJSON<{ data: { profile: {
        userID: number; schoolID: string | null; studentIDs: string[] | null;
        activeStudentID: string | null;
        verificationStatus: 'unverified' | 'pending' | 'verified' | 'rejected';
        verificationMethod: string | null; phone: string | null;
        phoneVerified: boolean; consentGivenAt: string | null;
        verifiedAt: string | null; createdAt: string; updatedAt: string
      } } }>('/user/profile/verify', { method: 'POST', body: JSON.stringify(data) }),
    bindPhone: (data: { phone: string; code?: string }) =>
      fetchJSON<{ data: { message: string } }>('/user/profile/bind-phone', { method: 'POST', body: JSON.stringify(data) }),
    getSchools: () =>
      fetchJSON<{ data: { schools: Array<{
        schoolID: string; schoolName: string; verificationMethod: string;
        consentText: string | null; enabled: boolean
      }> } }>('/user/schools'),
  },
}

// 导出类型
export type { components } from '@stuhelper/shared'
