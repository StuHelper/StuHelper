import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import { Universal } from 'koishi'
import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'

import {
  GUARD_GROUP_BINDING_TABLE,
  GUARD_MEMBER_TABLE,
  GUARD_TEMPLATE_TABLE,
} from '@stuhelper/koishi-shared'
import { MODERATION_EVENT_TABLE } from '@stuhelper/koishi-moderation-core'

import {
  respondAdmissionEvent,
  respondAdmissionSession,
  respondFreshmanForwards,
  respondPendingActions,
  waitFor,
} from './admission-test-support'
import groupGuardPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('数据库群绑定模板会驱动 admission 入群认证', async () => {
  const admissionEvents: unknown[] = []
  const server = createServer((req, res) => {
    if (respondAdmissionSession({ req, res, qqID: '10004', guildID: 'group-4' })) return
    if (respondPendingActions(req, res, () => [])) return
    if (respondAdmissionEvent({ req, res, events: admissionEvents })) return
    if (respondFreshmanForwards(req, res)) return
    assert.fail(`unexpected platform request: ${req.method} ${req.url}`)
  })

  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-guard-'))
  const muteActions: Array<{ groupId: string, memberId: string, duration: number }> = []
  const sentMessages: string[] = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, {
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    guard: {
      targetGroups: [],
      muteDurationSeconds: 600,
      kickAfterMinutes: 30,
      reminderTemplate: '静态模板不应命中。',
      exemptUsers: [],
    },
    scheduler: {
      scanIntervalSeconds: 1,
    },
    freshmanForward: {
      enabled: false,
    },
  })

  try {
    await root.start()
    await root.database.create(GUARD_TEMPLATE_TABLE, {
      id: 'dormitory',
      name: '宿舍群模板',
      muteDurationSeconds: 90,
      kickAfterMinutes: 10,
      reminderTemplate: '请先完成宿舍群学生认证。',
      exemptUsers: [],
      enabled: true,
      createdAt: new Date('2026-04-19T09:00:00Z'),
      updatedAt: new Date('2026-04-19T09:00:00Z'),
    })
    await root.database.create(GUARD_GROUP_BINDING_TABLE, {
      id: 'qq:group-4',
      platform: 'qq',
      guildId: 'group-4',
      templateId: 'dormitory',
      enabled: true,
      note: '宿舍群',
      createdAt: new Date('2026-04-19T09:00:00Z'),
      updatedAt: new Date('2026-04-19T09:00:00Z'),
    })

    const bot = root.bots[0] as unknown as Universal.Methods & { receive: ReceiveEvent }
    Object.assign(bot, { platform: 'onebot' })
    bot.muteGuildMember = async (groupId, memberId, duration) => {
      muteActions.push({ groupId, memberId, duration })
    }
    bot.kickGuildMember = async () => {
      throw new Error('kick should not be called in this test')
    }
    bot.sendMessage = async (_channelId, content) => {
      sentMessages.push(String(content))
      return ['msg-1']
    }

    bot.receive({
      type: 'guild-member-added',
      user: { id: '10004', name: '10004' },
      guild: { id: 'group-4' },
      channel: { id: 'group-4', type: Universal.Channel.Type.TEXT },
    })

    await waitFor(() => muteActions.length > 0 && sentMessages.length > 0)

    assert.equal(muteActions[0].groupId, 'group-4')
    assert.equal(muteActions[0].memberId, '10004')
    assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
    assert.match(sentMessages[0], /https:\/\/join\.stuhelper\.com\/verify\/token-10004\?qq=10004/)

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, { id: 'qq:514:group-4:10004' })
    assert.ok(record)
    assert.equal(record.platform, 'qq')
    assert.equal(record.admissionSessionID, 'session-10004')
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'remind')))
    await waitFor(async () => (await root.database.get(MODERATION_EVENT_TABLE, {})).length > 0)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('freshmanForward.enabled=false skips pending-forward backend scan', async () => {
  let pendingForwardRequests = 0
  const admissionEvents: unknown[] = []
  const server = createServer((req, res) => {
    if (respondAdmissionSession({ req, res, qqID: '10005', guildID: 'group-5' })) return
    if (respondPendingActions(req, res, () => [])) return
    if (respondAdmissionEvent({ req, res, events: admissionEvents })) return
    if (respondFreshmanForwards(req, res)) {
      pendingForwardRequests += 1
      return
    }
    assert.fail(`unexpected platform request: ${req.method} ${req.url}`)
  })

  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-guard-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, {
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    guard: {
      targetGroups: ['group-5'],
      muteDurationSeconds: 600,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成认证。',
      exemptUsers: [],
    },
    scheduler: {
      scanIntervalSeconds: 1,
    },
    freshmanForward: {
      enabled: false,
    },
  })

  try {
    await root.start()
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: ReceiveEvent }
    Object.assign(bot, { platform: 'onebot' })
    bot.muteGuildMember = async () => {}
    bot.kickGuildMember = async () => {
      throw new Error('kick should not be called in this test')
    }
    bot.sendMessage = async () => ['msg-1']

    bot.receive({
      type: 'guild-member-added',
      user: { id: '10005', name: '10005' },
      guild: { id: 'group-5' },
      channel: { id: 'group-5', type: Universal.Channel.Type.TEXT },
    })

    await waitFor(async () => (await root.database.get(GUARD_MEMBER_TABLE, {})).length > 0)
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'remind')))
    await waitFor(async () => (await root.database.get(MODERATION_EVENT_TABLE, {})).length > 0)
    await new Promise((resolve) => setTimeout(resolve, 1200))

    assert.equal(pendingForwardRequests, 0)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

type ReceiveEvent = (event: Partial<Universal.Event>) => void

function findEventByAction(events: unknown[], action: string) {
  return events.find((event) => eventAction(event) === action)
}

function eventAction(event: unknown) {
  if (!event || typeof event !== 'object') return undefined
  const body = (event as { body?: { action?: unknown } }).body
  return typeof body?.action === 'string' ? body.action : undefined
}

function closeServer(server: ReturnType<typeof createServer>) {
  return new Promise<void>((resolve, reject) => {
    server.close((error) => error ? reject(error) : resolve())
  })
}
