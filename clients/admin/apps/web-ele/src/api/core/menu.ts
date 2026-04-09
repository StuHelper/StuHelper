import type { RouteRecordStringComponent } from '@vben/types';

import { requestApi } from '#/api/shared-client';
import { unwrapData } from '#/api/shared-result';

/**
 * 获取用户所有菜单
 */
export async function getAllMenusApi() {
  return unwrapData<RouteRecordStringComponent[]>(
    await requestApi<RouteRecordStringComponent[]>('GET', '/menu/all'),
  );
}
