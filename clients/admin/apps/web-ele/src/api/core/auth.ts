import { createAuthApi } from '@stuhelper/shared/api';
import type { components } from '@stuhelper/shared';

import { sharedApiClient, sharedBaseApiClient } from '#/api/shared-client';
import { unwrapData, unwrapOptionalData } from '#/api/shared-result';

const authApi = createAuthApi(sharedApiClient);
const baseAuthApi = createAuthApi(sharedBaseApiClient);

export namespace AuthApi {
  export type LoginUrlResult = components['schemas']['LoginURLResponse'];
  export type MeResult = components['schemas']['UserInfo'];
}

/**
 * 从后端获取 OIDC 授权 URL，然后浏览器跳转
 * 传递当前页面 URL 作为登录后回跳地址
 */
export async function redirectToOIDCLogin(redirectPath?: string) {
  const currentUrl =
    redirectPath &&
    redirectPath.startsWith('/') &&
    !redirectPath.startsWith('//')
      ? new URL(redirectPath, window.location.origin).toString()
      : redirectPath || window.location.href;
  const data = unwrapData<AuthApi.LoginUrlResult>(
    await baseAuthApi.login(currentUrl),
  );
  const url = data.url;
  if (url) {
    window.location.href = url;
  }
}

/**
 * 尝试获取当前会话（用 baseRequestClient，不触发 token 刷新）
 * 首次访问时没有 session，会 401，此时不应走刷新逻辑
 */
export async function tryGetMe(): Promise<AuthApi.MeResult | null> {
  try {
    return unwrapOptionalData<AuthApi.MeResult>(await baseAuthApi.me());
  } catch {
    return null;
  }
}

/**
 * 获取当前会话用户信息（已登录后使用，走标准拦截器）
 */
export async function getMeApi() {
  return unwrapData<AuthApi.MeResult>(await authApi.me());
}

/**
 * 刷新 Token（Cookie 自动携带）
 */
export async function refreshTokenApi() {
  return unwrapData(await authApi.refresh());
}

/**
 * 退出登录
 */
export async function logoutApi() {
  return unwrapData(await authApi.logout());
}
