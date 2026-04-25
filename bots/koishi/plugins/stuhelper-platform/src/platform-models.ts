import { Context } from 'koishi'

import {
  AUDIT_EVENT_TABLE,
  GUILD_POLICY_TABLE,
  MODULE_CONFIG_TABLE,
  MODULE_STATE_TABLE,
  PERMISSION_POLICY_TABLE,
} from './constants'

export type ModuleStartupStatus = 'pending' | 'loaded' | 'error' | 'disabled'

export interface ModuleStateRecord {
  id: string
  moduleId: string
  enabled: boolean
  order: number
  version: string
  status: ModuleStartupStatus
  lastError: string | null
  createdAt: Date
  updatedAt: Date
}

export interface ModuleConfigRecord {
  id: string
  moduleId: string
  schemaVersion: string
  config: Record<string, unknown>
  createdAt: Date
  updatedAt: Date
}

export interface PermissionPolicyRecord {
  id: string
  roleId: string
  permissionId: string
  scope: string
  createdAt: Date
  updatedAt: Date
}

export interface GuildPolicyRecord {
  id: string
  guildId: string
  moduleId: string
  enabled: boolean
  config: Record<string, unknown> | null
  createdAt: Date
  updatedAt: Date
}

export interface AuditEventRecord {
  id: string
  actor: string
  moduleId: string
  action: string
  summary: string
  payload: Record<string, unknown>
  createdAt: Date
  updatedAt: Date
}

declare module 'koishi' {
  interface Tables {
    [MODULE_STATE_TABLE]: ModuleStateRecord
    [MODULE_CONFIG_TABLE]: ModuleConfigRecord
    [PERMISSION_POLICY_TABLE]: PermissionPolicyRecord
    [GUILD_POLICY_TABLE]: GuildPolicyRecord
    [AUDIT_EVENT_TABLE]: AuditEventRecord
  }
}

export function registerPlatformModels(ctx: Context) {
  extendModuleStateModel(ctx)
  extendModuleConfigModel(ctx)
  extendPermissionPolicyModel(ctx)
  extendGuildPolicyModel(ctx)
  extendAuditEventModel(ctx)
}

function extendModuleStateModel(ctx: Context) {
  ctx.model.extend(MODULE_STATE_TABLE, {
    id: 'string',
    moduleId: 'string',
    enabled: 'boolean',
    order: 'integer',
    version: 'string',
    status: 'string',
    lastError: 'text',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function extendModuleConfigModel(ctx: Context) {
  ctx.model.extend(MODULE_CONFIG_TABLE, {
    id: 'string',
    moduleId: 'string',
    schemaVersion: 'string',
    config: { type: 'json', initial: {} },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function extendPermissionPolicyModel(ctx: Context) {
  ctx.model.extend(PERMISSION_POLICY_TABLE, {
    id: 'string',
    roleId: 'string',
    permissionId: 'string',
    scope: 'string',
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function extendGuildPolicyModel(ctx: Context) {
  ctx.model.extend(GUILD_POLICY_TABLE, {
    id: 'string',
    guildId: 'string',
    moduleId: 'string',
    enabled: 'boolean',
    config: { type: 'json', initial: null },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}

function extendAuditEventModel(ctx: Context) {
  ctx.model.extend(AUDIT_EVENT_TABLE, {
    id: 'string',
    actor: 'string',
    moduleId: 'string',
    action: 'string',
    summary: 'text',
    payload: { type: 'json', initial: {} },
    createdAt: 'timestamp',
    updatedAt: 'timestamp',
  }, { primary: 'id' })
}
