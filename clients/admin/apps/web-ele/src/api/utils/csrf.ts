/**
 * CSRF cookie utilities shared across request pipelines.
 */

export const CSRF_COOKIE_NAME = 'csrf_token';
export const CSRF_HEADER_NAME = 'X-CSRF-Token';

export function readCookie(name: string): null | string {
  if (typeof document === 'undefined') {
    return null;
  }

  const target = `${encodeURIComponent(name)}=`;
  const cookies = document.cookie ? document.cookie.split(';') : [];
  for (const rawCookie of cookies) {
    const cookie = rawCookie.trim();
    if (!cookie.startsWith(target)) {
      continue;
    }

    try {
      return decodeURIComponent(cookie.slice(target.length));
    } catch (_error) {
      void _error;
      return null;
    }
  }

  return null;
}
