import { useAuthStore } from '@/stores/auth'
import { translate } from '@/i18n'

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

function normalizeStructuredError(value: unknown): StructuredApiError | null {
  if (typeof value === 'string') {
    return value ? { message: value } : null
  }

  if (!value || typeof value !== 'object') {
    return null
  }

  const record = value as Record<string, unknown>
  const code = typeof record.code === 'string' ? record.code : undefined
  const message = typeof record.message === 'string' ? record.message : undefined
  const details = 'details' in record ? record.details : undefined

  if (!code && !message && typeof details === 'undefined') {
    return null
  }

  return { code, message, details }
}

export function extractErrorMessage(result: ApiCallResult<unknown>): string {
  const envelope = result.data
  const envelopeError = normalizeStructuredError(envelope?.error)
  if (envelopeError?.message) return envelopeError.message
  if (typeof envelope?.error === 'string' && envelope.error) return envelope.error
  if (envelope?.message) return envelope.message

  const structuredError = normalizeStructuredError(result.error)
  if (structuredError?.message) return structuredError.message
  if (structuredError?.code) return structuredError.code

  if (result.error && typeof result.error === 'object') {
    const error = result.error as Record<string, unknown>
    const nestedError = normalizeStructuredError(error.error)
    if (nestedError?.message) return nestedError.message
    if (nestedError?.code) return nestedError.code
    const nestedMessage = error.message
    if (typeof nestedMessage === 'string' && nestedMessage) return nestedMessage
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
