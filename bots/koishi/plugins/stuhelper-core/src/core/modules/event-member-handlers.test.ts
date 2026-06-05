import assert from 'node:assert/strict'
import test from 'node:test'
import { readFileSync } from 'node:fs'
import { dirname, resolve } from 'node:path'
import { fileURLToPath } from 'node:url'

import type { EventRuntimeHost } from './event-support'
import { checkMuteExpires } from './event-member-handlers.ts'

const modulesDir = dirname(fileURLToPath(import.meta.url))

test('event member handlers use explicit push-message bot boundaries', () => {
  const source = readFileSync(resolve(modulesDir, './event-member-handlers.ts'), 'utf8')
  const serviceSource = readFileSync(resolve(modulesDir, '../services/stuhelper-group-center.service.ts'), 'utf8')

  assert.doesNotMatch(source, /bot: any/)
  assert.doesNotMatch(source, /as any/)
  assert.match(source, /type \{ PushMessageBot \}/)
  assert.match(source, /function isPushMessageBot/)
  assert.match(serviceSource, /export type PushMessageBot/)
})

test('checkMuteExpires pushes notifications and removes expired active mute records', async () => {
  const now = Date.now()
  const mutes = {
    'guild-1': {
      'expired-user': {
        startTime: now - 120_000,
        duration: 60_000,
      },
      'active-user': {
        startTime: now,
        duration: 60_000,
      },
      'left-user': {
        startTime: now - 120_000,
        duration: 60_000,
        leftGroup: true,
      },
    },
  }
  const pushed: Array<{ message: string; feature: string }> = []
  const host = createEventHost(mutes, pushed)

  await checkMuteExpires(host, createPushMessageBot())

  assert.deepEqual(Object.keys(mutes['guild-1']).sort(), ['active-user', 'left-user'])
  assert.deepEqual(pushed, [{
    message: '[禁言到期] 群 guild-1 用户 expired-user 的禁言已到期',
    feature: 'muteExpire',
  }])
  assert.deepEqual(host.updatedGuilds, ['guild-1'])
})

function createEventHost(
  mutes: Record<string, Record<string, {
    startTime: number
    duration: number
    remainingTime?: number
    leftGroup?: boolean
  }>>,
  pushed: Array<{ message: string; feature: string }>,
): EventRuntimeHost & { readonly updatedGuilds: string[] } {
  const updatedGuilds: string[] = []
  return {
    updatedGuilds,
    ctx: {
      stuhelperGroupCenter: {
        pushMessage: async (_bot: unknown, message: string, feature: string) => {
          pushed.push({ message, feature })
        },
      },
    },
    data: {
      mutes: {
        getAll: () => mutes,
        set: (guildId: string) => {
          updatedGuilds.push(guildId)
        },
      },
    },
    config: {},
  } as unknown as EventRuntimeHost & { readonly updatedGuilds: string[] }
}

function createPushMessageBot() {
  return {
    sendMessage: async () => {},
    sendPrivateMessage: async () => {},
  }
}
