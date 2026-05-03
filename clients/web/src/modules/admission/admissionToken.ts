import { ApiError } from '@/api/errors'

export type AdmissionMappedState = 'qqMismatch' | 'expired' | 'error'

const ADMISSION_PATH_PATTERN = /^\/admission\/a\/[^/]+$/

export function buildAdmissionReturnURL(
  pathWithQuery: string,
  origin = window.location.origin,
): string {
  const url = new URL(pathWithQuery, origin)

  if (url.origin !== origin) {
    throw new Error('Admission return URL must stay on the current origin')
  }
  if (!ADMISSION_PATH_PATTERN.test(url.pathname)) {
    throw new Error('Admission return URL must target /admission/a/:code')
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
    error.code === 'admission.token_expired'
  ) {
    return 'expired'
  }

  return 'error'
}
