import { ApiError } from '@/api/errors'

export type AdmissionMappedState = 'qqMismatch' | 'expired' | 'error'

const ADMISSION_PATH_PATTERN = /^\/verify\/[^/]+$/

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
  if (!(error instanceof ApiError)) {
    return 'error'
  }

  if (error.code === 'admission.qq_mismatch') {
    return 'qqMismatch'
  }
  if (
    error.code === 'admission.token_consumed' ||
    error.code === 'admission.token_expired' ||
    error.code === 'admission.token_not_found' ||
    error.code === 'admission.session_not_found'
  ) {
    return 'expired'
  }

  return 'error'
}

export function isAdmissionTokenConsumedError(error: unknown): boolean {
  return error instanceof ApiError && error.code === 'admission.token_consumed'
}

export function isAdmissionSessionExpiredError(error: unknown): boolean {
  return (
    error instanceof ApiError &&
    (
      error.code === 'admission.token_consumed' ||
      error.code === 'admission.token_expired' ||
      error.code === 'admission.token_not_found' ||
      error.code === 'admission.session_not_found'
    )
  )
}
