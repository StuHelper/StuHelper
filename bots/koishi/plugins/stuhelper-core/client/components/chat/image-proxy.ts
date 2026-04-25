import { imageApi } from '../../api'

const imageCache = new Map<string, string>()
const pendingLoads = new Map<string, Promise<string | null>>()

export function needsImageProxy(url: string): boolean {
  try {
    const urlObj = new URL(url)
    return ['gchat.qpic.cn', 'multimedia.nt.qq.com.cn', 'c2cpicdw.qpic.cn']
      .some((domain) => urlObj.hostname.endsWith(domain))
  } catch {
    return false
  }
}

export async function loadChatImage(url: string, file?: string): Promise<string | null> {
  const cacheKey = file ? `${url}#${file}` : url
  const cached = imageCache.get(cacheKey)
  if (cached) {
    return cached === 'error' ? null : cached
  }

  const pending = pendingLoads.get(cacheKey)
  if (pending) {
    return pending
  }

  const loadPromise = fetchImage(url, file, cacheKey)
  pendingLoads.set(cacheKey, loadPromise)
  return loadPromise
}

async function fetchImage(url: string, file: string | undefined, cacheKey: string): Promise<string | null> {
  try {
    const result = await imageApi.fetch(url, file)
    const resolved = resolveImageResult(result, url, file)
    imageCache.set(cacheKey, resolved ?? 'error')
    if (resolved && result?.data?.source === 'local' && file) {
      imageCache.set(url, resolved)
    }
    return resolved
  } catch (error) {
    console.warn('Image proxy failed:', url, file, error)
    imageCache.set(cacheKey, 'error')
    return null
  } finally {
    pendingLoads.delete(cacheKey)
  }
}

function resolveImageResult(result: any, url: string, file?: string): string | null {
  if (result?.success && result.data?.dataUrl) {
    return result.data.dataUrl
  }
  if (result?.success && result.data?.direct) {
    return url
  }
  if (file && imageCache.has(url)) {
    return imageCache.get(url) ?? null
  }
  return null
}
