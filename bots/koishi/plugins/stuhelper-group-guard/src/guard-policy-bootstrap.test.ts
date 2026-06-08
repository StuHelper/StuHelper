import assert from 'node:assert/strict'
import test from 'node:test'

import type {
  GuardGroupBindingRecord,
  GuardPolicyStore,
  GuardTemplateRecord,
  StuhelperGuardConfig,
} from '@stuhelper/koishi-shared'

import {
  bootstrapGuardPolicyFromStaticConfig,
  syncGuardPolicyFromAdmissionTargets,
} from './guard-policy-bootstrap'

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

test('syncGuardPolicyFromAdmissionTargets applies backend policy target enabled state', async () => {
  const store = createPolicyStore()
  const config = createGuardConfig()

  await bootstrapGuardPolicyFromStaticConfig(store as unknown as GuardPolicyStore, config)
  store.bindings[0].enabled = false

  const result = await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, config, [
    { policyID: 'policy-178', platform: 'qq', guildID: '178037297', guardEnabled: false },
    { policyID: 'policy-743', platform: 'qq', guildID: '743762161', guardEnabled: true },
    { policyID: 'policy-old-backend', platform: 'qq', guildID: 'old-backend' },
    { policyID: 'policy-other', platform: 'telegram', guildID: 'ignored', guardEnabled: true },
  ])

  assert.deepEqual(result, {
    templateCreated: false,
    bindingCreatedCount: 2,
    bindingUpdatedCount: 1,
  })
  assert.deepEqual(store.bindings.map((binding) => [binding.id, binding.enabled, binding.note]), [
    ['qq:178037297', false, 'synced from backend admission policies'],
    ['qq:743762161', true, 'synced from backend admission policies'],
    ['qq:old-backend', true, 'synced from backend admission policies'],
  ])
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
      const record = {
        id: `${input.platform}:${input.guildId}`,
        ...input,
        note: input.note ?? null,
        createdAt: now,
        updatedAt: now,
      }
      const index = bindings.findIndex((binding) => binding.id === record.id)
      if (index >= 0) {
        bindings[index] = { ...bindings[index], ...record, createdAt: bindings[index].createdAt }
        return
      }
      bindings.push(record)
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
