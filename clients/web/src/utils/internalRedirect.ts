import { isJoinAdmissionHost, isJoinAdmissionPath } from '@/router/join-domain'

export function readInternalRedirect(
  value: unknown,
  fallback = '/identity',
): string {
  if (typeof value !== 'string') return fallback
  const candidate = value.trim()
  if (!candidate.startsWith('/') || candidate.startsWith('//')) return fallback
  if (candidate.includes('\\')) return fallback
  return candidate
}

/**
 * Account-center flows normally return to a same-origin path. The only
 * cross-origin exception is the dedicated join surface, and even there only
 * its two current business routes are accepted. This keeps student/QQ/phone
 * flows decoupled without introducing an open redirect.
 */
export function readAccountFlowRedirect(
  value: unknown,
  fallback = '/identity',
): string {
  if (typeof value !== 'string') return fallback
  const candidate = value.trim()
  if (candidate.startsWith('/') && !candidate.startsWith('//') && !candidate.includes('\\')) {
    return candidate
  }
  if (typeof window === 'undefined') return fallback
  try {
    const parsed = new URL(candidate)
    const protocolAllowed = parsed.protocol === 'https:' ||
      (parsed.protocol === 'http:' && parsed.hostname.toLowerCase() === 'join.localhost')
    if (
      protocolAllowed &&
      parsed.username === '' &&
      parsed.password === '' &&
      isJoinAdmissionHost(parsed.hostname) &&
      isJoinAdmissionPath(parsed.pathname)
    ) {
      parsed.hash = ''
      return parsed.toString()
    }
  } catch {
    return fallback
  }
  return fallback
}

export function isCrossOriginAccountFlowRedirect(target: string): boolean {
  if (typeof window === 'undefined') return false
  try {
    return new URL(target, window.location.origin).origin !== window.location.origin
  } catch {
    return false
  }
}
