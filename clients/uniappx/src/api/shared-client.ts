import type { ApiClient } from '@stuhelper/shared/api'
import {
  AUTH_REFRESH_PATH,
  CSRF_COOKIE_NAME,
  CSRF_HEADER_NAME,
  NATIVE_SESSION_ID_HEADER,
  appendQuery,
  buildSecurityHeaders,
  createSessionApiClient,
  executeSessionRefresh,
  normalizeSchemaPath,
  serializePath,
  type HttpMethod,
  type RefreshSessionData,
  type RequestInitShape,
} from '@stuhelper/shared/api'
import type { ApiCallResult } from './result'
import {
  isNativeRuntime,
  readNativeAccessToken,
  readNativeRefreshToken,
  readNativeSessionID,
  updateNativeTokensAfterRefresh,
} from './native-session'

type RequestBody = UniNamespace.RequestOptions['data']
type UniRequestMethod = NonNullable<UniNamespace.RequestOptions['method']> | 'PATCH'
type UniRequestOptions = Omit<UniNamespace.RequestOptions, 'method'> & {
  method?: UniRequestMethod
}
type UniRequest = (options: UniRequestOptions) => UniNamespace.RequestTask
type ApiResultResponse = NonNullable<ApiCallResult<unknown>['response']>

const API_BASE_URL = import.meta.env.VITE_API_URL ?? ''
const CSRF_STORAGE_KEY = 'stuhelper:csrf-token'
const H5 = typeof window !== 'undefined' && typeof window.location?.origin === 'string'
const UNI_REQUEST_METHODS = {
  DELETE: 'DELETE',
  GET: 'GET',
  PATCH: 'PATCH',
  POST: 'POST',
  PUT: 'PUT',
} satisfies Record<HttpMethod, UniRequestMethod>
const SESSION_BOUND_AUTH_PATHS = new Set<string>([
  AUTH_REFRESH_PATH,
  '/api/v1/auth/logout',
])

class CSRFTokenStorageError extends Error {
  cause: unknown

  constructor(action: 'persist' | 'read', cause: unknown) {
    super(`failed to ${action} csrf token`)
    this.name = 'CSRFTokenStorageError'
    this.cause = cause
  }
}

function stripRelativeBasePrefix(baseUrl: string, path: string): string {
  if (!baseUrl) return path
  if (path === baseUrl) return '/'
  if (!path.startsWith(baseUrl)) return path

  const remainder = path.slice(baseUrl.length)
  return remainder.startsWith('/') ? remainder : `/${remainder}`
}

function readCookie(name: string): string | null {
  if (typeof document === 'undefined') return null
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

function readStoredCSRFToken(): string | null {
  try {
    if (typeof uni !== 'undefined' && typeof uni.getStorageSync === 'function') {
      const value = uni.getStorageSync(CSRF_STORAGE_KEY)
      return typeof value === 'string' && value ? value : null
    }

    if (typeof localStorage !== 'undefined') {
      return localStorage.getItem(CSRF_STORAGE_KEY) || null
    }
  } catch (error) {
    throw new CSRFTokenStorageError('read', error)
  }

  return null
}

function writeStoredCSRFToken(token: string): void {
  if (!token) return

  try {
    if (typeof uni !== 'undefined' && typeof uni.setStorageSync === 'function') {
      uni.setStorageSync(CSRF_STORAGE_KEY, token)
      return
    }

    if (typeof localStorage !== 'undefined') {
      localStorage.setItem(CSRF_STORAGE_KEY, token)
    }
  } catch (error) {
    throw new CSRFTokenStorageError('persist', error)
  }
}

function readHeader(
  headers: Record<string, unknown> | undefined,
  name: string,
): string | null {
  if (!headers) return null

  const target = name.toLowerCase()
  for (const [key, value] of Object.entries(headers)) {
    if (key.toLowerCase() !== target || value == null) continue
    const normalized = String(value).trim()
    return normalized || null
  }

  return null
}

function persistCSRFToken(headers?: Record<string, unknown>) {
  const csrfToken = readHeader(headers, CSRF_HEADER_NAME)
  if (csrfToken) {
    writeStoredCSRFToken(csrfToken)
  }
}

function injectNativeSessionHeader(schemaPath: string, headers: Record<string, string>): void {
  if (!isNativeRuntime()) return
  if (!SESSION_BOUND_AUTH_PATHS.has(schemaPath)) return
  if (headers[NATIVE_SESSION_ID_HEADER]) return

  const sessionID = readNativeSessionID()
  if (sessionID) {
    headers[NATIVE_SESSION_ID_HEADER] = sessionID
  }
}

function resolveCSRFToken(): string | null {
  return readCookie(CSRF_COOKIE_NAME) ?? readStoredCSRFToken()
}

function resolveRequestURL(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
  query?: Record<string, unknown>,
): string {
  const normalizedPath = appendQuery(
    serializePath(normalizeSchemaPath(API_BASE_URL, schemaPath), pathParams),
    query,
  )
  const trimmedBase = API_BASE_URL.replace(/\/$/, '')

  if (/^https?:\/\//.test(trimmedBase)) {
    return `${trimmedBase}${normalizedPath}`
  }

  if (!H5) {
    throw new Error('VITE_API_URL must be an absolute http(s) URL for non-H5 uniappx builds')
  }

  const relativePath = stripRelativeBasePrefix(trimmedBase, normalizedPath)
  return `${window.location.origin}${trimmedBase}${relativePath}`
}

function normalizeBody(data: unknown): unknown {
  if (typeof data !== 'string') return data
  try {
    return JSON.parse(data)
  } catch (_error) { void _error;
    return data
  }
}

function normalizeRequestBody(data: unknown): RequestBody {
  if (data == null) return undefined
  if (typeof data === 'string' || data instanceof ArrayBuffer) return data
  if (typeof data === 'object') {
    return data as Record<string, unknown>
  }
  return String(data)
}

function requestWithPatch(options: UniRequestOptions): UniNamespace.RequestTask {
  // @dcloudio/types omits PATCH from uni.request, but H5/native transports accept it.
  const request = uni.request as UniRequest
  return request(options)
}

function createResponse(status: number): ApiResultResponse {
  return {
    status,
  }
}

function buildSuccessResult<T>(
  status: number,
  data: unknown,
  headers?: Record<string, unknown>,
): ApiCallResult<T> {
  persistCSRFToken(headers)
  return {
    data: normalizeBody(data) as ApiCallResult<T>['data'],
    response: createResponse(status),
  }
}

function buildErrorResult<T>(
  status: number,
  data: unknown,
  headers?: Record<string, unknown>,
): ApiCallResult<T> {
  persistCSRFToken(headers)
  return {
    error: normalizeBody(data),
    response: createResponse(status),
  }
}

function toTransportErrorResult<T>(error: unknown): ApiCallResult<T> {
  return {
    error,
    response: createResponse(0),
  }
}

function performRequest<T>(
  method: HttpMethod,
  schemaPath: string,
  init?: RequestInitShape,
): Promise<ApiCallResult<T>> {
  let url: string
  try {
    url = resolveRequestURL(schemaPath, init?.params?.path, init?.params?.query)
  } catch (error) {
    return Promise.reject(error)
  }

  const requestHeaders: Record<string, string> = {}
  for (const [key, value] of Object.entries(init?.params?.header ?? {})) {
    if (value == null) continue
    requestHeaders[key] = String(value)
  }
  if (init?.body !== undefined && !requestHeaders['Content-Type']) {
    requestHeaders['Content-Type'] = 'application/json'
  }
  let headers: Record<string, string>
  try {
    injectNativeSessionHeader(schemaPath, requestHeaders)
    headers = buildSecurityHeaders(method, requestHeaders, {
      csrfToken: resolveCSRFToken(),
    })

    const nativeToken = readNativeAccessToken()
    if (nativeToken && !headers.Authorization) {
      headers.Authorization = `Bearer ${nativeToken}`
    }
  } catch (error) {
    return Promise.reject(error)
  }

  return new Promise((resolve) => {
    let settled = false
    let abortCleanup: (() => void) | null = null
    const finish = (result: ApiCallResult<T>) => {
      if (settled) return
      settled = true
      if (abortCleanup) {
        abortCleanup()
        abortCleanup = null
      }
      resolve(result)
    }

    const requestTask = requestWithPatch({
      data: normalizeRequestBody(init?.body),
      fail: (error) => {
        finish(toTransportErrorResult<T>(error))
      },
      header: headers,
      method: UNI_REQUEST_METHODS[method],
      success: (response) => {
        try {
          const status = typeof response.statusCode === 'number' ? response.statusCode : 0
          if (status >= 200 && status < 300) {
            finish(
              buildSuccessResult<T>(
                status,
                response.data,
                response.header as Record<string, unknown>,
              ),
            )
            return
          }
          finish(
            buildErrorResult<T>(
              status,
              response.data,
              response.header as Record<string, unknown>,
            ),
          )
        } catch (error) {
          finish(toTransportErrorResult<T>(error))
        }
      },
      timeout: 15000,
      url,
      withCredentials: true,
    })

    const signal = init?.signal
    if (!signal) return

    const abort = () => {
      requestTask.abort()
      finish({
        error: new Error('aborted'),
        response: createResponse(0),
      })
    }

    if (signal.aborted) {
      abort()
      return
    }

    signal.addEventListener('abort', abort, { once: true })
    abortCleanup = () => {
      signal.removeEventListener('abort', abort)
    }
  })
}

async function refreshSession() {
  let body: unknown
  if (isNativeRuntime()) {
    const refreshToken = readNativeRefreshToken()
    if (!refreshToken) {
      return { kind: 'unauthorized' } as const
    }
    body = { refreshToken }
  }

  return executeSessionRefresh({
    body,
    onSuccess(payload) {
      if (isNativeRuntime() && payload.accessToken) {
        updateNativeTokensAfterRefresh({
          accessToken: payload.accessToken,
          expiresIn: payload.expiresIn,
          refreshToken: payload.refreshToken,
        })
      }
    },
    request: (init) => performRequest<RefreshSessionData>('POST', AUTH_REFRESH_PATH, init),
  })
}

function hasRefreshSessionHint(): boolean {
  try {
    if (isNativeRuntime()) {
      return readNativeRefreshToken() !== null
    }
    return resolveCSRFToken() !== null
  } catch (_error) { void _error;
    return false
  }
}

export const apiClient: ApiClient = createSessionApiClient(
  {
    refresh: refreshSession,
    request: performRequest,
    shouldRefresh(schemaPath, status) {
      return status === 401 && schemaPath !== AUTH_REFRESH_PATH && hasRefreshSessionHint()
    },
  },
  {
    enableRefresh: true,
    reauthenticateOnUnauthorized: false,
  },
)
