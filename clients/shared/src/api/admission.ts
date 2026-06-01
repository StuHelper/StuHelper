import type { ApiClient } from './client'
import type { components, operations } from '../types/api.gen'

type AdmissionSession = components['schemas']['AdmissionSession']
type AdmissionMe = components['schemas']['AdmissionMe']
type FreshmanApplication = components['schemas']['FreshmanApplication']
type MemberBlacklistEntry = components['schemas']['MemberBlacklistEntry']
type MemberBlacklistCreateRequest =
  operations['createAdminMemberBlacklistEntry']['requestBody']['content']['application/json']
type MemberBlacklistReleaseRequest =
  operations['releaseAdminMemberBlacklistEntry']['requestBody']['content']['application/json']
type MemberBlacklistReleaseBySubjectRequest =
  operations['releaseAdminMemberBlacklistBySubject']['requestBody']['content']['application/json']
type CreateFreshmanApplicationRequest =
  operations['createFreshmanApplication']['requestBody']['content']['application/json']
type CameraCaptureRequest = components['schemas']['CameraCaptureRequest']
type FreshmanCameraHandoff = components['schemas']['FreshmanCameraHandoff']
type FreshmanCameraHandoffContinuationRequest =
  components['schemas']['FreshmanCameraHandoffContinuationRequest']
type SchoolEmailOTPRequest = components['schemas']['SchoolEmailOTPRequest']
type SchoolEmailOTPVerifyRequest = components['schemas']['SchoolEmailOTPVerifyRequest']
type SchoolEmailOTPResponse = components['schemas']['SchoolEmailOTPResponse']
type AdmissionPolicy = components['schemas']['AdmissionPolicy']
type FreshmanReviewRequest = components['schemas']['FreshmanReviewRequest']
type ListAdmissionSessionsParams =
  operations['listAdmissionSessions']['parameters']['query']
type ListFreshmanVerificationsParams =
  operations['listFreshmanVerifications']['parameters']['query']
type ListMemberBlacklistParams =
  operations['listAdminMemberBlacklist']['parameters']['query']

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

  createFreshmanCameraHandoff: (applicationID: string) =>
    client.POST('/api/v1/admission/freshman/applications/{id}/camera-handoffs', {
      params: { path: { id: applicationID } },
    }),

  getFreshmanCameraHandoff: (handoffID: string) =>
    client.GET('/api/v1/admission/freshman/camera-handoffs/{id}', {
      params: { path: { id: handoffID } },
    }),

  previewFreshmanMobileCameraHandoff: (token: string) =>
    client.GET('/api/v1/admission/freshman/mobile-camera-handoffs/{token}', {
      params: { path: { token } },
    }),

  uploadFreshmanMobileCameraCapture: (token: string, data: CameraCaptureRequest) =>
    client.POST('/api/v1/admission/freshman/mobile-camera-handoffs/{token}/camera-capture', {
      params: { path: { token } },
      body: data,
    }),

  chooseFreshmanMobileCameraContinuation: (
    token: string,
    data: FreshmanCameraHandoffContinuationRequest,
  ) =>
    client.POST('/api/v1/admission/freshman/mobile-camera-handoffs/{token}/continue', {
      params: { path: { token } },
      body: data,
    }),

  requestSchoolEmailOTP: (data: SchoolEmailOTPRequest) =>
    client.POST('/api/v1/admission/school-email/request-otp', { body: data }),

  verifySchoolEmailOTP: (data: SchoolEmailOTPVerifyRequest) =>
    client.POST('/api/v1/admission/school-email/verify-otp', { body: data }),

  listAdmissionPolicies: () =>
    client.GET('/api/v1/admin/admission/policies'),

  updateAdmissionPolicy: (id: string, data: AdmissionPolicy) =>
    client.PUT('/api/v1/admin/admission/policies/{id}', {
      params: { path: { id } },
      body: data,
    }),

  listAdmissionSessions: (params?: ListAdmissionSessionsParams) =>
    client.GET('/api/v1/admin/admission/sessions', {
      params: { query: params },
    }),

  listFreshmanVerifications: (params?: ListFreshmanVerificationsParams) =>
    client.GET('/api/v1/admin/freshman-verifications', {
      params: { query: params },
    }),

  getFreshmanVerification: (id: string) =>
    client.GET('/api/v1/admin/freshman-verifications/{id}', {
      params: { path: { id } },
    }),

  reviewFreshmanVerification: (id: string, data: FreshmanReviewRequest) =>
    client.PUT('/api/v1/admin/freshman-verifications/{id}', {
      params: { path: { id } },
      body: data,
    }),

  listMemberBlacklist: (params?: ListMemberBlacklistParams) =>
    client.GET('/api/v1/admin/member-blacklist', {
      params: { query: params },
    }),

  createMemberBlacklist: (data: MemberBlacklistCreateRequest) =>
    client.POST('/api/v1/admin/member-blacklist', { body: data }),

  releaseMemberBlacklist: (id: string, data: MemberBlacklistReleaseRequest) =>
    client.POST('/api/v1/admin/member-blacklist/{id}/release', {
      params: { path: { id } },
      body: data,
    }),

  releaseMemberBlacklistBySubject: (data: MemberBlacklistReleaseBySubjectRequest) =>
    client.POST('/api/v1/admin/member-blacklist/release-by-subject', {
      body: data,
    }),
})

export type {
  AdmissionMe,
  AdmissionPolicy,
  AdmissionSession,
  CameraCaptureRequest,
  CreateFreshmanApplicationRequest,
  FreshmanApplication,
  FreshmanCameraHandoff,
  FreshmanCameraHandoffContinuationRequest,
  FreshmanReviewRequest,
  ListAdmissionSessionsParams,
  ListFreshmanVerificationsParams,
  ListMemberBlacklistParams,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistReleaseBySubjectRequest,
  MemberBlacklistReleaseRequest,
  SchoolEmailOTPRequest,
  SchoolEmailOTPVerifyRequest,
  SchoolEmailOTPResponse,
}
