import type {
  GuardPolicyStore,
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

export async function bootstrapGuardPolicyFromStaticConfig(
  policyStore: GuardPolicyStore,
  config: StuhelperGuardConfig,
  logger?: BootstrapLogger,
): Promise<GuardPolicyBootstrapResult> {
  const targetGroups = normalizeTargetGroups(config.targetGroups)
  if (targetGroups.length === 0) {
    return { templateCreated: false, bindingCreatedCount: 0 }
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
  for (const guildId of targetGroups) {
    const id = createBindingID(ADMISSION_BUSINESS_PLATFORM, guildId)
    if (existingBindingIDs.has(id)) {
      continue
    }
    await policyStore.saveBinding({
      platform: ADMISSION_BUSINESS_PLATFORM,
      guildId,
      templateId: BOOTSTRAP_TEMPLATE_ID,
      enabled: true,
      note: 'bootstrapped from guard.targetGroups',
    })
    bindingCreatedCount += 1
  }

  if (templateCreated || bindingCreatedCount > 0) {
    logger?.info('已把静态入群认证 targetGroups 迁移到 WebUI 策略库：template=%s, bindings=%d', templateCreated ? 'created' : 'exists', bindingCreatedCount)
  }
  return { templateCreated, bindingCreatedCount }
}

function normalizeTargetGroups(groups: readonly string[]) {
  return [...new Set(groups.map((item) => item.trim()).filter(Boolean))]
}
