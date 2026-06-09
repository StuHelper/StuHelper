import assert from 'node:assert/strict'
import test from 'node:test'

import type {
  GuardGroupBindingRecord,
  GuardPolicyStore,
  GuardTemplateRecord,
} from '@stuhelper/koishi-shared'

import { syncGuardPolicyFromAdmissionTargets } from './guard-policy-bootstrap'

test('syncGuardPolicyFromAdmissionTargets creates default template and qq bindings idempotently', async () => {
  const store = createPolicyStore()

  const targets = [
    { policyID: 'policy-178', platform: 'qq', guildID: '178037297', guardEnabled: true },
    { policyID: 'policy-duplicate', platform: 'qq', guildID: '178037297', guardEnabled: true },
    { policyID: 'policy-empty', platform: 'qq', guildID: ' ', guardEnabled: true },
    { policyID: 'policy-other', platform: 'telegram', guildID: 'ignored', guardEnabled: true },
  ]

  const first = await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, targets)
  const second = await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, targets)

  assert.deepEqual(first, {
    templateCreated: true,
    bindingCreatedCount: 1,
    bindingUpdatedCount: 0,
    bindingDisabledCount: 0,
  })
  assert.deepEqual(second, {
    templateCreated: false,
    bindingCreatedCount: 0,
    bindingUpdatedCount: 1,
    bindingDisabledCount: 0,
  })
  assert.equal(store.templates[0].id, 'admission-default')
  assert.equal(store.templates[0].name, '入群认证默认模板')
  assert.equal(store.templates[0].muteDurationSeconds, 600)
  assert.equal(store.bindings[0].id, 'qq:178037297')
  assert.equal(store.bindings[0].platform, 'qq')
  assert.equal(store.bindings[0].guildId, '178037297')
  assert.equal(store.bindings[0].templateId, 'admission-default')
})

test('syncGuardPolicyFromAdmissionTargets applies backend policy target enabled state', async () => {
  const store = createPolicyStore()
  await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, [
    { policyID: 'policy-178', platform: 'qq', guildID: '178037297', guardEnabled: true },
  ])

  const result = await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, [
    { policyID: 'policy-178', platform: 'qq', guildID: '178037297', guardEnabled: false },
    { policyID: 'policy-743', platform: 'qq', guildID: '743762161', guardEnabled: true },
    {
      policyID: 'policy-review',
      platform: 'qq',
      guildID: 'review-only',
      guardEnabled: true,
      joinHandlingStrategy: 'join_request_review',
    },
    { policyID: 'policy-old-backend', platform: 'qq', guildID: 'old-backend' },
    { policyID: 'policy-other', platform: 'telegram', guildID: 'ignored', guardEnabled: true },
  ])

  assert.deepEqual(result, {
    templateCreated: false,
    bindingCreatedCount: 3,
    bindingUpdatedCount: 1,
    bindingDisabledCount: 0,
  })
  assert.deepEqual(store.bindings.map((binding) => [binding.id, binding.enabled, binding.note]), [
    ['qq:178037297', false, 'synced from backend admission policies'],
    ['qq:743762161', true, 'synced from backend admission policies'],
    ['qq:review-only', false, 'synced from backend admission policies'],
    ['qq:old-backend', true, 'synced from backend admission policies'],
  ])
})

test('syncGuardPolicyFromAdmissionTargets disables stale qq bindings absent from backend targets', async () => {
  const store = createPolicyStore()
  await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, [
    { policyID: 'policy-178', platform: 'qq', guildID: '178037297', guardEnabled: true },
    { policyID: 'policy-743', platform: 'qq', guildID: '743762161', guardEnabled: true },
  ])

  const result = await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, [
    { policyID: 'policy-743', platform: 'qq', guildID: '743762161', guardEnabled: true },
  ])

  assert.deepEqual(result, {
    templateCreated: false,
    bindingCreatedCount: 0,
    bindingUpdatedCount: 1,
    bindingDisabledCount: 1,
  })
  assert.deepEqual(store.bindings.map((binding) => [binding.id, binding.enabled, binding.note]), [
    ['qq:178037297', false, 'disabled because backend admission policy target is absent'],
    ['qq:743762161', true, 'synced from backend admission policies'],
  ])
})

test('syncGuardPolicyFromAdmissionTargets does not disable non-qq bindings as stale admission targets', async () => {
  const store = createPolicyStore()
  await store.saveBinding({
    platform: 'telegram',
    guildId: 'external',
    templateId: 'external-template',
    enabled: true,
    note: 'external binding',
  })

  const result = await syncGuardPolicyFromAdmissionTargets(store as unknown as GuardPolicyStore, [])

  assert.equal(result.bindingDisabledCount, 0)
  assert.deepEqual(store.bindings.map((binding) => [binding.id, binding.enabled, binding.note]), [
    ['telegram:external', true, 'external binding'],
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
