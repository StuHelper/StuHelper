import test from 'node:test'
import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

const clientDir = dirname(fileURLToPath(import.meta.url))

function readClientFile(relativePath: string): string {
  return readFileSync(resolve(clientDir, relativePath), 'utf8')
}

test('Drawer forwards the closed event to parent consumers', () => {
  const source = readClientFile('./components/primitives/Drawer.vue')

  assert.match(source, /@closed="handleClosed"/)
  assert.match(source, /emit\('closed'\)/)
})

test('SubscriptionView resets draft state on Drawer closed without a stray timeout', () => {
  const source = readClientFile('./components/SubscriptionView.vue')

  assert.match(source, /@closed="handleFormClosed"/)
  assert.match(source, /function handleFormClosed\(\) \{\s*resetDraft\(\)\s*\}/)
  assert.doesNotMatch(source, /setTimeout\(resetDraft,\s*280\)/)
})

test('WarnsView refreshes server state after successful count updates', () => {
  const source = readClientFile('./components/WarnsView.vue')

  assert.match(source, /await warnsApi\.update\(key, next\)/)
  assert.match(source, /pushSuccess\(next <= 0 \? '警告已清除' : '警告次数已更新'\)/)
  assert.match(source, /await refresh\(\)/)
})

test('ChatView delegates message rendering to a safe component instead of v-html', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.match(source, /<ChatMessageContent\b/)
  assert.doesNotMatch(source, /v-html="renderMessage\(msg\)"/)
  assert.doesNotMatch(source, /onclick="window\.open\('/)
})

test('ChatView uses a composite session key to avoid cross-platform collisions', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.match(source, /const buildSessionKey = \(params: \{/)
  assert.match(source, /return `\$\{params\.platform\}:\$\{params\.type\}:\$\{guildPart\}:\$\{params\.channelId\}`/)
  assert.match(source, /findSessionByKey\(sessionKey\)/)
})
