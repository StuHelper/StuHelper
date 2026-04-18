export const SSO_STATE_STORAGE_KEY = 'stuhelper:sso-state'

export type SSOStateValidationResult =
  | { ok: true }
  | { ok: false; reason: 'missing_saved_state' | 'mismatch' }

function normalizeState(value: unknown): string {
  return typeof value === 'string' ? value.trim() : ''
}

export function validateStoredSSOState(savedState: unknown, callbackState: unknown): SSOStateValidationResult {
  const expected = normalizeState(savedState)
  const actual = normalizeState(callbackState)

  if (!expected || !actual) {
    return { ok: false, reason: 'missing_saved_state' }
  }
  if (expected !== actual) {
    return { ok: false, reason: 'mismatch' }
  }

  return { ok: true }
}
