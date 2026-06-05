type BrowserStorageKind = 'local' | 'session'

function getBrowserStorage(kind: BrowserStorageKind): Storage | null {
  try {
    const storage = kind === 'local' ? globalThis.localStorage : globalThis.sessionStorage
    return storage ?? null
  } catch (_error) {
    void _error
    return null
  }
}

function safeGetStorageItem(kind: BrowserStorageKind, key: string): string | null {
  const storage = getBrowserStorage(kind)
  if (!storage) return null

  try {
    return storage.getItem(key)
  } catch (_error) {
    void _error
    return null
  }
}

function safeSetStorageItem(kind: BrowserStorageKind, key: string, value: string): boolean {
  const storage = getBrowserStorage(kind)
  if (!storage) return false

  try {
    storage.setItem(key, value)
    return true
  } catch (_error) {
    void _error
    return false
  }
}

function safeRemoveStorageItem(kind: BrowserStorageKind, key: string): void {
  const storage = getBrowserStorage(kind)
  if (!storage) return

  try {
    storage.removeItem(key)
  } catch (_error) {
    void _error
  }
}

export function safeGetLocalStorageItem(key: string): string | null {
  return safeGetStorageItem('local', key)
}

export function safeSetLocalStorageItem(key: string, value: string): boolean {
  return safeSetStorageItem('local', key, value)
}

export function safeRemoveLocalStorageItem(key: string): void {
  safeRemoveStorageItem('local', key)
}

export function safeGetSessionStorageItem(key: string): string | null {
  return safeGetStorageItem('session', key)
}

export function safeSetSessionStorageItem(key: string, value: string): boolean {
  return safeSetStorageItem('session', key, value)
}

export function safeRemoveSessionStorageItem(key: string): void {
  safeRemoveStorageItem('session', key)
}
