import assert from 'node:assert/strict'
import test from 'node:test'

import { resolveConsoleGuildScope } from './console-guild-scope'

test('resolveConsoleGuildScope rejects role assignments that reference missing roles', async () => {
  await assert.rejects(
    resolveConsoleGuildScope({
      auth: {
        id: 42,
        authority: 4,
      },
    }, {
      roles: [{ id: 'role-1001', guildIds: ['1001'] }],
      getUserRoleIds: (userId) => ({
        'onebot:operator': ['deleted-role'],
      }[userId] ?? []),
      listBindingsByAuthId: async () => [{ platform: 'onebot', pid: 'operator' }],
    }),
    /console role assignment references missing role: deleted-role/,
  )
})

