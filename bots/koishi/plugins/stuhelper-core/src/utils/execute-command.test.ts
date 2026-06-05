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
