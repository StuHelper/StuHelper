import type { ApiClient } from './client'
import type { components } from '../types/api.gen'

type SubmitIdentityRequest = components['schemas']['SubmitIdentityRequest']
type SubmitStudentVerificationRequest = components['schemas']['SubmitStudentVerificationRequest']
type BindPhoneRequest = components['schemas']['BindPhoneRequest']

export const createIdentityApi = (client: ApiClient) => ({
  getIdentity: () =>
    client.GET('/api/v1/user/identity'),

  submitIdentity: (data: SubmitIdentityRequest) =>
    client.POST('/api/v1/user/identity', { body: data }),

  getProfile: () =>
    client.GET('/api/v1/user/profile'),

  verifyStudent: (data: SubmitStudentVerificationRequest) =>
    client.POST('/api/v1/user/profile/verify', { body: data }),

  bindPhone: (data: BindPhoneRequest) =>
    client.POST('/api/v1/user/profile/bind-phone', { body: data }),

  getAcademicInfo: () =>
    client.GET('/api/v1/user/profile/academic-info'),

  listSchools: () =>
    client.GET('/api/v1/user/schools'),
})
