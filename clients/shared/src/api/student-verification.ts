import type { ApiClient } from './client'
import type { components } from '../types/api.gen'

type VerificationSchool = components['schemas']['VerificationSchool']
type VerificationApplication = components['schemas']['VerificationApplication']
type CreateVerificationApplicationRequest =
  components['schemas']['CreateVerificationApplicationRequest']
type VerifyRealNameRequest = components['schemas']['VerifyRealNameRequest']
type VerifySchoolSSORequest = components['schemas']['VerifySchoolSSORequest']
type StudentEmailIdentityRequest = components['schemas']['StudentEmailIdentityRequest']
type VerifyStudentEmailOTPRequest = components['schemas']['VerifyStudentEmailOTPRequest']
type StudentEmailOTPChallenge = components['schemas']['StudentEmailOTPChallenge']
type InboundEmailChallenge = components['schemas']['InboundEmailChallenge']
type StudentVerificationCredential = components['schemas']['StudentVerificationCredential']
type StudentEligibility = components['schemas']['StudentEligibility']
type PhoneStatus = components['schemas']['PhoneStatus']
type CreatePhoneOperationRequest = components['schemas']['CreatePhoneOperationRequest']
type PhoneBindingOperation = components['schemas']['PhoneBindingOperation']
type VerifyPhoneSMSRequest = components['schemas']['VerifyPhoneSMSRequest']
type UpsertManualReviewRequest = components['schemas']['UpsertManualReviewRequest']
type ManualReviewCase = components['schemas']['ManualReviewCase']
type ManualCameraCaptureRequest = components['schemas']['ManualCameraCaptureRequest']
type ManualCameraHandoff = components['schemas']['ManualCameraHandoff']
type ManualCameraContinuationRequest = components['schemas']['ManualCameraContinuationRequest']
type SubmitManualReviewRequest = components['schemas']['SubmitManualReviewRequest']
type ManualEmailOTPChallenge = components['schemas']['ManualEmailOTPChallenge']
type SchoolVerificationSuggestionRequest =
  components['schemas']['SchoolVerificationSuggestionRequest']
type SchoolVerificationSuggestion = components['schemas']['SchoolVerificationSuggestion']
type AdminVerificationSchoolConfig = components['schemas']['AdminVerificationSchoolConfig']
type AdminVerificationMethodConfig = components['schemas']['AdminVerificationMethodConfig']
type CreateAdminVerificationSchoolConfigRequest =
  components['schemas']['CreateAdminVerificationSchoolConfigRequest']
type UpdateAdminVerificationSchoolConfigRequest =
  components['schemas']['UpdateAdminVerificationSchoolConfigRequest']
type UpdateAdminVerificationMethodConfigRequest =
  components['schemas']['UpdateAdminVerificationMethodConfigRequest']
type ValidateAdminVerificationConfigRequest =
  components['schemas']['ValidateAdminVerificationConfigRequest']
type RosterSnapshot = components['schemas']['RosterSnapshot']
type RosterSnapshotSwitchRequest = components['schemas']['RosterSnapshotSwitchRequest']
type AdminManualReviewSummary = components['schemas']['ManualReviewCase']
type AdminManualReviewDetail = components['schemas']['AdminManualReviewDetail']
type AdminManualMaterialAccess = components['schemas']['AdminManualMaterialAccess']
type AdminManualReviewDecisionRequest =
  components['schemas']['AdminManualReviewDecisionRequest']
type AdminStudentCredential = components['schemas']['AdminStudentCredential']
type AdminCredentialRevokeRequest = components['schemas']['AdminCredentialRevokeRequest']
type AdminStudentSubjectConflict = components['schemas']['AdminStudentSubjectConflict']
type AdminSubjectConflictDecisionRequest =
  components['schemas']['AdminSubjectConflictDecisionRequest']
type AdminCampusConnectorHealth = components['schemas']['AdminCampusConnectorHealth']
type AdminRosterSyncCreateRequest = components['schemas']['AdminRosterSyncCreateRequest']
type AdminRosterSyncRequest = components['schemas']['AdminRosterSyncRequest']
type VerificationMethod = components['schemas']['VerificationMethod']
type VerificationCredentialStatus = components['schemas']['VerificationCredentialStatus']
type ManualReviewStatus = components['schemas']['ManualReviewStatus']

/**
 * Student verification and account-phone APIs.
 *
 * The two groups intentionally share a client factory but not a domain model:
 * a verified phone never implies student eligibility, and student verification
 * never requires a phone operation.
 */
export const createStudentVerificationApi = (client: ApiClient) => ({
  listSchools: () => client.GET('/api/v1/student-verification/schools'),

  createApplication: (data: CreateVerificationApplicationRequest) =>
    client.POST('/api/v1/student-verification/applications', { body: data }),

  getApplication: (applicationID: string) =>
    client.GET('/api/v1/student-verification/applications/{applicationID}', {
      params: { path: { applicationID } },
    }),

  cancelApplication: (applicationID: string) =>
    client.DELETE('/api/v1/student-verification/applications/{applicationID}', {
      params: { path: { applicationID } },
    }),

  verifyRealName: (applicationID: string, data: VerifyRealNameRequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/real-name/verify', {
      params: { path: { applicationID } },
      body: data,
    }),

  verifySchoolSSO: (applicationID: string, data: VerifySchoolSSORequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/school-sso/verify', {
      params: { path: { applicationID } },
      body: data,
    }),

  requestStudentEmailOTP: (applicationID: string, data: StudentEmailIdentityRequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/email/outbound/otp', {
      params: { path: { applicationID } },
      body: data,
    }),

  verifyStudentEmailOTP: (applicationID: string, data: VerifyStudentEmailOTPRequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/email/outbound/verify', {
      params: { path: { applicationID } },
      body: data,
    }),

  createInboundEmailChallenge: (applicationID: string, data: StudentEmailIdentityRequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/email/inbound/challenge', {
      params: { path: { applicationID } },
      body: data,
    }),

  getInboundEmailChallenge: (applicationID: string) =>
    client.GET('/api/v1/student-verification/applications/{applicationID}/email/inbound/challenge', {
      params: { path: { applicationID } },
    }),

  listCredentials: () => client.GET('/api/v1/student-verification/credentials'),

  revokeCredential: (credentialID: string) =>
    client.DELETE('/api/v1/student-verification/credentials/{credentialID}', {
      params: { path: { credentialID } },
    }),

  getEligibility: (schoolCode: string) =>
    client.GET('/api/v1/student-verification/eligibility', {
      params: { query: { schoolCode } },
    }),

  upsertManualReview: (applicationID: string, data: UpsertManualReviewRequest) =>
    client.PUT('/api/v1/student-verification/applications/{applicationID}/manual-review', {
      params: { path: { applicationID } },
      body: data,
    }),

  getManualReview: (applicationID: string) =>
    client.GET('/api/v1/student-verification/applications/{applicationID}/manual-review', {
      params: { path: { applicationID } },
    }),

  uploadManualCameraCapture: (applicationID: string, data: ManualCameraCaptureRequest) =>
    client.POST(
      '/api/v1/student-verification/applications/{applicationID}/manual-review/camera-captures',
      { params: { path: { applicationID } }, body: data },
    ),

  createManualCameraHandoff: (applicationID: string) =>
    client.POST(
      '/api/v1/student-verification/applications/{applicationID}/manual-review/camera-handoffs',
      { params: { path: { applicationID } } },
    ),

  getManualCameraHandoff: (applicationID: string, handoffID: string) =>
    client.GET(
      '/api/v1/student-verification/applications/{applicationID}/manual-review/camera-handoffs/{handoffID}',
      { params: { path: { applicationID, handoffID } } },
    ),

  submitManualReview: (applicationID: string, data: SubmitManualReviewRequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/manual-review/submit', {
      params: { path: { applicationID } },
      body: data,
    }),

  requestManualEmailOTP: (applicationID: string) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/manual-review/email/otp', {
      params: { path: { applicationID } },
    }),

  verifyManualEmailOTP: (applicationID: string, data: VerifyStudentEmailOTPRequest) =>
    client.POST('/api/v1/student-verification/applications/{applicationID}/manual-review/email/verify', {
      params: { path: { applicationID } },
      body: data,
    }),

  suggestSchool: (data: SchoolVerificationSuggestionRequest) =>
    client.POST('/api/v1/student-verification/school-suggestions', { body: data }),

  previewManualCameraHandoff: (token: string) =>
    client.GET('/api/v1/student-verification/manual-camera-handoffs/{token}', {
      params: { path: { token } },
    }),

  uploadManualHandoffCameraCapture: (token: string, data: ManualCameraCaptureRequest) =>
    client.POST('/api/v1/student-verification/manual-camera-handoffs/{token}/camera-capture', {
      params: { path: { token } },
      body: data,
    }),

  chooseManualCameraContinuation: (token: string, data: ManualCameraContinuationRequest) =>
    client.POST('/api/v1/student-verification/manual-camera-handoffs/{token}/continue', {
      params: { path: { token } },
      body: data,
    }),

  resumeManualCameraHandoff: (token: string) =>
    client.POST('/api/v1/student-verification/manual-camera-handoffs/{token}/resume', {
      params: { path: { token } },
    }),

  getPhoneStatus: () => client.GET('/api/v1/account/phone'),

  createPhoneOperation: (data: CreatePhoneOperationRequest) =>
    client.POST('/api/v1/account/phone/operations', { body: data }),

  createPhoneChangeOperation: (data: CreatePhoneOperationRequest) =>
    client.POST('/api/v1/account/phone/change-operations', { body: data }),

  getPhoneOperation: (operationID: string) =>
    client.GET('/api/v1/account/phone/operations/{operationID}', {
      params: { path: { operationID } },
    }),

  sendPhoneSMS: (operationID: string) =>
    client.POST('/api/v1/account/phone/operations/{operationID}/sms', {
      params: { path: { operationID } },
    }),

  verifyPhoneSMS: (operationID: string, data: VerifyPhoneSMSRequest) =>
    client.POST('/api/v1/account/phone/operations/{operationID}/sms/verify', {
      params: { path: { operationID } },
      body: data,
    }),

  unbindPhone: () => client.DELETE('/api/v1/account/phone'),
})

/**
 * School-scoped administration client for the target verification domain.
 * Responses intentionally expose neither raw evidence nor connector secrets.
 */
export const createStudentVerificationAdminApi = (client: ApiClient) => ({
  listSchools: () => client.GET('/api/v1/admin/student-verification/schools'),

  createSchool: (data: CreateAdminVerificationSchoolConfigRequest) =>
    client.POST('/api/v1/admin/student-verification/schools', { body: data }),

  getSchool: (schoolCode: string) =>
    client.GET('/api/v1/admin/student-verification/schools/{schoolCode}', {
      params: { path: { schoolCode } },
    }),

  updateSchool: (schoolCode: string, data: UpdateAdminVerificationSchoolConfigRequest) =>
    client.PUT('/api/v1/admin/student-verification/schools/{schoolCode}', {
      params: { path: { schoolCode } },
      body: data,
    }),

  validateSchool: (schoolCode: string, data: ValidateAdminVerificationConfigRequest) =>
    client.POST('/api/v1/admin/student-verification/schools/{schoolCode}/validate', {
      params: { path: { schoolCode } },
      body: data,
    }),

  updateMethod: (
    schoolCode: string,
    method: VerificationMethod,
    data: UpdateAdminVerificationMethodConfigRequest,
  ) =>
    client.PUT('/api/v1/admin/student-verification/schools/{schoolCode}/methods/{method}', {
      params: { path: { schoolCode, method } },
      body: data,
    }),

  validateMethod: (
    schoolCode: string,
    method: VerificationMethod,
    data: ValidateAdminVerificationConfigRequest,
  ) =>
    client.POST(
      '/api/v1/admin/student-verification/schools/{schoolCode}/methods/{method}/validate',
      { params: { path: { schoolCode, method } }, body: data },
    ),

  listRosterSnapshots: (schoolCode: string) =>
    client.GET('/api/v1/admin/student-verification/schools/{schoolCode}/roster-snapshots', {
      params: { path: { schoolCode } },
    }),

  listRosterSyncRequests: (schoolCode: string, limit = 20) =>
    client.GET(
      '/api/v1/admin/student-verification/schools/{schoolCode}/roster-sync-requests',
      { params: { path: { schoolCode }, query: { limit } } },
    ),

  createRosterSyncRequest: (
    schoolCode: string,
    data: AdminRosterSyncCreateRequest,
  ) =>
    client.POST(
      '/api/v1/admin/student-verification/schools/{schoolCode}/roster-sync-requests',
      { params: { path: { schoolCode } }, body: data },
    ),

  getRosterSnapshot: (schoolCode: string, snapshotID: string) =>
    client.GET(
      '/api/v1/admin/student-verification/schools/{schoolCode}/roster-snapshots/{snapshotID}',
      { params: { path: { schoolCode, snapshotID } } },
    ),

  activateRosterSnapshot: (
    schoolCode: string,
    snapshotID: string,
    data: RosterSnapshotSwitchRequest,
  ) =>
    client.POST(
      '/api/v1/admin/student-verification/schools/{schoolCode}/roster-snapshots/{snapshotID}/activate',
      { params: { path: { schoolCode, snapshotID } }, body: data },
    ),

  rollbackRosterSnapshot: (
    schoolCode: string,
    snapshotID: string,
    data: RosterSnapshotSwitchRequest,
  ) =>
    client.POST(
      '/api/v1/admin/student-verification/schools/{schoolCode}/roster-snapshots/{snapshotID}/rollback',
      { params: { path: { schoolCode, snapshotID } }, body: data },
    ),

  listManualReviews: (params: {
    limit?: number
    offset?: number
    schoolCode: string
    status?: 'approved' | 'pending' | 'rejected' | 'supplement_required'
  }) =>
    client.GET('/api/v1/admin/student-verification/manual-reviews', {
      params: { query: params },
    }),

  getManualReview: (caseID: string) =>
    client.GET('/api/v1/admin/student-verification/manual-reviews/{caseID}', {
      params: { path: { caseID } },
    }),

  accessManualReviewMaterial: (caseID: string, materialID: string) =>
    client.POST(
      '/api/v1/admin/student-verification/manual-reviews/{caseID}/materials/{materialID}/access',
      { params: { path: { caseID, materialID } } },
    ),

  decideManualReview: (caseID: string, data: AdminManualReviewDecisionRequest) =>
    client.POST('/api/v1/admin/student-verification/manual-reviews/{caseID}/decision', {
      params: { path: { caseID } },
      body: data,
    }),

  listCredentials: (params: {
    limit?: number
    offset?: number
    schoolCode: string
    status?: VerificationCredentialStatus
  }) =>
    client.GET('/api/v1/admin/student-verification/credentials', {
      params: { query: params },
    }),

  revokeCredential: (credentialID: string, data: AdminCredentialRevokeRequest) =>
    client.POST('/api/v1/admin/student-verification/credentials/{credentialID}/revoke', {
      params: { path: { credentialID } },
      body: data,
    }),

  listSubjectConflicts: (params: {
    limit?: number
    offset?: number
    schoolCode: string
    status?: 'dismissed' | 'open' | 'resolved' | 'under_review'
  }) =>
    client.GET('/api/v1/admin/student-verification/subject-conflicts', {
      params: { query: params },
    }),

  decideSubjectConflict: (conflictID: string, data: AdminSubjectConflictDecisionRequest) =>
    client.POST(
      '/api/v1/admin/student-verification/subject-conflicts/{conflictID}/decision',
      { params: { path: { conflictID } }, body: data },
    ),

  listConnectorHealth: (schoolCode?: string) =>
    client.GET('/api/v1/admin/student-verification/connectors', {
      params: { query: { schoolCode } },
    }),
})

export type {
  AdminCampusConnectorHealth,
  AdminRosterSyncCreateRequest,
  AdminRosterSyncRequest,
  AdminCredentialRevokeRequest,
  AdminManualMaterialAccess,
  AdminManualReviewDecisionRequest,
  AdminManualReviewDetail,
  AdminManualReviewSummary,
  AdminStudentCredential,
  AdminStudentSubjectConflict,
  AdminSubjectConflictDecisionRequest,
  AdminVerificationMethodConfig,
  AdminVerificationSchoolConfig,
  CreateAdminVerificationSchoolConfigRequest,
  CreatePhoneOperationRequest,
  CreateVerificationApplicationRequest,
  InboundEmailChallenge,
  ManualCameraCaptureRequest,
  ManualCameraContinuationRequest,
  ManualCameraHandoff,
  ManualEmailOTPChallenge,
  ManualReviewCase,
  ManualReviewStatus,
  PhoneBindingOperation,
  PhoneStatus,
  SchoolVerificationSuggestion,
  SchoolVerificationSuggestionRequest,
  StudentEligibility,
  StudentEmailIdentityRequest,
  StudentEmailOTPChallenge,
  StudentVerificationCredential,
  SubmitManualReviewRequest,
  RosterSnapshot,
  RosterSnapshotSwitchRequest,
  UpsertManualReviewRequest,
  VerificationApplication,
  VerificationCredentialStatus,
  VerificationMethod,
  VerificationSchool,
  VerifyPhoneSMSRequest,
  VerifyRealNameRequest,
  VerifySchoolSSORequest,
  VerifyStudentEmailOTPRequest,
  UpdateAdminVerificationMethodConfigRequest,
  UpdateAdminVerificationSchoolConfigRequest,
  ValidateAdminVerificationConfigRequest,
}
