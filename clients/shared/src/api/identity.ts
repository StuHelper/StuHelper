import type { ApiClient } from './client'

/**
 * Account surface and QQ binding APIs.
 *
 * Historical identity/profile/student/phone endpoints were intentionally
 * removed. Student eligibility and phone possession are exposed exclusively
 * by createStudentVerificationApi.
 */
export const createIdentityApi = (client: ApiClient) => ({
  getUserSurface: () =>
    client.GET('/api/v1/user/me'),

  getQQBinding: () =>
    client.GET('/api/v1/user/qq-binding'),

  createQQBindingCode: () =>
    client.POST('/api/v1/user/qq-binding/code'),
})
