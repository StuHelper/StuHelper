import assert from 'node:assert/strict'
import test from 'node:test'

import { createConsoleQuery, parseConsoleQuery } from './navigation'

test('parseConsoleQuery restores page workspace and entity context', () => {
  const params = new URLSearchParams('view=review&workspace=pending&guildId=1001&memberId=2002&itemId=rv-1&keyword=kick')
  const state = parseConsoleQuery(params)

  assert.deepEqual(state, {
    view: 'review',
    workspace: 'pending',
    guildId: '1001',
    memberId: '2002',
    itemId: 'rv-1',
    tab: null,
    keyword: 'kick',
  })
})

test('createConsoleQuery omits empty optional fields', () => {
  const params = createConsoleQuery({
    view: 'dashboard',
    workspace: null,
    guildId: null,
    memberId: null,
    itemId: null,
    tab: null,
    keyword: '',
  })

  assert.equal(params.toString(), 'view=dashboard')
})
