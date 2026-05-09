export const SSO_STATE_STORAGE_KEY = 'stuhelper:sso-state'
const SSO_STATE_MAX_LENGTH = 256
const SSO_STATE_PATTERN = /^[A-Za-z0-9_-]+$/

export type SSOStateValidationResult =
  | { ok: true }
  | { ok: false; reason: 'missing_saved_state' | 'mismatch' }

function normalizeState(value: unknown): string {
  if (typeof value !== 'string') {
    return ''
  }

  const normalized = value.trim()
  if (normalized === '' || normalized.length > SSO_STATE_MAX_LENGTH) {
    return ''
  }
  if (!SSO_STATE_PATTERN.test(normalized)) {
    return ''
  }

  return normalized
}

function getUniRuntime(): {
  getStorageSync?: (key: string) => unknown
  removeStorageSync?: (key: string) => void
  setStorageSync?: (key: string, value: string) => void
} | undefined {
  return (globalThis as typeof globalThis & {
    uni?: {
      getStorageSync?: (key: string) => unknown
      removeStorageSync?: (key: string) => void
      setStorageSync?: (key: string, value: string) => void
    }
  }).uni
}

export function persistSSOState(state: unknown): void {
  const normalized = normalizeState(state)
  if (!normalized) {
    throw new Error('missing native SSO state')
  }

  const runtime = getUniRuntime()
  if (!runtime?.setStorageSync) {
    throw new Error('native storage is unavailable for SSO state')
  }

  try {
    runtime.setStorageSync(SSO_STATE_STORAGE_KEY, normalized)
  } catch (_error) {
    void _error
    throw new Error('failed to persist native SSO state')
  }
}

export function readStoredSSOState(): string | null {
  const runtime = getUniRuntime()
  if (!runtime?.getStorageSync) {
    throw new Error('native storage is unavailable for SSO state')
  }

  try {
    const value = runtime.getStorageSync(SSO_STATE_STORAGE_KEY)
    const normalized = normalizeState(value)
    return normalized || null
  } catch (_error) {
    void _error
    throw new Error('failed to read native SSO state')
  }
}

export function clearStoredSSOState(): void {
  const runtime = getUniRuntime()
  if (!runtime?.removeStorageSync) {
    throw new Error('native storage is unavailable for SSO state')
  }

  try {
    runtime.removeStorageSync(SSO_STATE_STORAGE_KEY)
  } catch (_error) {
    void _error
    throw new Error('failed to clear native SSO state')
  }
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
