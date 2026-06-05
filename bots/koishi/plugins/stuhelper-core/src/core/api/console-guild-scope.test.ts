import assert from 'node:assert/strict'
import test from 'node:test'

import {
  resolveConsoleGuildScope,
  resolveRequiredConsoleGuildScope,
} from './console-guild-scope'

test('resolveConsoleGuildScope reads console auth from an unknown client boundary', async () => {
  const client: unknown = {
    auth: {
      id: 42,
      authority: 4,
    },
  }

  const scope = await resolveConsoleGuildScope(client, {
    roles: [],
    getUserRoleIds: () => [],
    listBindingsByAuthId: async (authId) => {
      assert.equal(authId, 42)
      return []
    },
  })

  assert.deepEqual(scope, { kind: 'all' })
})

test('resolveRequiredConsoleGuildScope rejects malformed console auth before scope lookup', async () => {
  await assert.rejects(
    resolveRequiredConsoleGuildScope({
      auth: {
        id: '42',
        authority: 4,
      },
    }, {
      roles: [],
      getUserRoleIds: () => [],
      listBindingsByAuthId: async () => {
        throw new Error('should not read bindings for malformed auth')
      },
    }),
    /console auth is required/,
  )
})

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
