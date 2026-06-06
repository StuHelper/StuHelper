import type { ApiClient } from './client'
import type { ApiCallResult } from './result'
import { extractResultData } from './result'

export type HttpMethod = 'DELETE' | 'GET' | 'PATCH' | 'POST' | 'PUT'

export type RequestHeadersShape =
  | Iterable<readonly unknown[]>
  | Record<string, unknown>
  | {
      forEach(callback: (value: unknown, key: string) => void): void
    }

export type RequestInitShape = {
  body?: unknown
  headers?: RequestHeadersShape
  params?: {
    header?: RequestHeadersShape
    path?: Record<string, unknown>
    query?: Record<string, unknown>
  }
  signal?: AbortSignal
}

export type RefreshSessionResult =
  | { kind: 'ok' }
  | { kind: 'unauthorized'; status?: number }
  | { kind: 'error'; error: unknown; status?: number }

export interface RefreshSessionData {
  message?: string
  expiresIn: number
  accessToken?: string
  refreshToken?: string
}

export interface RefreshSessionRequestOptions {
  body?: unknown
  onSuccess?(data: RefreshSessionData): Promise<void> | void
  request(init?: RequestInitShape): Promise<ApiCallResult<RefreshSessionData>>
  shouldTreatAsUnauthorized?(
    result: ApiCallResult<RefreshSessionData>,
    status: number,
  ): boolean
  unauthorizedStatuses?: number[]
}

export interface SessionTransport {
  onUnauthorized?(): Promise<void> | void
  refresh(): Promise<RefreshSessionResult>
  request<T>(
    method: HttpMethod,
    schemaPath: string,
    init?: RequestInitShape,
  ): Promise<ApiCallResult<T>>
  shouldRefresh?(schemaPath: string, status: number): boolean
}

export interface SessionClientOptions {
  enableRefresh?: boolean
  reauthenticateOnUnauthorized?: boolean
}

export const API_VERSION_PREFIX = '/api/v1'
export const AUTH_REFRESH_PATH = '/api/v1/auth/refresh'
export const CSRF_COOKIE_NAME = 'csrf_token'
export const CSRF_HEADER_NAME = 'X-CSRF-Token'
export const SAFE_HTTP_METHODS = new Set<HttpMethod | 'HEAD' | 'OPTIONS'>([
  'GET',
  'HEAD',
  'OPTIONS',
])

export function normalizeSchemaPath(baseUrl: string, schemaPath: string): string {
  const normalizedBaseUrl = baseUrl.replace(/\/$/, '')

  if (
    normalizedBaseUrl.endsWith(API_VERSION_PREFIX) &&
    schemaPath.startsWith(`${API_VERSION_PREFIX}/`)
  ) {
    return schemaPath.slice(API_VERSION_PREFIX.length)
  }

  return schemaPath
}

export function serializePath(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
): string {
  return schemaPath.replace(/\{([^}]+)\}/g, (_match, key) => {
    const value = pathParams?.[key]
    return encodeURIComponent(
      value === null || value === undefined ? '' : String(value),
    )
  })
}

export function appendQuery(
  path: string,
  query?: Record<string, unknown>,
): string {
  if (!query) return path

  const pairs: string[] = []
  for (const [key, value] of Object.entries(query)) {
    if (value == null || value === '') continue
    if (Array.isArray(value)) {
      for (const item of value) {
        if (item == null || item === '') continue
        pairs.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(item))}`)
      }
      continue
    }
    pairs.push(`${encodeURIComponent(key)}=${encodeURIComponent(String(value))}`)
  }

  if (pairs.length === 0) return path
  return `${path}${path.includes('?') ? '&' : '?'}${pairs.join('&')}`
}

export function needsCSRF(method: string): boolean {
  return !SAFE_HTTP_METHODS.has(method.toUpperCase() as HttpMethod | 'HEAD' | 'OPTIONS')
}

export function buildSecurityHeaders(
  method: HttpMethod,
  headers: Record<string, string>,
  options: {
    acceptLanguage?: string
    csrfToken?: string | null
  } = {},
): Record<string, string> {
  const nextHeaders = { ...headers }
  if (needsCSRF(method) && options.csrfToken) {
    nextHeaders[CSRF_HEADER_NAME] = options.csrfToken
  }
  if (options.acceptLanguage) {
    nextHeaders['Accept-Language'] = options.acceptLanguage
  }
  return nextHeaders
}

function assignHeader(
  target: Record<string, string>,
  key: unknown,
  value: unknown,
): void {
  if (key == null || value == null) return
  const name = String(key).trim()
  if (name === '') return
  target[name] = String(value)
}

function appendHeaders(
  target: Record<string, string>,
  source?: RequestHeadersShape,
): void {
  if (!source) return

  const maybeForEach = source as {
    forEach?: (callback: (value: unknown, key: string) => void) => void
  }
  if (typeof maybeForEach.forEach === 'function' && !Array.isArray(source)) {
    maybeForEach.forEach((value, key) => assignHeader(target, key, value))
    return
  }

  const maybeIterable = source as {
    [Symbol.iterator]?: () => IterableIterator<readonly unknown[]>
  }
  if (typeof maybeIterable[Symbol.iterator] === 'function') {
    for (const entry of maybeIterable as Iterable<readonly unknown[]>) {
      assignHeader(target, entry[0], entry[1])
    }
    return
  }

  for (const [key, value] of Object.entries(source)) {
    assignHeader(target, key, value)
  }
}

export function normalizeRequestHeaders(
  init?: Pick<RequestInitShape, 'headers' | 'params'>,
): Record<string, string> {
  const headers: Record<string, string> = {}
  appendHeaders(headers, init?.headers)
  appendHeaders(headers, init?.params?.header)
  return headers
}

export function extractRefreshSessionData(
  result: ApiCallResult<RefreshSessionData>,
): RefreshSessionData | null {
  const payload = extractResultData(result)
  if (!payload || typeof payload.expiresIn !== 'number') {
    return null
  }
  return payload
}

export async function executeSessionRefresh(
  options: RefreshSessionRequestOptions,
): Promise<RefreshSessionResult> {
  try {
    const result = await options.request(
      options.body === undefined ? undefined : { body: options.body },
    )
    const status = result.response?.status ?? 0

    if (!result.error && status >= 200 && status < 300) {
      const payload = extractRefreshSessionData(result)
      if (payload) {
        await options.onSuccess?.(payload)
      }
      return { kind: 'ok' }
    }

    const shouldTreatAsUnauthorized = options.shouldTreatAsUnauthorized?.(
      result,
      status,
    ) ?? (options.unauthorizedStatuses ?? [401]).includes(status)

    if (shouldTreatAsUnauthorized) {
      return { kind: 'unauthorized', status }
    }

    return {
      kind: 'error',
      error: result.error ?? result.data,
      status,
    }
  } catch (error) {
    return { kind: 'error', error }
  }
}

function defaultShouldRefresh(schemaPath: string, status: number): boolean {
  return status === 401 && schemaPath !== AUTH_REFRESH_PATH
}

function isReauthenticationStatus(status: number): boolean {
  return status === 401
}

export function createSessionApiClient(
  transport: SessionTransport,
  options: SessionClientOptions = {},
): ApiClient {
  let refreshPromise: null | Promise<RefreshSessionResult> = null

  async function runRefresh(): Promise<RefreshSessionResult> {
    if (refreshPromise) {
      return refreshPromise
    }

    refreshPromise = transport
      .refresh()
      .catch((error) => ({ kind: 'error', error } as const))
      .finally(() => {
        refreshPromise = null
      })

    return refreshPromise
  }

  async function maybeHandleUnauthorized(status: number) {
    if (!isReauthenticationStatus(status) || !options.reauthenticateOnUnauthorized) {
      return
    }
    await transport.onUnauthorized?.()
  }

  async function performRequest<T>(
    method: HttpMethod,
    schemaPath: string,
    init?: unknown,
  ): Promise<ApiCallResult<T>> {
    const requestInit = init as RequestInitShape | undefined
    const result = await transport.request<T>(method, schemaPath, requestInit)
    const status = result.response?.status ?? 0
    const shouldRefresh =
      options.enableRefresh !== false &&
      (transport.shouldRefresh?.(schemaPath, status) ??
        defaultShouldRefresh(schemaPath, status))

    if (!shouldRefresh) {
      await maybeHandleUnauthorized(status)
      return result
    }

    const refreshed = await runRefresh()
    if (refreshed.kind === 'ok') {
      return transport.request<T>(method, schemaPath, requestInit)
    }

    if (refreshed.kind === 'unauthorized') {
      await maybeHandleUnauthorized(refreshed.status ?? status)
      return {
        ...result,
        response: {
          ...result.response,
          status: refreshed.status ?? status,
        },
      }
    }

    return {
      error: refreshed.error,
      response: {
        ...result.response,
        status: refreshed.status ?? status,
      },
    }
  }

  return {
    DELETE: ((schemaPath: string, init?: unknown) =>
      performRequest('DELETE', schemaPath, init)) as ApiClient['DELETE'],
    GET: ((schemaPath: string, init?: unknown) =>
      performRequest('GET', schemaPath, init)) as ApiClient['GET'],
    PATCH: ((schemaPath: string, init?: unknown) =>
      performRequest('PATCH', schemaPath, init)) as ApiClient['PATCH'],
    POST: ((schemaPath: string, init?: unknown) =>
      performRequest('POST', schemaPath, init)) as ApiClient['POST'],
    PUT: ((schemaPath: string, init?: unknown) =>
      performRequest('PUT', schemaPath, init)) as ApiClient['PUT'],
  }
}
