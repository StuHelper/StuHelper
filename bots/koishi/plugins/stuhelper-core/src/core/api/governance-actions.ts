import type { Context } from 'koishi'
import { z } from 'zod'

import { GuardPolicyStore } from '@stuhelper/koishi-shared'
import { ModerationStore } from '@stuhelper/koishi-moderation-core'

const MAX_COMMAND_ID_LENGTH = 128
const MAX_ROLE_ID_LENGTH = 64
const MAX_TEMPLATE_ID_LENGTH = 64
const MAX_TEMPLATE_NAME_LENGTH = 128
const MAX_REMINDER_TEMPLATE_LENGTH = 2000
const MAX_EXEMPT_USER_COUNT = 500
const MAX_NOTE_LENGTH = 512
const MAX_MUTE_DURATION_SECONDS = 30 * 24 * 3600
const MAX_KICK_AFTER_MINUTES = 30 * 24 * 60
const MAX_MIN_AUTHORITY = 5

type CommandPolicyInput = {
  commandId: string
  roles: string[]
  minAuthority: number
}

type GuardTemplateInput = {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
  enabled: boolean
}

type GuardBindingInput = {
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note?: string | null
}

const commandPolicySchema = z.object({
  commandId: z.string().trim().min(1).max(MAX_COMMAND_ID_LENGTH),
  minAuthority: z.number().int().min(0).max(MAX_MIN_AUTHORITY).finite(),
  roles: z.array(z.string().trim().min(1).max(MAX_ROLE_ID_LENGTH)).max(MAX_EXEMPT_USER_COUNT),
}).strict()

const guardTemplateSchema = z.object({
  id: z.string().trim().min(1).max(MAX_TEMPLATE_ID_LENGTH),
  name: z.string().trim().min(1).max(MAX_TEMPLATE_NAME_LENGTH),
  muteDurationSeconds: z.number().int().min(0).max(MAX_MUTE_DURATION_SECONDS).finite(),
  kickAfterMinutes: z.number().int().min(0).max(MAX_KICK_AFTER_MINUTES).finite(),
  reminderTemplate: z.string().trim().min(1).max(MAX_REMINDER_TEMPLATE_LENGTH),
  exemptUsers: z.array(z.string().trim().min(1).max(MAX_ROLE_ID_LENGTH)).max(MAX_EXEMPT_USER_COUNT),
  enabled: z.boolean(),
}).strict()

const guardBindingSchema = z.object({
  platform: z.string().trim().min(1).max(MAX_ROLE_ID_LENGTH),
  guildId: z.string().trim().min(1).max(MAX_ROLE_ID_LENGTH),
  templateId: z.string().trim().min(1).max(MAX_TEMPLATE_ID_LENGTH),
  enabled: z.boolean(),
  note: z.union([
    z.string().trim().max(MAX_NOTE_LENGTH).transform((value) => value || null),
    z.null(),
    z.undefined(),
  ]),
}).strict()

export function registerGovernanceActionAPI(ctx: Context) {
  if (!ctx.console) {
    return
  }

  const moderationStore = new ModerationStore(ctx)
  const guardPolicyStore = new GuardPolicyStore(ctx)

  ctx.console.addListener('stuhelperGroupCenter/action/save-command-policy', async (input) => {
    await saveCommandPolicy(moderationStore, parseCommandPolicyInput(input))
    return '已保存命令策略。'
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/action/save-guard-template', async (input) => {
    return saveGuardTemplate(guardPolicyStore, parseGuardTemplateInput(input))
  }, { authority: 4 })

  ctx.console.addListener('stuhelperGroupCenter/action/save-guard-binding', async (input) => {
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
  await guardPolicyStore.saveTemplate({
    ...input,
    exemptUsers: [...input.exemptUsers],
  })
  return `已保存群模板：${input.name}`
}

async function saveGuardBinding(
  guardPolicyStore: GuardPolicyStore,
  input: GuardBindingInput,
) {
  const templates = await guardPolicyStore.listTemplates()
  const template = templates.find((item) => item.id === input.templateId)
  if (!template) {
    throw new Error(`guard template not found: ${input.templateId}`)
  }
  await guardPolicyStore.saveBinding({
    ...input,
    note: input.note || null,
  })
  return `已保存群绑定：${input.platform}/${input.guildId}`
}

export function parseCommandPolicyInput(input: unknown): CommandPolicyInput {
  const record = commandPolicySchema.parse(requireRecord(input, 'command policy'))
  return {
    commandId: record.commandId,
    minAuthority: record.minAuthority,
    roles: record.roles.map((item) => item.trim()),
  }
}

export function parseGuardTemplateInput(input: unknown): GuardTemplateInput {
  const record = guardTemplateSchema.parse(requireRecord(input, 'guard template'))
  return {
    id: record.id,
    name: record.name,
    muteDurationSeconds: record.muteDurationSeconds,
    kickAfterMinutes: record.kickAfterMinutes,
    reminderTemplate: record.reminderTemplate,
    exemptUsers: record.exemptUsers.map((item) => item.trim()),
    enabled: record.enabled,
  }
}

export function parseGuardBindingInput(input: unknown): GuardBindingInput {
  const record = guardBindingSchema.parse(requireRecord(input, 'guard binding'))
  return {
    platform: record.platform,
    guildId: record.guildId,
    templateId: record.templateId,
    enabled: record.enabled,
    note: record.note,
  }
}

function requireRecord(input: unknown, label: string) {
  if (!input || typeof input !== 'object' || Array.isArray(input)) {
    throw new Error(`${label} input must be an object`)
  }
  return input as Record<string, unknown>
}
