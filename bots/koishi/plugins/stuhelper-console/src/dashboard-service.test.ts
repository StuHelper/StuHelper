import assert from 'node:assert/strict'
import { mkdtemp, rm } from 'node:fs/promises'
import { tmpdir } from 'node:os'
import { join } from 'node:path'
import test from 'node:test'

import sqlite from '@koishijs/plugin-database-sqlite'

import {
  ModerationStore,
  registerModerationModels,
} from '@stuhelper/koishi-moderation-core'
import {
  GUARD_GROUP_BINDING_TABLE,
  GUARD_TEMPLATE_TABLE,
  registerGuardMemberModel,
  registerGuardPolicyModels,
} from '@stuhelper/koishi-shared'

import { StuhelperConsoleDataService } from './dashboard-service'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

test('控制台数据服务会返回排序后的举报列表', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-console-'))

  registerGuardMemberModel(root)
  registerGuardPolicyModels(root)
  registerModerationModels(root)
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })

  try {
    await root.start()
    const store = new ModerationStore(root)
    const olderReport = await store.createReport({
      platform: 'mock',
      botSelfId: '514',
      guildId: 'group-1',
      channelId: 'group-1',
      reporterMemberId: '20001',
      targetMemberId: '10001',
      reason: '广告刷屏',
      aiStatus: 'pending',
      aiSeverity: 'medium',
      aiSummary: null,
    })
    await root.database.set('stuhelper_moderation_report', { id: olderReport.id }, {
      createdAt: new Date('2026-04-19T09:00:00Z'),
      updatedAt: new Date('2026-04-19T09:00:00Z'),
    })

    const newerReport = await store.createReport({
      platform: 'mock',
      botSelfId: '514',
      guildId: 'group-1',
      channelId: 'group-1',
      reporterMemberId: '20002',
      targetMemberId: '10002',
      reason: '恶意引流',
      aiStatus: 'completed',
      aiSeverity: 'high',
      aiSummary: '高风险',
    })
    await root.database.set('stuhelper_moderation_report', { id: newerReport.id }, {
      createdAt: new Date('2026-04-19T10:00:00Z'),
      updatedAt: new Date('2026-04-19T10:00:00Z'),
    })

    const service = new StuhelperConsoleDataService(root, { title: '测试控制台' })
    const data = await service.get()

    assert.equal(data.title, '测试控制台')
    assert.equal(data.overview.openReports, 2)
    assert.deepEqual(data.recentReports.map((item) => item.targetMemberId), ['10002', '10001'])
    assert.equal(data.recentReports[0].createdAt, '2026-04-19T10:00:00.000Z')
    assert.equal(data.recentReports[1].createdAt, '2026-04-19T09:00:00.000Z')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('控制台数据服务会返回群模板和群绑定列表', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-console-'))

  registerGuardMemberModel(root)
  registerGuardPolicyModels(root)
  registerModerationModels(root)
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })

  try {
    await root.start()
    await root.database.create(GUARD_TEMPLATE_TABLE, {
      id: 'study-room',
      name: '自习室模板',
      muteDurationSeconds: 120,
      kickAfterMinutes: 20,
      reminderTemplate: '请完成自习室认证。',
      exemptUsers: ['10086'],
      enabled: true,
      createdAt: new Date('2026-04-19T09:00:00Z'),
      updatedAt: new Date('2026-04-19T09:00:00Z'),
    })
    await root.database.create(GUARD_GROUP_BINDING_TABLE, {
      id: 'mock:group-9',
      platform: 'mock',
      guildId: 'group-9',
      templateId: 'study-room',
      enabled: true,
      note: '九号群',
      createdAt: new Date('2026-04-19T09:10:00Z'),
      updatedAt: new Date('2026-04-19T09:10:00Z'),
    })

    const service = new StuhelperConsoleDataService(root, { title: '测试控制台' })
    const data = await service.get()

    assert.equal(data.guardTemplates.length, 1)
    assert.equal(data.guardTemplates[0].id, 'study-room')
    assert.equal(data.guardBindings.length, 1)
    assert.equal(data.guardBindings[0].platform, 'mock')
    assert.equal(data.guardBindings[0].guildId, 'group-9')
    assert.equal(data.guardBindings[0].templateId, 'study-room')
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})

test('控制台数据服务会暴露完整的命令策略候选清单', async () => {
  const runtime = createKoishiTestRuntime()
  const { root } = runtime
  const tempDir = await mkdtemp(join(tmpdir(), 'stuhelper-koishi-console-'))

  registerGuardMemberModel(root)
  registerGuardPolicyModels(root)
  registerModerationModels(root)
  runtime.register(sqlite, { path: join(tempDir, 'koishi.db') })

  try {
    await root.start()
    const service = new StuhelperConsoleDataService(root, { title: '测试控制台' })
    const data = await service.get()

    assert.deepEqual(data.supportedCommandIds, [
      'report',
      'dice',
      'mute-lottery',
      'guard-status',
      'guard-warnings',
      'guard-reviews',
      'guard-mute',
      'guard-kick-request',
      'guard-block-request',
    ])
  } finally {
    runtime.dispose()
    await rm(tempDir, { recursive: true, force: true })
  }
})
