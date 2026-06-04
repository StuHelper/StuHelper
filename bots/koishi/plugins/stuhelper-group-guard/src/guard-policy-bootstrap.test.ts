import assert from 'node:assert/strict'
import test from 'node:test'

import type {
  GuardGroupBindingRecord,
  GuardPolicyStore,
  GuardTemplateRecord,
  StuhelperGuardConfig,
} from '@stuhelper/koishi-shared'

import { bootstrapGuardPolicyFromStaticConfig } from './guard-policy-bootstrap'

test('bootstrapGuardPolicyFromStaticConfig promotes static target groups to qq bindings idempotently', async () => {
  const store = createPolicyStore()
  const config = createGuardConfig()

  const first = await bootstrapGuardPolicyFromStaticConfig(store as unknown as GuardPolicyStore, config)
  const second = await bootstrapGuardPolicyFromStaticConfig(store as unknown as GuardPolicyStore, config)

  assert.deepEqual(first, { templateCreated: true, bindingCreatedCount: 1 })
  assert.deepEqual(second, { templateCreated: false, bindingCreatedCount: 0 })
  assert.equal(store.templates[0].id, 'admission-default')
  assert.equal(store.templates[0].muteDurationSeconds, 2592000)
  assert.equal(store.bindings[0].id, 'qq:178037297')
  assert.equal(store.bindings[0].platform, 'qq')
  assert.equal(store.bindings[0].guildId, '178037297')
  assert.equal(store.bindings[0].templateId, 'admission-default')
})

function createPolicyStore() {
  const templates: GuardTemplateRecord[] = []
  const bindings: GuardGroupBindingRecord[] = []
  return {
    templates,
    bindings,
    listTemplates: async () => templates,
    listBindings: async () => bindings,
    saveTemplate: async (input: Omit<GuardTemplateRecord, 'createdAt' | 'updatedAt'>) => {
      const now = new Date('2026-06-04T08:00:00.000Z')
      templates.push({
        ...input,
        createdAt: now,
        updatedAt: now,
      })
    },
    saveBinding: async (input: Omit<GuardGroupBindingRecord, 'id' | 'createdAt' | 'updatedAt'>) => {
      const now = new Date('2026-06-04T08:00:00.000Z')
      bindings.push({
        id: `${input.platform}:${input.guildId}`,
        ...input,
        note: input.note ?? null,
        createdAt: now,
        updatedAt: now,
      })
    },
  }
}

function createGuardConfig(): StuhelperGuardConfig {
  return {
    targetGroups: ['178037297', '178037297', ' '],
    muteDurationSeconds: 2592000,
    kickAfterMinutes: 60,
    reminderTemplate: '请先认证',
    exemptUsers: [],
  }
}
