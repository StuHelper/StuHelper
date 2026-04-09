import type { ApiClient } from './client'
import type { components } from '../types/api.gen'

type ReviewIdentityRequest = components['schemas']['ReviewIdentityRequest']
type ReviewStudentVerificationRequest = components['schemas']['ReviewStudentVerificationRequest']
type UpdateSchoolConfigRequest = components['schemas']['UpdateSchoolConfigRequest']
type UpdateSystemConfigRequest = components['schemas']['UpdateSystemConfigRequest']

export const createUserAdminApi = (client: ApiClient) => ({
  listIdentityVerifications: (params?: { status?: 'pending' | 'verified' | 'rejected' | 'all'; page?: number; pageSize?: number }) =>
    client.GET('/api/v1/admin/identities', { params: { query: params } }),

  reviewIdentity: (userID: number, data: ReviewIdentityRequest) =>
    client.PUT('/api/v1/admin/identities/{userID}', { params: { path: { userID } }, body: data }),

  listStudentVerifications: (params?: { status?: 'pending' | 'verified' | 'rejected' | 'all'; schoolID?: number; page?: number; pageSize?: number }) =>
    client.GET('/api/v1/admin/student-verifications', { params: { query: params } }),

  reviewStudentVerification: (userID: number, data: ReviewStudentVerificationRequest) =>
    client.PUT('/api/v1/admin/student-verifications/{userID}', { params: { path: { userID } }, body: data }),

  listSchoolConfigs: () =>
    client.GET('/api/v1/admin/school-configs'),

  updateSchoolConfig: (schoolID: number, data: UpdateSchoolConfigRequest) =>
    client.PUT('/api/v1/admin/school-configs/{schoolID}', { params: { path: { schoolID } }, body: data }),

  listSystemConfigs: () =>
    client.GET('/api/v1/admin/system-configs'),

  updateSystemConfig: (key: string, data: UpdateSystemConfigRequest) =>
    client.PUT('/api/v1/admin/system-configs/{key}', { params: { path: { key } }, body: data }),
})
