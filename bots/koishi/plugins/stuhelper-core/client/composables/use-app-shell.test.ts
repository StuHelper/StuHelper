import test from 'node:test'
import assert from 'node:assert/strict'

import { createAppShellController } from './use-app-shell'

test('closeTransientOverlays clears shell overlays without resetting persistent rail state', () => {
  const shell = createAppShellController()

  shell.pinRail(true)
  shell.openSearch()
  shell.openEntity({ kind: 'user', id: '100001' })
  shell.openChat()
  shell.toggleChatMinimized()

  shell.closeTransientOverlays()

  assert.equal(shell.searchOpen.value, false)
  assert.equal(shell.entityTarget.value, null)
  assert.equal(shell.chatOpen.value, false)
  assert.equal(shell.chatMinimized.value, false)
  assert.equal(shell.railPinned.value, true)
  assert.equal(shell.railExpanded.value, true)
})
