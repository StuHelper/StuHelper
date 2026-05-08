import assert from 'node:assert/strict'
import test from 'node:test'

import { listAllMemberBlacklistPages } from './member-blacklist-pages'

test('listAllMemberBlacklistPages fetches until backend total is reached', async () => {
  const calls: Array<Record<string, unknown>> = []
  const backend = {
    async listMemberBlacklist(input: Record<string, unknown>) {
      calls.push(input)
      const page = Number(input.page)
      return {
        list: [memberBlacklistEntry(`entry-${page}`)],
        total: 2,
      }
    },
  }

  const result = await listAllMemberBlacklistPages(backend, {
    platform: 'qq',
    status: 'active',
  })

  assert.equal(result.total, 2)
  assert.deepEqual(result.list.map((entry) => entry.id), ['entry-1', 'entry-2'])
  assert.deepEqual(calls.map((call) => call.page), [1, 2])
  assert.ok(calls.every((call) => call.pageSize === 200))
})

test('listAllMemberBlacklistPages exposes inconsistent backend pagination', async () => {
  const backend = {
    async listMemberBlacklist(_input: Record<string, unknown>) {
      return { list: [], total: 1 }
    },
  }

  await assert.rejects(
    listAllMemberBlacklistPages(backend, { platform: 'qq', status: 'active' }),
    /pagination ended before total was reached/,
  )
})

function memberBlacklistEntry(id: string) {
  return {
    id,
    platform: 'qq',
    subjectType: 'qq_user' as const,
    subjectID: '10001',
    scopeType: 'guild' as const,
    guildID: '1001',
    source: 'manual_admin' as const,
    reasonCode: 'manual_blacklist' as const,
    reasonText: 'manual',
    metadata: {},
    createdByType: 'service_account' as const,
    createdByID: 'koishi-runtime',
    createdFrom: 'koishi_console' as const,
    createdAt: '2026-05-08T00:00:00Z',
    updatedAt: '2026-05-08T00:00:00Z',
  }
}
