export const SSO_STATE_STORAGE_KEY = 'stuhelper:sso-state'
export const SSO_REDIRECT_STORAGE_KEY = 'stuhelper:sso-redirect'
export const DEFAULT_SSO_REDIRECT = '/pages/user/index'
const SSO_STATE_MAX_LENGTH = 256
const SSO_STATE_PATTERN = /^[A-Za-z0-9_-]+$/
const SSO_REDIRECT_MAX_LENGTH = 2048

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

function containsControlCharacter(value: string): boolean {
  for (const character of value) {
    const codePoint = character.codePointAt(0) ?? 0
    if (codePoint <= 31 || codePoint === 127) {
      return true
    }
  }
  return false
}

function normalizedRedirect(value: unknown): string | null {
  if (typeof value !== 'string') {
    return null
  }

  let decoded = value.trim()
  if (decoded === '' || decoded.length > SSO_REDIRECT_MAX_LENGTH) {
    return null
  }

  for (let attempt = 0; attempt < 2; attempt += 1) {
    let next: string
    try {
      next = decodeURIComponent(decoded)
    } catch (_error) {
      void _error
      return null
    }
    if (next === decoded) {
      break
    }
    decoded = next
  }

  // Reject values that require a third decode. This prevents a multiply encoded
  // protocol-relative URL from crossing the internal-page boundary later.
  try {
    if (decodeURIComponent(decoded) !== decoded) {
      return null
    }
  } catch (_error) {
    void _error
    return null
  }

  if (
    decoded.length > SSO_REDIRECT_MAX_LENGTH
    || !decoded.startsWith('/pages/')
    || decoded.startsWith('//')
    || containsControlCharacter(decoded)
  ) {
    return null
  }
  return decoded
}

export function normalizeRedirectOption(value: unknown): string {
  return normalizedRedirect(value) ?? DEFAULT_SSO_REDIRECT
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

// The redirect is a best-effort UX hint, not an authentication secret. Storage
// failures must never prevent or invalidate the native SSO flow.
export function persistSSORedirect(path: unknown): void {
  const runtime = getUniRuntime()
  if (!runtime?.setStorageSync) {
    return
  }
  try {
    runtime.setStorageSync(SSO_REDIRECT_STORAGE_KEY, normalizeRedirectOption(path))
  } catch (_error) {
    void _error
  }
}

export function readStoredSSORedirect(): string | null {
  const runtime = getUniRuntime()
  if (!runtime?.getStorageSync) {
    return null
  }
  try {
    return normalizedRedirect(runtime.getStorageSync(SSO_REDIRECT_STORAGE_KEY))
  } catch (_error) {
    void _error
    return null
  }
}

export function clearStoredSSORedirect(): void {
  const runtime = getUniRuntime()
  if (!runtime?.removeStorageSync) {
    return
  }
  try {
    runtime.removeStorageSync(SSO_REDIRECT_STORAGE_KEY)
  } catch (_error) {
    void _error
  }
}

export function consumeStoredSSORedirect(): string {
  const target = normalizeRedirectOption(readStoredSSORedirect())
  clearStoredSSORedirect()
  return target
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
