import { ApiError } from '@/api/errors'

export type AdmissionMappedState = 'qqMismatch' | 'expired' | 'error'

const ADMISSION_PATH_PATTERN = /^\/verify\/[^/]+$/
const TERMINAL_ADMISSION_ERROR_CODES = new Set([
  'admission.token_consumed',
  'admission.token_expired',
  'admission.token_not_found',
  'admission.session_not_found',
])
const EXPIRED_ADMISSION_SESSION_ERROR_CODES = new Set([
  'admission.token_expired',
  'admission.token_not_found',
  'admission.session_not_found',
])

export function buildAdmissionReturnURL(
  pathWithQuery: string,
  origin = window.location.origin,
): string {
  const url = new URL(pathWithQuery, origin)

  if (url.origin !== origin) {
    throw new Error('Admission return URL must stay on the current origin')
  }
  if (!ADMISSION_PATH_PATTERN.test(url.pathname)) {
    throw new Error('Admission return URL must target /verify/:code')
  }

  return url.toString()
}

export function mapAdmissionApiError(error: unknown): AdmissionMappedState {
  const code = readAdmissionErrorCode(error)

  if (code === 'admission.qq_mismatch') {
    return 'qqMismatch'
  }
  if (code && TERMINAL_ADMISSION_ERROR_CODES.has(code)) {
    return 'expired'
  }

  return 'error'
}

export function isAdmissionTokenConsumedError(error: unknown): boolean {
  return readAdmissionErrorCode(error) === 'admission.token_consumed'
}

export function isAdmissionSessionExpiredError(error: unknown): boolean {
  const code = readAdmissionErrorCode(error)
  return Boolean(code && EXPIRED_ADMISSION_SESSION_ERROR_CODES.has(code))
}

function readAdmissionErrorCode(error: unknown): string | undefined {
  if (error instanceof ApiError) {
    return error.code
  }
  if (!error || typeof error !== 'object') {
    return undefined
  }
  const code = (error as { code?: unknown }).code
  if (typeof code === 'string' && code !== '') {
    return code
  }
  const nested = (error as { error?: unknown }).error
  if (nested && typeof nested === 'object') {
    const nestedCode = (nested as { code?: unknown }).code
    if (typeof nestedCode === 'string' && nestedCode !== '') {
      return nestedCode
    }
  }
  return undefined
}
