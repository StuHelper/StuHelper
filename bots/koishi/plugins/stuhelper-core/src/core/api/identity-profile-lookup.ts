import type { QQVerificationStatus } from '@stuhelper/koishi-shared'

import type { IdentityLookupError } from '../services'

const DEFAULT_LOOKUP_TTL_MS = 60_000
const DEFAULT_LOOKUP_CONCURRENCY = 8
const DEFAULT_LOOKUP_MAX_SIZE = 10_000

interface CacheEntry {
  expiresAt: number
  lastAccessedAt: number
  profile: QQVerificationStatus
}

interface IdentityProfileLookupOptions {
  ttlMs?: number
  concurrency?: number
  maxSize?: number
  now?: () => number
  getQQVerificationStatus: (memberId: string) => Promise<QQVerificationStatus>
}

export class IdentityProfileLookupError extends Error {
  readonly memberId: string

  constructor(memberId: string, cause: unknown) {
    super(cause instanceof Error ? cause.message : String(cause))
    this.name = 'IdentityProfileLookupError'
    this.memberId = memberId
    this.cause = cause
  }
}

export class IdentityProfileLookup {
  private readonly cache = new Map<string, CacheEntry>()
  private readonly ttlMs: number
  private readonly concurrency: number
  private readonly maxSize: number
  private readonly now: () => number
  private accessCounter = 0

  constructor(private readonly options: IdentityProfileLookupOptions) {
    this.ttlMs = options.ttlMs ?? DEFAULT_LOOKUP_TTL_MS
    this.concurrency = options.concurrency ?? DEFAULT_LOOKUP_CONCURRENCY
    this.maxSize = options.maxSize ?? DEFAULT_LOOKUP_MAX_SIZE
    this.now = options.now ?? Date.now
  }

  async lookup(memberIds: string[]) {
    compactIdentityProfileCache(this.cache, {
      now: this.now(),
      maxSize: this.maxSize,
    })
    const tasks = memberIds.map((memberId) => async () => {
      try {
        return await this.lookupOne(memberId)
      } catch (error) {
        return error as IdentityProfileLookupError
      }
    })
    const results = await runWithConcurrency(tasks, this.concurrency)
    const profiles: QQVerificationStatus[] = []
    const errors: IdentityLookupError[] = []

    for (const result of results) {
      if (result instanceof IdentityProfileLookupError) {
        errors.push({ memberId: result.memberId, message: result.message })
        continue
      }
      profiles.push(result)
    }

    return { profiles, errors }
  }

  private async lookupOne(memberId: string) {
    const cached = this.cache.get(memberId)
    if (cached && cached.expiresAt > this.now()) {
      cached.lastAccessedAt = this.nextAccessOrder()
      return cached.profile
    }

    try {
      const now = this.now()
      const profile = await this.options.getQQVerificationStatus(memberId)
      this.cache.set(memberId, {
        profile,
        expiresAt: now + this.ttlMs,
        lastAccessedAt: this.nextAccessOrder(),
      })
      compactIdentityProfileCache(this.cache, {
        now,
        maxSize: this.maxSize,
      })
      return profile
    } catch (cause) {
      throw new IdentityProfileLookupError(memberId, cause)
    }
  }

  private nextAccessOrder() {
    this.accessCounter += 1
    return this.accessCounter
  }
}

interface CompactCacheOptions {
  now: number
  maxSize: number
}

export function compactIdentityProfileCache(
  cache: Map<string, CacheEntry>,
  options: CompactCacheOptions,
) {
  for (const [memberId, entry] of cache) {
    if (entry.expiresAt <= options.now) {
      cache.delete(memberId)
    }
  }

  while (cache.size > options.maxSize) {
    const oldest = findLeastRecentlyUsedKey(cache)
    if (!oldest) {
      return
    }
    cache.delete(oldest)
  }
}

function findLeastRecentlyUsedKey(cache: Map<string, CacheEntry>) {
  let candidate: string | null = null
  let lastAccessedAt = Number.POSITIVE_INFINITY

  for (const [memberId, entry] of cache) {
    if (entry.lastAccessedAt >= lastAccessedAt) {
      continue
    }
    candidate = memberId
    lastAccessedAt = entry.lastAccessedAt
  }

  return candidate
}

async function runWithConcurrency<T>(
  tasks: Array<() => Promise<T>>,
  concurrency: number,
) {
  const results: T[] = []
  let nextIndex = 0
  const workers = Array.from({ length: Math.max(1, concurrency) }, async () => {
    while (nextIndex < tasks.length) {
      const currentIndex = nextIndex
      nextIndex += 1
      results[currentIndex] = await tasks[currentIndex]()
    }
  })
  await Promise.all(workers)
  return results
}
