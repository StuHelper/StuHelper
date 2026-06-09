import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import commands from '@koishijs/plugin-commands'
import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'
import { Universal, type Context } from 'koishi'

import {
  AdmissionRuntimeSettingsStore,
  AdminRuntimeSettingsStore,
  DEFAULT_ADMISSION_RUNTIME_SETTINGS,
  DEFAULT_ADMIN_RUNTIME_SETTINGS,
  GUARD_MEMBER_TABLE,
  type GuardMemberRecord,
  type StuhelperAdminMessageConfig,
} from '@stuhelper/koishi-shared'
import { COMMAND_POLICY_IDS, MODERATION_REVIEW_TABLE, ModerationStore } from '@stuhelper/koishi-moderation-core'

import adminPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('管理员可以查看当前群待认证成员', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, {
    platform: {
      baseUrl: 'http://127.0.0.1:8080',
      serviceToken: 'test-token',
    },
  })

  try {
    await root.start()
    await root.mock.initUser('20001', 5)
    await root.mock.initChannel('group-1')
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'pending-1',
      guildId: 'group-1',
      memberId: '10001',
      deadlineAt: new Date('2026-04-19T10:00:00Z'),
    }))

    const client = root.mock.client('20001', 'group-1')
    await client.shouldReply('群审状态', /待认证成员：\n10001 截止 2026-04-19T10:00:00.000Z/)
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('管理员命令使用 WebUI runtime settings 里的自定义提示文案', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await saveAdminRuntimeMessages(root, {
      guardPendingMembersHeader: '自定义待认证：',
      guardPendingMemberLine: '成员 {memberId} 到期 {deadlineAt}',
      commandAccessDenied: '自定义：无权限。',
    })
    await root.mock.initUser('20001', 5)
    await root.mock.initUser('20002', 3)
    await root.mock.initChannel('group-1')
    const now = new Date('2026-04-19T09:00:00Z')
    const moderationStore = new ModerationStore(root)
    await moderationStore.upsertCommandPolicy({
      commandId: COMMAND_POLICY_IDS.guardStatus,
      minAuthority: 5,
      roles: [],
      createdAt: now,
      updatedAt: now,
    })
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'pending-custom',
      guildId: 'group-1',
      memberId: '10001',
      deadlineAt: new Date('2026-04-19T10:00:00Z'),
    }))

    await root.mock.client('20001', 'group-1').shouldReply(
      '群审状态',
      /自定义待认证：\n成员 10001 到期 2026-04-19T10:00:00.000Z/,
    )
    await root.mock.client('20002', 'group-1').shouldReply('群审状态', '自定义：无权限。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('非管理员执行群审状态命令会被拒绝', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, {
    platform: {
      baseUrl: 'http://127.0.0.1:8080',
      serviceToken: 'test-token',
    },
  })

  try {
    await root.start()
    await root.mock.initUser('20002', 1)
    await root.mock.initChannel('group-1')

    const client = root.mock.client('20002', 'group-1')
    await client.shouldReply('群审状态', '权限不足。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('私聊显式群号也必须按目标群命令策略校验', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await root.mock.initUser('20003', 3)
    await root.mock.initChannel('group-1')

    const now = new Date('2026-04-19T09:00:00Z')
    const moderationStore = new ModerationStore(root)
    await moderationStore.upsertCommandPolicy({
      commandId: COMMAND_POLICY_IDS.guardStatus,
      minAuthority: 5,
      roles: [],
      createdAt: now,
      updatedAt: now,
    })

    const client = root.mock.client('20003')
    await client.shouldReply('群审状态 group-1', '命令权限不足。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('管理员可以查看成员警告次数', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await root.mock.initUser('20011', 3)
    await root.mock.initChannel('group-1')

    const moderationStore = new ModerationStore(root)
    await moderationStore.incrementWarning({
      guildId: 'group-1',
      memberId: '10001',
      reason: '广告刷屏',
      now: new Date('2026-04-19T10:00:00Z'),
    })

    const client = root.mock.client('20011', 'group-1')
    await client.shouldReply('群审警告 10001', '10001 当前累计警告 1 次，最近原因：广告刷屏')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('管理员可以查看待复核队列', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await root.mock.initUser('20012', 3)
    await root.mock.initChannel('group-1')

    const moderationStore = new ModerationStore(root)
    await moderationStore.createReview({
      platform: 'mock',
      botSelfId: '514',
      guildId: 'group-1',
      channelId: 'group-1',
      memberId: '10002',
      actionType: 'kick',
      status: 'pending',
      reason: '广告刷屏',
      operatorMemberId: null,
      resolutionNote: null,
      payload: null,
    })

    const client = root.mock.client('20012', 'group-1')
    await client.shouldReply('群审复核', /待复核队列：\n10002 \[kick] 广告刷屏/)
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('管理员可以提交踢人复核申请', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await root.mock.initUser('20013', 4)
    await root.mock.initChannel('group-1')

    const client = root.mock.client('20013', 'group-1')
    await client.shouldReply('群审踢人申请 10003 广告刷屏', '已提交踢人复核申请：10003，原因：广告刷屏')

    const reviews = await root.database.get(MODERATION_REVIEW_TABLE, {})
    assert.equal(reviews.length, 1)
    assert.equal(reviews[0].memberId, '10003')
    assert.equal(reviews[0].actionType, 'kick')
    assert.equal(reviews[0].status, 'pending')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('管理员可以批量禁言待认证成员', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))
  const muteActions: Array<{ guildId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await root.mock.initUser('20014', 3)
    await root.mock.initChannel('group-1')
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'pending-11',
      guildId: 'group-1',
      memberId: '10011',
      deadlineAt: new Date('2026-04-19T10:00:00Z'),
    }))
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'pending-12',
      guildId: 'group-1',
      memberId: '10012',
      deadlineAt: new Date('2026-04-19T10:00:00Z'),
    }))

    const bot = root.bots[0] as unknown as Universal.Methods
    bot.muteGuildMember = async (guildId, memberId, duration) => {
      muteActions.push({ guildId, memberId, duration })
    }

    const client = root.mock.client('20014', 'group-1')
    await client.shouldReply('群审禁言 120 10011,10012', '已批量禁言 2 名成员。')

    assert.deepEqual(muteActions, [
      { guildId: 'group-1', memberId: '10011', duration: 120000 },
      { guildId: 'group-1', memberId: '10012', duration: 120000 },
    ])
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('管理员命令执行开关由 admission runtime settings 控制', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-admin-'))

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(commands)
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(adminPlugin, createAdminPluginConfig())

  try {
    await root.start()
    await root.mock.initUser('20015', 5)
    await root.mock.initChannel('group-1')
    await new AdmissionRuntimeSettingsStore(
      root,
      DEFAULT_ADMISSION_RUNTIME_SETTINGS,
    ).saveSettings({
      adminCommandsEnabled: false,
    })

    const client = root.mock.client('20015', 'group-1')
    await client.shouldReply('群审状态', '管理员命令已由 StuHelper WebUI 关闭。')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

function createAdminPluginConfig() {
  return {
    platform: {
      baseUrl: 'http://127.0.0.1:8080',
      serviceToken: 'test-token',
    },
  }
}

async function saveAdminRuntimeMessages(
  root: Pick<Context, 'database'>,
  messages: Partial<StuhelperAdminMessageConfig>,
) {
  await new AdminRuntimeSettingsStore(
    root,
    DEFAULT_ADMIN_RUNTIME_SETTINGS,
  ).saveSettings({
    messages: {
      ...messages,
    },
  })
}

function createGuardMemberRecord(input: Pick<GuardMemberRecord, 'id' | 'guildId' | 'memberId' | 'deadlineAt'>): GuardMemberRecord {
  const now = new Date('2026-04-19T09:00:00Z')
  return {
    id: input.id,
    platform: 'mock',
    botSelfId: '514',
    guildId: input.guildId,
    channelId: input.guildId,
    memberId: input.memberId,
    memberName: input.memberId,
    verificationState: 'bound_unverified',
    admissionSessionID: null,
    backendSyncPending: false,
    joinedAt: now,
    deadlineAt: input.deadlineAt,
    nextReminderAt: null,
    manualReviewDeadlineAt: null,
    mutedAt: now,
    reminderSentAt: now,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: now,
    updatedAt: now,
  }
}
