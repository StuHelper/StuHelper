/**
 * Shared low-level request client bootstrap.
 *
 * All authenticated request / refresh / re-auth behavior lives in
 * /Users/zxy/Code/StuHelper/clients/admin/apps/web-ele/src/api/shared-client.ts.
 *
 * This module only exposes the raw RequestClient instance used by the shared
 * pipeline and a handful of base-client probes (for example auth tryGetMe).
 */
import { useAppConfig } from '@vben/hooks';
import { RequestClient } from '@vben/request';

const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);

export const baseRequestClient = new RequestClient({
  baseURL: apiURL,
  withCredentials: true,
});
