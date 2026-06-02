import type {
  AdmissionPolicy,
  AdmissionSession,
  CreatedAdmissionSession,
  FreshmanApplication,
  FreshmanReviewRequest,
  ListAdmissionSessionsParams,
  ListFreshmanVerificationsParams,
  ListMemberBlacklistParams,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistReleaseBySubjectRequest,
  MemberBlacklistReleaseRequest,
} from '@stuhelper/shared/api';

import { createAdmissionApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData, unwrapListData } from '#/api/shared-result';

const admissionApi = createAdmissionApi(sharedApiClient);

export type {
  AdmissionPolicy,
  AdmissionSession,
  CreatedAdmissionSession,
  FreshmanApplication,
  FreshmanReviewRequest,
  ListAdmissionSessionsParams,
  ListMemberBlacklistParams,
  MemberBlacklistCreateRequest,
  MemberBlacklistEntry,
  MemberBlacklistReleaseBySubjectRequest,
  MemberBlacklistReleaseRequest,
};

export async function listFreshmanVerifications(
  params?: ListFreshmanVerificationsParams,
) {
  return unwrapListData<FreshmanApplication>(
    await admissionApi.listFreshmanVerifications(params),
  );
}

export async function getFreshmanVerification(id: string) {
  return unwrapData<FreshmanApplication>(
    await admissionApi.getFreshmanVerification(id),
  );
}

export async function reviewFreshmanVerification(
  id: string,
  data: FreshmanReviewRequest,
) {
  return unwrapData<FreshmanApplication>(
    await admissionApi.reviewFreshmanVerification(id, data),
  );
}

export async function listAdmissionPolicies() {
  return unwrapData<AdmissionPolicy[]>(
    await admissionApi.listAdmissionPolicies(),
  );
}

export async function updateAdmissionPolicy(policy: AdmissionPolicy) {
  return unwrapData<AdmissionPolicy>(
    await admissionApi.updateAdmissionPolicy(policy.id, policy),
  );
}

export async function listAdmissionSessions(
  params?: ListAdmissionSessionsParams,
) {
  return unwrapListData<AdmissionSession>(
    await admissionApi.listAdmissionSessions(params),
  );
}

export async function resendAdminAdmissionSession(id: string) {
  return unwrapData<AdmissionSession>(
    await admissionApi.resendAdminAdmissionSession(id),
  );
}

export async function regenerateAdminAdmissionSession(id: string) {
  return unwrapData<CreatedAdmissionSession>(
    await admissionApi.regenerateAdminAdmissionSession(id),
  );
}

export async function cancelAdminAdmissionSession(id: string) {
  return unwrapData<AdmissionSession>(
    await admissionApi.cancelAdminAdmissionSession(id),
  );
}

export async function listMemberBlacklist(params?: ListMemberBlacklistParams) {
  return unwrapListData<MemberBlacklistEntry>(
    await admissionApi.listMemberBlacklist(params),
  );
}

export async function createMemberBlacklist(
  data: MemberBlacklistCreateRequest,
) {
  return unwrapData<MemberBlacklistEntry>(
    await admissionApi.createMemberBlacklist(data),
  );
}

export async function releaseMemberBlacklist(
  id: string,
  data: MemberBlacklistReleaseRequest,
) {
  return unwrapData<MemberBlacklistEntry>(
    await admissionApi.releaseMemberBlacklist(id, data),
  );
}

export async function releaseMemberBlacklistBySubject(
  data: MemberBlacklistReleaseBySubjectRequest,
) {
  return unwrapData<MemberBlacklistEntry>(
    await admissionApi.releaseMemberBlacklistBySubject(data),
  );
}
