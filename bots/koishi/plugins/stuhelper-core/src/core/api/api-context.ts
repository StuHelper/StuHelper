import type { DataService, Events as ConsoleEvents } from '@koishijs/console'
import type { PlatformClient } from '@stuhelper/koishi-shared'
import type { Context } from 'koishi'

import type { StuhelperGroupCenterService } from '../services/stuhelper-group-center.service'
import type { ConsoleGuildScope } from './console-guild-scope'

export type AuthorityListenerRegistrar = <K extends keyof ConsoleEvents>(
  event: K,
  callback: ConsoleEvents[K],
  options?: DataService.Options,
) => void

export type ConsoleScopeResolver = (client: unknown) => Promise<ConsoleGuildScope>

export interface WebSocketAPIContext {
  readonly ctx: Context
  readonly service: StuhelperGroupCenterService
  readonly addAuthorityListener: AuthorityListenerRegistrar
  readonly resolveConsoleScope: ConsoleScopeResolver
  readonly platform: PlatformClient
  readonly packageVersion: string
}
