import { DataService } from '@koishijs/console'
import type { Context } from 'koishi'

import { STUHELPER_PLATFORM_SERVICE } from './constants'
import type {
  StuhelperAuditEventSnapshot,
  StuhelperModuleSnapshot,
  StuhelperPlatformService,
} from './platform-service'

const CONSOLE_AUTHORITY = 4
const CONSOLE_ACTOR = 'console'
const DEFAULT_AUDIT_LIMIT = 50

export interface StuhelperPlatformData {
  readonly generatedAt: string
  readonly modules: readonly StuhelperModuleSnapshot[]
  readonly auditEvents: readonly StuhelperAuditEventSnapshot[]
}

export type PlatformConsoleService = Pick<
  StuhelperPlatformService,
  'listModules' | 'listAuditEvents' | 'setModuleEnabled' | 'saveModuleConfig'
>

declare module '@koishijs/console' {
  interface Events {
    'stuhelper-platform/refresh'(): Promise<void>
    'stuhelper-platform/module.set-enabled'(input: unknown): Promise<void>
    'stuhelper-platform/module.save-config'(input: unknown): Promise<void>
    'stuhelper-platform/audit.list'(input?: unknown): Promise<readonly StuhelperAuditEventSnapshot[]>
  }

  namespace Console {
    interface Services {
      stuhelperPlatform: StuhelperPlatformDataService
    }
  }
}

export interface StuhelperPlatformDataServiceConfig {
  readonly service: PlatformConsoleService
}

export class StuhelperPlatformDataService extends DataService<StuhelperPlatformData> {
  private readonly service: PlatformConsoleService

  constructor(ctx: Context, config: StuhelperPlatformDataServiceConfig) {
    super(ctx, STUHELPER_PLATFORM_SERVICE, { authority: CONSOLE_AUTHORITY })
    this.service = config.service
  }

  async get(): Promise<StuhelperPlatformData> {
    const [modules, auditEvents] = await Promise.all([
      this.service.listModules(),
      this.service.listAuditEvents(DEFAULT_AUDIT_LIMIT),
    ])
    return {
      generatedAt: new Date().toISOString(),
      modules,
      auditEvents,
    }
  }
}

export function registerPlatformConsoleRoutes(ctx: Context, service: PlatformConsoleService): void {
  ctx.console.addListener('stuhelper-platform/refresh', async () => {
    await refreshPlatformData(ctx)
  }, { authority: CONSOLE_AUTHORITY })

  ctx.console.addListener('stuhelper-platform/module.set-enabled', async (input) => {
    const payload = parseSetEnabledInput(input)
    await service.setModuleEnabled(payload.moduleId, payload.enabled, CONSOLE_ACTOR)
    await refreshPlatformData(ctx)
  }, { authority: CONSOLE_AUTHORITY })

  ctx.console.addListener('stuhelper-platform/module.save-config', async (input) => {
    const payload = parseSaveConfigInput(input)
    await service.saveModuleConfig(payload.moduleId, payload.config, CONSOLE_ACTOR)
    await refreshPlatformData(ctx)
  }, { authority: CONSOLE_AUTHORITY })

  ctx.console.addListener('stuhelper-platform/audit.list', async (input) => {
    const payload = parseAuditListInput(input)
    return service.listAuditEvents(payload.limit)
  }, { authority: CONSOLE_AUTHORITY })
}

async function refreshPlatformData(ctx: Context): Promise<void> {
  await ctx.console.refresh(STUHELPER_PLATFORM_SERVICE)
}

function parseSetEnabledInput(input: unknown): { readonly moduleId: string; readonly enabled: boolean } {
  const record = requireRecord(input, 'module state')
  return {
    moduleId: readString(record.moduleId, 'moduleId'),
    enabled: readBoolean(record.enabled, 'enabled'),
  }
}

function parseSaveConfigInput(input: unknown): {
  readonly moduleId: string
  readonly config: Record<string, unknown>
} {
  const record = requireRecord(input, 'module config')
  return {
    moduleId: readString(record.moduleId, 'moduleId'),
    config: readConfig(record.config),
  }
}

function parseAuditListInput(input: unknown): { readonly limit?: number } {
  if (input === undefined) return {}
  const record = requireRecord(input, 'audit list')
  return { limit: readOptionalPositiveInteger(record.limit, 'limit') }
}

function requireRecord(input: unknown, label: string): Record<string, unknown> {
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

function readBoolean(value: unknown, field: string): boolean {
  if (typeof value !== 'boolean') {
    throw new Error(`${field} must be a boolean`)
  }
  return value
}

function readConfig(value: unknown): Record<string, unknown> {
  if (!value || typeof value !== 'object' || Array.isArray(value)) {
    throw new Error('config must be an object')
  }
  return { ...(value as Record<string, unknown>) }
}

function readOptionalPositiveInteger(value: unknown, field: string): number | undefined {
  if (value === undefined) return undefined
  if (typeof value !== 'number' || !Number.isInteger(value) || value < 1) {
    throw new Error(`${field} must be a positive integer`)
  }
  return value
}
