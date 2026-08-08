import type { ApiClient } from './client'
import type { components } from '../types/api.gen'

type UpdateSystemConfigRequest = components['schemas']['UpdateSystemConfigRequest']

/** User-system administration that is unrelated to student verification. */
export const createUserAdminApi = (client: ApiClient) => ({
  listSystemConfigs: () =>
    client.GET('/api/v1/admin/system-configs'),

  updateSystemConfig: (key: string, data: UpdateSystemConfigRequest) =>
    client.PUT('/api/v1/admin/system-configs/{key}', { params: { path: { key } }, body: data }),
})
