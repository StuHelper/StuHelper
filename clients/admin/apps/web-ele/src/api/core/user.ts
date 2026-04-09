import type { UserInfo } from '@vben/types';

import type { AuthApi } from './auth';

import { preferences } from '@vben/preferences';

import { getMeApi } from './auth';

/**
 * Raw /auth/me response enriched with Vben-compatible UserInfo.
 * Preserves backend-only fields (e.g. accountSettingsUrl) alongside
 * the Vben-shaped userInfo for the route guard and auth store.
 */
export interface MeWithUserInfo {
  userInfo: UserInfo;
  me: AuthApi.MeResult;
}

/**
 * Pure mapping: MeResult → Vben UserInfo.
 * Single source of truth for the field mapping — used by both
 * initSession() (cold start via tryGetMe) and fetchUserInfo()
 * (hot reload via getMeApi).
 */
export function mapMeToUserInfo(me: AuthApi.MeResult): UserInfo {
  return {
    userId: String(me.id),
    username: me.name,
    realName: me.displayName,
    avatar: me.avatar ?? '',
    desc: '',
    // Vestigial from Vben's built-in auth pattern. This project uses
    // cookie-based OIDC auth via Zitadel, so token is always empty.
    token: '',
    roles: me.roles,
    homePath: preferences.app.defaultHomePath,
  };
}

/**
 * 获取用户信息（从 /auth/me 转换为 Vben UserInfo 格式）
 * Returns both the Vben-shaped UserInfo and the raw /auth/me payload
 * so the caller can extract backend-only fields.
 */
export async function getUserInfoApi(): Promise<MeWithUserInfo> {
  const me = await getMeApi();
  return { userInfo: mapMeToUserInfo(me), me };
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
