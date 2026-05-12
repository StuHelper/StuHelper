import { CSRF_COOKIE_NAME } from '@stuhelper/shared/api'
import { tokenExpiry } from './auth'

export function readCookie(name: string): string | null {
  const cookies = document.cookie ? document.cookie.split(';') : []
  const target = `${encodeURIComponent(name)}=`
  for (const raw of cookies) {
    const cookie = raw.trim()
    if (!cookie.startsWith(target)) continue
    try {
      return decodeURIComponent(cookie.slice(target.length))
    } catch (_error) { void _error;
      return null
    }
  }
  return null
}

export function hasStoredSessionHint(): boolean {
  return readCookie(CSRF_COOKIE_NAME) !== null || tokenExpiry.get() !== null
}
