import type { components } from '@stuhelper/shared/types';

import type { ApiCallResult } from '#/api/shared-result';

import { createAuthApi, extractResultErrorCode } from '@stuhelper/shared/api';

import { sharedApiClient, sharedBaseApiClient } from '#/api/shared-client';
import { unwrapData } from '#/api/shared-result';

const authApi = createAuthApi(sharedApiClient);
const baseAuthApi = createAuthApi(sharedBaseApiClient);

export namespace AuthApi {
  export type LoginUrlResult = components['schemas']['LoginURLResponse'];
  export type MeResult = components['schemas']['UserInfo'];
}

export type SessionProbeResult =
  | { kind: 'fatal_error'; message: string }
  | { kind: 'forbidden' }
  | { kind: 'ok'; me: AuthApi.MeResult }
  | { kind: 'retryable_error'; message: string }
  | { kind: 'unauthenticated' };

export type LogoutResult =
  | { kind: 'error'; message: string }
  | { kind: 'ok' }
  | { kind: 'unauthenticated' };

function classifySessionProbe(
  result: ApiCallResult<AuthApi.MeResult>,
): SessionProbeResult {
  if (result.data && 'data' in result.data && result.data.data) {
    return { kind: 'ok', me: result.data.data as AuthApi.MeResult };
  }

  const status = result.response?.status;
  const code = extractResultErrorCode(result);

  if (status === 401 || code?.startsWith('A00101')) {
    return { kind: 'unauthenticated' };
  }
  if (status === 403 || code === 'A0010200' || code === 'A0010201') {
    return { kind: 'forbidden' };
  }
  if (status !== undefined && status >= 500) {
    return { kind: 'retryable_error', message: 'admin.result.requestFailed' };
  }
  return { kind: 'fatal_error', message: 'admin.result.requestFailed' };
}

export function getAccountSettingsUrl(me: AuthApi.MeResult): string {
  const value = (me as Record<string, unknown>).accountSettingsUrl;
  return typeof value === 'string' ? value : '';
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
export async function tryGetMe(): Promise<SessionProbeResult> {
  return classifySessionProbe(await baseAuthApi.me());
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

function classifyLogoutResult(result: ApiCallResult<unknown>): LogoutResult {
  const status = result.response?.status;
  const code = extractResultErrorCode(result);

  if (!result.error && status !== undefined && status >= 200 && status < 300) {
    return { kind: 'ok' };
  }
  if (status === 401 || code?.startsWith('A00101')) {
    return { kind: 'unauthenticated' };
  }
  return { kind: 'error', message: 'admin.result.requestFailed' };
}

/**
 * 退出登录
 */
export async function logoutApi() {
  return classifyLogoutResult(await baseAuthApi.logout());
}
