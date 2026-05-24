import type { components } from '@stuhelper/shared/types';

import { createAdminApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData, unwrapListData } from '#/api/shared-result';

const adminApi = createAdminApi(sharedApiClient);

export type OpenPlatformAppWithScopes =
  components['schemas']['OpenPlatformAppWithScopes'];
export type OpenPlatformApprovedApp =
  components['schemas']['OpenPlatformApprovedApp'];
export type OpenPlatformAppLifecycleResult =
  components['schemas']['OpenPlatformAppLifecycleResult'];
export type OpenPlatformImportedApp =
  components['schemas']['OpenPlatformImportedApp'];
export type OpenPlatformImportCasdoorAppRequest =
  components['schemas']['OpenPlatformImportCasdoorAppRequest'];
export type OpenPlatformRotatedSecret =
  components['schemas']['OpenPlatformRotatedSecret'];
export type OpenPlatformRedirectURIRequest =
  components['schemas']['OpenPlatformRedirectURIRequest'];
export type OpenPlatformAuditEvent =
  components['schemas']['OpenPlatformAuditEvent'];
export type OpenPlatformAdminUserAuthorizedApp =
  components['schemas']['OpenPlatformAdminUserAuthorizedApp'];
export type OpenPlatformTokenProbeEvidence =
  components['schemas']['OpenPlatformTokenProbeEvidence'];
export type OpenPlatformDisclosureReport =
  components['schemas']['OpenPlatformDisclosureReport'];
export type OpenPlatformResourceAccessAction =
  components['schemas']['OpenPlatformResourceAccessAction'];
export type OpenPlatformResourceGrant =
  components['schemas']['OpenPlatformResourceGrant'];
export type OpenPlatformResourceGrantRequest =
  components['schemas']['OpenPlatformResourceGrantRequest'];
export type OpenPlatformResourceGrantResult =
  components['schemas']['OpenPlatformResourceGrantResult'];
export type OpenPlatformResourceType =
  components['schemas']['OpenPlatformResourceType'];
export type OpenPlatformScope = components['schemas']['OpenPlatformScope'];
export type OpenPlatformTokenProbeResult =
  components['schemas']['OpenPlatformTokenProbeEvidence']['result'];
export type OpenPlatformAppStatus =
  | 'all'
  | components['schemas']['OpenPlatformApp']['status'];

export async function getOpenPlatformAppList(params: {
  page?: number;
  pageSize?: number;
  status?: OpenPlatformAppStatus;
}) {
  return unwrapListData<OpenPlatformAppWithScopes>(
    await adminApi.getOpenPlatformApps(params),
  );
}

export async function getOpenPlatformAuditEventList(params: {
  appID?: number;
  eventType?: string;
  page?: number;
  pageSize?: number;
  scope?: OpenPlatformScope;
  userID?: number;
}) {
  return unwrapListData<OpenPlatformAuditEvent>(
    await adminApi.getOpenPlatformAuditEvents(params),
  );
}

export async function getOpenPlatformConsentList(params: {
  appID?: number;
  page?: number;
  pageSize?: number;
  userID?: number;
}) {
  return unwrapListData<OpenPlatformAdminUserAuthorizedApp>(
    await adminApi.getOpenPlatformConsents(params),
  );
}

export async function getOpenPlatformTokenProbeEvidenceList(params: {
  appID?: number;
  clientID?: string;
  page?: number;
  pageSize?: number;
  result?: OpenPlatformTokenProbeResult;
  reviewerUserID?: number;
}) {
  return unwrapListData<OpenPlatformTokenProbeEvidence>(
    await adminApi.getOpenPlatformTokenProbeEvidence(params),
  );
}

export async function getOpenPlatformDisclosureReport(params?: {
  windowHours?: number;
}) {
  return unwrapData<OpenPlatformDisclosureReport>(
    await adminApi.getOpenPlatformDisclosureReport(params),
  );
}

export async function getOpenPlatformResourceGrants(
  appID: number,
  resourceType: OpenPlatformResourceType,
) {
  return unwrapData<OpenPlatformResourceGrantResult>(
    await adminApi.getOpenPlatformResourceGrants(appID, resourceType),
  );
}

export async function grantOpenPlatformResourceAccess(
  appID: number,
  data: OpenPlatformResourceGrantRequest,
) {
  return unwrapData<OpenPlatformResourceGrantResult>(
    await adminApi.grantOpenPlatformResourceAccess(appID, data),
  );
}

export async function revokeOpenPlatformResourceAccess(
  appID: number,
  data: OpenPlatformResourceGrantRequest,
) {
  return unwrapData<OpenPlatformResourceGrantResult>(
    await adminApi.revokeOpenPlatformResourceAccess(appID, data),
  );
}

export async function revokeOpenPlatformConsent(params: {
  appID: number;
  reason: string;
  scopes?: OpenPlatformScope[];
  userID: number;
}) {
  return unwrapData(
    await adminApi.revokeOpenPlatformConsent(params.appID, {
      reason: params.reason,
      scopes: params.scopes,
      userID: params.userID,
    }),
  );
}

export async function approveOpenPlatformScope(
  appID: number,
  scope: OpenPlatformScope,
  decisionNote?: string,
) {
  return unwrapData(
    await adminApi.approveOpenPlatformScope(
      appID,
      scope,
      decisionNote ? { decisionNote } : undefined,
    ),
  );
}

export async function rejectOpenPlatformScope(
  appID: number,
  scope: OpenPlatformScope,
  decisionNote?: string,
) {
  return unwrapData(
    await adminApi.rejectOpenPlatformScope(
      appID,
      scope,
      decisionNote ? { decisionNote } : undefined,
    ),
  );
}

export async function approveOpenPlatformApp(appID: number) {
  return unwrapData<OpenPlatformApprovedApp>(
    await adminApi.approveOpenPlatformApp(appID),
  );
}

export async function importOpenPlatformCasdoorApp(
  data: OpenPlatformImportCasdoorAppRequest,
) {
  return unwrapData<OpenPlatformImportedApp>(
    await adminApi.importOpenPlatformCasdoorApp(data),
  );
}

export async function approveOpenPlatformRedirectURIRequest(
  appID: number,
  requestID: number,
  decisionNote?: string,
) {
  return unwrapData<OpenPlatformRedirectURIRequest>(
    await adminApi.approveOpenPlatformRedirectURIRequest(
      appID,
      requestID,
      decisionNote ? { decisionNote } : undefined,
    ),
  );
}

export async function rejectOpenPlatformRedirectURIRequest(
  appID: number,
  requestID: number,
  decisionNote?: string,
) {
  return unwrapData<OpenPlatformRedirectURIRequest>(
    await adminApi.rejectOpenPlatformRedirectURIRequest(
      appID,
      requestID,
      decisionNote ? { decisionNote } : undefined,
    ),
  );
}

export async function rotateOpenPlatformAppSecret(
  appID: number,
  reason?: string,
) {
  return unwrapData<OpenPlatformRotatedSecret>(
    await adminApi.rotateOpenPlatformAppSecret(
      appID,
      reason ? { reason } : undefined,
    ),
  );
}

export async function suspendOpenPlatformApp(appID: number, reason: string) {
  return unwrapData<OpenPlatformAppLifecycleResult>(
    await adminApi.suspendOpenPlatformApp(appID, { reason }),
  );
}

export async function resumeOpenPlatformApp(appID: number, reason: string) {
  return unwrapData<OpenPlatformAppLifecycleResult>(
    await adminApi.resumeOpenPlatformApp(appID, { reason }),
  );
}

export async function revokeOpenPlatformApp(appID: number, reason: string) {
  return unwrapData<OpenPlatformAppLifecycleResult>(
    await adminApi.revokeOpenPlatformApp(appID, { reason }),
  );
}
