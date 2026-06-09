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

test('release admission action 默认只解除禁言不发送群提示', async () => {
  let sendCalled = false
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []
  const bot = {
    kickGuildMember: async () => {},
    muteGuildMember: async (guildId: string, memberId: string, duration: number) => {
      muteActions.push({ guildId, memberId, duration })
    },
    sendMessage: async () => {
      sendCalled = true
      return ['message-1']
    },
  } as unknown as Universal.Methods

  const result = await executeAdmissionAction(bot, releaseAction(), null)

  assert.equal(sendCalled, false)
  assert.deepEqual(muteActions, [{ guildId: 'guild-1', memberId: '10001', duration: 0 }])
  assert.deepEqual(result.event, { action: 'release', success: true })
})

test('release admission action 可以通过消息模板恢复群提示', async () => {
  const sentMessages: string[] = []
  const bot = {
    kickGuildMember: async () => {},
    muteGuildMember: async () => {},
    sendMessage: async (_channelId: string, content: string) => {
      sentMessages.push(content)
      return ['message-1']
    },
  } as unknown as Universal.Methods

  const result = await executeAdmissionAction(bot, releaseAction(), null, {
    admissionReleaseCompleted: '{at} 自定义解除禁言提示：{memberId}',
  })

  assert.equal(sentMessages.length, 1)
  assert.match(sentMessages[0], /自定义解除禁言提示：10001/)
  assert.deepEqual(result.event, { action: 'release', success: true, messageID: 'message-1' })
})

test('remind admission action respects direct-only reminder delivery', async () => {
  const groupMessages: string[] = []
  const privateMessages: Array<{ userId: string, content: string, guildId?: string }> = []
  const bot = {
    kickGuildMember: async () => {},
    muteGuildMember: async () => {},
    sendMessage: async (_channelId: string, content: string) => {
      groupMessages.push(content)
      throw new Error('group reminder should not be sent')
    },
    getFriendList: async () => ({ data: [] }),
    sendPrivateMessage: async (userId: string, content: string, guildId?: string) => {
      privateMessages.push({ userId, content, guildId })
      return ['direct-message-1']
    },
  } as unknown as Universal.Methods

  const result = await executeAdmissionAction(bot, remindAction(), null, undefined, {
    groupEnabled: false,
    directEnabled: true,
  })

  assert.deepEqual(groupMessages, [])
  assert.equal(privateMessages.length, 1)
  assert.equal(privateMessages[0].userId, '10001')
  assert.equal(privateMessages[0].guildId, 'guild-1')
  assert.match(privateMessages[0].content, /https:\/\/join\.stuhelper\.com\/verify\/remind-token/)
  assert.deepEqual(result.event, { action: 'remind', success: true, messageID: 'direct-message-1' })
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

function remindAction(): AdmissionPendingAction {
  return {
    sessionID: 'session-remind',
    action: 'remind',
    guildID: 'guild-1',
    channelID: 'channel-1',
    qqID: '10001',
    deadlineAt: new Date(Date.now() + DEADLINE_OFFSET_MS).toISOString(),
    authURL: 'https://join.stuhelper.com/verify/remind-token',
  }
}

function releaseAction(): AdmissionPendingAction {
  return {
    sessionID: 'session-release',
    action: 'release',
    guildID: 'guild-1',
    channelID: 'guild-1',
    qqID: '10001',
    deadlineAt: new Date(Date.now() + DEADLINE_OFFSET_MS).toISOString(),
  }
}
