import assert from 'node:assert/strict'
import test from 'node:test'

import type { Universal } from 'koishi'

import {
  resolveAdmissionReminderDeliveryConfig,
  sendAdmissionReminderMessage,
} from './admission-reminder-delivery'

test('admission reminder delivery defaults to group reminder only', async () => {
  const sentMessages: Array<{ channelId: string, content: string }> = []
  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async (channelId: string, content: string) => {
        sentMessages.push({ channelId, content })
        return ['group-message-1']
      },
      sendPrivateMessage: async () => {
        throw new Error('private message should not be sent by default')
      },
    } as unknown as Universal.Methods,
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: '认证链接',
  })

  assert.deepEqual(sentMessages, [{ channelId: 'channel-1', content: '认证链接' }])
  assert.deepEqual(result, {
    messageID: 'group-message-1',
    groupMessageID: 'group-message-1',
    directMessageID: undefined,
  })
})

test('admission reminder direct delivery sends friend private message without guild scope', async () => {
  const privateMessages: Array<{ userId: string, content: string, guildId?: string }> = []
  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async () => {
        throw new Error('group message should not be sent')
      },
      getFriendList: async () => [{ id: '10001' }],
      sendPrivateMessage: async (userId: string, content: string, guildId?: string) => {
        privateMessages.push({ userId, content, guildId })
        return ['direct-message-1']
      },
    } as unknown as Universal.Methods,
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: '认证链接',
    delivery: { groupEnabled: false, directEnabled: true },
  })

  assert.deepEqual(privateMessages, [{ userId: '10001', content: '认证链接', guildId: undefined }])
  assert.equal(result.messageID, 'direct-message-1')
})

test('admission reminder direct delivery sends non-friend temporary session with guild scope', async () => {
  const privateMessages: Array<{ userId: string, content: string, guildId?: string }> = []
  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async () => {
        throw new Error('group message should not be sent')
      },
      getFriendList: async () => ({ data: [] }),
      sendPrivateMessage: async (userId: string, content: string, guildId?: string) => {
        privateMessages.push({ userId, content, guildId })
        return ['temp-message-1']
      },
    } as unknown as Universal.Methods,
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: '认证链接',
    delivery: { groupEnabled: false, directEnabled: true },
  })

  assert.deepEqual(privateMessages, [{ userId: '10001', content: '认证链接', guildId: 'guild-1' }])
  assert.equal(result.messageID, 'temp-message-1')
})

test('admission reminder direct delivery falls back to OneBot temporary internal API', async () => {
  const internalPayloads: unknown[] = []
  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async () => {
        throw new Error('group message should not be sent')
      },
      getFriendList: async () => ({ data: [] }),
      sendPrivateMessage: async () => {
        throw new Error('standard temporary session unavailable')
      },
      internal: {
        sendPrivateMsg: async (payload: unknown) => {
          internalPayloads.push(payload)
          return { message_id: 12345 }
        },
      },
    } as unknown as Universal.Methods,
    guildId: '743762161',
    channelId: '743762161',
    memberId: '4883553',
    content: '认证链接',
    delivery: { groupEnabled: false, directEnabled: true },
  })

  assert.deepEqual(internalPayloads, [{
    user_id: 4883553,
    group_id: 743762161,
    message: '认证链接',
  }])
  assert.equal(result.messageID, '12345')
})

test('admission reminder delivery accepts one successful enabled channel', async () => {
  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async () => ['group-message-1'],
      getFriendList: async () => ({ data: [] }),
      sendPrivateMessage: async () => {
        throw new Error('temporary session unavailable')
      },
    } as unknown as Universal.Methods,
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: '认证链接',
    delivery: { groupEnabled: true, directEnabled: true },
  })

  assert.equal(result.messageID, 'group-message-1')
})

test('admission reminder delivery logs partial channel failures instead of swallowing them', async () => {
  const warns: Array<{ message: string, context: Record<string, unknown> }> = []
  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async () => {
        throw new Error('group send failed')
      },
      getFriendList: async () => ({ data: [{ id: '10001' }] }),
      sendPrivateMessage: async () => ['direct-message-1'],
    } as unknown as Universal.Methods,
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: '认证链接',
    delivery: { groupEnabled: true, directEnabled: true },
    logger: {
      warn(message, context) {
        warns.push({ message, context })
      },
    },
  })

  assert.equal(result.directMessageID, 'direct-message-1')
  assert.equal(warns.length, 1)
  assert.equal(warns[0].message, 'admission reminder delivery attempt failed')
  assert.equal(warns[0].context.channel, 'group')
  assert.equal(warns[0].context.guildId, 'guild-1')
  assert.equal(warns[0].context.memberId, '10001')
  assert.match(String(warns[0].context.error), /group send failed/)
})

test('admission reminder delivery allows disabling all reminder channels', async () => {
  assert.deepEqual(
    resolveAdmissionReminderDeliveryConfig({ groupEnabled: false, directEnabled: false }),
    { groupEnabled: false, directEnabled: false },
  )

  const result = await sendAdmissionReminderMessage({
    bot: {
      sendMessage: async () => {
        throw new Error('group message should not be sent')
      },
      sendPrivateMessage: async () => {
        throw new Error('direct message should not be sent')
      },
    } as unknown as Universal.Methods,
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: '认证链接',
    delivery: { groupEnabled: false, directEnabled: false },
  })
  assert.deepEqual(result, {})
})

test('admission reminder delivery resolver exposes independent runtime switches', () => {
  assert.deepEqual(resolveAdmissionReminderDeliveryConfig(), {
    groupEnabled: true,
    directEnabled: false,
  })
  assert.deepEqual(resolveAdmissionReminderDeliveryConfig({ groupEnabled: true, directEnabled: false }), {
    groupEnabled: true,
    directEnabled: false,
  })
  assert.deepEqual(resolveAdmissionReminderDeliveryConfig({ groupEnabled: false, directEnabled: true }), {
    groupEnabled: false,
    directEnabled: true,
  })
  assert.deepEqual(resolveAdmissionReminderDeliveryConfig({ groupEnabled: true, directEnabled: true }), {
    groupEnabled: true,
    directEnabled: true,
  })
})
