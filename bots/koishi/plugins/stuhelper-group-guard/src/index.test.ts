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

import groupGuardPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('未认证成员入群后会被禁言并收到提醒，认证完成后自动解禁', async () => {
  let verificationState: 'bound_unverified' | 'verified' = 'bound_unverified'
  const server = createServer((req, res) => {
    if (respondAdmissionSession(req, res, '10001', 'group-1')) return
    assert.equal(req.method, 'GET')
    assert.equal(req.headers.authorization, 'Bearer test-token')
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify({
      success: true,
      data: {
        qqID: '10001',
        userID: 42,
        qqNickname: '10001',
        boundAt: '2026-04-19T00:00:00Z',
        verificationState,
        profileVerificationStatus: verificationState === 'verified' ? 'verified' : 'pending',
        studentVerified: verificationState === 'verified',
      },
    }))
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
      targetGroups: ['group-1'],
      muteDurationSeconds: 600,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成 StuHelper 注册、QQ 绑定与学生认证。',
      exemptUsers: [],
    },
    scheduler: {
      scanIntervalSeconds: 1,
    },
  })

  try {
    await root.start()
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: (event: Partial<Universal.Event>) => void }
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
      user: { id: '10001', name: '10001' },
      guild: { id: 'group-1' },
      channel: { id: 'group-1', type: Universal.Channel.Type.TEXT },
    })

    await waitFor(() => muteActions.length > 0 && sentMessages.length > 0)

    assert.equal(muteActions[0].groupId, 'group-1')
    assert.equal(muteActions[0].memberId, '10001')
    assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
    assert.match(sentMessages[0], /https:\/\/auth\.stuhelper\.com\/admission\/a\/token-10001\?qq=10001/)

    const records = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.equal(records.length, 1)
    assert.equal(records[0].releasedAt, null)

    verificationState = 'verified'
    await sleep(1200)

    assert.equal(muteActions[1]?.duration, 0)
    const released = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.ok(released[0].releasedAt instanceof Date)
  } finally {
    runtime.dispose()
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('超时未认证成员会被自动踢出', async () => {
  const server = createServer((req, res) => {
    if (respondAdmissionSession(req, res, '10002', 'group-2')) return
    assert.equal(req.method, 'GET')
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify({
      success: true,
      data: {
        qqID: '10002',
        userID: 42,
        qqNickname: '10002',
        boundAt: '2026-04-19T00:00:00Z',
        verificationState: 'bound_unverified',
        profileVerificationStatus: 'pending',
        studentVerified: false,
      },
    }))
  })

  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-guard-'))
  const kickActions: Array<{ groupId: string, memberId: string }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, {
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    guard: {
      targetGroups: ['group-2'],
      muteDurationSeconds: 600,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成认证。',
      exemptUsers: [],
    },
    scheduler: {
      scanIntervalSeconds: 1,
    },
  })

  try {
    await root.start()
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: (event: Partial<Universal.Event>) => void }
    bot.muteGuildMember = async () => {}
    bot.kickGuildMember = async (groupId, memberId) => {
      kickActions.push({ groupId, memberId })
    }
    bot.sendMessage = async () => ['msg-1']

    bot.receive({
      type: 'guild-member-added',
      user: { id: '10002', name: '10002' },
      guild: { id: 'group-2' },
      channel: { id: 'group-2', type: Universal.Channel.Type.TEXT },
    })

    await waitFor(async () => {
      const records = await root.database.get(GUARD_MEMBER_TABLE, {})
      return records.length > 0
    })

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, {})
    await root.database.set(GUARD_MEMBER_TABLE, { id: record.id }, {
      deadlineAt: new Date(Date.now() - 1000),
    })

    await sleep(1200)

    assert.deepEqual(kickActions[0], { groupId: 'group-2', memberId: '10002' })
    const records = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.ok(records[0].kickedAt instanceof Date)
  } finally {
    runtime.dispose()
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('扫描待认证成员时会路由到记录绑定的 bot 实例', async () => {
  let verificationState: 'bound_unverified' | 'verified' = 'bound_unverified'
  const server = createServer((req, res) => {
    if (respondAdmissionSession(req, res, '10003', 'group-3')) return
    assert.equal(req.method, 'GET')
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify({
      success: true,
      data: {
        qqID: '10003',
        userID: 42,
        qqNickname: '10003',
        boundAt: '2026-04-19T00:00:00Z',
        verificationState,
        profileVerificationStatus: verificationState === 'verified' ? 'verified' : 'pending',
        studentVerified: verificationState === 'verified',
      },
    }))
  })

  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-guard-'))
  const firstBotMuteActions: Array<{ groupId: string, memberId: string, duration: number }> = []
  const secondBotMuteActions: Array<{ groupId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(MockBot, { selfId: '515' })
  runtime.register(groupGuardPlugin, {
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    guard: {
      targetGroups: ['group-3'],
      muteDurationSeconds: 600,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成认证。',
      exemptUsers: [],
    },
    scheduler: {
      scanIntervalSeconds: 1,
    },
  })

  try {
    await root.start()
    const bots = root.bots as unknown as Array<Universal.Methods & {
      selfId: string
      receive: (event: Partial<Universal.Event>) => void
    }>
    const firstBot = bots.find((item) => item.selfId === '514')
    const secondBot = bots.find((item) => item.selfId === '515')
    assert.ok(firstBot)
    assert.ok(secondBot)

    firstBot.muteGuildMember = async (groupId, memberId, duration) => {
      firstBotMuteActions.push({ groupId, memberId, duration })
    }
    secondBot.muteGuildMember = async (groupId, memberId, duration) => {
      secondBotMuteActions.push({ groupId, memberId, duration })
    }
    firstBot.kickGuildMember = async () => {
      throw new Error('first bot should not kick members in this test')
    }
    secondBot.kickGuildMember = async () => {
      throw new Error('second bot should not kick members in this test')
    }
    firstBot.sendMessage = async () => ['msg-1']
    secondBot.sendMessage = async () => ['msg-2']

    secondBot.receive({
      type: 'guild-member-added',
      selfId: '515',
      platform: 'mock',
      user: { id: '10003', name: '10003' },
      guild: { id: 'group-3' },
      channel: { id: 'group-3', type: Universal.Channel.Type.TEXT },
    })

    await waitFor(async () => {
      const records = await root.database.get(GUARD_MEMBER_TABLE, {})
      return records.length > 0
    })

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.equal(record.botSelfId, '515')
    assert.equal(record.platform, 'mock')
    assert.equal(secondBotMuteActions[0].groupId, 'group-3')
    assert.equal(secondBotMuteActions[0].memberId, '10003')
    assert.ok(secondBotMuteActions[0].duration > 29 * 24 * 60 * 60 * 1000)

    verificationState = 'verified'
    await sleep(1200)

    assert.equal(firstBotMuteActions.length, 0)
    assert.deepEqual(secondBotMuteActions[1], { groupId: 'group-3', memberId: '10003', duration: 0 })
  } finally {
    runtime.dispose()
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('数据库群绑定模板会驱动 admission 入群认证', async () => {
  const server = createServer((req, res) => {
    if (respondAdmissionSession(req, res, '10004', 'group-4')) return
    assert.equal(req.method, 'GET')
    res.setHeader('content-type', 'application/json')
    res.end(JSON.stringify({
      success: true,
      data: {
        qqID: '10004',
        userID: 42,
        qqNickname: '10004',
        boundAt: '2026-04-19T00:00:00Z',
        verificationState: 'bound_unverified',
        profileVerificationStatus: 'pending',
        studentVerified: false,
      },
    }))
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

    const bot = root.bots[0] as unknown as Universal.Methods & { receive: (event: Partial<Universal.Event>) => void }
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
    await new Promise<void>((resolve, reject) => server.close((error) => error ? reject(error) : resolve()))
    await rm(tempDir, { recursive: true, force: true })
  }
})

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function waitFor(check: () => boolean | Promise<boolean>, timeoutMs = 1000, intervalMs = 20) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (await check()) {
      return
    }
    await sleep(intervalMs)
  }
  throw new Error('waitFor timed out')
}

function respondAdmissionSession(req: any, res: any, qqID: string, guildID: string) {
  if (req.method !== 'POST') return false
  assert.equal(req.headers.authorization, 'Bearer test-token')
  assert.equal(req.url, '/api/v1/bot/admission/sessions')
  res.setHeader('content-type', 'application/json')
  res.end(JSON.stringify({
    success: true,
    data: {
      token: `token-${qqID}`,
      authURL: `https://auth.stuhelper.com/admission/a/token-${qqID}?qq=${qqID}`,
      session: admissionSessionData(qqID, guildID),
    },
  }))
  return true
}

function admissionSessionData(qqID: string, guildID: string) {
  const now = Date.now()
  return {
    id: `session-${qqID}`,
    platform: 'mock',
    guildID,
    channelID: guildID,
    qqID,
    status: 'joined_muted',
    tokenExpiresAt: new Date(now + 60 * 60 * 1000).toISOString(),
    linkWaitDeadlineAt: new Date(now + 60 * 60 * 1000).toISOString(),
    submissionWaitDeadlineAt: new Date(now + 24 * 60 * 60 * 1000).toISOString(),
    initialMuteUntil: new Date(now + 30 * 24 * 60 * 60 * 1000).toISOString(),
    projectionPending: false,
  }
}
