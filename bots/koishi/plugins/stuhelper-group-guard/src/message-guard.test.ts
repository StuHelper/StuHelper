import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'
import { Universal } from 'koishi'

import {
  MODERATION_EVENT_TABLE,
  MODERATION_KEYWORD_RULE_TABLE,
  MODERATION_MESSAGE_LEDGER_TABLE,
  MODERATION_WARNING_TABLE,
} from '@stuhelper/koishi-moderation-core'
import {
  AdmissionRuntimeSettingsStore,
  DEFAULT_ADMISSION_RUNTIME_SETTINGS,
  DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS,
  GroupGuardBehaviorSettingsStore,
} from '@stuhelper/koishi-shared'

import groupGuardPlugin from './index.ts'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('关键词命中后会删除消息、累计警告并执行自动禁言', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-message-guard-'))
  const deleteActions: string[] = []
  const muteActions: Array<{ groupId: string, memberId: string, duration: number }> = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, {
    platform: {
      baseUrl: 'http://127.0.0.1:18080',
      serviceToken: 'test-token',
    },
    scheduler: {
      scanIntervalSeconds: 60,
    },
  })

  try {
    await root.start()
    await root.database.create(MODERATION_KEYWORD_RULE_TABLE, {
      id: 'keyword-1',
      guildId: 'group-1',
      pattern: '广告',
      matchMode: 'includes',
      action: 'delete',
      enabled: true,
      muteSeconds: 0,
      note: '违禁广告',
      createdAt: new Date('2026-06-05T03:00:00.000Z'),
      updatedAt: new Date('2026-06-05T03:00:00.000Z'),
    })
    await saveAdmissionRuntimeSettings(root, { moderationEnabled: true })
    await saveGroupGuardBehaviorSettings(root, {
      moderation: {
        warningThresholdExpression: 'warnings >= 1',
        defaultMuteSeconds: 180,
        antiRecallNotify: true,
      },
    })
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: (event: Partial<Universal.Event>) => void }
    bot.deleteMessage = async (_channelId, messageId) => {
      deleteActions.push(messageId)
    }
    bot.muteGuildMember = async (groupId, memberId, duration) => {
      muteActions.push({ groupId, memberId, duration })
    }
    bot.sendMessage = async () => ['msg-1']

    const client = root.mock.client('10001', 'group-1')
    await client.receive('这是广告内容')

    const ledgerRecord = await waitForValue(async () => {
      const records = await root.database.get(MODERATION_MESSAGE_LEDGER_TABLE, {})
      return records[0]
    })

    await sleep(50)

    assert.deepEqual(deleteActions, [ledgerRecord.messageId])
    assert.deepEqual(muteActions[0], { groupId: 'group-1', memberId: '10001', duration: 180000 })

    const warnings = await root.database.get(MODERATION_WARNING_TABLE, {})
    assert.equal(warnings[0].total, 1)

    const events = await root.database.get(MODERATION_EVENT_TABLE, {})
    assert.ok(events.some((event) => event.type === 'keyword_hit'))
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('撤回事件会记录到事件日志并发送防撤回提示', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-message-guard-'))
  const sentMessages: string[] = []

  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })
  runtime.register(groupGuardPlugin, {
    platform: {
      baseUrl: 'http://127.0.0.1:18080',
      serviceToken: 'test-token',
    },
    scheduler: {
      scanIntervalSeconds: 60,
    },
  })

  try {
    await root.start()
    await saveAdmissionRuntimeSettings(root, { moderationEnabled: true })
    const bot = root.bots[0] as unknown as Universal.Methods & { receive: (event: Partial<Universal.Event>) => void }
    bot.muteGuildMember = async () => {}
    bot.sendMessage = async (_channelId, content) => {
      sentMessages.push(String(content))
      return ['msg-2']
    }

    const client = root.mock.client('10002', 'group-1')
    await client.receive('我要撤回这条消息')

    const ledgerRecord = await waitForValue(async () => {
      const records = await root.database.get(MODERATION_MESSAGE_LEDGER_TABLE, {})
      return records[0]
    })

    bot.receive({
      type: 'message-deleted',
      message: { id: ledgerRecord.messageId, messageId: ledgerRecord.messageId, content: '' },
      user: { id: '10002', name: '10002' },
      guild: { id: 'group-1' },
      channel: { id: 'group-1', type: Universal.Channel.Type.TEXT },
    })

    await sleep(50)

    const events = await root.database.get(MODERATION_EVENT_TABLE, {})
    assert.ok(events.some((event) => event.type === 'message_deleted'))
    assert.ok(sentMessages.some((item) => item.includes('撤回')))
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

function sleep(ms: number) {
  return new Promise((resolve) => setTimeout(resolve, ms))
}

async function waitForValue<T>(load: () => Promise<T | undefined>, timeoutMs = 500, intervalMs = 20) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    const value = await load()
    if (value !== undefined) {
      return value
    }
    await sleep(intervalMs)
  }
  throw new Error('waitForValue timeout')
}

async function saveGroupGuardBehaviorSettings(
  root: {
    database: {
      create(table: string, record: Record<string, unknown>): Promise<unknown>
      get(table: string, query: Record<string, unknown>): Promise<Record<string, unknown>[]>
      set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>): Promise<unknown>
    }
  },
  overrides: Partial<typeof DEFAULT_GROUP_GUARD_BEHAVIOR_SETTINGS>,
) {
  // 走真实 store 保存：settings 读缓存按 (database, table) 共享，
  // 绕过 store 的裸 DB 写不会被运行中的插件实例观察到。
  await new GroupGuardBehaviorSettingsStore(root).saveSettings(overrides)
}

async function saveAdmissionRuntimeSettings(
  root: {
    database: {
      create(table: string, record: Record<string, unknown>): Promise<unknown>
      get(table: string, query: Record<string, unknown>): Promise<Record<string, unknown>[]>
      set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>): Promise<unknown>
    }
  },
  overrides: Partial<typeof DEFAULT_ADMISSION_RUNTIME_SETTINGS>,
) {
  await new AdmissionRuntimeSettingsStore(root, DEFAULT_ADMISSION_RUNTIME_SETTINGS).saveSettings(overrides)
}
