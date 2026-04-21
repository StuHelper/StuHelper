import type { Context } from 'koishi'

import { GuardPolicyStore } from '@stuhelper/koishi-shared'
import { ModerationStore } from '@stuhelper/koishi-moderation-core'

interface CommandPolicyInput {
  commandId: string
  roles: string[]
  minAuthority: number
}

interface GuardTemplateInput {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
  enabled: boolean
}

interface GuardBindingInput {
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note?: string | null
}

export function registerGovernanceActionAPI(ctx: Context) {
  if (!ctx.console) {
    return
  }

  const moderationStore = new ModerationStore(ctx)
  const guardPolicyStore = new GuardPolicyStore(ctx)

  ctx.console.addListener('stuhelperGroupCenter/action/save-command-policy' as any, async (input) => {
    await saveCommandPolicy(moderationStore, parseCommandPolicyInput(input))
    return '已保存命令策略。'
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/action/save-guard-template' as any, async (input) => {
    return saveGuardTemplate(guardPolicyStore, parseGuardTemplateInput(input))
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/action/save-guard-binding' as any, async (input) => {
    return saveGuardBinding(guardPolicyStore, parseGuardBindingInput(input))
  }, { authority: 4 })
}

async function saveCommandPolicy(
  moderationStore: ModerationStore,
  input: CommandPolicyInput,
) {
  const now = new Date()
  const current = await moderationStore.getCommandPolicy(input.commandId)
  await moderationStore.upsertCommandPolicy({
    commandId: input.commandId,
    roles: [...input.roles],
    minAuthority: input.minAuthority,
    createdAt: current?.createdAt || now,
    updatedAt: now,
  })
}

async function saveGuardTemplate(
  guardPolicyStore: GuardPolicyStore,
  input: GuardTemplateInput,
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

async function saveGuardBinding(
  guardPolicyStore: GuardPolicyStore,
  input: GuardBindingInput,
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

function parseCommandPolicyInput(input: unknown): CommandPolicyInput {
  const record = requireRecord(input, 'command policy')
  return {
    commandId: readString(record.commandId, 'commandId'),
    minAuthority: readNumber(record.minAuthority, 'minAuthority'),
    roles: readStringArray(record.roles, 'roles'),
  }
}

function parseGuardTemplateInput(input: unknown): GuardTemplateInput {
  const record = requireRecord(input, 'guard template')
  return {
    id: readString(record.id, 'id'),
    name: readString(record.name, 'name'),
    muteDurationSeconds: readNumber(record.muteDurationSeconds, 'muteDurationSeconds'),
    kickAfterMinutes: readNumber(record.kickAfterMinutes, 'kickAfterMinutes'),
    reminderTemplate: readString(record.reminderTemplate, 'reminderTemplate'),
    exemptUsers: readStringArray(record.exemptUsers, 'exemptUsers'),
    enabled: readBoolean(record.enabled, 'enabled'),
  }
}

function parseGuardBindingInput(input: unknown): GuardBindingInput {
  const record = requireRecord(input, 'guard binding')
  return {
    platform: readString(record.platform, 'platform'),
    guildId: readString(record.guildId, 'guildId'),
    templateId: readString(record.templateId, 'templateId'),
    enabled: readBoolean(record.enabled, 'enabled'),
    note: readOptionalNullableString(record.note, 'note'),
  }
}

function requireRecord(input: unknown, label: string) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as Record<string, unknown>
}

function readString(value: unknown, field: string): string {
  if (typeof value !== 'string' || !value.trim()) {
    throw new Error(`${field} must be a non-empty string`)
  }
  return value
}

function readStringArray(value: unknown, field: string): string[] {
  if (!Array.isArray(value) || value.some((item) => typeof item !== 'string')) {
    throw new Error(`${field} must be a string array`)
  }
  return [...value]
}

function readNumber(value: unknown, field: string): number {
  if (typeof value !== 'number' || Number.isNaN(value)) {
    throw new Error(`${field} must be a valid number`)
  }
  return value
}

function readBoolean(value: unknown, field: string): boolean {
  if (typeof value !== 'boolean') {
    throw new Error(`${field} must be a boolean`)
  }
  return value
}

function readOptionalNullableString(value: unknown, field: string): string | null | undefined {
  if (value === undefined) return undefined
  if (value === null) return null
  if (typeof value !== 'string') {
    throw new Error(`${field} must be a string or null`)
  }
  return value
}
