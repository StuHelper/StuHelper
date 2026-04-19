export type VerificationState =
  | 'unbound'
  | 'bound_unverified'
  | 'verified'
  | 'muted_pending_verification'
  | 'expired_pending_kick'

export interface StuhelperPlatformConfig {
  baseUrl: string
  serviceToken: string
}

export interface StuhelperBindingConfig {
  command: string
  codeTtlMinutes: number
}

export interface StuhelperGuardConfig {
  targetGroups: string[]
  muteDurationSeconds: number
  kickAfterMinutes: number
  reminderTemplate: string
  exemptUsers: string[]
}

export interface StuhelperAdminConfig {
  enableCommands: boolean
}

export interface StuhelperSchedulerConfig {
  scanIntervalSeconds: number
}

export interface StuhelperCoreConfig {
  platform: StuhelperPlatformConfig
  binding: StuhelperBindingConfig
  guard: StuhelperGuardConfig
  admin: StuhelperAdminConfig
  scheduler: StuhelperSchedulerConfig
}

export interface StuhelperBindingPluginConfig {
  platform: StuhelperPlatformConfig
  binding: StuhelperBindingConfig
}

export interface StuhelperGroupGuardPluginConfig {
  platform: StuhelperPlatformConfig
  guard: StuhelperGuardConfig
  scheduler: StuhelperSchedulerConfig
}

export interface StuhelperAdminPluginConfig {
  platform: StuhelperPlatformConfig
  admin: StuhelperAdminConfig
}
