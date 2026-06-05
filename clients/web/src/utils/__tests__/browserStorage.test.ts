import { beforeEach, describe, expect, it } from 'vitest'
import {
  safeGetLocalStorageItem,
  safeGetSessionStorageItem,
  safeRemoveLocalStorageItem,
  safeRemoveSessionStorageItem,
  safeSetLocalStorageItem,
  safeSetSessionStorageItem,
} from '../browserStorage'

class MemoryStorage implements Storage {
  private store = new Map<string, string>()

  get length() {
    return this.store.size
  }

  clear() {
    this.store.clear()
  }

  getItem(key: string) {
    return this.store.get(key) ?? null
  }

  key(index: number) {
    return Array.from(this.store.keys())[index] ?? null
  }

  removeItem(key: string) {
    this.store.delete(key)
  }

  setItem(key: string, value: string) {
    this.store.set(key, value)
  }
}

class ThrowingStorage implements Storage {
  get length() {
    return 0
  }

  clear() {
    throw new Error('storage unavailable')
  }

  getItem() {
    throw new Error('storage unavailable')
  }

  key() {
    throw new Error('storage unavailable')
  }

  removeItem() {
    throw new Error('storage unavailable')
  }

  setItem() {
    throw new Error('storage unavailable')
  }
}

const localStorageMock = new MemoryStorage()
const sessionStorageMock = new MemoryStorage()

describe('browser storage helpers', () => {
  beforeEach(() => {
    Object.defineProperty(globalThis, 'localStorage', {
      value: localStorageMock,
      configurable: true,
    })

    Object.defineProperty(globalThis, 'sessionStorage', {
      value: sessionStorageMock,
      configurable: true,
    })

    localStorageMock.clear()
    sessionStorageMock.clear()
  })

  it('reads, writes, and removes localStorage values', () => {
    expect(safeSetLocalStorageItem('key', 'value')).toBe(true)

    expect(safeGetLocalStorageItem('key')).toBe('value')

    safeRemoveLocalStorageItem('key')
    expect(safeGetLocalStorageItem('key')).toBeNull()
  })

  it('reads, writes, and removes sessionStorage values', () => {
    expect(safeSetSessionStorageItem('key', 'value')).toBe(true)

    expect(safeGetSessionStorageItem('key')).toBe('value')

    safeRemoveSessionStorageItem('key')
    expect(safeGetSessionStorageItem('key')).toBeNull()
  })

  it('degrades when a storage API throws', () => {
    Object.defineProperty(globalThis, 'localStorage', {
      value: new ThrowingStorage(),
      configurable: true,
    })

    expect(safeSetLocalStorageItem('key', 'value')).toBe(false)
    expect(safeGetLocalStorageItem('key')).toBeNull()
    expect(() => safeRemoveLocalStorageItem('key')).not.toThrow()
  })

  it('degrades when a storage global getter throws', () => {
    Object.defineProperty(globalThis, 'sessionStorage', {
      configurable: true,
      get() {
        throw new Error('storage getter unavailable')
      },
    })

    expect(safeSetSessionStorageItem('key', 'value')).toBe(false)
    expect(safeGetSessionStorageItem('key')).toBeNull()
    expect(() => safeRemoveSessionStorageItem('key')).not.toThrow()
  })
})
