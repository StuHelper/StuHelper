import type { Context } from 'koishi'

import type { StuhelperGuardConfig } from '../types/index'
import {
  GUARD_GROUP_BINDING_TABLE,
  GUARD_TEMPLATE_TABLE,
  type GuardGroupBindingRecord,
  type GuardTemplateRecord,
} from './policy'

const STATIC_TEMPLATE_ID = '__static__'
const STATIC_TEMPLATE_NAME = '静态默认模板'

export interface EffectiveGuardPolicy {
  source: 'binding' | 'static'
  templateId: string
  templateName: string
  platform: string
  guildId: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
}

interface GuardBindingInput {
  platform: string
  guildId: string
  templateId: string
  enabled: boolean
  note?: string | null
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

export class GuardPolicyStore {
  constructor(
    private readonly ctx: Context,
    private readonly fallbackConfig?: StuhelperGuardConfig,
  ) {}

  async listTemplates() {
    return this.ctx.database.get(GUARD_TEMPLATE_TABLE, {}) as Promise<GuardTemplateRecord[]>
  }

  async listBindings() {
    return this.ctx.database.get(GUARD_GROUP_BINDING_TABLE, {}) as Promise<GuardGroupBindingRecord[]>
  }

  async saveTemplate(input: GuardTemplateInput) {
    const now = new Date()
    const [existing] = await this.ctx.database.get(GUARD_TEMPLATE_TABLE, { id: input.id }) as GuardTemplateRecord[]
    if (existing) {
      const { id: _id, ...changes } = input
      await this.ctx.database.set(GUARD_TEMPLATE_TABLE, { id: input.id }, {
        ...changes,
        updatedAt: now,
      })
      return
    }

    await this.ctx.database.create(GUARD_TEMPLATE_TABLE, {
      ...input,
      createdAt: now,
      updatedAt: now,
    } satisfies GuardTemplateRecord)
  }

  async saveBinding(input: GuardBindingInput) {
    const now = new Date()
    const id = createBindingID(input.platform, input.guildId)
    const [existing] = await this.ctx.database.get(GUARD_GROUP_BINDING_TABLE, { id }) as GuardGroupBindingRecord[]
    if (existing) {
      await this.ctx.database.set(GUARD_GROUP_BINDING_TABLE, { id }, {
        ...input,
        note: input.note || null,
        updatedAt: now,
      })
      return
    }

    await this.ctx.database.create(GUARD_GROUP_BINDING_TABLE, {
      id,
      ...input,
      note: input.note || null,
      createdAt: now,
      updatedAt: now,
    } satisfies GuardGroupBindingRecord)
  }

  async resolvePolicy(platform: string, guildId: string) {
    const [binding] = await this.ctx.database.get(GUARD_GROUP_BINDING_TABLE, {
      id: createBindingID(platform, guildId),
    }) as GuardGroupBindingRecord[]

    if (binding) {
      if (!binding.enabled) {
        return null
      }
      const template = await this.requireEnabledTemplate(binding.templateId)
      return createBoundPolicy(binding, template)
    }

    if (!this.fallbackConfig || !this.fallbackConfig.targetGroups.includes(guildId)) {
      return null
    }

    return {
      source: 'static',
      templateId: STATIC_TEMPLATE_ID,
      templateName: STATIC_TEMPLATE_NAME,
      platform,
      guildId,
      muteDurationSeconds: this.fallbackConfig.muteDurationSeconds,
      kickAfterMinutes: this.fallbackConfig.kickAfterMinutes,
      reminderTemplate: this.fallbackConfig.reminderTemplate,
      exemptUsers: [...this.fallbackConfig.exemptUsers],
    } satisfies EffectiveGuardPolicy
  }

  private async requireEnabledTemplate(templateId: string) {
    const [template] = await this.ctx.database.get(GUARD_TEMPLATE_TABLE, { id: templateId }) as GuardTemplateRecord[]
    if (!template) {
      throw new Error(`guard template not found: ${templateId}`)
    }
    if (!template.enabled) {
      throw new Error(`guard template is disabled: ${templateId}`)
    }
    return template
  }
}

export function createBindingID(platform: string, guildId: string) {
  return `${platform}:${guildId}`
}

function createBoundPolicy(binding: GuardGroupBindingRecord, template: GuardTemplateRecord) {
  return {
    source: 'binding',
    templateId: template.id,
    templateName: template.name,
    platform: binding.platform,
    guildId: binding.guildId,
    muteDurationSeconds: template.muteDurationSeconds,
    kickAfterMinutes: template.kickAfterMinutes,
    reminderTemplate: template.reminderTemplate,
    exemptUsers: [...template.exemptUsers],
  } satisfies EffectiveGuardPolicy
}
