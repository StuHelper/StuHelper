import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createConsoleHash,
  createConsoleQuery,
  mergeConsoleLocation,
  parseConsoleHash,
  parseConsoleLocation,
  parseConsoleQuery,
} from './navigation'

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

test('parseConsoleHash restores view and contextual state', () => {
  const state = parseConsoleHash('#review?workspace=pending&guildId=1001&itemId=rv-1&keyword=kick')

  assert.deepEqual(state, {
    view: 'review',
    workspace: 'pending',
    guildId: '1001',
    memberId: null,
    itemId: 'rv-1',
    tab: null,
    keyword: 'kick',
  })
})

test('parseConsoleLocation prefers hash state while keeping query as legacy fallback', () => {
  const hashState = parseConsoleLocation(new URL('http://127.0.0.1:5140/stuhelper?view=config#review'))
  const legacyState = parseConsoleLocation(new URL('http://127.0.0.1:5140/stuhelper?view=config'))

  assert.equal(hashState.view, 'review')
  assert.equal(legacyState.view, 'config')
})

test('parseConsoleLocation falls back to dashboard for explicit unknown view ids', () => {
  const legacyState = parseConsoleLocation(new URL('http://127.0.0.1:5140/stuhelper?view=missing'))
  const hashState = parseConsoleLocation(new URL('http://127.0.0.1:5140/stuhelper#missing'))
  const contextualHashState = parseConsoleLocation(
    new URL('http://127.0.0.1:5140/stuhelper#missing?workspace=pending&keyword=kick'),
  )

  assert.equal(legacyState.view, 'dashboard')
  assert.equal(hashState.view, 'dashboard')
  assert.deepEqual(contextualHashState, {
    view: 'dashboard',
    workspace: 'pending',
    guildId: null,
    memberId: null,
    itemId: null,
    tab: null,
    keyword: 'kick',
  })
})

test('createConsoleHash omits empty optional fields', () => {
  const hash = createConsoleHash({
    view: 'dashboard',
    workspace: null,
    guildId: null,
    memberId: null,
    itemId: null,
    tab: null,
    keyword: '',
  })

  assert.equal(hash, '#dashboard')
})

test('mergeConsoleLocation removes legacy navigation query and writes hash state', () => {
  const url = mergeConsoleLocation(new URL('http://127.0.0.1:5140/stuhelper?view=config&keep=1'), {
    view: 'review',
    workspace: 'pending',
    guildId: null,
    memberId: null,
    itemId: null,
    tab: null,
    keyword: '',
  })

  assert.equal(url.pathname, '/stuhelper')
  assert.equal(url.search, '?keep=1')
  assert.equal(url.hash, '#review?workspace=pending')
})
