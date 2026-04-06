import type { UserInfo } from '@vben/types';

import { preferences } from '@vben/preferences';

import { getMeApi } from './auth';

/**
 * 获取用户信息（从 /auth/me 转换为 Vben UserInfo 格式）
 */
export async function getUserInfoApi(): Promise<UserInfo> {
  const me = await getMeApi();
  return {
    userId: String(me.id),
    username: me.name,
    realName: me.displayName,
    avatar: me.avatar ?? '',
    desc: '',
    token: '',
    roles: me.roles,
    homePath: preferences.app.defaultHomePath,
  };
}

/**
 * 获取用户权限码（capabilities）
 */
export async function getAccessCodesApi(): Promise<string[]> {
  const me = await getMeApi();
  return me.globalCapabilities?.length
    ? me.globalCapabilities
    : me.capabilities;
}
