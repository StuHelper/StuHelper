import { ApiError } from '@/api/errors'
import {
  safeGetSessionStorageItem,
  safeRemoveSessionStorageItem,
  safeSetSessionStorageItem,
} from '@/utils/browserStorage'

export type AdmissionMappedState = 'qqMismatch' | 'invalid' | 'expired' | 'error'

const ADMISSION_PATH_PATTERN = /^\/verify\/[^/]+\/?$/
const TERMINAL_ADMISSION_ERROR_CODES = new Set([
  'admission.token_consumed',
  'admission.token_expired',
])
const INVALID_ADMISSION_LINK_ERROR_CODES = new Set([
  'admission.token_not_found',
  'admission.session_not_found',
])
const EXPIRED_ADMISSION_SESSION_ERROR_CODES = new Set([
  'admission.token_expired',
  'admission.token_not_found',
  'admission.session_not_found',
])
const LINKED_ADMISSION_SESSION_KEY_PREFIX = 'admission_linked_session:'

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
  url.search = ''
  url.hash = ''

  return url.toString()
}

export function mapAdmissionApiError(error: unknown): AdmissionMappedState {
  const code = readAdmissionErrorCode(error)

  if (code === 'admission.qq_mismatch' || isQQBindingConflictError(error)) {
    return 'qqMismatch'
  }
  if (code && INVALID_ADMISSION_LINK_ERROR_CODES.has(code)) {
    return 'invalid'
  }
  if (code && TERMINAL_ADMISSION_ERROR_CODES.has(code)) {
    return 'expired'
  }

  return 'error'
}

function isQQBindingConflictError(error: unknown): boolean {
  const message = readAdmissionErrorMessage(error)
  return /qq binding conflict/i.test(message ?? '')
}

export function isAdmissionTokenConsumedError(error: unknown): boolean {
  return readAdmissionErrorCode(error) === 'admission.token_consumed'
}

export function isAdmissionSessionExpiredError(error: unknown): boolean {
  const code = readAdmissionErrorCode(error)
  return Boolean(code && EXPIRED_ADMISSION_SESSION_ERROR_CODES.has(code))
}

export function isFreshmanCameraHandoffLockedError(error: unknown): boolean {
  const message = readAdmissionErrorMessage(error)
  return /admission camera handoff locked|camera handoff locked|handoff locked/i.test(
    message ?? '',
  )
}

export function rememberLinkedAdmissionSession(
  token: string,
  admissionSessionID: string,
): void {
  if (!token || !admissionSessionID) return
  safeSetSessionStorageItem(linkedAdmissionSessionKey(token), admissionSessionID)
}

export function readLinkedAdmissionSessionID(token: string): string | null {
  if (!token) return null
  return safeGetSessionStorageItem(linkedAdmissionSessionKey(token))
}

export function forgetLinkedAdmissionSession(token: string): void {
  if (!token) return
  safeRemoveSessionStorageItem(linkedAdmissionSessionKey(token))
}

function linkedAdmissionSessionKey(token: string): string {
  return `${LINKED_ADMISSION_SESSION_KEY_PREFIX}${token}`
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

function readAdmissionErrorMessage(error: unknown): string | undefined {
  if (error instanceof Error && error.message) {
    return error.message
  }
  if (!error || typeof error !== 'object') {
    return undefined
  }
  const message = (error as { message?: unknown }).message
  if (typeof message === 'string' && message !== '') {
    return message
  }
  const nested = (error as { error?: unknown }).error
  if (nested && typeof nested === 'object') {
    const nestedMessage = (nested as { message?: unknown }).message
    if (typeof nestedMessage === 'string' && nestedMessage !== '') {
      return nestedMessage
    }
  }
  return undefined
}
