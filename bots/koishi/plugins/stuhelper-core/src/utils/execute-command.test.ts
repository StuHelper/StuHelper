import assert from 'node:assert/strict'
import test from 'node:test'

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
  }
  const session = { user: { authority: 1 } }

  const result = await executeCommand({ ctx: ctx as any, session, commandName: 'warn' })

  assert.equal(result, 'ok')
  assert.equal(authorityDuringExecution, 1)
  assert.equal(session.user.authority, 1)
})
