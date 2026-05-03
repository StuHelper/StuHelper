import type {
  AdmissionPolicy,
  AdmissionSession,
  FreshmanApplication,
  FreshmanReviewRequest,
  ListAdmissionSessionsParams,
  ListFreshmanVerificationsParams,
} from '@stuhelper/shared/api';
import type { ApiCallResult } from '#/api/shared-result';

import { createAdmissionApi, isResultFailure } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData, unwrapListData } from '#/api/shared-result';

const admissionApi = createAdmissionApi(sharedApiClient);

export type {
  AdmissionPolicy,
  AdmissionSession,
  FreshmanApplication,
  FreshmanReviewRequest,
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

export async function releaseAdmissionBlacklist(qqID: string): Promise<void> {
  const result = await admissionApi.releaseAdmissionBlacklist(qqID);
  assertSuccess(result);
}

function assertSuccess(result: ApiCallResult<unknown>): void {
  if (!isResultFailure(result)) return;
  unwrapData(result as ApiCallResult<never>);
}
