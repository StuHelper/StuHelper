import assert from 'node:assert/strict'
import test from 'node:test'

import type { QQVerificationStatus } from '@stuhelper/koishi-shared'

import {
  compactIdentityProfileCache,
  IdentityProfileLookup,
  IdentityProfileLookupError,
} from './identity-profile-lookup'

test('IdentityProfileLookup caches profiles within the TTL window', async () => {
  let calls = 0
  let now = Date.parse('2026-04-21T08:00:00.000Z')
  const lookup = new IdentityProfileLookup({
    ttlMs: 60_000,
    concurrency: 2,
    now: () => now,
    getQQVerificationStatus: async (memberId) => {
      calls += 1
      return createProfile(memberId)
    },
  })

  const first = await lookup.lookup(['1001', '1002'])
  const second = await lookup.lookup(['1001'])

  assert.equal(first.profiles.length, 2)
  assert.equal(second.profiles.length, 1)
  assert.equal(calls, 2)

  now += 61_000
  await lookup.lookup(['1001'])
  assert.equal(calls, 3)
})

test('IdentityProfileLookup preserves memberId with structured errors', async () => {
  const lookup = new IdentityProfileLookup({
    concurrency: 1,
    getQQVerificationStatus: async (memberId) => {
      throw new IdentityProfileLookupError(memberId, new Error('platform unavailable'))
    },
  })

  const result = await lookup.lookup(['2001'])

  assert.deepEqual(result.profiles, [])
  assert.deepEqual(result.errors, [{
    memberId: '2001',
    message: 'platform unavailable',
  }])
})

test('IdentityProfileLookup limits concurrent platform requests', async () => {
  let active = 0
  let maxActive = 0
  const lookup = new IdentityProfileLookup({
    concurrency: 2,
    getQQVerificationStatus: async (memberId) => {
      active += 1
      maxActive = Math.max(maxActive, active)
      await new Promise((resolve) => setTimeout(resolve, 10))
      active -= 1
      return createProfile(memberId)
    },
  })

  const result = await lookup.lookup(['3001', '3002', '3003', '3004'])

  assert.equal(result.profiles.length, 4)
  assert.equal(maxActive, 2)
})

test('compactIdentityProfileCache removes expired entries before enforcing max size', () => {
  const cache = new Map([
    ['expired-a', createCacheEntry('expired-a', 10, 10)],
    ['expired-b', createCacheEntry('expired-b', 20, 20)],
    ['fresh-c', createCacheEntry('fresh-c', 120, 30)],
  ])

  compactIdentityProfileCache(cache, {
    now: 100,
    maxSize: 2,
  })

  assert.deepEqual([...cache.keys()], ['fresh-c'])
})

test('IdentityProfileLookup evicts the least recently used cache entry when max size is exceeded', async () => {
  let calls = 0
  let now = Date.parse('2026-04-21T08:00:00.000Z')
  const lookup = new IdentityProfileLookup({
    ttlMs: 60_000,
    concurrency: 2,
    maxSize: 2,
    now: () => now,
    getQQVerificationStatus: async (memberId) => {
      calls += 1
      return createProfile(memberId)
    },
  })

  await lookup.lookup(['4001', '4002'])
  await lookup.lookup(['4001'])
  await lookup.lookup(['4003'])
  await lookup.lookup(['4002'])

  assert.equal(calls, 4)
})

test('IdentityProfileLookup drops expired cache entries before storing new profiles', async () => {
  let calls = 0
  let now = Date.parse('2026-04-21T08:00:00.000Z')
  const lookup = new IdentityProfileLookup({
    ttlMs: 60_000,
    concurrency: 1,
    maxSize: 2,
    now: () => now,
    getQQVerificationStatus: async (memberId) => {
      calls += 1
      return createProfile(memberId)
    },
  })

  await lookup.lookup(['5001', '5002'])
  now += 61_000
  await lookup.lookup(['5003'])
  await lookup.lookup(['5001'])

  assert.equal(calls, 4)
})

function createProfile(memberId: string): QQVerificationStatus {
  return {
    qqID: memberId,
    userID: Number(memberId),
    boundAt: '2026-04-20T08:00:00.000Z',
    verificationState: 'verified',
    profileVerificationStatus: 'verified',
    studentVerified: true,
  }
}

function createCacheEntry(memberId: string, expiresAt: number, lastAccessedAt: number) {
  return {
    expiresAt,
    lastAccessedAt,
    profile: createProfile(memberId),
  }
}
