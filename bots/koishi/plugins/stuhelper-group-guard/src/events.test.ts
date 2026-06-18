import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import { registerGroupGuardEvents } from './events'

test('group guard events keep optional message guard narrowed without assertions', () => {
  const source = readFileSync(new URL('./events.ts', import.meta.url), 'utf8')

  assert.doesNotMatch(source, /messageGuard!/)
})

test('group guard event listeners return service promises to Koishi', async () => {
  const ctx = createEventContext()
  const memberAdded = Promise.resolve()
  const memberRequest = Promise.resolve()
  const memberMessageCalls: unknown[] = []
  const messageGuardCalls: unknown[] = []
  const message = Promise.resolve()
  const messageDeleted = Promise.resolve()

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => memberAdded,
      handleGuildMemberRequest: () => memberRequest,
      handleMessage: (session: unknown) => {
        memberMessageCalls.push(session)
        return Promise.resolve(false)
      },
      scanPendingMembers: () => Promise.resolve(),
    },
    messageGuard: {
      handleMessage: (session: unknown) => {
        messageGuardCalls.push(session)
        return message
      },
      handleMessageDeleted: () => messageDeleted,
    },
    logger: createLogger(),
    scanIntervalSeconds: 30,
  } as any)

  assert.equal(ctx.handlers.get('guild-member-added')?.({}), memberAdded)
  assert.equal(ctx.handlers.get('guild-member-request')?.({}), memberRequest)
  await ctx.handlers.get('message')?.({ id: 'message-1' })
  assert.deepEqual(memberMessageCalls, [{ id: 'message-1' }])
  assert.deepEqual(messageGuardCalls, [{ id: 'message-1' }])
  assert.equal(ctx.handlers.get('message-deleted')?.({}), messageDeleted)
})

test('group guard message listener keeps admission code handling when moderation is disabled', async () => {
  const ctx = createEventContext()
  const memberMessageCalls: unknown[] = []
  const messageGuardCalls: unknown[] = []

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => Promise.resolve(),
      handleGuildMemberRequest: () => Promise.resolve(),
      handleMessage: (session: unknown) => {
        memberMessageCalls.push(session)
        return Promise.resolve(true)
      },
      scanPendingMembers: () => Promise.resolve(),
    },
    messageGuard: {
      handleMessage: (session: unknown) => {
        messageGuardCalls.push(session)
        return Promise.resolve()
      },
      handleMessageDeleted: () => Promise.resolve(),
    },
    runtimeSettings: {
      isModerationEnabled: async () => false,
    },
    logger: createLogger(),
    scanIntervalSeconds: 30,
  } as any)

  await ctx.handlers.get('message')?.({ id: 'message-2' })

  assert.deepEqual(memberMessageCalls, [{ id: 'message-2' }])
  assert.deepEqual(messageGuardCalls, [])
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
      handleMessage: () => Promise.resolve(false),
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
