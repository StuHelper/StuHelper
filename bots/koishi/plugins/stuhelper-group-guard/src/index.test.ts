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

import {
  respondAdmissionSession,
  respondFreshmanForwards,
  respondPendingActions,
  waitFor,
} from './admission-test-support'
import groupGuardPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('数据库群绑定模板会驱动 admission 入群认证', async () => {
  const server = createServer((req, res) => {
    if (respondAdmissionSession(req, res, '10004', 'group-4')) return
    if (respondPendingActions(req, res, () => [])) return
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
      id: 'mock:group-4',
      platform: 'mock',
      guildId: 'group-4',
      templateId: 'dormitory',
      enabled: true,
      note: '宿舍群',
      createdAt: new Date('2026-04-19T09:00:00Z'),
      updatedAt: new Date('2026-04-19T09:00:00Z'),
    })

    const bot = root.bots[0] as unknown as Universal.Methods & { receive: ReceiveEvent }
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
    assert.match(sentMessages[0], /auth\.stuhelper\.com/)

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, { id: 'mock:514:group-4:10004' })
    assert.ok(record)
    assert.equal(record.admissionSessionID, 'session-10004')
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

type ReceiveEvent = (event: Partial<Universal.Event>) => void

function closeServer(server: ReturnType<typeof createServer>) {
  return new Promise<void>((resolve, reject) => {
    server.close((error) => error ? reject(error) : resolve())
  })
}
