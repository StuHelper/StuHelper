import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import sqlite from '@koishijs/plugin-database-sqlite'
import { Context } from 'koishi'

import {
  MODERATION_COMMAND_POLICY_TABLE,
  MODERATION_FUN_PROFILE_TABLE,
  MODERATION_KEYWORD_RULE_TABLE,
  MODERATION_MESSAGE_LEDGER_TABLE,
  MODERATION_REVIEW_TABLE,
} from './constants'
import { registerModerationModels } from './models'
import { ModerationStore } from './store'
import type {
  CommandPolicyRecord,
  FunProfileRecord,
  KeywordRuleRecord,
  MessageLedgerRecord,
  ReviewQueueRecord,
} from './types'

test('ModerationStore updates existing records without primary key churn', async () => {
  const createdAt = new Date('2026-06-05T03:00:00.000Z')
  const updatedAt = new Date('2026-06-05T04:00:00.000Z')
  const writes: Array<{
    table: string
    query: unknown
    patch: Record<string, unknown>
  }> = []
  const existingByTable: Record<string, unknown[]> = {
    [MODERATION_MESSAGE_LEDGER_TABLE]: [createMessage({ createdAt })],
    [MODERATION_KEYWORD_RULE_TABLE]: [createKeywordRule({ createdAt })],
    [MODERATION_COMMAND_POLICY_TABLE]: [createCommandPolicy({ createdAt })],
    [MODERATION_FUN_PROFILE_TABLE]: [createFunProfile({ createdAt })],
  }
  const store = new ModerationStore({
    database: {
      async get(table: string) {
        return existingByTable[table] || []
      },
      async set(table: string, query: unknown, patch: Record<string, unknown>) {
        writes.push({ table, query, patch })
      },
      async create() {
        assert.fail('existing records should be updated, not created')
      },
    },
  } as never)

  await store.saveMessage(createMessage({
    createdAt: updatedAt,
    content: 'new content',
    deletedAt: updatedAt,
  }))
  await store.upsertKeywordRule(createKeywordRule({
    createdAt: updatedAt,
    updatedAt,
    enabled: false,
  }))
  await store.upsertCommandPolicy(createCommandPolicy({
    createdAt: updatedAt,
    updatedAt,
    minAuthority: 4,
  }))
  await store.saveFunProfile(createFunProfile({
    createdAt: updatedAt,
    updatedAt,
    muteDrawCount: 7,
  }))

  assert.equal(writes.length, 4)
  assertUpdatePatch(writes[0], MODERATION_MESSAGE_LEDGER_TABLE, { messageId: 'message-1' }, [
    'messageId',
    'platform',
    'botSelfId',
    'guildId',
    'channelId',
    'memberId',
    'createdAt',
  ])
  assert.equal(writes[0].patch.content, 'new content')
  assert.equal(writes[0].patch.normalizedContent, 'content')
  assert.equal(writes[0].patch.quoteMessageId, null)
  assert.equal(writes[0].patch.deletedAt, updatedAt)

  assertUpdatePatch(writes[1], MODERATION_KEYWORD_RULE_TABLE, { id: 'rule-1' }, [
    'id',
    'createdAt',
  ])
  assert.equal(writes[1].patch.enabled, false)
  assert.equal(writes[1].patch.updatedAt, updatedAt)

  assertUpdatePatch(writes[2], MODERATION_COMMAND_POLICY_TABLE, { commandId: 'report' }, [
    'commandId',
    'createdAt',
  ])
  assert.equal(writes[2].patch.minAuthority, 4)
  assert.equal(writes[2].patch.updatedAt, updatedAt)

  assertUpdatePatch(writes[3], MODERATION_FUN_PROFILE_TABLE, { memberId: '10001' }, [
    'memberId',
    'createdAt',
  ])
  assert.equal(writes[3].patch.muteDrawCount, 7)
  assert.equal(writes[3].patch.updatedAt, updatedAt)
})

test('ModerationStore reads and deletes keyword rules by id', async () => {
  const rule = createKeywordRule()
  const deletes: Array<{ table: string; query: Record<string, unknown> }> = []
  const store = new ModerationStore({
    database: {
      async get(table: string, query: Record<string, unknown>) {
        assert.equal(table, MODERATION_KEYWORD_RULE_TABLE)
        assert.deepEqual(query, { id: 'rule-1' })
        return [rule]
      },
      async remove(table: string, query: Record<string, unknown>) {
        deletes.push({ table, query })
      },
    },
  } as never)

  assert.deepEqual(await store.getKeywordRule('rule-1'), rule)
  await store.deleteKeywordRule('rule-1')
  assert.deepEqual(deletes, [{
    table: MODERATION_KEYWORD_RULE_TABLE,
    query: { id: 'rule-1' },
  }])
})

test('ModerationStore owns optimistic review state transitions', async () => {
  const review = createReview()
  const rejectAt = new Date('2026-06-05T04:10:00.000Z')
  const claimAt = new Date('2026-06-05T04:11:00.000Z')
  const executedAt = new Date('2026-06-05T04:12:00.000Z')
  const rolledBackAt = new Date('2026-06-05T04:13:00.000Z')
  const writes: Array<{
    query: Record<string, unknown>
    patch: Record<string, unknown>
  }> = []
  const store = new ModerationStore({
    database: {
      async set(table: string, query: Record<string, unknown>, patch: Record<string, unknown>) {
        assert.equal(table, MODERATION_REVIEW_TABLE)
        writes.push({ query, patch })
        return { matched: 1, modified: 1 }
      },
    },
  } as never)

  assert.equal(await store.tryUpdatePendingReview({
    review,
    status: 'rejected',
    operatorMemberId: 'admin-42',
    resolutionNote: '证据不足',
    updatedAt: rejectAt,
  }), true)
  assert.equal(await store.tryClaimPendingReview({
    review,
    operatorMemberId: 'admin-42',
    resolutionNote: '证据充分',
    claimedAt: claimAt,
  }), true)
  assert.equal(await store.tryFinalizeClaimedReview({
    reviewId: review.id,
    operatorMemberId: 'admin-42',
    resolutionNote: '证据充分',
    claimedAt: claimAt,
    executedAt,
  }), true)
  await store.rollbackClaimedReview({
    reviewId: review.id,
    claimedAt: claimAt,
    rolledBackAt,
  })

  assert.deepEqual(writes, [
    {
      query: { id: 'rv-1', status: 'pending', updatedAt: review.updatedAt },
      patch: {
        status: 'rejected',
        operatorMemberId: 'admin-42',
        resolutionNote: '证据不足',
        updatedAt: rejectAt,
      },
    },
    {
      query: { id: 'rv-1', status: 'pending', updatedAt: review.updatedAt },
      patch: {
        status: 'approved',
        operatorMemberId: 'admin-42',
        resolutionNote: '证据充分',
        updatedAt: claimAt,
      },
    },
    {
      query: { id: 'rv-1', status: 'approved', updatedAt: claimAt },
      patch: {
        status: 'executed',
        operatorMemberId: 'admin-42',
        resolutionNote: '证据充分',
        updatedAt: executedAt,
      },
    },
    {
      query: { id: 'rv-1', status: 'approved', updatedAt: claimAt },
      patch: {
        status: 'pending',
        operatorMemberId: null,
        resolutionNote: null,
        updatedAt: rolledBackAt,
      },
    },
  ])
  for (const write of writes) {
    assert.equal('id' in write.patch, false)
    assert.equal('platform' in write.patch, false)
    assert.equal('botSelfId' in write.patch, false)
    assert.equal('guildId' in write.patch, false)
    assert.equal('memberId' in write.patch, false)
  }
})

test('ModerationStore reports lost review claims without throwing', async () => {
  const store = new ModerationStore({
    database: {
      async set() {
        return { matched: 0, modified: 0 }
      },
    },
  } as never)

  assert.equal(await store.tryClaimPendingReview({
    review: createReview(),
    operatorMemberId: 'admin-42',
    resolutionNote: null,
    claimedAt: new Date('2026-06-05T04:11:00.000Z'),
  }), false)
})

function assertUpdatePatch(
  write: { table: string; query: unknown; patch: Record<string, unknown> },
  table: string,
  query: unknown,
  omittedFields: string[],
) {
  assert.equal(write.table, table)
  assert.deepEqual(write.query, query)
  for (const field of omittedFields) {
    assert.equal(field in write.patch, false, `${table} update patch should omit ${field}`)
  }
}

function createMessage(overrides: Partial<MessageLedgerRecord> = {}): MessageLedgerRecord {
  return {
    messageId: 'message-1',
    platform: 'qq',
    botSelfId: 'bot-1',
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    content: 'content',
    normalizedContent: 'content',
    quoteMessageId: null,
    createdAt: new Date('2026-06-05T03:00:00.000Z'),
    deletedAt: null,
    ...overrides,
  }
}

function createKeywordRule(overrides: Partial<KeywordRuleRecord> = {}): KeywordRuleRecord {
  return {
    id: 'rule-1',
    guildId: 'guild-1',
    pattern: 'spam',
    matchMode: 'includes',
    action: 'warn',
    enabled: true,
    muteSeconds: 0,
    note: null,
    createdAt: new Date('2026-06-05T03:00:00.000Z'),
    updatedAt: new Date('2026-06-05T03:00:00.000Z'),
    ...overrides,
  }
}

function createCommandPolicy(overrides: Partial<CommandPolicyRecord> = {}): CommandPolicyRecord {
  return {
    commandId: 'report',
    roles: ['admin'],
    minAuthority: 3,
    createdAt: new Date('2026-06-05T03:00:00.000Z'),
    updatedAt: new Date('2026-06-05T03:00:00.000Z'),
    ...overrides,
  }
}

function createFunProfile(overrides: Partial<FunProfileRecord> = {}): FunProfileRecord {
  return {
    memberId: '10001',
    muteDrawCount: 1,
    pityBonus: 0,
    lastDrawAt: null,
    createdAt: new Date('2026-06-05T03:00:00.000Z'),
    updatedAt: new Date('2026-06-05T03:00:00.000Z'),
    ...overrides,
  }
}

test('listPendingReviews 对空 guildId 返回空数组且不触发全表查询', async () => {
  const queries: unknown[] = []
  const store = new ModerationStore({
    database: {
      async get(table: string, query: unknown) {
        assert.equal(table, MODERATION_REVIEW_TABLE)
        queries.push(query)
        return [
          createReview({ id: 'rv-pending', status: 'pending' }),
          createReview({ id: 'rv-executed', status: 'executed' }),
        ]
      },
    },
  } as never)

  assert.deepEqual(await store.listPendingReviews(''), [])
  assert.deepEqual(await store.listPendingReviews('   '), [])
  assert.equal(queries.length, 0)

  const scoped = await store.listPendingReviews(' guild-1 ')
  assert.deepEqual(queries, [{ guildId: 'guild-1' }])
  assert.deepEqual(scoped.map((item) => item.id), ['rv-pending'])
})

test('listAllPendingReviews 显式执行全局查询', async () => {
  const queries: unknown[] = []
  const store = new ModerationStore({
    database: {
      async get(table: string, query: unknown) {
        assert.equal(table, MODERATION_REVIEW_TABLE)
        queries.push(query)
        return [
          createReview({ id: 'rv-1', guildId: 'guild-1', status: 'pending' }),
          createReview({ id: 'rv-2', guildId: 'guild-2', status: 'stuck_manual' }),
          createReview({ id: 'rv-3', guildId: 'guild-3', status: 'rejected' }),
        ]
      },
    },
  } as never)

  const records = await store.listAllPendingReviews()
  assert.deepEqual(queries, [{}])
  assert.deepEqual(records.map((item) => item.id), ['rv-1', 'rv-2'])
})

test('incrementWarning 并发命中不丢失计数（原子自增）', async () => {
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-moderation-store-'))
  const root = new Context()
  root.plugin(sqlite, { path: join(tempDir, 'koishi.db') })

  try {
    await root.start()
    registerModerationModels(root)
    const store = new ModerationStore(root)
    const now = new Date('2026-06-12T03:00:00.000Z')

    const concurrentHits = 5
    await Promise.all(Array.from({ length: concurrentHits }, (_, index) =>
      store.incrementWarning({
        guildId: 'guild-1',
        memberId: '10001',
        reason: `hit-${index}`,
        now,
      }),
    ))

    const counter = await store.getWarningCounter('guild-1', '10001')
    assert.equal(counter?.total, concurrentHits)

    const after = await store.incrementWarning({
      guildId: 'guild-1',
      memberId: '10001',
      reason: 'final',
      now,
    })
    assert.equal(after.total, concurrentHits + 1)
    assert.equal(after.lastReason, 'final')
  } finally {
    await root.stop()
    await rm(tempDir, { recursive: true, force: true })
  }
})

function createReview(overrides: Partial<ReviewQueueRecord> = {}): ReviewQueueRecord {
  return {
    id: 'rv-1',
    platform: 'qq',
    botSelfId: 'bot-1',
    guildId: 'guild-1',
    channelId: 'channel-1',
    memberId: '10001',
    actionType: 'kick',
    status: 'pending',
    reason: 'spam',
    operatorMemberId: null,
    resolutionNote: null,
    payload: null,
    createdAt: new Date('2026-06-05T04:00:00.000Z'),
    updatedAt: new Date('2026-06-05T04:00:00.000Z'),
    ...overrides,
  }
}

test('查询下推：listRecentMessages/listRecentEvents 在数据库层排序截断，listKeywordRules 命中 guildId 与通配', async () => {
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-moderation-store-'))
  const root = new Context()
  root.plugin(sqlite, { path: join(tempDir, 'koishi.db') })

  try {
    await root.start()
    registerModerationModels(root)
    const store = new ModerationStore(root)
    const base = new Date('2026-06-12T00:00:00.000Z').getTime()

    for (let index = 0; index < 5; index += 1) {
      await store.saveMessage(createMessage({
        messageId: `msg-${index}`,
        guildId: index < 4 ? 'guild-1' : 'guild-2',
        createdAt: new Date(base + index * 60_000),
      }))
      await store.appendEvent({
        platform: 'qq',
        botSelfId: 'bot-1',
        guildId: 'guild-1',
        channelId: 'channel-1',
        memberId: '10001',
        type: 'keyword_hit',
        level: 'low',
        summary: `event-${index}`,
        payload: null,
      })
    }
    await store.upsertKeywordRule(createKeywordRule({ id: 'rule-guild', guildId: 'guild-1' }))
    await store.upsertKeywordRule(createKeywordRule({ id: 'rule-wildcard', guildId: '*' }))
    await store.upsertKeywordRule(createKeywordRule({ id: 'rule-other', guildId: 'guild-9' }))

    const messages = await store.listRecentMessages('guild-1', 2)
    assert.deepEqual(messages.map((record) => record.messageId), ['msg-3', 'msg-2'])

    const events = await store.listRecentEvents(3)
    assert.equal(events.length, 3)

    const rules = await store.listKeywordRules('guild-1')
    assert.deepEqual(rules.map((record) => record.id).sort(), ['rule-guild', 'rule-wildcard'])
  } finally {
    await root.stop()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('pruneExpired 按保留期删除过期消息台账与事件日志', async () => {
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-moderation-store-'))
  const root = new Context()
  root.plugin(sqlite, { path: join(tempDir, 'koishi.db') })

  try {
    await root.start()
    registerModerationModels(root)
    const store = new ModerationStore(root)
    const now = new Date('2026-06-12T00:00:00.000Z')
    const daysAgo = (days: number) => new Date(now.getTime() - days * 24 * 60 * 60 * 1000)

    await store.saveMessage(createMessage({ messageId: 'msg-old', createdAt: daysAgo(20) }))
    await store.saveMessage(createMessage({ messageId: 'msg-fresh', createdAt: daysAgo(3) }))
    await root.database.create('stuhelper_moderation_event', {
      id: 'event-old',
      platform: 'qq',
      botSelfId: 'bot-1',
      guildId: 'guild-1',
      channelId: 'channel-1',
      memberId: '10001',
      type: 'keyword_hit',
      level: 'low',
      summary: 'old',
      payload: null,
      createdAt: daysAgo(40),
      updatedAt: daysAgo(40),
    })
    await root.database.create('stuhelper_moderation_event', {
      id: 'event-fresh',
      platform: 'qq',
      botSelfId: 'bot-1',
      guildId: 'guild-1',
      channelId: 'channel-1',
      memberId: '10001',
      type: 'keyword_hit',
      level: 'low',
      summary: 'fresh',
      payload: null,
      createdAt: daysAgo(5),
      updatedAt: daysAgo(5),
    })

    const result = await store.pruneExpired({
      messageRetentionDays: 14,
      eventRetentionDays: 30,
      now,
    })

    assert.equal(result.removedMessages, 1)
    assert.equal(result.removedEvents, 1)
    assert.equal(await store.getMessage('msg-old'), undefined)
    assert.ok(await store.getMessage('msg-fresh'))
    const events = await store.listRecentEvents(10)
    assert.deepEqual(events.map((record) => record.id), ['event-fresh'])
  } finally {
    await root.stop()
    await rm(tempDir, { recursive: true, force: true })
  }
})
