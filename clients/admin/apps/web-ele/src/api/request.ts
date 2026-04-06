/**
 * StuHelper Admin API 请求客户端
 *
 * 适配 Go 后端响应格式 {success, data, error}
 * 使用 HttpOnly Cookie 认证（非 Bearer token）
 */
import type { RequestClientOptions } from '@vben/request';

import { useAppConfig } from '@vben/hooks';
import { preferences } from '@vben/preferences';
import {
  authenticateResponseInterceptor,
  errorMessageResponseInterceptor,
  RequestClient,
} from '@vben/request';
import { useAccessStore } from '@vben/stores';

import { ElMessage } from 'element-plus';

import { useAuthStore } from '#/store';

const { apiURL } = useAppConfig(import.meta.env, import.meta.env.PROD);
const CSRF_COOKIE_NAME = 'csrf_token';
const CSRF_HEADER_NAME = 'X-CSRF-Token';

function readCookie(name: string): string | null {
  if (typeof document === 'undefined') {
    return null;
  }

  const target = `${encodeURIComponent(name)}=`;
  const cookies = document.cookie ? document.cookie.split(';') : [];
  for (const rawCookie of cookies) {
    const cookie = rawCookie.trim();
    if (!cookie.startsWith(target)) {
      continue;
    }

    try {
      return decodeURIComponent(cookie.slice(target.length));
    } catch {
      return null;
    }
  }

  return null;
}

function withSecurityHeaders(headers: Record<string, string>): Record<string, string> {
  const nextHeaders = { ...headers };
  const csrfToken = readCookie(CSRF_COOKIE_NAME);
  if (csrfToken) {
    nextHeaders[CSRF_HEADER_NAME] = csrfToken;
  }
  nextHeaders['Accept-Language'] = preferences.app.locale;
  return nextHeaders;
}

function extractErrorMessage(payload: unknown): string | null {
  if (!payload || typeof payload !== 'object') {
    return null;
  }

  const record = payload as Record<string, unknown>;
  const error = record.error;
  if (error && typeof error === 'object') {
    const errorRecord = error as Record<string, unknown>;
    const message = errorRecord.message;
    if (typeof message === 'string' && message.trim()) {
      return message;
    }
    const nestedError = errorRecord.error;
    if (typeof nestedError === 'string' && nestedError.trim()) {
      return nestedError;
    }
  }

  const message = record.message;
  if (typeof message === 'string' && message.trim()) {
    return message;
  }

  return null;
}

function createRequestClient(baseURL: string, options?: RequestClientOptions) {
  const client = new RequestClient({
    ...options,
    baseURL,
  });

  /**
   * 重新认证：重定向到 Zitadel 登录页
   */
  async function doReAuthenticate() {
    console.warn('Session expired, redirecting to login.');
    const accessStore = useAccessStore();
    const authStore = useAuthStore();
    accessStore.setAccessToken(null);
    if (
      preferences.app.loginExpiredMode === 'modal' &&
      accessStore.isAccessChecked
    ) {
      accessStore.setLoginExpired(true);
    } else {
      await authStore.logout();
    }
  }

  /**
   * 刷新 token：调用后端 refresh 端点（Cookie 自动携带）
   */
  async function doRefreshToken() {
    await client.post<{ expiresIn?: number; message?: string }>(
      '/auth/refresh',
    );
    const newToken = 'cookie-session';
    const accessStore = useAccessStore();
    accessStore.setAccessToken(newToken);
    return newToken;
  }

  function formatToken(token: null | string) {
    return token ? `Bearer ${token}` : null;
  }

  // 请求拦截：Cookie 自动携带，添加语言头
  client.addRequestInterceptor({
    fulfilled: async (config) => {
      config.withCredentials = true;
      const headers = withSecurityHeaders(
        (config.headers ?? {}) as Record<string, string>,
      );

      // 如果有 access token（从 cookie 中转存），添加到 header
      const accessStore = useAccessStore();
      if (accessStore.accessToken) {
        const authorization = formatToken(accessStore.accessToken);
        if (authorization) {
          headers.Authorization = authorization;
        }
      }

      config.headers = headers as any;
      return config;
    },
  });

  // StuHelper 响应格式适配：{success: bool, data: T, error?: string}
  client.addResponseInterceptor({
    fulfilled: (response) => {
      const { data: responseData, status } = response;

      if (status >= 200 && status < 400 && responseData) {
        // 后端返回 {success, data, error} 格式
        if (responseData.success === true) {
          return responseData.data;
        }
        // success=false，抛出错误让下游处理
        if (responseData.success === false) {
          const errorMessage =
            extractErrorMessage(responseData) ?? 'Request failed';
          const error = new Error(errorMessage);
          (error as any).response = response;
          throw error;
        }
      }

      // 非标准格式（如文件下载等），直接返回原始数据
      return responseData;
    },
  });

  // 401 处理
  client.addResponseInterceptor(
    authenticateResponseInterceptor({
      client,
      doReAuthenticate,
      doRefreshToken,
      enableRefreshToken: preferences.app.enableRefreshToken,
      formatToken,
    }),
  );

  // 通用错误提示
  client.addResponseInterceptor(
    errorMessageResponseInterceptor((msg: string, error) => {
      const responseData = error?.response?.data ?? {};
      const errorMessage = extractErrorMessage(responseData) ?? '';
      ElMessage.error(errorMessage || msg);
    }),
  );

  return client;
}

export const requestClient = createRequestClient(apiURL);

export const baseRequestClient = new RequestClient({
  baseURL: apiURL,
  withCredentials: true,
});
