import type { ApiClient } from './client'
import type { components, operations } from '../types/api.gen'

type AdmissionSession = components['schemas']['AdmissionSession']
type AdmissionMe = components['schemas']['AdmissionMe']
type CreatedAdmissionSession = components['schemas']['CreatedAdmissionSession']
type MemberBlacklistEntry = components['schemas']['MemberBlacklistEntry']
type MemberBlacklistCreateRequest =
  operations['createAdminMemberBlacklistEntry']['requestBody']['content']['application/json']
type MemberBlacklistReleaseRequest =
  operations['releaseAdminMemberBlacklistEntry']['requestBody']['content']['application/json']
type MemberBlacklistReleaseBySubjectRequest =
  operations['releaseAdminMemberBlacklistBySubject']['requestBody']['content']['application/json']
type AdmissionPolicy = components['schemas']['AdmissionPolicy']
type AdmissionPolicyCreateRequest = components['schemas']['AdmissionPolicyCreateRequest']
type ListAdmissionSessionsParams =
  operations['listAdmissionSessions']['parameters']['query']
type ListMemberBlacklistParams =
  operations['listAdminMemberBlacklist']['parameters']['query']

export const createAdmissionApi = (client: ApiClient) => ({
  getAdmissionSession: (token: string) =>
    client.GET('/api/v1/admission/sessions/{token}', {
      params: {
        path: { token },
      },
    }),

  linkAdmissionSession: (token: string) =>
    client.POST('/api/v1/admission/sessions/{token}/link', {
      params: {
        path: { token },
      },
    }),

  getAdmissionMe: (admissionSessionID?: string) =>
    admissionSessionID
      ? client.GET('/api/v1/admission/me', {
          params: {
            query: { admissionSessionID },
          },
        })
      : client.GET('/api/v1/admission/me'),

  listAdmissionPolicies: () =>
    client.GET('/api/v1/admin/admission/policies'),

  createAdmissionPolicy: (data: AdmissionPolicyCreateRequest) =>
    client.POST('/api/v1/admin/admission/policies', {
      body: data,
    }),

  updateAdmissionPolicy: (id: string, data: AdmissionPolicy) =>
    client.PUT('/api/v1/admin/admission/policies/{id}', {
      params: { path: { id } },
      body: data,
    }),

  listAdmissionSessions: (params?: ListAdmissionSessionsParams) =>
    client.GET('/api/v1/admin/admission/sessions', {
      params: { query: params },
    }),

  resendAdminAdmissionSession: (id: string) =>
    client.POST('/api/v1/admin/admission/sessions/{id}/resend', {
      params: { path: { id } },
    }),

  regenerateAdminAdmissionSession: (id: string) =>
    client.POST('/api/v1/admin/admission/sessions/{id}/regenerate', {
      params: { path: { id } },
    }),

  cancelAdminAdmissionSession: (id: string) =>
    client.POST('/api/v1/admin/admission/sessions/{id}/cancel', {
      params: { path: { id } },
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
  AdmissionPolicyCreateRequest,
  AdmissionSession,
  CreatedAdmissionSession,
  ListAdmissionSessionsParams,
  ListMemberBlacklistParams,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistReleaseBySubjectRequest,
  MemberBlacklistReleaseRequest,
}
