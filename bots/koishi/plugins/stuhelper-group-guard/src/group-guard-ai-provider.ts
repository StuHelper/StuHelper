import {
  DEFAULT_GROUP_GUARD_AI_SETTINGS,
  type GroupGuardAISettings,
} from '@stuhelper/koishi-shared'

export type GroupGuardAISettingsProvider = () => GroupGuardAISettings | Promise<GroupGuardAISettings>

export interface GroupGuardAISettingsStoreRef {
  getAISettings(): Promise<GroupGuardAISettings>
}

export function createGroupGuardAISettingsProvider(
  store?: GroupGuardAISettingsStoreRef,
): GroupGuardAISettingsProvider {
  if (!store) {
    return () => DEFAULT_GROUP_GUARD_AI_SETTINGS
  }
  return () => store.getAISettings()
}

export async function getGroupGuardAISettings(provider?: GroupGuardAISettingsProvider) {
  return provider ? await provider() : DEFAULT_GROUP_GUARD_AI_SETTINGS
}
