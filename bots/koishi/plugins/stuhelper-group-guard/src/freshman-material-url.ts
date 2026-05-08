import { isIP } from 'node:net'

const FRESHMAN_MATERIAL_HOSTS_ENV = 'STUHELPER_FRESHMAN_MATERIAL_HOSTS'
const HTTPS_PROTOCOL = 'https:'
const IPV4_LOCAL_PREFIXES = ['10.', '127.', '169.254.', '192.168.']
const IPV6_LOCAL_HOSTS = new Set(['::1', '::'])

export function validateFreshmanMaterialURL(value: string) {
  const url = parseFreshmanMaterialURL(value)
  assertFreshmanMaterialHostAllowed(url)
  return url.href
}

function parseFreshmanMaterialURL(value: string) {
  let url: URL
  try {
    url = new URL(value)
  } catch {
    throw new Error('freshman material URL must be absolute')
  }
  if (url.protocol !== HTTPS_PROTOCOL) {
    throw new Error('freshman material URL must use https')
  }
  if (url.username || url.password) {
    throw new Error('freshman material URL must not include credentials')
  }
  return url
}

function assertFreshmanMaterialHostAllowed(url: URL) {
  const host = normalizeHost(url.hostname)
  if (isLocalHostname(host) || isPrivateIP(host)) {
    throw new Error(`freshman material URL host is not public: ${host}`)
  }
  const allowedHosts = readAllowedHosts()
  if (!allowedHosts.has(host)) {
    throw new Error(`freshman material URL host is not allowlisted: ${host}`)
  }
}

function readAllowedHosts() {
  const raw = process.env[FRESHMAN_MATERIAL_HOSTS_ENV]
  if (!raw?.trim()) {
    throw new Error(`${FRESHMAN_MATERIAL_HOSTS_ENV} is required for freshman material forwarding`)
  }
  const hosts = raw.split(',').map((item) => normalizeHost(item)).filter(Boolean)
  if (!hosts.length) {
    throw new Error(`${FRESHMAN_MATERIAL_HOSTS_ENV} must contain at least one host`)
  }
  return new Set(hosts)
}

function normalizeHost(value: string) {
  return value.trim().toLowerCase().replace(/^\[/, '').replace(/\]$/, '').replace(/\.$/, '')
}

function isLocalHostname(host: string) {
  return host === 'localhost' || host.endsWith('.localhost') || host.endsWith('.local')
}

function isPrivateIP(host: string) {
  const version = isIP(host)
  if (version === 4) return isPrivateIPv4(host)
  if (version === 6) return isPrivateIPv6(host)
  return false
}

function isPrivateIPv4(host: string) {
  if (IPV4_LOCAL_PREFIXES.some((prefix) => host.startsWith(prefix))) return true
  const [first, second] = host.split('.').map(Number)
  return first === 172 && second >= 16 && second <= 31
}

function isPrivateIPv6(host: string) {
  if (IPV6_LOCAL_HOSTS.has(host)) return true
  return host.startsWith('fc') || host.startsWith('fd') || host.startsWith('fe80:')
}
