import type {
  GuardPolicyStore,
  AdmissionPolicyTarget,
  StuhelperGuardConfig,
} from '@stuhelper/koishi-shared'
import { createBindingID } from '@stuhelper/koishi-shared'

const ADMISSION_BUSINESS_PLATFORM = 'qq'
const BOOTSTRAP_TEMPLATE_ID = 'admission-default'
const BOOTSTRAP_TEMPLATE_NAME = '入群认证默认模板'

interface BootstrapLogger {
  info(message: string, ...args: unknown[]): void
}

export interface GuardPolicyBootstrapResult {
  readonly templateCreated: boolean
  readonly bindingCreatedCount: number
}

export interface GuardPolicyTargetSyncResult extends GuardPolicyBootstrapResult {
  readonly bindingUpdatedCount: number
}

export async function bootstrapGuardPolicyFromStaticConfig(
  policyStore: GuardPolicyStore,
  config: StuhelperGuardConfig,
  logger?: BootstrapLogger,
): Promise<GuardPolicyBootstrapResult> {
  const result = await ensureGuardPolicyBindings(
    policyStore,
    config,
    normalizeTargetGroups(config.targetGroups),
    'bootstrapped from guard.targetGroups',
    false,
  )

  if (result.templateCreated || result.bindingCreatedCount > 0) {
    logger?.info('已把静态入群认证 targetGroups 迁移到 WebUI 策略库：template=%s, bindings=%d', result.templateCreated ? 'created' : 'exists', result.bindingCreatedCount)
  }
  return {
    templateCreated: result.templateCreated,
    bindingCreatedCount: result.bindingCreatedCount,
  }
}

export async function syncGuardPolicyFromAdmissionTargets(
  policyStore: GuardPolicyStore,
  config: StuhelperGuardConfig,
  targets: readonly AdmissionPolicyTarget[],
  logger?: BootstrapLogger,
): Promise<GuardPolicyTargetSyncResult> {
  const targetGroups = normalizeTargetGroups(
    targets
      .filter((target) => target.platform === ADMISSION_BUSINESS_PLATFORM)
      .map((target) => target.guildID),
  )
  const result = await ensureGuardPolicyBindings(
    policyStore,
    config,
    targetGroups,
    'synced from backend admission policies',
    true,
  )

  if (result.templateCreated || result.bindingCreatedCount > 0 || result.bindingUpdatedCount > 0) {
    logger?.info(
      '已从后端 admission policy 同步入群认证目标群：template=%s, created=%d, updated=%d',
      result.templateCreated ? 'created' : 'exists',
      result.bindingCreatedCount,
      result.bindingUpdatedCount,
    )
  }
  return result
}

async function ensureGuardPolicyBindings(
  policyStore: GuardPolicyStore,
  config: StuhelperGuardConfig,
  targetGroups: readonly string[],
  note: string,
  refreshExistingBindings: boolean,
): Promise<GuardPolicyTargetSyncResult> {
  if (targetGroups.length === 0) {
    return { templateCreated: false, bindingCreatedCount: 0, bindingUpdatedCount: 0 }
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
      muteDurationSeconds: config.muteDurationSeconds,
      kickAfterMinutes: config.kickAfterMinutes,
      reminderTemplate: config.reminderTemplate,
      exemptUsers: [...config.exemptUsers],
      enabled: true,
    })
  }

  const existingBindingIDs = new Set(bindings.map((binding) => binding.id))
  let bindingCreatedCount = 0
  let bindingUpdatedCount = 0
  for (const guildId of targetGroups) {
    const id = createBindingID(ADMISSION_BUSINESS_PLATFORM, guildId)
    const exists = existingBindingIDs.has(id)
    if (exists && !refreshExistingBindings) {
      continue
    }
    await policyStore.saveBinding({
      platform: ADMISSION_BUSINESS_PLATFORM,
      guildId,
      templateId: BOOTSTRAP_TEMPLATE_ID,
      enabled: true,
      note,
    })
    if (exists) {
      bindingUpdatedCount += 1
    } else {
      existingBindingIDs.add(id)
      bindingCreatedCount += 1
    }
  }

  return { templateCreated, bindingCreatedCount, bindingUpdatedCount }
}

function normalizeTargetGroups(groups: readonly string[]) {
  return [...new Set(groups.map((item) => item.trim()).filter(Boolean))]
}
