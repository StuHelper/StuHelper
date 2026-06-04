import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { createServer, type IncomingMessage, type ServerResponse } from 'node:http'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import commands from '@koishijs/plugin-commands'
import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'
import { Universal } from 'koishi'

import {
  ModerationStore,
  MODERATION_EVENT_TABLE,
  MODERATION_FUN_PROFILE_TABLE,
  MODERATION_REPORT_TABLE,
} from '@stuhelper/koishi-moderation-core'
import { GUARD_MEMBER_TABLE } from '@stuhelper/koishi-shared'

import groupGuardPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('举报命令会创建举报记录并返回人工处理提示', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig())

  try {
    await root.start()
    await root.mock.initUser('10001', 1)
    await root.mock.initChannel('group-1')

    const client = root.mock.client('10001', 'group-1')
    await client.shouldReply('举报 10002 广告刷屏', '举报已记录。当前未启用 AI 审核，事件已进入人工处理范围。')

    const reports = await root.database.get(MODERATION_REPORT_TABLE, {})
    assert.equal(reports.length, 1)
    assert.equal(reports[0].targetMemberId, '10002')
    assert.equal(reports[0].reason, '广告刷屏')

    const events = await root.database.get(MODERATION_EVENT_TABLE, {})
    assert.ok(events.some((event) => event.type === 'report_created'))
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('抽禁言命令会按保底规则写入画像并执行禁言', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig({
    fun: {
      diceSides: 100,
      muteLotteryBaseSeconds: 60,
      muteLotteryMaxSeconds: 600,
      muteLotteryPityThreshold: 1,
      muteLotteryPitySeconds: 300,
    },
  }))

  try {
    await root.start()
    await root.mock.initUser('10003', 1)
    await root.mock.initChannel('group-1')

    const bot = root.bots[0] as unknown as Universal.Methods
    bot.muteGuildMember = async (guildId, memberId, duration) => {
      muteActions.push({ guildId, memberId, duration })
    }

    const client = root.mock.client('10003', 'group-1')
    await client.shouldReply('抽禁言', '保底触发，10003 本次自助禁言 300 秒。')

    assert.deepEqual(muteActions[0], { guildId: 'group-1', memberId: '10003', duration: 300000 })

    const profiles = await root.database.get(MODERATION_FUN_PROFILE_TABLE, {})
    assert.equal(profiles.length, 1)
    assert.equal(profiles[0].muteDrawCount, 0)

    const events = await root.database.get(MODERATION_EVENT_TABLE, {})
    assert.ok(events.some((event) => event.type === 'action_executed'))
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('命令权限策略会限制举报命令并允许角色放行', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig())

  try {
    await root.start()
    await root.mock.initUser('10004', 1)
    await root.mock.initChannel('group-1')

    const moderationStore = new ModerationStore(root)
    const now = new Date()
    await moderationStore.upsertCommandPolicy({
      commandId: 'report',
      roles: ['moderator'],
      minAuthority: 4,
      createdAt: now,
      updatedAt: now,
    })

    const client = root.mock.client('10004', 'group-1')
    await client.shouldReply('举报 10005 广告刷屏', '命令权限不足。')

    await moderationStore.setMemberRoles('group-1', '10004', ['moderator'])
    await client.shouldReply('举报 10005 广告刷屏', '举报已记录。当前未启用 AI 审核，事件已进入人工处理范围。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('公开命令可以关闭以避免接管既有生产命令', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig({
    commands: { enabled: false },
  }))

  try {
    await root.start()
    assert.equal(root.$commander.resolve('举报'), undefined)
    assert.equal(root.$commander.resolve('骰子'), undefined)
    assert.equal(root.$commander.resolve('抽禁言'), undefined)
    assert.notEqual(root.$commander.resolve('查询入群认证'), undefined)
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('入群认证管理员命令可以查询、重发和重新生成认证链接', async () => {
  const requests: CapturedAdmissionAdminRequest[] = []
  const server = createServer((req, res) => respondAdmissionAdminRequest(req, res, requests))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig({
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    commands: { enabled: false },
    moderation: { enabled: false },
    freshmanForward: { enabled: false },
    scheduler: { scanIntervalSeconds: 3600 },
  }))

  try {
    await root.start()
    await root.mock.initUser('90001', 5)
    await root.mock.initChannel('group-1')
    const bot = root.bots[0] as unknown as Universal.Methods & { platform?: string }
    bot.platform = 'onebot'
    bot.muteGuildMember = async (guildId, memberId, duration) => {
      muteActions.push({ guildId, memberId, duration })
    }

    const client = root.mock.client('90001', 'group-1')
    await root.database.create(GUARD_MEMBER_TABLE, activeAdmissionGuardRecord())
    await assertSingleReply(client, '查询入群认证 10001', /状态：已绑定 QQ，等待学生认证/)
    await assertSingleReply(
      client,
      '查询入群认证 10002',
      /QQ 绑定：已完成[\s\S]*学生认证：未通过[\s\S]*曾绑定账号但未完成学生认证/,
    )
    await assertSingleReply(client, '重发认证链接 10001', /https:\/\/join\.stuhelper\.com\/verify\/token-current/)
    await assertSingleReply(client, '重新生成认证链接 10001', /https:\/\/join\.stuhelper\.com\/verify\/token-new/)
    await waitForRequestCount(requests, 6)

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, { id: 'qq:514:group-1:10001' })
    assert.ok(record)
    assert.equal(record.admissionSessionID, 'session-token-new')
    assert.equal(record.backendSyncPending, false)
    assert.ok(record.reminderSentAt instanceof Date)
    assert.equal(record.releasedAt, null)
    assert.equal(record.kickedAt, null)

    assert.deepEqual(requests.map((item) => [item.method, item.path]), [
      ['GET', '/api/v1/bot/admission/sessions/member?platform=qq&guildID=group-1&qqID=10001'],
      ['GET', '/api/v1/bot/admission/sessions/member?platform=qq&guildID=group-1&qqID=10002'],
      ['POST', '/api/v1/bot/admission/sessions/member/resend'],
      ['POST', '/api/v1/bot/admission/sessions/session-token-current/events'],
      ['POST', '/api/v1/bot/admission/sessions/member/regenerate'],
      ['POST', '/api/v1/bot/admission/sessions/session-token-new/events'],
    ])
    assert.ok(requests.every((item) => item.authorization === 'Bearer test-token'))
    assert.deepEqual(requests[2].body, { platform: 'qq', guildID: 'group-1', qqID: '10001' })
    assert.equal(requests[3].body.action, 'remind')
    assert.equal(requests[3].body.success, true)
    assert.equal(requests[4].body.botSelfID, '514')
    assert.equal(requests[5].body.action, 'remind')
    assert.equal(requests[5].body.success, true)
    assert.equal(muteActions[0].guildId, 'group-1')
    assert.equal(muteActions[0].memberId, '10001')
    assert.ok(muteActions[0].duration > 0)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('入群认证管理员命令会抑制短时间重复重新生成链接', async () => {
  const requests: CapturedAdmissionAdminRequest[] = []
  const server = createServer((req, res) => respondAdmissionAdminRequest(req, res, requests))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig({
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    commands: { enabled: false },
    moderation: { enabled: false },
    freshmanForward: { enabled: false },
    scheduler: { scanIntervalSeconds: 3600 },
  }))

  try {
    await root.start()
    await root.mock.initUser('90001', 5)
    await root.mock.initChannel('group-1')
    const bot = root.bots[0] as unknown as Universal.Methods & { platform?: string }
    bot.platform = 'onebot'
    bot.muteGuildMember = async (guildId, memberId, duration) => {
      muteActions.push({ guildId, memberId, duration })
    }

    await root.database.create(GUARD_MEMBER_TABLE, activeAdmissionGuardRecord())
    const client = root.mock.client('90001', 'group-1')

    const firstReplies = await client.receive('重新生成认证链接 10001')
    assert.equal(firstReplies.length, 1)
    assert.match(firstReplies[0], /https:\/\/join\.stuhelper\.com\/verify\/token-new/)

    const duplicateReplies = await client.receive('重新生成认证链接 10001')
    assert.deepEqual(duplicateReplies, [])

    assert.equal(requests.filter((item) => item.path === '/api/v1/bot/admission/sessions/member/regenerate').length, 1)
    assert.equal(requests.filter((item) => item.path.endsWith('/events')).length, 1)
    assert.equal(muteActions.length, 1)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('重新生成认证链接遇到已认证 QQ 时解除禁言且不重发链接', async () => {
  const requests: CapturedAdmissionAdminRequest[] = []
  const server = createServer((req, res) => respondAdmissionAdminRequest(req, res, requests))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig({
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    commands: { enabled: false },
    moderation: { enabled: false },
    freshmanForward: { enabled: false },
    scheduler: { scanIntervalSeconds: 3600 },
  }))

  try {
    await root.start()
    await root.mock.initUser('90001', 5)
    await root.mock.initChannel('group-1')
    const bot = root.bots[0] as unknown as Universal.Methods & { platform?: string }
    bot.platform = 'onebot'
    bot.muteGuildMember = async (guildId, memberId, duration) => {
      muteActions.push({ guildId, memberId, duration })
    }

    await root.database.create(GUARD_MEMBER_TABLE, activeAdmissionGuardRecord('10003'))
    const client = root.mock.client('90001', 'group-1')

    const replies = await client.receive('重新生成认证链接 10003')
    assert.equal(replies.length, 1)
    assert.match(replies[0], /10003[\s\S]*已完成 StuHelper 学生身份认证[\s\S]*已解除禁言/)
    assert.doesNotMatch(replies[0], /https:\/\/join\.stuhelper\.com\/verify\//)

    await waitForRequestCount(requests, 2)
    assert.deepEqual(requests.map((item) => [item.method, item.path]), [
      ['POST', '/api/v1/bot/admission/sessions/member/regenerate'],
      ['POST', '/api/v1/bot/admission/sessions/session-token-verified/events'],
    ])
    assert.equal(requests[1].body.action, 'release')
    assert.equal(requests[1].body.success, true)
    assert.deepEqual(muteActions, [{ guildId: 'group-1', memberId: '10003', duration: 0 }])

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, { id: 'qq:514:group-1:10003' })
    assert.ok(record)
    assert.equal(record.admissionSessionID, 'session-token-verified')
    assert.ok(record.releasedAt instanceof Date)
    assert.equal(record.reminderSentAt, null)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('入群认证管理员命令可以跳过本群认证、清空失败次数和解除拉黑', async () => {
  const requests: CapturedAdmissionAdminRequest[] = []
  const server = createServer((req, res) => respondAdmissionAdminRequest(req, res, requests))
  await new Promise<void>((resolve) => server.listen(0, resolve))
  const address = server.address()
  assert.ok(address && typeof address === 'object')

  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-commands-'))
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, createGroupGuardConfig({
    platform: {
      baseUrl: `http://127.0.0.1:${address.port}`,
      serviceToken: 'test-token',
    },
    commands: { enabled: false },
    moderation: { enabled: false },
    freshmanForward: { enabled: false },
    scheduler: { scanIntervalSeconds: 3600 },
  }))

  try {
    await root.start()
    await root.mock.initUser('90001', 5)
    await root.mock.initChannel('group-1')
    const bot = root.bots[0] as unknown as Universal.Methods & { platform?: string }
    bot.platform = 'onebot'
    bot.muteGuildMember = async (guildId, memberId, duration) => {
      muteActions.push({ guildId, memberId, duration })
    }

    await root.database.create(GUARD_MEMBER_TABLE, activeAdmissionGuardRecord('10004'))
    const client = root.mock.client('90001', 'group-1')

    await assertSingleReply(client, '跳过入群认证 10004', /10004[\s\S]*已跳过本群入群认证[\s\S]*不代表 StuHelper 学生认证已通过/)
    await assertSingleReply(client, '清空入群未认证次数 10004', /QQ 10004[\s\S]*原次数：2/)
    await assertSingleReply(client, '解除入群拉黑 10004', /已解除 QQ 10004 在本群的入群拉黑状态/)

    await waitForRequestCount(requests, 3)
    assert.deepEqual(requests.map((item) => [item.method, item.path]), [
      ['POST', '/api/v1/bot/admission/sessions/member/skip'],
      ['POST', '/api/v1/bot/admission/failures/reset'],
      ['POST', '/api/v1/bot/member-blacklist/release-by-subject'],
    ])
    assert.deepEqual(requests[0].body, {
      platform: 'qq',
      guildID: 'group-1',
      qqID: '10004',
      operatorQQID: '90001',
    })
    assert.equal(requests[2].body.releaseReasonCode, 'release_only')
    assert.equal(requests[2].body.subjectID, '10004')
    assert.deepEqual(muteActions, [{ guildId: 'group-1', memberId: '10004', duration: 0 }])

    const [record] = await root.database.get(GUARD_MEMBER_TABLE, { id: 'qq:514:group-1:10004' })
    assert.ok(record)
    assert.equal(record.admissionSessionID, 'session-token-current')
    assert.ok(record.releasedAt instanceof Date)
  } finally {
    runtime.dispose()
    await closeServer(server)
    await rm(tempDir, { recursive: true, force: true })
  }
})

function createGroupGuardConfig(overrides?: Partial<ReturnType<typeof createBaseGroupGuardConfig>>) {
  return {
    ...createBaseGroupGuardConfig(),
    ...overrides,
    guard: {
      ...createBaseGroupGuardConfig().guard,
      ...overrides?.guard,
    },
    scheduler: {
      ...createBaseGroupGuardConfig().scheduler,
      ...overrides?.scheduler,
    },
    moderation: {
      ...createBaseGroupGuardConfig().moderation,
      ...overrides?.moderation,
    },
    fun: {
      ...createBaseGroupGuardConfig().fun,
      ...overrides?.fun,
    },
    ai: {
      ...createBaseGroupGuardConfig().ai,
      ...overrides?.ai,
    },
    commands: {
      ...createBaseGroupGuardConfig().commands,
      ...overrides?.commands,
    },
    admissionCommands: {
      ...createBaseGroupGuardConfig().admissionCommands,
      ...overrides?.admissionCommands,
    },
    freshmanForward: {
      ...createBaseGroupGuardConfig().freshmanForward,
      ...overrides?.freshmanForward,
    },
  }
}

function createBaseGroupGuardConfig() {
  return {
    platform: {
      baseUrl: 'http://127.0.0.1:18080',
      serviceToken: 'test-token',
    },
    guard: {
      targetGroups: ['group-1'],
      muteDurationSeconds: 600,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成认证。',
      exemptUsers: [],
    },
    scheduler: {
      scanIntervalSeconds: 60,
    },
    moderation: {
      repeatThreshold: 3,
      repeatWindowSize: 3,
      warningThresholdExpression: 'warnings >= 3',
      defaultMuteSeconds: 180,
      antiRecallNotify: true,
      keywordRules: [],
    },
    fun: {
      diceSides: 100,
      muteLotteryBaseSeconds: 60,
      muteLotteryMaxSeconds: 600,
      muteLotteryPityThreshold: 5,
      muteLotteryPitySeconds: 300,
    },
    ai: {
      enabled: false,
      endpoint: '',
      apiKey: '',
      model: '',
    },
    commands: {
      enabled: true,
    },
    admissionCommands: {
      enabled: true,
      minAuthority: 4,
      operatorQQIDs: [],
    },
    freshmanForward: {
      enabled: false,
    },
  }
}

interface CapturedAdmissionAdminRequest {
  readonly method: string
  readonly path: string
  readonly authorization: string
  readonly body: Record<string, unknown>
}

async function respondAdmissionAdminRequest(
  req: IncomingMessage,
  res: ServerResponse,
  requests: CapturedAdmissionAdminRequest[],
) {
  const url = new URL(req.url || '/', 'http://127.0.0.1')
  const body = req.method === 'POST' ? await readJSONBody(req) : {}
  requests.push({
    method: req.method || 'GET',
    path: url.pathname + url.search,
    authorization: req.headers.authorization || '',
    body,
  })
  res.setHeader('content-type', 'application/json')
  if (req.method === 'GET' && url.pathname === '/api/v1/bot/admission/sessions/member') {
    if (url.searchParams.get('qqID') === '10002') {
      res.end(JSON.stringify({
        success: true,
        data: admissionAdminSession('expired_kicked', 'token-expired', { userID: 6 }),
      }))
      return
    }
    res.end(JSON.stringify({ success: true, data: admissionAdminSession('linked', 'token-current') }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/bot/admission/sessions/member/resend') {
    res.end(JSON.stringify({ success: true, data: admissionAdminSession('linked', 'token-current') }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/bot/admission/sessions/member/regenerate') {
    if (body.qqID === '10003') {
      res.statusCode = 201
      res.end(JSON.stringify({
        success: true,
        data: {
          session: admissionAdminSession('verified', 'token-verified', {
            qqID: '10003',
            userID: 7,
            tokenConsumedAt: new Date().toISOString(),
          }),
          token: 'token-verified',
          authURL: 'https://join.stuhelper.com/verify/token-verified',
        },
      }))
      return
    }
    res.statusCode = 201
    res.end(JSON.stringify({
      success: true,
      data: {
        session: admissionAdminSession('joined_muted', 'token-new'),
        token: 'token-new',
        authURL: 'https://join.stuhelper.com/verify/token-new',
      },
    }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/bot/admission/sessions/member/skip') {
    res.end(JSON.stringify({
      success: true,
      data: admissionAdminSession('cancelled', 'token-current', {
        qqID: body.qqID,
        cancelledAt: new Date().toISOString(),
      }),
    }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/bot/admission/failures/reset') {
    res.end(JSON.stringify({
      success: true,
      data: {
        platform: body.platform,
        guildID: body.guildID,
        qqID: body.qqID,
        previousFailureCount: 2,
      },
    }))
    return
  }
  if (req.method === 'POST' && url.pathname === '/api/v1/bot/member-blacklist/release-by-subject') {
    res.end(JSON.stringify({
      success: true,
      data: {
        id: 'blacklist-1',
        platform: body.platform,
        subjectType: body.subjectType,
        subjectID: body.subjectID,
        scopeType: body.scopeType,
        guildID: body.guildID,
        source: 'admission_failure',
        reasonCode: 'admission_timeout_limit',
        reasonText: 'admission failure limit reached',
        metadata: {},
        createdByType: 'system',
        createdByID: 'system',
        createdFrom: 'admission_worker',
        releasedAt: new Date().toISOString(),
        releasedByType: 'qq_operator',
        releasedByID: body.operatorQQID,
        releaseReasonCode: body.releaseReasonCode,
        releaseReason: body.releaseReason,
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      },
    }))
    return
  }
  if (
    req.method === 'POST' &&
    url.pathname.startsWith('/api/v1/bot/admission/sessions/') &&
    url.pathname.endsWith('/events')
  ) {
    res.end(JSON.stringify({ success: true }))
    return
  }
  res.statusCode = 404
  res.end(JSON.stringify({ success: false, error: { message: 'not found' } }))
}

function admissionAdminSession(status: string, token: string, overrides: Record<string, unknown> = {}) {
  const now = Date.now()
  return {
    id: `session-${token}`,
    platform: 'qq',
    guildID: 'group-1',
    channelID: 'group-1',
    qqID: '10001',
    status,
    authURL: `https://join.stuhelper.com/verify/${token}`,
    tokenExpiresAt: new Date(now + 60 * 60 * 1000).toISOString(),
    linkWaitDeadlineAt: new Date(now + 60 * 60 * 1000).toISOString(),
    submissionWaitDeadlineAt: new Date(now + 24 * 60 * 60 * 1000).toISOString(),
    initialMuteUntil: new Date(now + 30 * 24 * 60 * 60 * 1000).toISOString(),
    projectionPending: false,
    ...overrides,
  }
}

function activeAdmissionGuardRecord(qqID = '10001') {
  const now = new Date()
  return {
    id: `qq:514:group-1:${qqID}`,
    platform: 'qq',
    botSelfId: '514',
    guildId: 'group-1',
    channelId: 'group-1',
    memberId: qqID,
    memberName: qqID,
    verificationState: 'bound_unverified',
    admissionSessionID: 'session-token-current',
    backendSyncPending: false,
    joinedAt: now,
    deadlineAt: new Date(now.getTime() + 60 * 60 * 1000),
    nextReminderAt: new Date(now.getTime() + 10 * 60 * 1000),
    manualReviewDeadlineAt: null,
    mutedAt: now,
    reminderSentAt: null,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: now,
    updatedAt: now,
  }
}

async function readJSONBody(req: IncomingMessage) {
  const chunks: Buffer[] = []
  for await (const chunk of req) {
    chunks.push(Buffer.isBuffer(chunk) ? chunk : Buffer.from(chunk))
  }
  if (!chunks.length) return {}
  return JSON.parse(Buffer.concat(chunks).toString()) as Record<string, unknown>
}

async function closeServer(server: ReturnType<typeof createServer>) {
  await new Promise<void>((resolve, reject) => {
    server.close((error) => {
      if (error) reject(error)
      else resolve()
    })
  })
}

async function waitForRequestCount(requests: readonly unknown[], expected: number) {
  const deadline = Date.now() + 1000
  while (requests.length < expected && Date.now() < deadline) {
    await new Promise((resolve) => setTimeout(resolve, 10))
  }
}

async function assertSingleReply(
  client: { receive(message: string): Promise<string[]> },
  message: string,
  expected: RegExp,
) {
  const replies = await client.receive(message)
  assert.equal(replies.length, 1)
  assert.match(replies[0], expected)
}
