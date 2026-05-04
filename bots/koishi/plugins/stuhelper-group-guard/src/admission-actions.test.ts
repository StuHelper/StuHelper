import assert from 'node:assert/strict'
import test from 'node:test'

import type { Universal } from 'koishi'

import { executeAdmissionAction } from './admission-actions'
import type { AdmissionPendingAction } from '@stuhelper/koishi-shared'

const DEADLINE_OFFSET_MS = 60_000

test('未知 admission action 不会默认执行拉黑', async () => {
  let kicked = false
  const bot = {
    kickGuildMember: async () => {
      kicked = true
    },
    muteGuildMember: async () => {},
    sendMessage: async () => ['message-1'],
  } as unknown as Universal.Methods

  await assert.rejects(
    () => executeAdmissionAction(bot, unknownAction(), null),
    /unknown admission action: mute/,
  )
  assert.equal(kicked, false)
})

function unknownAction(): AdmissionPendingAction {
  return {
    sessionID: 'session-unknown',
    action: 'mute' as AdmissionPendingAction['action'],
    guildID: 'guild-1',
    channelID: 'guild-1',
    qqID: '10001',
    deadlineAt: new Date(Date.now() + DEADLINE_OFFSET_MS).toISOString(),
  }
}
