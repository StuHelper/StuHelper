import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context, Session } from 'koishi'

import { executeCommand } from './index.ts'

test('executeCommand does not elevate session authority unless explicitly requested', async () => {
  let authorityDuringExecution = 0
  const ctx = {
    logger: () => ({ error() {}, info() {} }),
    $commander: {
      get: () => ({
        execute: async ({ session }: { session: { user: { authority: number } } }) => {
          authorityDuringExecution = session.user.authority
          return 'ok'
        },
      }),
    },
  } as unknown as Context
  const session = { user: { authority: 1 } } as unknown as Session<'authority'>

  const result = await executeCommand({ ctx, session, commandName: 'warn' })

  assert.equal(result, 'ok')
  assert.equal(authorityDuringExecution, 1)
  assert.equal(session.user?.authority, 1)
})

test('executeCommand elevates only the cloned command session when requested', async () => {
  let authorityDuringExecution = 0
  const ctx = {
    logger: () => ({ error() {}, info() {} }),
    $commander: {
      get: () => ({
        execute: async ({ session }: { session: { user: { authority: number } } }) => {
          authorityDuringExecution = session.user.authority
          return 'ok'
        },
      }),
    },
  } as unknown as Context
  const session = { user: { authority: 1 } } as unknown as Session<'authority'>

  const result = await executeCommand({ ctx, session, commandName: 'warn', useAdmin: true })

  assert.equal(result, 'ok')
  assert.equal(authorityDuringExecution, 5)
  assert.equal(session.user?.authority, 1)
})

test('executeCommand redacts sensitive arguments, results, and errors before logging', async () => {
  const logs: unknown[] = []
  let shouldFail = false
  const ctx = {
    logger: () => ({
      error: (...args: unknown[]) => logs.push(args),
      info: (...args: unknown[]) => logs.push(args),
    }),
    $commander: {
      get: () => ({
        execute: async () => {
          if (shouldFail) throw new Error('Authorization: Bearer error_secret')
          return {
            token: 'result_secret',
            message: 'Authorization: Bearer result_auth_secret',
          }
        },
      }),
    },
  } as unknown as Context
  const session = { user: { authority: 1 } } as unknown as Session<'authority'>

  await executeCommand({
    ctx,
    session,
    commandName: 'warn',
    args: ['--token', 'arg_secret'],
  })
  shouldFail = true
  const failed = await executeCommand({
    ctx,
    session,
    commandName: 'warn',
    args: ['--api-key=sk-arg-secret'],
  })

  const payload = JSON.stringify(logs)
  assert.doesNotMatch(payload, /arg_secret/)
  assert.doesNotMatch(payload, /sk-arg-secret/)
  assert.doesNotMatch(payload, /result_secret/)
  assert.doesNotMatch(payload, /result_auth_secret/)
  assert.doesNotMatch(payload, /error_secret/)
  assert.doesNotMatch(String(failed), /error_secret/)
  assert.match(payload, /\[REDACTED\]/)
  assert.match(String(failed), /\[REDACTED\]/)
})
