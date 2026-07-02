import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import test from 'node:test'

import { registerGroupGuardEvents } from './events'

test('group guard events keep optional message guard narrowed without assertions', () => {
  const source = readFileSync(new URL('./events.ts', import.meta.url), 'utf8')

  assert.doesNotMatch(source, /messageGuard!/)
})

test('group guard event listeners await services and keep message deletion promises', async () => {
  const ctx = createEventContext()
  const memberMessageCalls: unknown[] = []
  const messageGuardCalls: unknown[] = []
  const message = Promise.resolve()
  const messageDeleted = Promise.resolve()
  let memberAddedCalls = 0
  let memberRequestCalls = 0

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => {
        memberAddedCalls += 1
        return Promise.resolve()
      },
      handleGuildMemberRequest: () => {
        memberRequestCalls += 1
        return Promise.resolve()
      },
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

  await ctx.handlers.get('guild-member-added')?.({})
  await ctx.handlers.get('guild-member-request')?.({})
  assert.equal(memberAddedCalls, 1)
  assert.equal(memberRequestCalls, 1)
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

test('guild-member-added handler failures are logged with member context', async () => {
  const ctx = createEventContext()
  const errors: unknown[][] = []

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => Promise.reject(new Error('backend unavailable')),
      handleGuildMemberRequest: () => Promise.resolve(),
      scanPendingMembers: () => Promise.resolve(),
    },
    logger: {
      error: (...args: unknown[]) => {
        errors.push(args)
      },
    },
    scanIntervalSeconds: 30,
  } as any)

  const session = { platform: 'onebot', guildId: 'guild-1', userId: 'member-1' }
  await assert.doesNotReject(async () => {
    await ctx.handlers.get('guild-member-added')?.(session)
  })

  assert.equal(errors.length, 1)
  assert.match(String(errors[0]?.[0]), /guild-member-added handling failed/)
  assert.deepEqual(errors[0]?.[1], {
    platform: 'onebot',
    guildId: 'guild-1',
    memberId: 'member-1',
  })
  assert.equal((errors[0]?.[2] as Error).message, 'backend unavailable')
})

test('guild-member-request handler failures are logged with member context', async () => {
  const ctx = createEventContext()
  const errors: unknown[][] = []

  registerGroupGuardEvents(ctx as any, {
    memberGuard: {
      handleGuildMemberAdded: () => Promise.resolve(),
      handleGuildMemberRequest: () => {
        throw new Error('decision service down')
      },
      scanPendingMembers: () => Promise.resolve(),
    },
    logger: {
      error: (...args: unknown[]) => {
        errors.push(args)
      },
    },
    scanIntervalSeconds: 30,
  } as any)

  const session = { platform: 'onebot', guildId: 'guild-2', userId: 'member-2' }
  await assert.doesNotReject(async () => {
    await ctx.handlers.get('guild-member-request')?.(session)
  })

  assert.equal(errors.length, 1)
  assert.match(String(errors[0]?.[0]), /guild-member-request handling failed/)
  assert.deepEqual(errors[0]?.[1], {
    platform: 'onebot',
    guildId: 'guild-2',
    memberId: 'member-2',
  })
  assert.equal((errors[0]?.[2] as Error).message, 'decision service down')
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
