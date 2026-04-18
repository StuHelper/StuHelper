import type { ApiCallResult, ApiEnvelope } from './shared-result'

import {
  AUTH_REFRESH_PATH,
  buildSecurityHeaders,
  createSessionApiClient,
  executeSessionRefresh,
  normalizeSchemaPath,
  serializePath,
  type HttpMethod,
  type RefreshSessionData,
  type RequestInitShape,
} from '@stuhelper/shared/api'
import { preferences } from '@vben/preferences'

import { baseRequestClient } from '#/api/request'
import {
  CSRF_COOKIE_NAME,
  readCookie,
} from '#/api/utils/csrf'

type TransportError = {
  response?: {
    data?: unknown
    status?: number
  }
}

function logAdminAuthWarning(
  event: string,
  error: unknown,
  extra?: Record<string, unknown>,
) {
  console.warn('[admin-auth]', event, extra ?? {}, error)
}

function withSecurityHeaders(
  method: HttpMethod,
  headers: Record<string, string>,
): Record<string, string> {
  return buildSecurityHeaders(method, headers, {
    acceptLanguage: preferences.app.locale,
    csrfToken: readCookie(CSRF_COOKIE_NAME),
  })
}

function serializeSchemaPath(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
): string {
  return serializePath(
    normalizeSchemaPath(baseRequestClient.getBaseUrl() ?? '', schemaPath),
    pathParams,
  )
}

async function redirectToOIDCLogin() {
  if (typeof window === 'undefined') {
    return
  }

  try {
    const { resetAllStores } = await import('@vben/stores')
    resetAllStores()
  } catch (error) {
    logAdminAuthWarning('resetAllStores failed during forced re-auth', error)
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
      url: serializeSchemaPath('/api/v1/auth/login'),
      withCredentials: true,
    })

    const url = response.data?.data?.url
    if (url) {
      window.location.replace(url)
      return
    }
  } catch (error) {
    logAdminAuthWarning(
      'failed to fetch OIDC login URL during forced re-auth',
      error,
      {
        redirect: window.location.href,
      },
    )
  }

  window.location.replace('/admin/')
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
    })

    return {
      data: response.data,
      response: {
        status: response.status,
      },
    }
  } catch (error) {
    const responseError = error as TransportError
    return {
      error: responseError.response?.data ?? error,
      response: {
        status: responseError.response?.status,
      },
    }
  }
}

async function refreshSession() {
  return executeSessionRefresh({
    request: (init) =>
      doRequest<RefreshSessionData>('POST', AUTH_REFRESH_PATH, {
        params: { header: {} },
        ...init,
      }),
  })
}

const adminSessionTransport = {
  onUnauthorized: redirectToOIDCLogin,
  refresh: refreshSession,
  request: doRequest,
}

export const sharedApiClient = createSessionApiClient(adminSessionTransport, {
  enableRefresh: true,
  reauthenticateOnUnauthorized: true,
})

export const sharedBaseApiClient = createSessionApiClient(adminSessionTransport, {
  enableRefresh: false,
  reauthenticateOnUnauthorized: false,
})
