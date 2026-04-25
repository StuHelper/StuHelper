import { GuardPolicyStore } from '@stuhelper/koishi-shared'
import { ModerationStore } from '@stuhelper/koishi-moderation-core'

import type {
  StuhelperCommandPolicyInput,
  StuhelperGuardBindingInput,
  StuhelperGuardTemplateInput,
  StuhelperKeywordRuleInput,
  StuhelperMemberRoleInput,
} from './console-types'

export async function saveMemberRoles(
  moderationStore: ModerationStore,
  input: StuhelperMemberRoleInput,
) {
  await moderationStore.setMemberRoles(input.guildId, input.memberId, input.roles)
}

export async function saveKeywordRule(
  moderationStore: ModerationStore,
  input: StuhelperKeywordRuleInput,
) {
  const now = new Date()
  const current = (await moderationStore.listAllKeywordRules()).find((item) => item.id === input.id)
  await moderationStore.upsertKeywordRule({
    ...input,
    note: input.note || null,
    createdAt: current?.createdAt || now,
    updatedAt: now,
  })
}

export async function saveCommandPolicy(
  moderationStore: ModerationStore,
  input: StuhelperCommandPolicyInput,
) {
  const now = new Date()
  const current = await moderationStore.getCommandPolicy(input.commandId)
  await moderationStore.upsertCommandPolicy({
    commandId: input.commandId,
    roles: input.roles,
    minAuthority: input.minAuthority,
    createdAt: current?.createdAt || now,
    updatedAt: now,
  })
}

export async function saveGuardTemplate(
  guardPolicyStore: GuardPolicyStore,
  input: StuhelperGuardTemplateInput,
) {
  if (!input.id.trim() || !input.name.trim() || !input.reminderTemplate.trim()) {
    throw new Error('guard template id、名称和提醒文案不能为空')
  }
  await guardPolicyStore.saveTemplate({
    ...input,
    id: input.id.trim(),
    name: input.name.trim(),
    reminderTemplate: input.reminderTemplate.trim(),
    exemptUsers: [...input.exemptUsers],
  })
  return `已保存群模板：${input.name}`
}

export async function saveGuardBinding(
  guardPolicyStore: GuardPolicyStore,
  input: StuhelperGuardBindingInput,
) {
  if (!input.platform.trim() || !input.guildId.trim() || !input.templateId.trim()) {
    throw new Error('platform、guildId 和 templateId 不能为空')
  }
  const templates = await guardPolicyStore.listTemplates()
  const templateId = input.templateId.trim()
  const template = templates.find((item) => item.id === templateId)
  if (!template) {
    throw new Error(`guard template not found: ${templateId}`)
  }
  await guardPolicyStore.saveBinding({
    ...input,
    platform: input.platform.trim(),
    guildId: input.guildId.trim(),
    templateId,
    note: input.note?.trim() || null,
  })
  return `已保存群绑定：${input.platform.trim()}/${input.guildId.trim()}`
}
