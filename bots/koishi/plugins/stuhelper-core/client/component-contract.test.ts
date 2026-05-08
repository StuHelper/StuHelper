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

test('dangerous moderation actions require shared confirmation before API mutation', () => {
  const reviewSource = readClientFile('./components/ReviewView.vue')
  const blacklistSource = readClientFile('./components/BlacklistView.vue')
  const warnsSource = readClientFile('./components/WarnsView.vue')

  assert.match(reviewSource, /<ConfirmDialog\b/)
  assert.ok(
    reviewSource.indexOf('const confirmed = await confirm(') < reviewSource.indexOf('await consolePageApi.workItemAction('),
  )
  assert.match(reviewSource, /if \(!confirmed\) return/)

  assert.match(blacklistSource, /<ConfirmDialog\b/)
  assert.ok(
    blacklistSource.indexOf('const confirmed = await confirm(') < blacklistSource.indexOf('await blacklistApi.remove(id'),
  )
  assert.match(blacklistSource, /if \(!confirmed\) return/)

  assert.match(warnsSource, /<ConfirmDialog\b/)
  assert.ok(
    warnsSource.indexOf('const confirmed = await confirm(') < warnsSource.indexOf('await warnsApi.update(key, next)'),
  )
  assert.match(warnsSource, /if \(!confirmed\) \{\s*await refresh\(\)\s*return\s*\}/)
})

test('SettingsView sidebar icon binding uses explicit string icon names', () => {
  const source = readClientFile('./components/SettingsView.vue')

  assert.doesNotMatch(source, /<k-icon :name="section"/)
  assert.match(source, /<k-icon :name="section\.icon"/)
  assert.match(source, /icon: 'stuhelperGroupCenter:octicons\./)
})

test('RolesView has no debug console output in browser code', () => {
  const source = readClientFile('./components/RolesView.vue')

  assert.doesNotMatch(source, /console\.(log|error|warn|info)\(/)
})

test('ChatView uses console API avatars instead of hard-coded QQ avatar URLs', () => {
  const source = readClientFile('./components/ChatView.vue')

  assert.doesNotMatch(source, /p\.qlogo\.cn/)
  assert.doesNotMatch(source, /q1\.qlogo\.cn/)
  assert.match(source, /displayAvatar = info\.avatar/)
})

test('shell portals keep StuHelper design tokens when mounted under body', () => {
  const searchSource = readClientFile('./components/shell/SearchPanel.vue')
  const overlaySource = readClientFile('./components/shell/EntityOverlay.vue')
  const dockSource = readClientFile('./components/shell/ChatDock.vue')
  const confirmSource = readClientFile('./components/primitives/ConfirmDialog.vue')
  const drawerSource = readClientFile('./components/primitives/Drawer.vue')

  assert.match(searchSource, /class="stuhelperGroupCenter-portal sh-search"/)
  assert.match(overlaySource, /class="stuhelperGroupCenter-portal"/)
  assert.match(dockSource, /class="stuhelperGroupCenter-portal sh-dock"/)
  assert.match(confirmSource, /modal-class="stuhelperGroupCenter-portal"/)
  assert.match(drawerSource, /modal-class="stuhelperGroupCenter-portal"/)
})

test('EntityChip owns click propagation so row-level handlers do not also fire', () => {
  const source = readClientFile('./components/primitives/EntityChip.vue')

  assert.match(source, /:is="tag"/)
  assert.match(source, /@click\.stop="handleClick"/)
  assert.match(source, /@keydown\.enter\.stop\.prevent="handleClick"/)
  assert.match(source, /const tag = computed\(\(\) => props\.inline \? 'span' : 'button'\)/)
  assert.match(source, /event\.stopPropagation\(\)/)
})

test('SearchPanel escape does not bubble into the AppShell escape chain', () => {
  const searchSource = readClientFile('./components/shell/SearchPanel.vue')
  const shellSource = readClientFile('./components/shell/AppShell.vue')

  assert.match(searchSource, /@keydown\.escape\.stop\.prevent="close"/)
  assert.match(shellSource, /if \(event\.defaultPrevented\) return/)
})

test('mobile shell exposes a CommandBar rail toggle outside the collapsed rail', () => {
  const commandSource = readClientFile('./components/shell/CommandBar.vue')
  const styleSource = readClientFile('./styles/shell.css')

  assert.match(commandSource, /class="sh-cmd__menu"/)
  assert.match(commandSource, /@click="shell\.toggleRail\(\)"/)
  assert.match(styleSource, /\.sh-shell__rail \{\s*position: fixed;/)
  assert.match(styleSource, /\.sh-shell\[data-rail-expanded='true'\] \.sh-shell__rail/)
})
