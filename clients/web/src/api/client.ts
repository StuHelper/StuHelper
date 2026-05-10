import type { ApiClient } from '@stuhelper/shared/api'
import {
  AUTH_REFRESH_PATH,
  appendQuery,
  buildSecurityHeaders,
  createSessionApiClient,
  executeSessionRefresh,
  parseApiError,
  serializePath,
  type HttpMethod,
  type RefreshSessionData,
  type RequestInitShape,
} from '@stuhelper/shared/api'
import {
  ApiError,
  httpStatusToDefaultCode,
  isAuthError,
  isCsrfError,
} from './errors'
import { useAuthStore } from '@/stores/auth'
import { tokenExpiry } from '@/utils/auth'

const RAW_API_BASE_URL = import.meta.env.VITE_API_URL?.trim() || ''
const AUTH_REFRESH_TIMEOUT_MS = 10_000
const DEFAULT_REQUEST_TIMEOUT_MS = (() => {
  const raw = Number(import.meta.env.VITE_API_TIMEOUT_MS ?? 15_000)
  return Number.isFinite(raw) && raw > 0 ? raw : 15_000
})()
const CSRF_COOKIE_NAME = 'csrf_token'
const inflightGetRequests = new Map<string, Promise<unknown>>()
let didWarnMissingApiBaseUrl = false

interface ErrorResponse {
  headers?: Pick<Headers, 'get'>
  status: number
  statusText?: string
}

function stripSchemaPrefix(baseUrl: string): string {
  const trimmed = baseUrl.trim().replace(/\/$/, '')
  return trimmed.replace(/\/api(?:\/v1)?$/, '')
}

export function resolveApiBaseUrl(rawBaseUrl: string = RAW_API_BASE_URL): string {
  const normalizedBaseUrl = stripSchemaPrefix(rawBaseUrl)

  if (/^https?:\/\//.test(normalizedBaseUrl)) {
    return normalizedBaseUrl
  }

  if (window.location?.origin) {
    if (import.meta.env.DEV && normalizedBaseUrl === '' && !didWarnMissingApiBaseUrl) {
      didWarnMissingApiBaseUrl = true
      console.warn('[api] VITE_API_URL not set, falling back to window.location.origin')
    }
    return window.location.origin
  }

  return normalizedBaseUrl
}

function readCookie(name: string): string | null {
  const cookies = document.cookie ? document.cookie.split(';') : []
  const target = `${encodeURIComponent(name)}=`
  for (const raw of cookies) {
    const cookie = raw.trim()
    if (!cookie.startsWith(target)) continue
    try {
      return decodeURIComponent(cookie.slice(target.length))
    } catch (_error) { void _error;
      return null
    }
  }
  return null
}

function resolveApiPath(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
  query?: Record<string, unknown>,
): string {
  return appendQuery(serializePath(schemaPath, pathParams), query)
}

/**
 * 将 API 路径解析为绝对 URL。
 */
export function resolveApiURL(path: string): URL {
  const base = resolveApiBaseUrl()
  return new URL(path, base)
}

function withBrowserSecurity(request: Request): Request {
  const headers = buildSecurityHeaders(
    request.method as HttpMethod,
    Object.fromEntries(request.headers.entries()),
    {
      csrfToken: readCookie(CSRF_COOKIE_NAME),
    },
  )

  return new Request(request, {
    credentials: 'include',
    headers,
  })
}

function isRefreshRequestPath(pathname: string): boolean {
  return pathname === AUTH_REFRESH_PATH
}

function normalizeClientError(error: unknown): ApiError {
  if (error instanceof ApiError) {
    return error
  }

  if (!navigator.onLine) {
    return new ApiError({ code: 'OFFLINE', message: 'offline' })
  }

  if (error instanceof DOMException && error.name === 'AbortError') {
    return new ApiError({ code: 'TIMEOUT', message: 'request timeout' })
  }

  return new ApiError({
    code: 'NETWORK_ERROR',
    message: error instanceof Error ? error.message : 'network error',
  })
}

function hasRefreshSessionHint(): boolean {
  return readCookie(CSRF_COOKIE_NAME) !== null || tokenExpiry.get() !== null
}

async function fetchWithTimeout(
  request: Request,
  timeoutMs: number,
): Promise<Response> {
  const controller = new AbortController()
  const upstreamSignal = request.signal
  const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs)
  const abortFromUpstream = () => controller.abort()

  if (upstreamSignal) {
    if (upstreamSignal.aborted) {
      controller.abort()
    } else {
      upstreamSignal.addEventListener('abort', abortFromUpstream, {
        once: true,
      })
    }
  }

  try {
    return await window.fetch(
      new Request(request, { signal: controller.signal }),
    )
  } finally {
    window.clearTimeout(timeoutId)
    upstreamSignal?.removeEventListener('abort', abortFromUpstream)
  }
}

async function performBrowserFetch(request: Request): Promise<Response> {
  const timeoutMs = isRefreshRequestPath(new URL(request.url, window.location.origin).pathname)
    ? AUTH_REFRESH_TIMEOUT_MS
    : DEFAULT_REQUEST_TIMEOUT_MS
  return fetchWithTimeout(request, timeoutMs)
}

function normalizeRequestBody(body: unknown): BodyInit | null | undefined {
  if (body == null) return undefined
  if (
    typeof body === 'string' ||
    body instanceof Blob ||
    body instanceof FormData ||
    body instanceof URLSearchParams ||
    body instanceof ArrayBuffer ||
    ArrayBuffer.isView(body)
  ) {
    return body as BodyInit
  }
  if (typeof body === 'object') {
    return JSON.stringify(body)
  }
  return String(body)
}

function requestHeadersToObject(headers: Headers): Record<string, unknown> {
  return Object.fromEntries(headers.entries())
}

async function browserRequest<T>(
  method: HttpMethod,
  schemaPath: string,
  init?: RequestInitShape,
) {
  const path = resolveApiPath(schemaPath, init?.params?.path, init?.params?.query)
  const url = resolveApiURL(path)
  const requestHeaders = new Headers(
    Object.entries(init?.params?.header ?? {}).reduce<Record<string, string>>(
      (acc, [key, value]) => {
        if (value != null) acc[key] = String(value)
        return acc
      },
      {},
    ),
  )
  const body = normalizeRequestBody(init?.body)
  if (body !== undefined && !requestHeaders.has('Content-Type')) {
    requestHeaders.set('Content-Type', 'application/json')
  }

  const request = withBrowserSecurity(
    new Request(url, {
      body,
      credentials: 'include',
      headers: requestHeaders,
      method,
      signal: init?.signal,
    }),
  )

  const response = await performBrowserFetch(request)
  const payload = await response.clone().json().catch(() => null)

  if (response.ok) {
    return {
      data: payload as T,
      response,
    }
  }

  return {
    error: payload,
    response,
  }
}

async function refreshSession() {
  return executeSessionRefresh({
    async onSuccess(payload) {
      tokenExpiry.set(payload.expiresIn)
    },
    async request(init) {
      const result = await browserRequest<RefreshSessionData>('POST', AUTH_REFRESH_PATH, init)
      const status = result.response?.status ?? 0
      if (!result.error && status >= 200 && status < 300) {
        return result
      }

      return {
        ...result,
        error: buildApiError(result.response ?? { status }, result.error ?? result.data),
      }
    },
    shouldTreatAsUnauthorized(result, status) {
      if (status === 401) {
        return true
      }
      const error = result.error
      return (
        error instanceof ApiError &&
        !isCsrfError(error.code) &&
        isAuthError(error.code)
      )
    },
  }).then((result) => {
    if (
      result.kind === 'error' &&
      result.error instanceof ApiError &&
      isAuthError(result.error.code)
    ) {
      return { kind: 'unauthorized', status: result.status } as const
    }
    if (
      result.kind === 'error' &&
      !(result.error instanceof ApiError)
    ) {
      return {
        kind: 'error',
        error: normalizeClientError(result.error),
        status: result.status,
      } as const
    }
    return result
  })
}

function responseHeader(response: ErrorResponse, name: string): string | null {
  return response.headers?.get(name) ?? null
}

function buildApiError(response: ErrorResponse, payload: unknown): ApiError {
  const errorData = parseApiError(payload)

  return new ApiError({
    message:
      errorData.message || response.statusText || 'API request failed',
    code: errorData.code || httpStatusToDefaultCode(response.status),
    status: response.status,
    details: errorData.details,
    requestID: responseHeader(response, 'X-Request-Id') || undefined,
  })
}

const browserSessionClient = createSessionApiClient(
  {
    onUnauthorized() {
      useAuthStore().clearSession()
    },
    refresh: refreshSession,
    request: browserRequest,
    shouldRefresh(schemaPath, status) {
      return status === 401 && schemaPath !== AUTH_REFRESH_PATH && hasRefreshSessionHint()
    },
  },
  {
    enableRefresh: true,
    reauthenticateOnUnauthorized: true,
  },
)

function callBrowserSessionClient(
  method: HttpMethod,
  schemaPath: string,
  init?: unknown,
) {
  switch (method) {
    case 'DELETE':
      return browserSessionClient.DELETE(schemaPath as never, init as never)
    case 'GET':
      return browserSessionClient.GET(schemaPath as never, init as never)
    case 'PATCH':
      return browserSessionClient.PATCH(schemaPath as never, init as never)
    case 'POST':
      return browserSessionClient.POST(schemaPath as never, init as never)
    case 'PUT':
      return browserSessionClient.PUT(schemaPath as never, init as never)
  }
}

async function authenticatedFetch(input: Request): Promise<Response> {
  const url = new URL(input.url, window.location.origin)
  const body = ['GET', 'HEAD'].includes(input.method.toUpperCase())
    ? undefined
    : await input.clone().text().catch(() => undefined)
  const result = await callBrowserSessionClient(
    input.method.toUpperCase() as HttpMethod,
    url.pathname + url.search,
    {
      body,
      params: {
        header: requestHeadersToObject(input.headers),
      },
      signal: input.signal,
    },
  )

  if (!result.response) {
    throw normalizeClientError(result.error)
  }
  return result.response as Response
}

async function invokeClientMethod(
  method: HttpMethod,
  schemaPath: string,
  init?: unknown,
) {
  const execute = async () => {
    const result = await callBrowserSessionClient(method, schemaPath, init)
    const status = result.response?.status ?? 0
    if (result.error || status >= 400) {
      throw buildApiError(result.response ?? { status }, result.error ?? result.data)
    }
    return result
  }

  try {
    if (method === 'GET' && canDeduplicateRequest(init)) {
      const key = buildInflightGetRequestKey(schemaPath, init)
      const existing = inflightGetRequests.get(key)
      if (existing) {
        return await existing
      }
      const pending = execute().finally(() => {
        inflightGetRequests.delete(key)
      })
      inflightGetRequests.set(key, pending)
      return await pending
    }

    return await execute()
  } catch (error) {
    if (error instanceof ApiError) {
      throw error
    }
    throw normalizeClientError(error)
  }
}

function canDeduplicateRequest(init?: unknown): boolean {
  if (!init || typeof init !== 'object') {
    return true
  }
  return !('signal' in init)
}

function buildInflightGetRequestKey(schemaPath: string, init?: unknown): string {
  return JSON.stringify({
    schemaPath,
    init: init ?? null,
  })
}

export const apiClientOptions = {
  baseUrl: resolveApiBaseUrl(),
  credentials: 'include' as const,
  fetch: ((input: RequestInfo | URL, init?: RequestInit) => {
    const request = new Request(input, init)
    return authenticatedFetch(request)
  }) as typeof fetch,
}

export const apiClient: ApiClient = {
  DELETE: ((schemaPath: string, init?: unknown) =>
    invokeClientMethod('DELETE', schemaPath, init)) as ApiClient['DELETE'],
  GET: ((schemaPath: string, init?: unknown) =>
    invokeClientMethod('GET', schemaPath, init)) as ApiClient['GET'],
  PATCH: ((schemaPath: string, init?: unknown) =>
    invokeClientMethod('PATCH', schemaPath, init)) as ApiClient['PATCH'],
  POST: ((schemaPath: string, init?: unknown) =>
    invokeClientMethod('POST', schemaPath, init)) as ApiClient['POST'],
  PUT: ((schemaPath: string, init?: unknown) =>
    invokeClientMethod('PUT', schemaPath, init)) as ApiClient['PUT'],
}

export const __testing__ = {
  DEFAULT_REQUEST_TIMEOUT_MS,
  AUTH_REFRESH_TIMEOUT_MS,
  readCookie,
  withBrowserSecurity,
  fetchWithTimeout,
  performBrowserFetch,
  authenticatedFetch,
  buildApiError,
  resolveApiBaseUrl,
  normalizeClientError,
  resolveApiURL,
  buildInflightGetRequestKey,
  canDeduplicateRequest,
}
