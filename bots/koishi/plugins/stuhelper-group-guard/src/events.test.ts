import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import { registerGroupGuardEvents } from './events'

test('group guard events keep optional message guard narrowed without assertions', () => {
  const source = readFileSync(new URL('./events.ts', import.meta.url), 'utf8')

  assert.doesNotMatch(source, /messageGuard!/)
})

test('group guard event listeners return service promises to Koishi', () => {
  const ctx = createEventContext()
  const memberAdded = Promise.resolve()
  const memberRequest = Promise.resolve()
  const message = Promise.resolve()
  const messageDeleted = Promise.resolve()

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => memberAdded,
      handleGuildMemberRequest: () => memberRequest,
      scanPendingMembers: () => Promise.resolve(),
    },
    messageGuard: {
      handleMessage: () => message,
      handleMessageDeleted: () => messageDeleted,
    },
    logger: createLogger(),
    scanIntervalSeconds: 30,
  } as any)

  assert.equal(ctx.handlers.get('guild-member-added')?.({}), memberAdded)
  assert.equal(ctx.handlers.get('guild-member-request')?.({}), memberRequest)
  assert.equal(ctx.handlers.get('message')?.({}), message)
  assert.equal(ctx.handlers.get('message-deleted')?.({}), messageDeleted)
})

test('scheduled group guard scans log rejected promises explicitly', async () => {
  const ctx = createEventContext()
  const errors: unknown[][] = []
  let catchAttached = false
  const rejectedScan = {
    catch(handler: (error: Error) => unknown) {
      catchAttached = true
      handler(new Error('scan failed'))
      return Promise.resolve()
    },
  }

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => Promise.resolve(),
      handleGuildMemberRequest: () => Promise.resolve(),
      scanPendingMembers: () => rejectedScan,
    },
    messageGuard: {
      handleMessage: () => Promise.resolve(),
      handleMessageDeleted: () => Promise.resolve(),
    },
    logger: {
      error: (...args: unknown[]) => {
        errors.push(args)
      },
    },
    scanIntervalSeconds: 30,
  } as any)

  const interval = ctx.intervals[0]
  assert.ok(interval)
  const result = interval.callback()
  await result

  assert.equal(catchAttached, true)
  assert.equal(errors.length, 1)
  assert.match(String(errors[0]?.[0]), /scheduled scan failed/)
  assert.equal((errors[0]?.[1] as Error).message, 'scan failed')
})

function createEventContext() {
  const handlers = new Map<string, (session: unknown) => unknown>()
  const intervals: Array<{ callback: () => Promise<void> | void; ms: number }> = []

  return {
    handlers,
    intervals,
    bots: [],
    on(event: string, handler: (session: unknown) => unknown) {
      handlers.set(event, handler)
    },
    setInterval(callback: () => Promise<void> | void, ms: number) {
      intervals.push({ callback, ms })
    },
  }
}

function createLogger() {
  return {
    error() {},
  }
}
