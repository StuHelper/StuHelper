import type { ApiClient } from './client'
import type { components, operations } from '../types/api.gen'

type AdmissionSession = components['schemas']['AdmissionSession']
type AdmissionMe = components['schemas']['AdmissionMe']
type FreshmanApplication = components['schemas']['FreshmanApplication']
type CreateFreshmanApplicationRequest =
  operations['createFreshmanApplication']['requestBody']['content']['application/json']
type CameraCaptureRequest = components['schemas']['CameraCaptureRequest']
type SchoolEmailOTPRequest = components['schemas']['SchoolEmailOTPRequest']
type SchoolEmailOTPVerifyRequest = components['schemas']['SchoolEmailOTPVerifyRequest']

function withQQQuery(qq?: string) {
  return qq ? { qq } : undefined
}

export const createAdmissionApi = (client: ApiClient) => ({
  getAdmissionSession: (token: string, qq?: string) =>
    client.GET('/api/v1/admission/sessions/{token}', {
      params: {
        path: { token },
        query: withQQQuery(qq),
      },
    }),

  linkAdmissionSession: (token: string, qq?: string) =>
    client.POST('/api/v1/admission/sessions/{token}/link', {
      params: {
        path: { token },
        query: withQQQuery(qq),
      },
    }),

  getAdmissionMe: () =>
    client.GET('/api/v1/admission/me'),

  submitFreshmanApplication: (data: CreateFreshmanApplicationRequest) =>
    client.POST('/api/v1/admission/freshman/applications', { body: data }),

  uploadCameraCapture: (applicationID: string, data: CameraCaptureRequest) =>
    client.POST('/api/v1/admission/freshman/applications/{id}/camera-captures', {
      params: { path: { id: applicationID } },
      body: data,
    }),

  requestSchoolEmailOTP: (data: SchoolEmailOTPRequest) =>
    client.POST('/api/v1/admission/school-email/request-otp', { body: data }),

  verifySchoolEmailOTP: (data: SchoolEmailOTPVerifyRequest) =>
    client.POST('/api/v1/admission/school-email/verify-otp', { body: data }),
})

export type {
  AdmissionMe,
  AdmissionSession,
  CameraCaptureRequest,
  CreateFreshmanApplicationRequest,
  FreshmanApplication,
  SchoolEmailOTPRequest,
  SchoolEmailOTPVerifyRequest,
}
