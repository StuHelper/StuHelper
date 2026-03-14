import { createApiClient } from '@stuhelper/shared/api'

const API_BASE_URL = import.meta.env.VITE_API_URL || '/api'
const CSRF_COOKIE_NAME = 'csrf_token'
const CSRF_HEADER_NAME = 'X-CSRF-Token'
const AUTH_REFRESH_PATH = '/api/v1/auth/refresh'
const AUTH_REFRESH_TIMEOUT_MS = 10_000

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

function needsCSRF(method: string): boolean {
  return !['GET', 'HEAD', 'OPTIONS'].includes(method.toUpperCase())
}

function withBrowserSecurity(request: Request): Request {
  const headers = new Headers(request.headers)
  if (needsCSRF(request.method)) {
    const csrfToken = readCookie(CSRF_COOKIE_NAME)
    if (csrfToken) {
      headers.set(CSRF_HEADER_NAME, csrfToken)
    }
  }
  return new Request(request, { credentials: 'include', headers })
}

function isRefreshRequest(request: Request): boolean {
  const url = new URL(request.url, window.location.origin)
  return url.pathname === AUTH_REFRESH_PATH
}

async function fetchWithTimeout(request: Request, timeoutMs: number): Promise<Response> {
  const controller = new AbortController()
  const timeoutId = window.setTimeout(() => controller.abort(), timeoutMs)
  try {
    return await window.fetch(new Request(request, { signal: controller.signal }))
  } finally {
    window.clearTimeout(timeoutId)
  }
}

let refreshPromise: Promise<boolean> | null = null

async function refreshAccessToken(): Promise<boolean> {
  if (refreshPromise) return refreshPromise

  refreshPromise = (async () => {
    try {
      const headers = new Headers()
      const csrfToken = readCookie(CSRF_COOKIE_NAME)
      if (csrfToken) headers.set(CSRF_HEADER_NAME, csrfToken)

      const response = await fetchWithTimeout(
        new Request(new URL(AUTH_REFRESH_PATH, window.location.origin), {
          method: 'POST',
          credentials: 'include',
          headers,
        }),
        AUTH_REFRESH_TIMEOUT_MS,
      )
      return response.ok
    } catch {
      return false
    } finally {
      refreshPromise = null
    }
  })()

  return refreshPromise
}

async function authenticatedFetch(input: Request): Promise<Response> {
  const initialRequest = withBrowserSecurity(input.clone())
  const response = await (isRefreshRequest(initialRequest)
    ? fetchWithTimeout(initialRequest, AUTH_REFRESH_TIMEOUT_MS)
    : window.fetch(initialRequest))

  if (response.status !== 401 || isRefreshRequest(initialRequest)) {
    return response
  }

  const refreshed = await refreshAccessToken()
  if (!refreshed) return response

  const retryRequest = withBrowserSecurity(input.clone())
  return window.fetch(retryRequest)
}

export const apiClient = createApiClient({
  baseUrl: API_BASE_URL,
  credentials: 'include',
  fetch: authenticatedFetch,
})

apiClient.use({
  async onResponse({ response }) {
    if (response.ok) return response
    const payload = await response.clone().json().catch(() => null)
    const errorData =
      payload && typeof payload === 'object' && 'error' in payload
        ? (payload as Record<string, unknown>).error
        : null
    const msg =
      errorData && typeof errorData === 'object' && 'message' in (errorData as Record<string, unknown>)
        ? String((errorData as Record<string, unknown>).message)
        : response.statusText || 'API request failed'
    const err = new Error(msg) as Error & { status: number; code?: string }
    err.status = response.status
    if (errorData && typeof errorData === 'object' && 'code' in (errorData as Record<string, unknown>)) {
      err.code = String((errorData as Record<string, unknown>).code)
    }
    throw err
  },
})
