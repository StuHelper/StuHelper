import type { components } from '@stuhelper/shared/types';

import { createAdminApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData, unwrapListData } from '#/api/shared-result';

const adminApi = createAdminApi(sharedApiClient);

export type AuthorizationGrant = components['schemas']['AuthorizationGrant'];
export type AuthorizationRole = components['schemas']['AuthorizationRole'];
export type AssignableAuthorizationRole =
  components['schemas']['AssignableAuthorizationRole'];
export type AuthorizationDesiredState =
  components['schemas']['AuthorizationDesiredState'];
export type AuthorizationProjectionStatus =
  components['schemas']['AuthorizationProjectionStatus'];
export type CreateAuthorizationGrantRequest =
  components['schemas']['CreateAuthorizationGrantRequest'];
export type AuthorizationGrantMutationResult =
  components['schemas']['AuthorizationGrantMutationResult'];
export type AuthorizationProjectionReconcileResult =
  components['schemas']['AuthorizationProjectionReconcileResult'];

export interface ListAuthorizationGrantsParams {
  desiredState?: AuthorizationDesiredState;
  page?: number;
  pageSize?: number;
  projectionStatus?: AuthorizationProjectionStatus;
  role?: AuthorizationRole;
  subjectUserID?: number;
}

export async function listAuthorizationGrants(
  params?: ListAuthorizationGrantsParams,
) {
  return unwrapListData<AuthorizationGrant>(
    await adminApi.getAuthorizationGrants(params),
  );
}

export async function createAuthorizationGrant(
  data: CreateAuthorizationGrantRequest,
) {
  return unwrapData<AuthorizationGrantMutationResult>(
    await adminApi.createAuthorizationGrant(data),
  );
}

export async function revokeAuthorizationGrant(
  grantID: number,
  reason: string,
) {
  return unwrapData<AuthorizationGrantMutationResult>(
    await adminApi.revokeAuthorizationGrant(grantID, { reason }),
  );
}

export async function reconcileAuthorizationGrant(
  grantID: number,
  reason: string,
) {
  return unwrapData<AuthorizationGrantMutationResult>(
    await adminApi.reconcileAuthorizationGrant(grantID, { reason }),
  );
}

export async function reconcileAllAuthorizationProjections(reason: string) {
  return unwrapData<AuthorizationProjectionReconcileResult>(
    await adminApi.reconcileAllAuthorizationProjections({ reason }),
  );
}
