import type { components } from '@stuhelper/shared';

import { createUserAdminApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData, unwrapListData } from '#/api/shared-result';

const userAdminApi = createUserAdminApi(sharedApiClient);

export type IdentityVerification =
  components['schemas']['AdminIdentityReviewItem'];
export type StudentVerification =
  components['schemas']['AdminStudentVerificationItem'];
export type SchoolConfig = components['schemas']['AdminSchoolConfig'];
export type SystemConfig = components['schemas']['SystemConfig'];
export type UpdateSchoolConfigPayload =
  components['schemas']['UpdateSchoolConfigRequest'];

export async function getIdentityList(params: {
  page?: number;
  pageSize?: number;
  status?: 'all' | 'pending' | 'rejected' | 'verified';
}) {
  return unwrapListData<IdentityVerification>(
    await userAdminApi.listIdentityVerifications(params),
  );
}

export async function reviewIdentity(
  userId: number,
  data: { approved: boolean; rejectionReason?: string },
) {
  return unwrapData(await userAdminApi.reviewIdentity(userId, data));
}

export async function getStudentVerificationList(params: {
  page?: number;
  pageSize?: number;
  schoolId?: string;
  status?: 'all' | 'pending' | 'rejected' | 'verified';
}) {
  const schoolID =
    params.schoolId && params.schoolId.trim() !== ''
      ? Number(params.schoolId)
      : undefined;

  return unwrapListData<StudentVerification>(
    await userAdminApi.listStudentVerifications({
      page: params.page,
      pageSize: params.pageSize,
      ...(schoolID === undefined || Number.isNaN(schoolID) ? {} : { schoolID }),
      status: params.status,
    }),
  );
}

export async function reviewStudentVerification(
  userId: number,
  data: { approved: boolean; rejectionReason?: string },
) {
  return unwrapData(await userAdminApi.reviewStudentVerification(userId, data));
}

export async function getSchoolConfigList() {
  return unwrapData<SchoolConfig[]>(await userAdminApi.listSchoolConfigs());
}

export async function updateSchoolConfig(
  schoolId: number,
  data: UpdateSchoolConfigPayload,
) {
  return unwrapData(await userAdminApi.updateSchoolConfig(schoolId, data));
}

export async function getSystemConfigList() {
  return unwrapData<SystemConfig[]>(await userAdminApi.listSystemConfigs());
}

export async function updateSystemConfig(key: string, data: { value: string }) {
  return unwrapData(await userAdminApi.updateSystemConfig(key, data));
}
