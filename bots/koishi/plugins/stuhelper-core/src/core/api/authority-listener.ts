import type { DataService, Events as ConsoleEvents } from '@koishijs/console'
import type { Context } from 'koishi'

const CONSOLE_MIN_AUTHORITY = 4

type ConsoleInstance = NonNullable<Context['console']>

export function createAuthority4ListenerRegistrar(console: ConsoleInstance) {
  return function addAuthorityListener<K extends keyof ConsoleEvents>(
    event: K,
    callback: ConsoleEvents[K],
    options?: DataService.Options,
  ) {
    console.addListener(event, callback, {
      authority: CONSOLE_MIN_AUTHORITY,
      ...options,
    })
  }
}
