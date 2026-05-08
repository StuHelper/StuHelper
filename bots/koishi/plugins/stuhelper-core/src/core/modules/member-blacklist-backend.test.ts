import assert from 'node:assert/strict'
import test from 'node:test'

import { listVisibleMemberBlacklists } from './member-blacklist-backend'

test('listVisibleMemberBlacklists paginates global and guild scopes', async () => {
  const calls: Array<Record<string, unknown>> = []
  const backend = {
    async listMemberBlacklist(input: Record<string, unknown>) {
      calls.push(input)
      const scope = String(input.scopeType)
      const page = Number(input.page)
      return {
        list: [memberBlacklistEntry(`${scope}-${page}`)],
        total: 2,
      }
    },
  }

  const result = await listVisibleMemberBlacklists(backend, 'qq', 'guild-1')

  assert.deepEqual(
    result.map((entry) => entry.id).sort(),
    ['global-1', 'global-2', 'guild-1', 'guild-2'],
  )
  assert.equal(calls.length, 4)
  assert.ok(calls.every((call) => call.platform === 'qq'))
  assert.ok(calls.every((call) => call.subjectType === 'qq_user'))
  assert.ok(calls.every((call) => call.status === 'active'))
})

function memberBlacklistEntry(id: string) {
  const isGlobal = id.startsWith('global')
  return {
    id,
    platform: 'qq',
    subjectType: 'qq_user' as const,
    subjectID: '10001',
    scopeType: isGlobal ? 'global' as const : 'guild' as const,
    guildID: isGlobal ? null : 'guild-1',
    source: 'manual_admin' as const,
    reasonCode: 'manual_blacklist' as const,
    reasonText: 'manual',
    metadata: {},
    createdByType: 'qq_operator' as const,
    createdByID: '90001',
    createdFrom: 'qq_command' as const,
    createdAt: '2026-05-08T00:00:00Z',
    updatedAt: '2026-05-08T00:00:00Z',
  }
}
