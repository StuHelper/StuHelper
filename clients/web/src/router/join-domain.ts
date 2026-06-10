const JOIN_ADMISSION_HOSTS = new Set([
  'join.stuhelper.com',
  'join.localhost',
])

const JOIN_ADMISSION_PATH_PATTERNS = [
  /^\/start\/?$/,
  /^\/verify\/[^/]+\/?$/,
  /^\/admission\/freshman\/camera\/[^/]+\/?$/,
]

export function currentHostname(): string {
  return typeof window === 'undefined' ? '' : window.location.hostname
}

export function isJoinAdmissionHost(hostname: string | undefined): boolean {
  return JOIN_ADMISSION_HOSTS.has((hostname ?? '').trim().toLowerCase())
}

export function isJoinAdmissionPath(path: string): boolean {
  return JOIN_ADMISSION_PATH_PATTERNS.some((pattern) => pattern.test(path))
}

export function shouldBlockJoinHostRoute(
  hostname: string | undefined,
  path: string,
): boolean {
  return isJoinAdmissionHost(hostname) && !isJoinAdmissionPath(path)
}

export function shouldBlockAdmissionPathOutsideJoinHost(
  hostname: string | undefined,
  path: string,
): boolean {
  return isJoinAdmissionPath(path) && !isJoinAdmissionHost(hostname)
}
