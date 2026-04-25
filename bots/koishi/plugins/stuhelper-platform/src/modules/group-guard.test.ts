import assert from 'node:assert/strict'
import test from 'node:test'

import * as entry from '../index'
import { createModuleRegistry } from '../module-registry'
import { StuhelperPlatformService } from '../platform-service'
import { createGroupGuardModule, groupGuardModule } from './group-guard'

const DEFAULT_MUTE_DURATION_SECONDS = 600
const CUSTOM_DICE_SIDES = 20
const GROUP_GUARD_MODULE_ID = 'group-guard'

class FakeDatabase {
  async get() {
    return []
  }

  async upsert() {}

  async create() {}
}

test('group guard manifest and config defaults define platform managed policy', () => {
  assert.deepEqual(groupGuardModule.manifest, {
    id: GROUP_GUARD_MODULE_ID,
    name: '群守卫',
    description: '管理群成员准入、消息巡检、举报和趣味群管命令。',
    version: '0.1.0',
    category: 'moderation',
    defaultEnabled: true,
    order: 10,
  })
  assert.deepEqual(groupGuardModule.configSchema.defaults(), {
    platform: { baseUrl: 'http://127.0.0.1:8080', serviceToken: '' },
    guard: {
      targetGroups: [],
      muteDurationSeconds: DEFAULT_MUTE_DURATION_SECONDS,
      kickAfterMinutes: 30,
      reminderTemplate: '请先完成 StuHelper 注册、QQ 绑定与学生认证。',
      exemptUsers: [],
    },
    scheduler: { scanIntervalSeconds: 60 },
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
      muteLotteryBaseSeconds: 120,
      muteLotteryMaxSeconds: 600,
      muteLotteryPityThreshold: 5,
      muteLotteryPitySeconds: 300,
    },
    ai: { enabled: false, endpoint: '', apiKey: '', model: '' },
  })
})

test('group guard config parser accepts explicit unknown input and rejects invalid fields', async () => {
  const parsed = groupGuardModule.configSchema.parse({
    guard: {
      targetGroups: ['10001'],
      muteDurationSeconds: DEFAULT_MUTE_DURATION_SECONDS,
      kickAfterMinutes: 5,
      reminderTemplate: '欢迎 {user}',
      exemptUsers: ['admin'],
    },
    scheduler: { scanIntervalSeconds: 30 },
    fun: { diceSides: CUSTOM_DICE_SIDES },
  })

  assert.deepEqual(parsed.guard.targetGroups, ['10001'])
  assert.equal(parsed.guard.kickAfterMinutes, 5)
  assert.equal(parsed.scheduler.scanIntervalSeconds, 30)
  assert.equal(parsed.fun.diceSides, CUSTOM_DICE_SIDES)
  await assert.rejects(
    async () => groupGuardModule.configSchema.parse({ fun: { diceSides: 'many' } }),
    /fun\.diceSides must be a positive integer/,
  )
  await assert.rejects(
    async () => groupGuardModule.configSchema.parse({ ignored: true }),
    /unknown group guard config field: ignored/,
  )
})

test('group guard metadata describes permissions commands events and webui sections', () => {
  assert.deepEqual(groupGuardModule.permissions.map((permission) => permission.id), [
    'group-guard.view',
    'group-guard.manage',
    'group-guard.command.report',
    'group-guard.command.dice',
    'group-guard.command.mute-lottery',
  ])
  assert.deepEqual(groupGuardModule.commands.map((command) => command.name), [
    '举报 <targetMemberId> <reason:text>',
    '骰子 [sides:natural]',
    '抽禁言',
  ])
  assert.deepEqual(groupGuardModule.events.map((event) => event.name), [
    'guild-member-added',
    'message',
    'message-deleted',
    'scan-tick',
  ])
  assert.deepEqual(groupGuardModule.webui.map((item) => item.section), [
    'module',
    'policy',
  ])
})

test('group guard setup applies runtime adapter and writes audit event', async () => {
  const auditEvents: unknown[] = []
  const appliedConfigs: unknown[] = []
  const module = createGroupGuardModule({
    prepare: () => {},
    start: (_ctx, config) => {
      appliedConfigs.push(config)
    },
  })

  await module.setup({
    koishi: {} as never,
    config: module.configSchema.defaults(),
    audit: async (event) => {
      auditEvents.push(event)
    },
  })

  assert.equal(appliedConfigs.length, 1)
  assert.deepEqual(auditEvents, [{
    moduleId: GROUP_GUARD_MODULE_ID,
    action: 'module.setup',
    summary: '群守卫模块已加载',
    payload: { targetGroupCount: 0, scanIntervalSeconds: 60 },
  }])
})

test('platform index exports and service listing include group guard module', async () => {
  assert.equal(entry.groupGuardModule, groupGuardModule)

  const service = entry.createPlatformService({ database: new FakeDatabase() } as never)
  const modules = await service.listModules()

  assert.deepEqual(modules.map((module) => module.manifest.id), [GROUP_GUARD_MODULE_ID])
  assert.equal(modules[0].enabled, true)
})

test('direct service listing includes group guard metadata', async () => {
  const service = new StuhelperPlatformService({
    registry: createModuleRegistry([groupGuardModule]),
    store: {
      getModuleState: async (_moduleId, defaultEnabled) => ({
        enabled: defaultEnabled,
        status: 'loaded',
        lastError: null,
      }),
      getModuleConfig: async () => null,
      listAuditEvents: async () => [],
      setModuleEnabled: async () => {},
      saveModuleConfig: async () => {},
    },
    runtime: {
      syncModule: async () => {},
      restartModule: async () => {},
    },
  })

  const [module] = await service.listModules()

  assert.equal(module.manifest.id, GROUP_GUARD_MODULE_ID)
  assert.deepEqual(module.commands.map((command) => command.permission), [
    'group-guard.command.report',
    'group-guard.command.dice',
    'group-guard.command.mute-lottery',
  ])
})
