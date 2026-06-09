import type {
  GuardPolicyStore,
  AdmissionPolicyTarget,
} from '@stuhelper/koishi-shared'
import {
  createBindingID,
} from '@stuhelper/koishi-shared'

import { isPostJoinGuardStrategy } from './post-join-guard-strategy'

const ADMISSION_BUSINESS_PLATFORM = 'qq'
const BOOTSTRAP_TEMPLATE_ID = 'admission-default'
const BOOTSTRAP_TEMPLATE_NAME = '入群认证默认模板'
const BOOTSTRAP_MUTE_DURATION_SECONDS = 600
const BOOTSTRAP_KICK_AFTER_MINUTES = 30
const BOOTSTRAP_REMINDER_TEMPLATE = '请先完成 StuHelper 注册、QQ 绑定与学生认证。'
const SYNCED_BINDING_NOTE = 'synced from backend admission policies'
const STALE_BINDING_NOTE = 'disabled because backend admission policy target is absent'

interface BootstrapLogger {
  info(message: string, ...args: unknown[]): void
}

interface NormalizedAdmissionTargetGroup {
  readonly guildId: string
  readonly enabled: boolean
}

export interface GuardPolicyBootstrapResult {
  readonly templateCreated: boolean
  readonly bindingCreatedCount: number
}

export interface GuardPolicyTargetSyncResult extends GuardPolicyBootstrapResult {
  readonly bindingUpdatedCount: number
  readonly bindingDisabledCount: number
}

export async function syncGuardPolicyFromAdmissionTargets(
  policyStore: GuardPolicyStore,
  targets: readonly AdmissionPolicyTarget[],
  logger?: BootstrapLogger,
): Promise<GuardPolicyTargetSyncResult> {
  const targetGroups = normalizeAdmissionTargetGroups(targets)
  const result = await ensureGuardPolicyBindings(
    policyStore,
    targetGroups,
    SYNCED_BINDING_NOTE,
  )

  if (
    result.templateCreated ||
    result.bindingCreatedCount > 0 ||
    result.bindingUpdatedCount > 0 ||
    result.bindingDisabledCount > 0
  ) {
    logger?.info(
      '已从后端 admission policy 同步入群认证目标群：template=%s, created=%d, updated=%d, staleDisabled=%d',
      result.templateCreated ? 'created' : 'exists',
      result.bindingCreatedCount,
      result.bindingUpdatedCount,
      result.bindingDisabledCount,
    )
  }
  return result
}

async function ensureGuardPolicyBindings(
  policyStore: GuardPolicyStore,
  targetGroups: readonly NormalizedAdmissionTargetGroup[],
  note: string,
): Promise<GuardPolicyTargetSyncResult> {
  if (targetGroups.length === 0) {
    return disableStaleBackendBindings(policyStore, [], [], false)
  }

  const [templates, bindings] = await Promise.all([
    policyStore.listTemplates(),
    policyStore.listBindings(),
  ])
  const templateCreated = !templates.some((template) => template.id === BOOTSTRAP_TEMPLATE_ID)
  if (templateCreated) {
    await policyStore.saveTemplate({
      id: BOOTSTRAP_TEMPLATE_ID,
      name: BOOTSTRAP_TEMPLATE_NAME,
      muteDurationSeconds: BOOTSTRAP_MUTE_DURATION_SECONDS,
      kickAfterMinutes: BOOTSTRAP_KICK_AFTER_MINUTES,
      reminderTemplate: BOOTSTRAP_REMINDER_TEMPLATE,
      exemptUsers: [],
      enabled: true,
    })
  }

  const existingBindingIDs = new Set(bindings.map((binding) => binding.id))
  let bindingCreatedCount = 0
  let bindingUpdatedCount = 0
  for (const target of targetGroups) {
    const id = createBindingID(ADMISSION_BUSINESS_PLATFORM, target.guildId)
    const exists = existingBindingIDs.has(id)
    await policyStore.saveBinding({
      platform: ADMISSION_BUSINESS_PLATFORM,
      guildId: target.guildId,
      templateId: BOOTSTRAP_TEMPLATE_ID,
      enabled: target.enabled,
      note,
    })
    if (exists) {
      bindingUpdatedCount += 1
    } else {
      existingBindingIDs.add(id)
      bindingCreatedCount += 1
    }
  }

  const staleResult = await disableStaleBackendBindings(
    policyStore,
    targetGroups,
    bindings,
    templateCreated,
  )

  return {
    templateCreated,
    bindingCreatedCount,
    bindingUpdatedCount,
    bindingDisabledCount: staleResult.bindingDisabledCount,
  }
}

async function disableStaleBackendBindings(
  policyStore: GuardPolicyStore,
  targetGroups: readonly NormalizedAdmissionTargetGroup[],
  currentBindings: Awaited<ReturnType<GuardPolicyStore['listBindings']>>,
  templateCreated: boolean,
): Promise<GuardPolicyTargetSyncResult> {
  const bindings = currentBindings.length > 0
    ? currentBindings
    : await policyStore.listBindings()
  const targetGuildIds = new Set(targetGroups.map((target) => target.guildId))
  let bindingDisabledCount = 0

  for (const binding of bindings) {
    if (binding.platform !== ADMISSION_BUSINESS_PLATFORM) continue
    if (targetGuildIds.has(binding.guildId)) continue
    if (!binding.enabled && binding.note === STALE_BINDING_NOTE) continue
    await policyStore.saveBinding({
      platform: binding.platform,
      guildId: binding.guildId,
      templateId: binding.templateId,
      enabled: false,
      note: STALE_BINDING_NOTE,
    })
    bindingDisabledCount += 1
  }

  return {
    templateCreated,
    bindingCreatedCount: 0,
    bindingUpdatedCount: 0,
    bindingDisabledCount,
  }
}

function normalizeAdmissionTargetGroups(
  targets: readonly AdmissionPolicyTarget[],
): NormalizedAdmissionTargetGroup[] {
  const normalized = new Map<string, NormalizedAdmissionTargetGroup>()
  for (const target of targets) {
    if (target.platform !== ADMISSION_BUSINESS_PLATFORM) continue
    const guildId = target.guildID.trim()
    if (!guildId) continue
    normalized.set(guildId, {
      guildId,
      enabled: target.guardEnabled !== false && isPostJoinGuardStrategy(target.joinHandlingStrategy),
    })
  }
  return [...normalized.values()]
}
