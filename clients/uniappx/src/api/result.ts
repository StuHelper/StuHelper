import { useAuthStore } from '@/stores/auth'
import { translate } from '@/i18n'
import { parseApiError, extractApiErrorMessage } from '@stuhelper/shared/api'

export interface StructuredApiError {
  code?: string
  details?: unknown
  message?: string
}

export interface ApiEnvelope<T> {
  success?: boolean
  data?: T
  error?: string | StructuredApiError
  message?: string
  code?: string
}

export interface ApiCallResult<T = unknown> {
  data?: ApiEnvelope<T>
  error?: unknown
  response?: {
    status?: number
  }
}

function readStatus(result: ApiCallResult<unknown>): number | undefined {
  return result.response?.status
}

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
  // 从 envelope 提取（使用 shared 统一解析）
  if (result.data) {
    const msg = extractApiErrorMessage(result.data, '')
    if (msg) return msg
  }

  // runtime error
  if (result.error) {
    const msg = extractApiErrorMessage(result.error, '')
    if (msg) return msg

    // 嵌套 error 对象
    if (typeof result.error === 'object') {
      const nested = (result.error as Record<string, unknown>).error
      if (nested) {
        const nestedMsg = extractApiErrorMessage({ error: nested }, '')
        if (nestedMsg) return nestedMsg
      }
    }
  }

  const status = readStatus(result)
  if (status === 401) return translate('common.sessionExpired')
  return translate('common.retryLater')
}

function handleUnauthorized(status?: number) {
  if (status === 401) {
    useAuthStore().clearSession()
  }
}

export function unwrapData<T>(result: ApiCallResult<T>): T {
  const status = readStatus(result)
  const payload = result.data

  if (payload && typeof payload === 'object' && 'data' in payload && payload.data !== undefined) {
    return payload.data as T
  }

  handleUnauthorized(status)
  throw new Error(extractErrorMessage(result))
}

export function unwrapOptionalData<T>(result: ApiCallResult<T>): T | null {
  const status = readStatus(result)
  const payload = result.data

  if (payload && typeof payload === 'object' && 'data' in payload) {
    return (payload.data ?? null) as T | null
  }

  if (status === 401 || status === 404) {
    handleUnauthorized(status)
    return null
  }

  throw new Error(extractErrorMessage(result))
}

export function assertMutationSuccess(result: ApiCallResult<unknown>): void {
  const status = readStatus(result)
  if (status === 401) {
    handleUnauthorized(status)
  }

  if (status !== undefined && status >= 400) {
    throw new Error(extractErrorMessage(result))
  }

  if (typeof result.error !== 'undefined') {
    throw new Error(extractErrorMessage(result))
  }

  const payload = result.data
  if (payload && typeof payload === 'object' && 'success' in payload && payload.success === false) {
    throw new Error(extractErrorMessage(result))
  }
}

export function unwrapListData<T>(result: ApiCallResult<{ list?: T[]; total?: number }>): {
  list: T[]
  total: number
} {
  const payload = unwrapData(result)
  return {
    list: Array.isArray(payload?.list) ? payload.list : [],
    total: typeof payload?.total === 'number' ? payload.total : 0,
  }
}
