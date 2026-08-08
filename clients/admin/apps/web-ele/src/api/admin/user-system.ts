import type { components } from '@stuhelper/shared/types';

import { createUserAdminApi } from '@stuhelper/shared/api';

import { sharedApiClient } from '#/api/shared-client';
import { unwrapData } from '#/api/shared-result';

const userAdminApi = createUserAdminApi(sharedApiClient);

export type SystemConfig = components['schemas']['SystemConfig'];

export async function getSystemConfigList() {
  return unwrapData<SystemConfig[]>(await userAdminApi.listSystemConfigs());
}

export async function updateSystemConfig(key: string, data: { value: string }) {
  return unwrapData(await userAdminApi.updateSystemConfig(key, data));
}
