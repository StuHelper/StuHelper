import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import sqlite from '@koishijs/plugin-database-sqlite'
import MockBot from '@koishijs/plugin-mock'
import { Context, Universal } from 'koishi'

import {
  MODERATION_EVENT_TABLE,
  MODERATION_REVIEW_TABLE,
  ModerationActionService,
  ModerationStore,
  registerModerationModels,
} from '@stuhelper/koishi-moderation-core'
import {
  GUARD_GROUP_BINDING_TABLE,
  GUARD_MEMBER_TABLE,
  GUARD_TEMPLATE_TABLE,
  registerGuardMemberModel,
  registerGuardPolicyModels,
  type GuardMemberRecord,
} from '@stuhelper/koishi-shared'

import { registerConsoleListeners, handleGuardBatchAction, handleReviewAction } from './controller'
import { STUHELPER_CONSOLE_SERVICE } from './constants'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('控制台批量踢出会改为创建人工复核队列', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-console-'))
  const kickActions: Array<{ guildId: string, memberId: string, permanent: boolean }> = []

  registerGuardMemberModel(root)
  registerModerationModels(root)
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })

  try {
    await root.start()
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'guard-1',
      guildId: 'group-1',
      memberId: '10001',
    }))
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'guard-2',
      guildId: 'group-1',
      memberId: '10002',
    }))

    const bot = root.bots[0] as unknown as Universal.Methods
    bot.kickGuildMember = async (guildId, memberId, permanent) => {
      kickActions.push({ guildId, memberId, permanent })
    }

    const store = new ModerationStore(root)
    const actions = new ModerationActionService(store)

    const message = await handleGuardBatchAction(root, store, actions, {
      action: 'kick',
      memberIds: ['guard-1', 'guard-2'],
      reason: '待人工复核',
      permanent: true,
    })

    assert.equal(message, '已提交 2 条人工复核申请。')
    assert.deepEqual(kickActions, [])

    const reviews = await root.database.get(MODERATION_REVIEW_TABLE, {})
    assert.equal(reviews.length, 2)
    assert.ok(reviews.every((review) => review.status === 'pending'))
    assert.ok(reviews.every((review) => review.actionType === 'kick_and_block'))

    const events = await root.database.get(MODERATION_EVENT_TABLE, {})
    assert.equal(events.filter((event) => event.type === 'review_created').length, 2)

    const guardMembers = await root.database.get(GUARD_MEMBER_TABLE, {})
    assert.ok(guardMembers.every((record) => record.kickedAt === null))
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('控制台执行复核会真正踢人并更新状态', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-console-'))
  const kickActions: Array<{ guildId: string, memberId: string, permanent: boolean }> = []

  registerGuardMemberModel(root)
  registerModerationModels(root)
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })
  runtime.register(MockBot, { selfId: '514' })

  try {
    await root.start()
    await root.database.create(GUARD_MEMBER_TABLE, createGuardMemberRecord({
      id: 'guard-3',
      guildId: 'group-1',
      memberId: '10003',
    }))

    const bot = root.bots[0] as unknown as Universal.Methods
    bot.sendMessage = async () => ['msg-1']
    bot.kickGuildMember = async (guildId, memberId, permanent) => {
      kickActions.push({ guildId, memberId, permanent })
    }

    const store = new ModerationStore(root)
    const actions = new ModerationActionService(store)
    const review = await store.createReview({
      platform: 'mock',
      botSelfId: '514',
      guildId: 'group-1',
      channelId: 'group-1',
      memberId: '10003',
      actionType: 'kick_and_block',
      status: 'pending',
      reason: '人工确认',
      operatorMemberId: null,
      resolutionNote: null,
      payload: { source: 'console-test' },
    })

    const message = await handleReviewAction(root, store, actions, {
      reviewId: review.id,
      action: 'execute',
      note: '批准执行',
    })

    assert.equal(message, '已执行复核动作：10003')
    assert.deepEqual(kickActions, [{ guildId: 'group-1', memberId: '10003', permanent: true }])

    const [updatedReview] = await root.database.get(MODERATION_REVIEW_TABLE, { id: review.id })
    assert.equal(updatedReview.status, 'executed')
    assert.equal(updatedReview.operatorMemberId, 'console')
    assert.equal(updatedReview.resolutionNote, '批准执行')

    const [updatedGuardMember] = await root.database.get(GUARD_MEMBER_TABLE, { id: 'guard-3' })
    assert.ok(updatedGuardMember.kickedAt instanceof Date)
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('控制台保存群模板与群绑定事件会写入 SQLite', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-console-'))

  registerGuardPolicyModels(root)
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })

  const consoleHarness = attachConsoleHarness(root)
  registerConsoleListeners(root)

  try {
    await root.start()

    const templateMessage = await consoleHarness.emit('stuhelper-console/save-guard-template', {
      id: 'dormitory',
      name: '宿舍群模板',
      muteDurationSeconds: 900,
      kickAfterMinutes: 45,
      reminderTemplate: '请先完成 StuHelper 注册、QQ 绑定与学生认证。',
      exemptUsers: ['10086', '10010'],
      enabled: true,
    })

    const bindingMessage = await consoleHarness.emit('stuhelper-console/save-guard-binding', {
      platform: 'onebot',
      guildId: '2026001',
      templateId: 'dormitory',
      enabled: true,
      note: '  2026 级宿舍群  ',
    })

    assert.equal(templateMessage, '已保存群模板：宿舍群模板')
    assert.equal(bindingMessage, '已保存群绑定：onebot/2026001')

    const [template] = await root.database.get(GUARD_TEMPLATE_TABLE, { id: 'dormitory' })
    assert.equal(template.name, '宿舍群模板')
    assert.equal(template.muteDurationSeconds, 900)
    assert.deepEqual(template.exemptUsers, ['10086', '10010'])

    const [binding] = await root.database.get(GUARD_GROUP_BINDING_TABLE, { id: 'onebot:2026001' })
    assert.equal(binding.platform, 'onebot')
    assert.equal(binding.guildId, '2026001')
    assert.equal(binding.templateId, 'dormitory')
    assert.equal(binding.note, '2026 级宿舍群')

    assert.equal(consoleHarness.authorities.get('stuhelper-console/save-guard-template'), 4)
    assert.equal(consoleHarness.authorities.get('stuhelper-console/save-guard-binding'), 4)
    assert.deepEqual(consoleHarness.refreshes, [
      STUHELPER_CONSOLE_SERVICE,
      STUHELPER_CONSOLE_SERVICE,
    ])
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

function createGuardMemberRecord(
  input: Pick<GuardMemberRecord, 'id' | 'guildId' | 'memberId'>,
): GuardMemberRecord {
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
    joinedAt: now,
    deadlineAt: new Date('2026-04-19T10:00:00Z'),
    mutedAt: now,
    reminderSentAt: now,
    releasedAt: null,
    kickedAt: null,
    lastError: null,
    createdAt: now,
    updatedAt: now,
  }
}

function attachConsoleHarness(root: Context) {
  const listeners = new Map<string, (input?: unknown) => Promise<unknown> | unknown>()
  const authorities = new Map<string, number>()
  const refreshes: string[] = []

  Object.assign(root, {
    console: {
      addListener(name: string, listener: (input?: unknown) => Promise<unknown> | unknown, options?: { authority?: number }) {
        listeners.set(name, listener)
        authorities.set(name, options?.authority ?? 0)
      },
      async refresh(service: string) {
        refreshes.push(service)
      },
    },
  })

  return {
    refreshes,
    authorities,
    async emit(name: string, input?: unknown) {
      const listener = listeners.get(name)
      assert.ok(listener, `console listener not found: ${name}`)
      return listener(input)
    },
  }
}
