import type { ApiClient } from '@stuhelper/shared/api';

import { preferences } from '@vben/preferences';

import { baseRequestClient } from '#/api/request';
import { CSRF_COOKIE_NAME, CSRF_HEADER_NAME, readCookie } from '#/api/utils/csrf';

import type { ApiCallResult, ApiEnvelope } from './shared-result';

type HttpMethod = 'DELETE' | 'GET' | 'POST' | 'PUT';

const SAFE_METHODS = new Set<HttpMethod | 'HEAD' | 'OPTIONS'>([
  'GET',
  'HEAD',
  'OPTIONS',
]);

type RequestInitShape = {
  params?: {
    header?: Record<string, unknown>;
    path?: Record<string, unknown>;
    query?: Record<string, unknown>;
  };
  body?: unknown;
  signal?: AbortSignal;
};

const API_VERSION_PREFIX = '/api/v1';

let refreshPromise: null | Promise<boolean> = null;
let redirectPromise: null | Promise<void> = null;

function withSecurityHeaders(
  method: HttpMethod,
  headers: Record<string, string>,
): Record<string, string> {
  const nextHeaders = { ...headers };

  if (!SAFE_METHODS.has(method)) {
    const csrfToken = readCookie(CSRF_COOKIE_NAME);
    if (csrfToken) {
      nextHeaders[CSRF_HEADER_NAME] = csrfToken;
    }
  }

  nextHeaders['Accept-Language'] = preferences.app.locale;
  return nextHeaders;
}

function normalizeSchemaPath(schemaPath: string): string {
  const baseUrl = baseRequestClient.getBaseUrl() ?? '';
  const normalizedBaseUrl = baseUrl.replace(/\/$/, '');

  if (
    normalizedBaseUrl.endsWith(API_VERSION_PREFIX) &&
    schemaPath.startsWith(`${API_VERSION_PREFIX}/`)
  ) {
    return schemaPath.slice(API_VERSION_PREFIX.length);
  }

  return schemaPath;
}

function serializePath(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
): string {
  return schemaPath.replace(/\{([^}]+)\}/g, (_match, key) => {
    const value = pathParams?.[key];
    return encodeURIComponent(value == null ? '' : String(value));
  });
}

async function refreshSession(): Promise<boolean> {
  if (refreshPromise) {
    return refreshPromise;
  }

  refreshPromise = (async () => {
    try {
      await baseRequestClient.instance.request({
        headers: withSecurityHeaders('POST', {}),
        method: 'POST',
        url: normalizeSchemaPath('/api/v1/auth/refresh'),
        withCredentials: true,
      });

      return true;
    } catch {
      return false;
    } finally {
      refreshPromise = null;
    }
  })();

  return refreshPromise;
}

async function redirectToOIDCLogin() {
  if (typeof window === 'undefined') {
    return;
  }

  if (redirectPromise) {
    return redirectPromise;
  }

  redirectPromise = (async () => {
    try {
      const { resetAllStores } = await import('@vben/stores');
      resetAllStores();
    } catch {
      // ignore store reset failures during forced re-auth
    }

    try {
      const response = await baseRequestClient.instance.request<
        ApiEnvelope<{ url?: string }>
      >({
        headers: withSecurityHeaders('GET', {}),
        method: 'GET',
        params: {
          redirect: window.location.href,
        },
        url: normalizeSchemaPath('/api/v1/auth/login'),
        withCredentials: true,
      });

      const url = response.data?.data?.url;
      if (url) {
        window.location.replace(url);
        return;
      }
    } catch {
      // ignore and fall back below
    }

    window.location.replace('/admin/');
  })();

  return redirectPromise;
}

async function performRequest<T>(
  method: HttpMethod,
  schemaPath: string,
  init?: RequestInitShape,
  enableRefresh: boolean = true,
  shouldReAuthenticate: boolean = enableRefresh,
): Promise<ApiCallResult<T>> {
  const url = serializePath(
    normalizeSchemaPath(schemaPath),
    init?.params?.path,
  );

  try {
    const response = await baseRequestClient.instance.request<ApiEnvelope<T>>({
      data: init?.body,
      headers: withSecurityHeaders(
        method,
        (init?.params?.header as Record<string, string> | undefined) ?? {},
      ),
      method,
      params: init?.params?.query,
      signal: init?.signal,
      url,
      withCredentials: true,
    });

    return {
      data: response.data,
      response: {
        status: response.status,
      },
    };
  } catch (error) {
    const responseError = error as {
      response?: {
        data?: unknown;
        status?: number;
      };
    };
    const status = responseError.response?.status;

    if (
      status === 401 &&
      enableRefresh &&
      schemaPath !== '/api/v1/auth/refresh'
    ) {
      const refreshed = await refreshSession();
      if (refreshed) {
        return performRequest<T>(
          method,
          schemaPath,
          init,
          false,
          shouldReAuthenticate,
        );
      }

      if (shouldReAuthenticate) {
        await redirectToOIDCLogin();
        return {
          error: responseError.response?.data ?? error,
          response: {
            status,
          },
        };
      }
    }

    if (
      status === 401 &&
      shouldReAuthenticate &&
      schemaPath !== '/api/v1/auth/refresh'
    ) {
      await redirectToOIDCLogin();
    }

    return {
      error: responseError.response?.data ?? error,
      response: {
        status,
      },
    };
  }
}

export async function requestApi<T>(
  method: HttpMethod,
  schemaPath: string,
  init?: RequestInitShape,
  options?: {
    enableRefresh?: boolean;
    shouldReAuthenticate?: boolean;
  },
): Promise<ApiCallResult<T>> {
  const enableRefresh = options?.enableRefresh ?? true;
  return performRequest<T>(
    method,
    schemaPath,
    init,
    enableRefresh,
    options?.shouldReAuthenticate ?? enableRefresh,
  );
}

function buildApiClient(enableRefresh: boolean): ApiClient {
  return {
    DELETE: ((schemaPath: string, init?: RequestInitShape) =>
      requestApi('DELETE', schemaPath, init, {
        enableRefresh,
        shouldReAuthenticate: enableRefresh,
      })) as ApiClient['DELETE'],
    GET: ((schemaPath: string, init?: RequestInitShape) =>
      requestApi('GET', schemaPath, init, {
        enableRefresh,
        shouldReAuthenticate: enableRefresh,
      })) as ApiClient['GET'],
    POST: ((schemaPath: string, init?: RequestInitShape) =>
      requestApi('POST', schemaPath, init, {
        enableRefresh,
        shouldReAuthenticate: enableRefresh,
      })) as ApiClient['POST'],
    PUT: ((schemaPath: string, init?: RequestInitShape) =>
      requestApi('PUT', schemaPath, init, {
        enableRefresh,
        shouldReAuthenticate: enableRefresh,
      })) as ApiClient['PUT'],
  };
}

export const sharedApiClient = buildApiClient(true);
export const sharedBaseApiClient = buildApiClient(false);
