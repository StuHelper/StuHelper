import type { Awaitable, Context, Session } from 'koishi'

export type StuhelperModuleCategory =
  | 'system'
  | 'moderation'
  | 'communication'
  | 'governance'

export interface StuhelperModuleManifest {
  readonly id: string
  readonly name: string
  readonly description: string
  readonly version: string
  readonly category: StuhelperModuleCategory
  readonly defaultEnabled: boolean
  readonly order: number
}

export type StuhelperModuleConfig = Record<string, unknown>

export interface StuhelperConfigSchema<TConfig extends StuhelperModuleConfig> {
  parse(value: unknown): TConfig
  defaults(): TConfig
}

export interface StuhelperPermissionDefinition {
  readonly id: string
  readonly label: string
  readonly description: string
}

export interface StuhelperCommandDefinition<TConfig extends StuhelperModuleConfig> {
  readonly name: string
  readonly description: string
  readonly permission: string
}

export interface StuhelperEventDefinition<TConfig extends StuhelperModuleConfig> {
  readonly name: string
}

export interface StuhelperWebuiContribution {
  readonly id: string
  readonly label: string
  readonly section: 'overview' | 'module' | 'policy' | 'audit' | 'settings'
}

export interface StuhelperModuleContext<TConfig extends StuhelperModuleConfig> {
  readonly koishi: Context
  readonly config: TConfig
  readonly session?: Session
  audit(event: StuhelperAuditInput): Awaitable<void>
}

export interface StuhelperAuditInput {
  readonly moduleId: string
  readonly action: string
  readonly summary: string
  readonly payload: Record<string, unknown>
}

export interface StuhelperModule<TConfig extends StuhelperModuleConfig = StuhelperModuleConfig> {
  readonly manifest: StuhelperModuleManifest
  readonly configSchema: StuhelperConfigSchema<TConfig>
  readonly permissions: readonly StuhelperPermissionDefinition[]
  readonly commands: readonly StuhelperCommandDefinition<TConfig>[]
  readonly events: readonly StuhelperEventDefinition<TConfig>[]
  readonly webui: readonly StuhelperWebuiContribution[]
  prepare?(koishi: Context): void
  setup(context: StuhelperModuleContext<TConfig>): Awaitable<void>
}
