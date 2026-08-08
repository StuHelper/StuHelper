import type {
  AdminCampusConnectorHealth,
  AdminCredentialRevokeRequest,
  AdminManualMaterialAccess,
  AdminManualReviewDecisionRequest,
  AdminManualReviewDetail,
  AdminManualReviewSummary,
  AdminRosterSyncCreateRequest,
  AdminRosterSyncRequest,
  AdminStudentCredential,
  AdminStudentSubjectConflict,
  AdminSubjectConflictDecisionRequest,
  AdminVerificationMethodConfig,
  AdminVerificationSchoolConfig,
  CreateAdminVerificationSchoolConfigRequest,
  RosterSnapshot,
  RosterSnapshotSwitchRequest,
  UpdateAdminVerificationMethodConfigRequest,
  UpdateAdminVerificationSchoolConfigRequest,
  ValidateAdminVerificationConfigRequest,
  VerificationCredentialStatus,
  VerificationMethod,
} from '@stuhelper/shared/api';

import { createStudentVerificationAdminApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData } from '#/api/shared-result';

const api = createStudentVerificationAdminApi(sharedApiClient);

export type {
  AdminCampusConnectorHealth,
  AdminManualReviewDetail,
  AdminManualReviewSummary,
  AdminRosterSyncRequest,
  AdminStudentCredential,
  AdminStudentSubjectConflict,
  AdminVerificationMethodConfig,
  AdminVerificationSchoolConfig,
  RosterSnapshot,
  VerificationMethod,
};

export async function listVerificationSchools() {
  return unwrapData<AdminVerificationSchoolConfig[]>(await api.listSchools());
}

export async function createVerificationSchool(
  data: CreateAdminVerificationSchoolConfigRequest,
) {
  return unwrapData<AdminVerificationSchoolConfig>(
    await api.createSchool(data),
  );
}

export async function getVerificationSchool(schoolCode: string) {
  return unwrapData<AdminVerificationSchoolConfig>(
    await api.getSchool(schoolCode),
  );
}

export async function updateVerificationSchool(
  schoolCode: string,
  data: UpdateAdminVerificationSchoolConfigRequest,
) {
  return unwrapData<AdminVerificationSchoolConfig>(
    await api.updateSchool(schoolCode, data),
  );
}

export async function validateVerificationSchool(
  schoolCode: string,
  data: ValidateAdminVerificationConfigRequest,
) {
  return unwrapData<AdminVerificationSchoolConfig>(
    await api.validateSchool(schoolCode, data),
  );
}

export async function updateVerificationMethod(
  schoolCode: string,
  method: VerificationMethod,
  data: UpdateAdminVerificationMethodConfigRequest,
) {
  return unwrapData<AdminVerificationMethodConfig>(
    await api.updateMethod(schoolCode, method, data),
  );
}

export async function validateVerificationMethod(
  schoolCode: string,
  method: VerificationMethod,
  data: ValidateAdminVerificationConfigRequest,
) {
  return unwrapData<AdminVerificationMethodConfig>(
    await api.validateMethod(schoolCode, method, data),
  );
}

export async function listRosterSnapshots(schoolCode: string) {
  return unwrapData<RosterSnapshot[]>(
    await api.listRosterSnapshots(schoolCode),
  );
}

export async function listRosterSyncRequests(schoolCode: string) {
  return unwrapData<AdminRosterSyncRequest[]>(
    await api.listRosterSyncRequests(schoolCode),
  );
}

export async function createRosterSyncRequest(
  schoolCode: string,
  data: AdminRosterSyncCreateRequest,
) {
  return unwrapData<AdminRosterSyncRequest>(
    await api.createRosterSyncRequest(schoolCode, data),
  );
}

export async function activateRosterSnapshot(
  schoolCode: string,
  snapshotID: string,
  data: RosterSnapshotSwitchRequest,
) {
  return unwrapData<RosterSnapshot>(
    await api.activateRosterSnapshot(schoolCode, snapshotID, data),
  );
}

export async function rollbackRosterSnapshot(
  schoolCode: string,
  snapshotID: string,
  data: RosterSnapshotSwitchRequest,
) {
  return unwrapData<RosterSnapshot>(
    await api.rollbackRosterSnapshot(schoolCode, snapshotID, data),
  );
}

export async function listManualStudentReviews(params: {
  limit?: number;
  offset?: number;
  schoolCode: string;
  status?: 'approved' | 'pending' | 'rejected' | 'supplement_required';
}) {
  return unwrapData<AdminManualReviewSummary[]>(
    await api.listManualReviews(params),
  );
}

export async function getManualStudentReview(caseID: string) {
  return unwrapData<AdminManualReviewDetail>(await api.getManualReview(caseID));
}

export async function accessManualStudentReviewMaterial(
  caseID: string,
  materialID: string,
) {
  return unwrapData<AdminManualMaterialAccess>(
    await api.accessManualReviewMaterial(caseID, materialID),
  );
}

export async function decideManualStudentReview(
  caseID: string,
  data: AdminManualReviewDecisionRequest,
) {
  return unwrapData(await api.decideManualReview(caseID, data));
}

export async function listStudentCredentials(params: {
  limit?: number;
  offset?: number;
  schoolCode: string;
  status?: VerificationCredentialStatus;
}) {
  return unwrapData<AdminStudentCredential[]>(
    await api.listCredentials(params),
  );
}

export async function revokeStudentCredential(
  credentialID: string,
  data: AdminCredentialRevokeRequest,
) {
  return unwrapData<AdminStudentCredential>(
    await api.revokeCredential(credentialID, data),
  );
}

export async function listStudentSubjectConflicts(params: {
  limit?: number;
  offset?: number;
  schoolCode: string;
  status?: 'dismissed' | 'open' | 'resolved' | 'under_review';
}) {
  return unwrapData<AdminStudentSubjectConflict[]>(
    await api.listSubjectConflicts(params),
  );
}

export async function decideStudentSubjectConflict(
  conflictID: string,
  data: AdminSubjectConflictDecisionRequest,
) {
  return unwrapData<AdminStudentSubjectConflict>(
    await api.decideSubjectConflict(conflictID, data),
  );
}

export async function listCampusConnectorHealth(schoolCode?: string) {
  return unwrapData<AdminCampusConnectorHealth[]>(
    await api.listConnectorHealth(schoolCode),
  );
}
