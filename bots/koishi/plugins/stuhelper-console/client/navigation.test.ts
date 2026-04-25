import assert from 'node:assert/strict'
import test from 'node:test'

import {
  createConsoleSearch,
  parseConsoleSearch,
  updateConsoleUrl,
} from './navigation'

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

test('updateConsoleUrl preserves unrelated URL parts and replaces console params', () => {
  const currentUrl = new URL(
    'https://console.example.dev/stuhelper?foo=bar&queue=stale&id=old&source=dashboard#detail',
  )

  const nextUrl = updateConsoleUrl(currentUrl, {
    section: 'audit',
    queue: null,
    id: '',
    source: 'nav',
  })

  assert.equal(
    nextUrl.href,
    'https://console.example.dev/stuhelper?foo=bar&section=audit&source=nav#detail',
  )
  assert.equal(
    currentUrl.href,
    'https://console.example.dev/stuhelper?foo=bar&queue=stale&id=old&source=dashboard#detail',
  )
})
