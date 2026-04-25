import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
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
  }
}
