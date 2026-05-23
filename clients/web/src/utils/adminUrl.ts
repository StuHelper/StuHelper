const DEFAULT_ADMIN_PATH = '/admin/'

function withTrailingSlash(value: string) {
  return value.endsWith('/') ? value : `${value}/`
}

function normalizeURLPath(url: URL) {
  if (url.pathname === '/') {
    url.pathname = DEFAULT_ADMIN_PATH
    return url.toString()
  }
  url.pathname = withTrailingSlash(url.pathname)
  return url.toString()
}

function normalizeRelativePath(value: string) {
  const path = value.startsWith('/') ? value : `/${value}`
  return withTrailingSlash(path)
}

export function resolveAdminConsoleURL(rawURL?: string) {
  const value = rawURL?.trim()
  if (!value) {
    return DEFAULT_ADMIN_PATH
  }

  try {
    return normalizeURLPath(new URL(value))
  } catch (error) {
    void error
    return normalizeRelativePath(value)
  }
}
