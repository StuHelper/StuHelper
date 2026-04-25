import { send } from '@koishijs/client'

import type { PlatformConfig } from './model'

type PlatformEventName =
  | 'stuhelper-platform/refresh'
  | 'stuhelper-platform/module.set-enabled'
  | 'stuhelper-platform/module.save-config'

export function refreshPlatformData(): Promise<void> {
  return sendPlatform<void>('stuhelper-platform/refresh')
}

export function setModuleEnabled(moduleId: string, enabled: boolean): Promise<void> {
  return sendPlatform<void>('stuhelper-platform/module.set-enabled', {
    moduleId,
    enabled,
  })
}

export function saveModuleConfig(moduleId: string, config: PlatformConfig): Promise<void> {
  return sendPlatform<void>('stuhelper-platform/module.save-config', {
    moduleId,
    config,
  })
}

function sendPlatform<T>(type: PlatformEventName, payload?: unknown): Promise<T> {
  const args = payload === undefined ? [] : [payload]
  return requireResponse(send(type, ...args) as Promise<T> | undefined)
}

function requireResponse<T>(response: Promise<T> | undefined): Promise<T> {
  if (!response) {
    throw new Error('Koishi console socket is not connected')
  }
  return response
}
