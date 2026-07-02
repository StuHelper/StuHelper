/**
 * Auth/session error codes shared with the backend error catalog
 * (`server/internal/pkg/errs/codes.go`). Keep names aligned with the Go
 * constants so the contract stays greppable across the stack.
 */

/** A00101xx — login/session family (e.g. login required, account disabled). */
export const AUTH_SESSION_ERROR_CODE_PREFIX = 'A00101'
/** ErrForbidden — 权限不足。 */
export const AUTH_FORBIDDEN_CODE = 'A0010200'
/** ErrAccessDenied — 访问被拒绝。 */
export const AUTH_ACCESS_DENIED_CODE = 'A0010201'
/** ErrCSRFTokenInvalid — CSRF Token 无效。 */
export const CSRF_TOKEN_INVALID_CODE = 'A0010202'
/** ErrCSRFTokenMissing — CSRF Token 缺失。 */
export const CSRF_TOKEN_MISSING_CODE = 'A0010203'
/** ErrMFARequired — 需要注册 MFA。 */
export const MFA_ENROLLMENT_REQUIRED_CODE = 'A0010204'
/** ErrStepUpRequired — 需要重新完成 MFA 验证。 */
export const STEP_UP_REQUIRED_CODE = 'A0010205'

/** HTTP statuses the backend uses to signal a required step-up (412/428). */
export const STEP_UP_REQUIRED_STATUSES: ReadonlySet<number> = new Set([412, 428])

/** True when the error code belongs to the login/session (401) family. */
export function isAuthSessionErrorCode(code: string | undefined): boolean {
  return code !== undefined && code.startsWith(AUTH_SESSION_ERROR_CODE_PREFIX)
}

export const DEFAULT_HTTP_STATUS_ERROR_CODES = {
  400: 'A0000400',
  401: 'A0010100',
  403: 'A0010200',
  404: 'A0000404',
  409: 'A0000409',
  422: 'A0000422',
  429: 'A0000429',
  500: 'B0000001',
  502: 'C0000001',
  503: 'B0000004',
  504: 'B0000006',
} as const satisfies Record<number, string>

export const DEFAULT_INTERNAL_ERROR_CODE = 'B0000001'

export function defaultHttpStatusErrorCode(status: number): string {
  return DEFAULT_HTTP_STATUS_ERROR_CODES[status as keyof typeof DEFAULT_HTTP_STATUS_ERROR_CODES]
    ?? DEFAULT_INTERNAL_ERROR_CODE
}
