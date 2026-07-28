import { Context } from 'koishi'

import type { AdmissionJoinHandlingStrategy } from '../types/index'

export const GUARD_TEMPLATE_TABLE = 'stuhelper_guard_template'
export const GUARD_GROUP_BINDING_TABLE = 'stuhelper_guard_group_binding'

declare module 'koishi' {
  interface Tables {
    [GUARD_TEMPLATE_TABLE]: GuardTemplateRecord
    [GUARD_GROUP_BINDING_TABLE]: GuardGroupBindingRecord
  }
}

export interface GuardTemplateRecord {
  id: string
  name: string
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
  enabled: boolean
  createdAt: Date
  updatedAt: Date
}

export interface GuardGroupBindingRecord {
  id: string
  platform: string
  guildId: string
  templateId: string
  joinHandlingStrategy?: AdmissionJoinHandlingStrategy
  kickAfterMinutesOverride?: number | null
  enabled: boolean
  note: string | null
  createdAt: Date
  updatedAt: Date
}

export function registerGuardPolicyModels(ctx: Context) {
  ctx.model.extend(GUARD_TEMPLATE_TABLE, {
    id: 'string',
    name: 'string',
    muteDurationSeconds: 'unsigned',
    kickAfterMinutes: 'unsigned',
    reminderTemplate: 'text',
    exemptUsers: { type: 'json', initial: [] },
    enabled: 'boolean',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })

  ctx.model.extend(GUARD_GROUP_BINDING_TABLE, {
    id: 'string',
    platform: 'string',
    guildId: 'string',
    templateId: 'string',
    joinHandlingStrategy: { type: 'string', initial: 'post_join_guard' },
    kickAfterMinutesOverride: { type: 'unsigned', initial: null },
    enabled: 'boolean',
    note: 'text',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}
