import assert from 'node:assert/strict'
import test from 'node:test'

import { createConsoleSearch, parseConsoleSearch } from './navigation'

test('parseConsoleSearch falls back to dashboard for unknown sections', () => {
  const state = parseConsoleSearch(new URLSearchParams('section=unknown&id=rv_42'))
  assert.deepEqual(state, {
    section: 'dashboard',
    queue: null,
    id: 'rv_42',
    source: 'direct',
  })
})

test('createConsoleSearch round-trips queue context', () => {
  const state = {
    section: 'enforcement',
    queue: 'review',
    id: 'rv_42',
    source: 'dashboard',
  } as const
  const search = createConsoleSearch(state)
  const parsed = parseConsoleSearch(search)

  assert.deepEqual(parsed, state)
})
