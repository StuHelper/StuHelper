import type { ApiClient } from '@stuhelper/shared/api'
import type { ApiCallResult } from './result'

type HttpMethod = 'GET' | 'POST' | 'PUT' | 'PATCH' | 'DELETE'
type RequestBody = UniNamespace.RequestOptions['data']

type RequestInitShape = {
  params?: {
    path?: Record<string, unknown>
    query?: Record<string, unknown>
    header?: Record<string, unknown>
  }
  body?: unknown
  signal?: AbortSignal
}

type PerformRequestOptions = {
  skipRefresh?: boolean
}

const API_BASE_URL = import.meta.env.VITE_API_URL ?? ''
const API_VERSION_PREFIX = '/api/v1'
const CSRF_COOKIE_NAME = 'csrf_token'
const CSRF_HEADER_NAME = 'X-CSRF-Token'
const CSRF_STORAGE_KEY = 'stuhelper:csrf-token'
const H5 = typeof window !== 'undefined' && typeof window.location?.origin === 'string'
const NATIVE_TOKEN_STORAGE_KEY = 'stuhelper:native-tokens'
const AUTH_REFRESH_PATH = '/api/v1/auth/refresh'
const AUTO_REFRESH_EXCLUDED_PATHS = new Set([
  '/api/v1/auth/login',
  '/api/v1/auth/signup',
  AUTH_REFRESH_PATH,
  '/api/v1/auth/logout',
  '/api/v1/auth/logout-all',
  '/api/v1/auth/phone/request-otp',
  '/api/v1/auth/phone/verify-otp',
  '/api/v1/auth/exchange-native',
])

let refreshPromise: Promise<boolean> | null = null

/** 判断当前是否运行在原生 App 环境 */
function isNativeRuntime(): boolean {
  return typeof plus !== 'undefined'
}

/** 从本地存储读取原生 App 的 access token */
function readNativeAccessToken(): string | null {
  if (!isNativeRuntime()) return null
  try {
    const raw = uni.getStorageSync(NATIVE_TOKEN_STORAGE_KEY)
    if (!raw || typeof raw !== 'string') return null
    const parsed = JSON.parse(raw) as { accessToken?: string; expiresAt?: number }
    if (!parsed.accessToken) return null
    // 预留 30s 缓冲判断过期
    if (typeof parsed.expiresAt === 'number' && Date.now() >= parsed.expiresAt - 30_000) return null
    return parsed.accessToken
  } catch {
    return null
  }
}

function serializePath(schemaPath: string, pathParams?: Record<string, unknown>): string {
  return schemaPath.replace(/\{([^}]+)\}/g, (_match, key) => {
    const value = pathParams?.[key]
    return encodeURIComponent(value == null ? '' : String(value))
  })
}

function appendQuery(path: string, query?: Record<string, unknown>): string {
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

function normalizeSchemaPath(schemaPath: string): string {
  const trimmedBase = API_BASE_URL.replace(/\/$/, '')
  if (trimmedBase.endsWith(API_VERSION_PREFIX) && schemaPath.startsWith(`${API_VERSION_PREFIX}/`)) {
    return schemaPath.slice(API_VERSION_PREFIX.length)
  }
  return schemaPath
}

function stripRelativeBasePrefix(baseUrl: string, path: string): string {
  if (!baseUrl) return path

  if (path === baseUrl) {
    return '/'
  }

  if (!path.startsWith(baseUrl)) {
    return path
  }

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
    } catch {
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
      const value = localStorage.getItem(CSRF_STORAGE_KEY)
      return value || null
    }
  } catch {
    // ignore storage access failures
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
  } catch {
    // ignore storage access failures
  }
}

function readHeader(
  headers: Record<string, unknown> | undefined,
  name: string
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

function resolveCSRFToken(): string | null {
  return readCookie(CSRF_COOKIE_NAME) ?? readStoredCSRFToken()
}

function needsCSRF(method: string): boolean {
  return !['GET', 'HEAD', 'OPTIONS'].includes(method.toUpperCase())
}

function resolveRequestURL(
  schemaPath: string,
  pathParams?: Record<string, unknown>,
  query?: Record<string, unknown>
): string {
  const normalizedPath = appendQuery(
    serializePath(normalizeSchemaPath(schemaPath), pathParams),
    query
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
  } catch {
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

function createResponse(status: number, headers?: Record<string, unknown>): Response {
  const normalizedHeaders = Object.fromEntries(
    Object.entries(headers ?? {}).map(([key, value]) => [key.toLowerCase(), String(value)])
  )

  return {
    ok: status >= 200 && status < 300,
    status,
    headers: {
      get(name: string) {
        return normalizedHeaders[name.toLowerCase()] ?? null
      },
    },
  } as unknown as Response
}

function buildSuccessResult<T>(
  status: number,
  data: unknown,
  headers?: Record<string, unknown>
): ApiCallResult<T> {
  persistCSRFToken(headers)
  return {
    data: normalizeBody(data) as ApiCallResult<T>['data'],
    response: createResponse(status, headers),
  }
}

function buildErrorResult<T>(
  status: number,
  data: unknown,
  headers?: Record<string, unknown>
): ApiCallResult<T> {
  persistCSRFToken(headers)
  return {
    error: normalizeBody(data),
    response: createResponse(status, headers),
  }
}

function shouldAutoRefresh(schemaPath: string, status: number, options?: PerformRequestOptions): boolean {
  return status === 401 && !options?.skipRefresh && !AUTO_REFRESH_EXCLUDED_PATHS.has(schemaPath)
}

function toTransportErrorResult<T>(error: unknown): ApiCallResult<T> {
  return {
    error,
    response: createResponse(0),
  }
}

async function refreshSession(): Promise<boolean> {
  if (!refreshPromise) {
    refreshPromise = performRequest<unknown>('POST', AUTH_REFRESH_PATH, undefined, { skipRefresh: true })
      .then((result) => {
        const status = result.response?.status ?? 0
        return !result.error && status >= 200 && status < 300
      })
      .catch(() => false)
      .finally(() => {
        refreshPromise = null
      })
  }
  return refreshPromise
}

function performRequest<T>(
  method: HttpMethod,
  schemaPath: string,
  init?: unknown,
  options?: PerformRequestOptions,
): Promise<ApiCallResult<T>> {
  const requestInit = (init ?? {}) as RequestInitShape
  let url: string
  try {
    url = resolveRequestURL(
      schemaPath,
      requestInit.params?.path,
      requestInit.params?.query
    )
  } catch (error) {
    return Promise.reject(error)
  }
  const headers: Record<string, string> = {}

  for (const [key, value] of Object.entries(requestInit.params?.header ?? {})) {
    if (value == null) continue
    headers[key] = String(value)
  }

  if (requestInit.body !== undefined && !headers['Content-Type']) {
    headers['Content-Type'] = 'application/json'
  }

  if (needsCSRF(method)) {
    const csrfToken = resolveCSRFToken()
    if (csrfToken) {
      headers[CSRF_HEADER_NAME] = csrfToken
    }
  }

  // 原生 App：从本地存储注入 Bearer token（替代 cookie）
  const nativeToken = readNativeAccessToken()
  if (nativeToken && !headers['Authorization']) {
    headers['Authorization'] = `Bearer ${nativeToken}`
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

    const requestTask = uni.request({
      url,
      method: method as unknown as UniNamespace.RequestOptions['method'],
      data: normalizeRequestBody(requestInit.body),
      header: headers,
      timeout: 15000,
      withCredentials: true,
      success: (response) => {
        const status = typeof response.statusCode === 'number' ? response.statusCode : 0
        if (status >= 200 && status < 300) {
          finish(
            buildSuccessResult<T>(
              status,
              response.data,
              response.header as Record<string, unknown>
            )
          )
          return
        }
        const errorResult = buildErrorResult<T>(
          status,
          response.data,
          response.header as Record<string, unknown>
        )
        if (shouldAutoRefresh(schemaPath, status, options)) {
          void refreshSession().then((refreshed) => {
            if (!refreshed) {
              finish(errorResult)
              return
            }
            void performRequest<T>(method, schemaPath, init, { skipRefresh: true })
              .then(finish)
              .catch((error) => finish(toTransportErrorResult<T>(error)))
    }) as unknown as UniNamespace.RequestTask
          return
        }
        finish(errorResult)
      },
      fail: (error) => {
        finish(toTransportErrorResult<T>(error))
      },
    }) as UniNamespace.RequestTask

    const signal = requestInit.signal
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

const GET = ((schemaPath: string, init?: unknown) => performRequest('GET', schemaPath, init)) as ApiClient['GET']
const POST = ((schemaPath: string, init?: unknown) => performRequest('POST', schemaPath, init)) as ApiClient['POST']
const PUT = ((schemaPath: string, init?: unknown) => performRequest('PUT', schemaPath, init)) as ApiClient['PUT']
const PATCH = ((schemaPath: string, init?: unknown) => performRequest('PATCH', schemaPath, init)) as ApiClient['PATCH']
const DELETE = ((schemaPath: string, init?: unknown) => performRequest('DELETE', schemaPath, init)) as ApiClient['DELETE']

export const apiClient: ApiClient = {
  GET,
  POST,
  PUT,
  PATCH,
  DELETE,
}
