import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { createServer } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import { Universal } from 'koishi'
import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'

import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'
import { MODERATION_EVENT_TABLE } from '@stuhelper/koishi-moderation-core'

import {
  admissionAction,
  respondAdmissionEvent,
  respondAdmissionPolicyTargets,
  respondAdmissionSession,
  respondFreshmanForwards,
  respondPendingActions,
  sleep,
  waitFor,
} from './admission-test-support'
import groupGuardPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('未认证成员入群后会被禁言并收到提醒，认证完成后自动解禁', async () => {
  let pendingActions: unknown[] = []
  const admissionEvents: unknown[] = []
  const server = createServer((req, res) => {
    if (respondAdmissionSession({ req, res, qqID: '10001', guildID: 'group-1' })) return
    if (respondPendingActions(req, res, () => pendingActions)) return
    if (respondAdmissionEvent({ req, res, events: admissionEvents, afterEvent: () => { pendingActions = [] } })) return
    if (respondFreshmanForwards(req, res)) return
    if (respondAdmissionPolicyTargets(req, res, [admissionPolicyTarget('group-1')])) return
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
  runtime.register(groupGuardPlugin, groupGuardConfig(address.port))

  try {
    await root.start()
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: ReceiveEvent }
    forceQQAdmissionBot(bot)
    bot.muteGuildMember = async (groupId, memberId, duration) => {
      muteActions.push({ groupId, memberId, duration })
    }
    bot.kickGuildMember = async () => { throw new Error('kick should not be called in this test') }
    bot.sendMessage = async (_channelId, content) => {
      sentMessages.push(String(content))
      return ['msg-1']
    }

    receiveJoin(bot, '10001', 'group-1')
    await waitFor(() => muteActions.length > 0 && sentMessages.length > 0)

    assert.equal(muteActions[0].groupId, 'group-1')
    assert.equal(muteActions[0].memberId, '10001')
    assert.ok(muteActions[0].duration > 29 * 24 * 60 * 60 * 1000)
    assert.match(sentMessages[0], /https:\/\/join\.stuhelper\.com\/verify\/token-10001/)

    const records = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.equal(records.length, 1)
    assert.equal(records[0].releasedAt, null)
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'remind')))

    pendingActions = [admissionAction('10001', 'group-1', 'release')]
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'release')), 2500)
    await waitFor(async () => {
      const [record] = await root.database.get(GUARD_MEMBER_TABLE, {})
      return record?.releasedAt instanceof Date
    }, 2500)

    assert.equal(muteActions[1]?.duration, 0)
    assert.deepEqual(findEventByAction(admissionEvents, 'release'), successEvent('session-10001', 'release'))
    const released = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.ok(released[0].releasedAt instanceof Date)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('重复入群事件只创建一个认证链接', async () => {
  let admissionSessionRequests = 0
  const admissionEvents: unknown[] = []
  const server = createServer((req, res) => {
    if (req.method === 'POST' && req.url === '/api/v1/bot/admission/sessions') {
      admissionSessionRequests += 1
    }
    if (respondAdmissionSession({ req, res, qqID: '10004', guildID: 'group-dup' })) return
    if (respondPendingActions(req, res, [])) return
    if (respondAdmissionEvent({ req, res, events: admissionEvents })) return
    if (respondFreshmanForwards(req, res)) return
    if (respondAdmissionPolicyTargets(req, res, [admissionPolicyTarget('group-dup')])) return
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
  runtime.register(groupGuardPlugin, groupGuardConfig(address.port))

  try {
    await root.start()
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: ReceiveEvent }
    forceOneBotAdmissionBot(bot)
    bot.muteGuildMember = async (groupId, memberId, duration) => {
      muteActions.push({ groupId, memberId, duration })
    }
    bot.kickGuildMember = async () => { throw new Error('kick should not be called in this test') }
    bot.sendMessage = async (_channelId, content) => {
      sentMessages.push(String(content))
      return ['msg-1']
    }

    receiveJoin(bot, '10004', 'group-dup', undefined, 'onebot')
    receiveJoin(bot, '10004', 'group-dup', undefined, 'onebot')
    await waitFor(() => sentMessages.length > 0)
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'remind')))
    await waitFor(async () => {
      const events = await root.database.get(MODERATION_EVENT_TABLE, {
        memberId: '10004',
        type: 'join_guarded',
      })
      return events.length === 1
    })

    assert.equal(admissionSessionRequests, 1)
    assert.equal(muteActions.length, 1)
    assert.equal(sentMessages.length, 1)
    assert.match(sentMessages[0], /https:\/\/join\.stuhelper\.com\/verify\/token-10004/)

    const records = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.equal(records.length, 1)
    assert.equal(records[0].platform, 'qq')
    assert.equal(records[0].botSelfId, '514')
    assert.equal(records[0].guildId, 'group-dup')
    assert.equal(records[0].memberId, '10004')
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('超时未认证成员会被自动踢出', async () => {
  let pendingActions: unknown[] = []
  const admissionEvents: unknown[] = []
  const server = createServer((req, res) => {
    if (respondAdmissionSession({ req, res, qqID: '10002', guildID: 'group-2' })) return
    if (respondPendingActions(req, res, () => pendingActions)) return
    if (respondAdmissionEvent({ req, res, events: admissionEvents, afterEvent: () => { pendingActions = [] } })) return
    if (respondFreshmanForwards(req, res)) return
    if (respondAdmissionPolicyTargets(req, res, [admissionPolicyTarget('group-2')])) return
    assert.fail(`unexpected platform request: ${req.method} ${req.url}`)
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
  runtime.register(groupGuardPlugin, groupGuardConfig(address.port))

  try {
    await root.start()
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: ReceiveEvent }
    forceQQAdmissionBot(bot)
    bot.muteGuildMember = async () => {}
    bot.kickGuildMember = async (groupId, memberId) => { kickActions.push({ groupId, memberId }) }
    bot.sendMessage = async () => ['msg-1']

    receiveJoin(bot, '10002', 'group-2')
    await waitFor(async () => (await root.database.get(GUARD_MEMBER_TABLE, {})).length > 0)
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'remind')))

    pendingActions = [admissionAction('10002', 'group-2', 'kick')]
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'kick')), 2500)
    await waitFor(async () => {
      const [record] = await root.database.get(GUARD_MEMBER_TABLE, {})
      return record?.kickedAt instanceof Date
    }, 2500)

    assert.deepEqual(kickActions[0], { groupId: 'group-2', memberId: '10002' })
    assert.deepEqual(findEventByAction(admissionEvents, 'kick'), successEvent('session-10002', 'kick', 'msg-1'))
    const records = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.ok(records[0].kickedAt instanceof Date)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('扫描待认证成员时会路由到记录绑定的 bot 实例', async () => {
  let releaseEnabled = false
  const admissionEvents: unknown[] = []
  const server = createServer((req, res) => {
    if (respondAdmissionSession({ req, res, qqID: '10003', guildID: 'group-3' })) return
    if (respondPendingActions(req, res, (url) => {
      if (!releaseEnabled || url.searchParams.get('botSelfID') !== '515') return []
      return [admissionAction('10003', 'group-3', 'release')]
    })) return
    if (respondAdmissionEvent({ req, res, events: admissionEvents, afterEvent: () => { releaseEnabled = false } })) return
    if (respondFreshmanForwards(req, res)) return
    if (respondAdmissionPolicyTargets(req, res, [admissionPolicyTarget('group-3')])) return
    assert.fail(`unexpected platform request: ${req.method} ${req.url}`)
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
  runtime.register(groupGuardPlugin, groupGuardConfig(address.port))

  try {
    await root.start()
    const bots = root.bots as unknown as Array<Universal.Methods & { selfId: string, receive: ReceiveEvent }>
    const firstBot = bots.find((item) => item.selfId === '514')
    const secondBot = bots.find((item) => item.selfId === '515')
    assert.ok(firstBot)
    assert.ok(secondBot)
    forceQQAdmissionBot(firstBot)
    forceQQAdmissionBot(secondBot)

    firstBot.muteGuildMember = async (groupId, memberId, duration) => {
      firstBotMuteActions.push({ groupId, memberId, duration })
    }
    secondBot.muteGuildMember = async (groupId, memberId, duration) => {
      secondBotMuteActions.push({ groupId, memberId, duration })
    }
    firstBot.kickGuildMember = async () => { throw new Error('first bot should not kick members in this test') }
    secondBot.kickGuildMember = async () => { throw new Error('second bot should not kick members in this test') }
    firstBot.sendMessage = async () => ['msg-1']
    secondBot.sendMessage = async () => ['msg-2']

    receiveJoin(secondBot, '10003', 'group-3', '515')
    await waitFor(async () => {
      const records = await root.database.get(GUARD_MEMBER_TABLE, {})
      return records.length > 0
        && secondBotMuteActions.length > 0
        && Boolean(findEventByAction(admissionEvents, 'remind'))
    })

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.equal(record.botSelfId, '515')
    assert.equal(record.platform, 'qq')
    assert.equal(secondBotMuteActions[0].groupId, 'group-3')
    assert.equal(secondBotMuteActions[0].memberId, '10003')
    assert.ok(secondBotMuteActions[0].duration > 29 * 24 * 60 * 60 * 1000)

    releaseEnabled = true
    await waitFor(() => Boolean(findEventByAction(admissionEvents, 'release')), 2500)
    await waitFor(async () => {
      const [record] = await root.database.get(GUARD_MEMBER_TABLE, {})
      return record?.releasedAt instanceof Date
    }, 2500)

    assert.equal(firstBotMuteActions.length, 0)
    assert.deepEqual(secondBotMuteActions[1], { groupId: 'group-3', memberId: '10003', duration: 0 })
    assert.deepEqual(findEventByAction(admissionEvents, 'release'), successEvent('session-10003', 'release'))
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

type ReceiveEvent = (event: Partial<Universal.Event>) => void

function groupGuardConfig(port: number) {
  return {
    platform: { baseUrl: `http://127.0.0.1:${port}`, serviceToken: 'test-token' },
    scheduler: { scanIntervalSeconds: 1 },
  }
}

function admissionPolicyTarget(guildID: string) {
  return {
    policyID: `policy-${guildID}`,
    platform: 'qq',
    guildID,
    guardEnabled: true,
    joinHandlingStrategy: 'post_join_guard',
  }
}

function forceQQAdmissionBot(bot: { platform?: string }) {
  bot.platform = 'qq'
}

function forceOneBotAdmissionBot(bot: { platform?: string }) {
  bot.platform = 'onebot'
}

function receiveJoin(
  bot: { receive: ReceiveEvent },
  userID: string,
  guildID: string,
  selfId?: string,
  platform = 'qq',
) {
  bot.receive({
    type: 'guild-member-added',
    selfId,
    platform,
    user: { id: userID, name: userID },
    guild: { id: guildID },
    channel: { id: guildID, type: Universal.Channel.Type.TEXT },
  })
}

function successEvent(sessionID: string, action: string, messageID?: string) {
  return { sessionID, body: { action, success: true, ...(messageID ? { messageID } : {}) } }
}

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
