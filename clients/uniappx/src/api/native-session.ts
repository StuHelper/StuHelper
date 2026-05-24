const TOKEN_EXPIRY_BUFFER_MS = 30_000
export const NATIVE_TOKEN_STORAGE_KEY = 'stuhelper:native-tokens'
export const NATIVE_TOKEN_SECURE_STORAGE_SERVICE = 'stuhelper.native-session'

export interface NativeSecureStorageBridge {
  getItem: (service: string, key: string) => string | null | undefined
  removeItem: (service: string, key: string) => void
  setItem: (service: string, key: string, value: string) => void
}

declare global {
  var stuhelperSecureStorage: NativeSecureStorageBridge | undefined
}

class NativeSessionStorageError extends Error {
  cause: unknown

  constructor(action: 'clear' | 'persist' | 'read', cause: unknown) {
    super(`failed to ${action} native session tokens`)
    this.name = 'NativeSessionStorageError'
    this.cause = cause
  }
}

/** 原生 App 本地存储的 token 结构。 */
export interface NativeTokens {
  accessToken: string
  refreshToken: string
  sessionID: string
  /** token 到期时间戳（毫秒）。 */
  expiresAt: number
}

/** 判断当前是否运行在原生 App 环境。 */
export function isNativeRuntime(): boolean {
  return typeof plus !== 'undefined'
}

function secureStorageBridge(action: 'clear' | 'persist' | 'read'): NativeSecureStorageBridge {
  const bridge = globalThis.stuhelperSecureStorage
  if (
    !bridge
    || typeof bridge.getItem !== 'function'
    || typeof bridge.removeItem !== 'function'
    || typeof bridge.setItem !== 'function'
  ) {
    throw new NativeSessionStorageError(action, new Error('native secure storage bridge is unavailable'))
  }
  return bridge
}

function clearLegacyNativeTokenStorage(action: 'clear' | 'persist' | 'read'): void {
  try {
    if (typeof uni !== 'undefined' && typeof uni.removeStorageSync === 'function') {
      uni.removeStorageSync(NATIVE_TOKEN_STORAGE_KEY)
    }
  } catch (error) {
    throw new NativeSessionStorageError(action, error)
  }
}

function parseNativeTokens(raw: unknown): NativeTokens | null {
  if (typeof raw !== 'string' || raw === '') {
    return null
  }

  try {
    const parsed = JSON.parse(raw) as Partial<NativeTokens>
    if (typeof parsed.accessToken !== 'string' || parsed.accessToken === '') return null
    if (typeof parsed.refreshToken !== 'string' || parsed.refreshToken === '') return null
    if (typeof parsed.sessionID !== 'string' || parsed.sessionID === '') return null
    if (typeof parsed.expiresAt !== 'number' || !Number.isFinite(parsed.expiresAt)) return null
    return {
      accessToken: parsed.accessToken,
      refreshToken: parsed.refreshToken,
      sessionID: parsed.sessionID,
      expiresAt: parsed.expiresAt,
    }
  } catch (_error) {
    void _error
    return null
  }
}

function readNativeTokenSnapshot(): NativeTokens | null {
  if (!isNativeRuntime()) return null

  try {
    clearLegacyNativeTokenStorage('read')
    return parseNativeTokens(
      secureStorageBridge('read').getItem(
        NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
        NATIVE_TOKEN_STORAGE_KEY,
      ),
    )
  } catch (error) {
    if (error instanceof NativeSessionStorageError) {
      throw error
    }
    throw new NativeSessionStorageError('read', error)
  }
}

/** 从本地存储读取原生 token。 */
export function readNativeTokens(): NativeTokens | null {
  return readNativeTokenSnapshot()
}

/** 判断 access token 是否已接近过期。 */
export function isNativeTokenExpired(tokens: NativeTokens): boolean {
  return Date.now() >= tokens.expiresAt - TOKEN_EXPIRY_BUFFER_MS
}

/** 读取仍可直接用于请求头的 access token。 */
export function readNativeAccessToken(): string | null {
  const tokens = readNativeTokenSnapshot()
  if (!tokens || isNativeTokenExpired(tokens)) return null
  return tokens.accessToken
}

/** 读取 refresh token。 */
export function readNativeRefreshToken(): string | null {
  return readNativeTokenSnapshot()?.refreshToken ?? null
}

/** 读取 native session ID。 */
export function readNativeSessionID(): string | null {
  return readNativeTokenSnapshot()?.sessionID ?? null
}

/** 持久化原生 token 到本地存储。 */
export function writeNativeTokens(tokens: NativeTokens): void {
  if (!isNativeRuntime()) return

  try {
    clearLegacyNativeTokenStorage('persist')
    secureStorageBridge('persist').setItem(
      NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
      NATIVE_TOKEN_STORAGE_KEY,
      JSON.stringify(tokens),
    )
  } catch (error) {
    if (error instanceof NativeSessionStorageError) {
      throw error
    }
    throw new NativeSessionStorageError('persist', error)
  }
}

/** 清除本地存储的原生 token。 */
export function clearNativeTokens(): void {
  if (!isNativeRuntime()) return

  try {
    clearLegacyNativeTokenStorage('clear')
    secureStorageBridge('clear').removeItem(
      NATIVE_TOKEN_SECURE_STORAGE_SERVICE,
      NATIVE_TOKEN_STORAGE_KEY,
    )
  } catch (error) {
    if (error instanceof NativeSessionStorageError) {
      throw error
    }
    throw new NativeSessionStorageError('clear', error)
  }
}

/** refresh 成功后更新本地 token，但保留原 sessionID。 */
export function updateNativeTokensAfterRefresh(payload: {
  accessToken: string
  refreshToken?: string | null
  expiresIn?: number | null
}): void {
  const existing = readNativeTokenSnapshot()
  if (!existing) return

  writeNativeTokens({
    ...existing,
    accessToken: payload.accessToken,
    expiresAt: typeof payload.expiresIn === 'number'
      ? Date.now() + payload.expiresIn * 1000
      : existing.expiresAt,
    refreshToken: typeof payload.refreshToken === 'string' && payload.refreshToken !== ''
      ? payload.refreshToken
      : existing.refreshToken,
  })
}
