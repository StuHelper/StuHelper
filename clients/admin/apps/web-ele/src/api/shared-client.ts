import type {
  HttpMethod,
  RefreshSessionData,
  RequestInitShape,
} from '@stuhelper/shared/api';

import type { ApiCallResult, ApiEnvelope } from './shared-result';

import { preferences } from '@vben/preferences';

import {
  AUTH_REFRESH_PATH,
  buildSecurityHeaders,
  createSessionApiClient,
  executeSessionRefresh,
  normalizeSchemaPath,
  parseApiError,
  serializePath,
} from '@stuhelper/shared/api';

import { baseRequestClient } from '#/api/request';
import { CSRF_COOKIE_NAME, readCookie } from '#/api/utils/csrf';
import { adminLogger } from '#/utils/admin-logger';

type TransportError = {
  response?: {
    data?: unknown;
    status?: number;
  };
};

const STEP_UP_REQUIRED_CODE = 'A0010205';
const MFA_ENROLLMENT_REQUIRED_CODE = 'A0010204';

function logAdminAuthWarning(
  event: string,
  error: unknown,
  extra?: Record<string, unknown>,
) {
  adminLogger.warn(event, error, extra);
}

function normalizeAdminAuthError(
  error: unknown,
  fallbackMessage: string,
): Error {
  if (error instanceof Error) {
    return error;
  }
  return new Error(fallbackMessage);
}

function withSecurityHeaders(
  method: HttpMethod,
  headers: Record<string, string>,
): Record<string, string> {
  return buildSecurityHeaders(method, headers, {
    acceptLanguage: preferences.app.locale,
    csrfToken: readCookie(CSRF_COOKIE_NAME),
  });
}

function serializeSchemaPath(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
): string {
  return serializePath(
    normalizeSchemaPath(baseRequestClient.getBaseUrl() ?? '', schemaPath),
    pathParams,
  );
}

async function redirectToOIDCLogin() {
  if (typeof window === 'undefined') {
    return;
  }

  try {
    const { resetAllStores } = await import('@vben/stores');
    resetAllStores();
  } catch (error) {
    logAdminAuthWarning('resetAllStores failed during forced re-auth', error);
  }

  try {
    const response = await baseRequestClient.instance.request<
      ApiEnvelope<{ url?: string }>
    >({
      headers: withSecurityHeaders('GET', {}),
      method: 'GET',
      params: {
        app: 'admin',
        redirect: window.location.href,
      },
      url: serializeSchemaPath('/api/v1/auth/login'),
      withCredentials: true,
    });

    const url = response.data?.data?.url;
    if (url) {
      window.location.replace(url);
      return;
    }

    throw new Error('forced reauthentication login URL is missing');
  } catch (error) {
    logAdminAuthWarning(
      'failed to fetch OIDC login URL during forced re-auth',
      error,
      {
        redirect: window.location.href,
      },
    );
    throw normalizeAdminAuthError(error, 'forced reauthentication failed');
  }
}

async function redirectToStepUp() {
  if (typeof window === 'undefined') {
    return;
  }

  const response = await baseRequestClient.instance.request<
    ApiEnvelope<{ url?: string }>
  >({
    headers: withSecurityHeaders('GET', {}),
    method: 'GET',
    params: {
      platform: 'web',
      redirect: window.location.href,
    },
    url: serializeSchemaPath('/api/v1/auth/step-up'),
    withCredentials: true,
  });

  const url = response.data?.data?.url;
  if (!url) {
    throw new Error('step-up URL is missing');
  }
  window.location.replace(url);
}

async function redirectToMFAEnrollment() {
  if (typeof window === 'undefined') {
    return;
  }

  const response = await baseRequestClient.instance.request<
    ApiEnvelope<{ accountSettingsUrl?: string }>
  >({
    headers: withSecurityHeaders('GET', {}),
    method: 'GET',
    url: serializeSchemaPath('/api/v1/auth/me'),
    withCredentials: true,
  });

  const url = response.data?.data?.accountSettingsUrl;
  if (!url) {
    throw new Error('MFA enrollment URL is missing');
  }
  window.location.replace(url);
}

async function handleRecoverableAdminAuthError(
  status: number | undefined,
  payload: unknown,
) {
  const errorCode = parseApiError(payload).code;
  if (status === 403 && errorCode === MFA_ENROLLMENT_REQUIRED_CODE) {
    await redirectToMFAEnrollment();
  }
  if (status === 412 && errorCode === STEP_UP_REQUIRED_CODE) {
    await redirectToStepUp();
  }
}

async function doRequest<T>(
  method: HttpMethod,
  schemaPath: string,
  init?: RequestInitShape,
): Promise<ApiCallResult<T>> {
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
      url: serializeSchemaPath(schemaPath, init?.params?.path),
      withCredentials: true,
    });

    return {
      data: response.data,
      response: {
        status: response.status,
      },
    };
  } catch (error) {
    const responseError = error as TransportError;
    await handleRecoverableAdminAuthError(
      responseError.response?.status,
      responseError.response?.data,
    );
    return {
      error: responseError.response?.data ?? error,
      response: {
        status: responseError.response?.status,
      },
    };
  }
}

async function refreshSession() {
  return executeSessionRefresh({
    request: (init) =>
      doRequest<RefreshSessionData>('POST', AUTH_REFRESH_PATH, {
        params: { header: {} },
        ...init,
      }),
  });
}

const adminSessionTransport = {
  onUnauthorized: redirectToOIDCLogin,
  refresh: refreshSession,
  request: doRequest,
};

export const sharedApiClient = createSessionApiClient(adminSessionTransport, {
  enableRefresh: true,
  reauthenticateOnUnauthorized: true,
});

export const sharedBaseApiClient = createSessionApiClient(
  adminSessionTransport,
  {
    enableRefresh: false,
    reauthenticateOnUnauthorized: false,
  },
);
