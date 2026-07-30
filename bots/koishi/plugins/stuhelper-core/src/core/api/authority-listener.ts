import type { DataService, Events as ConsoleEvents } from '@koishijs/console'
import type { Context } from 'koishi'

const CONSOLE_MIN_AUTHORITY = 4

export function createAuthority4ListenerRegistrar(ctx: Context) {
  const console = ctx.console
  return function addAuthorityListener<K extends keyof ConsoleEvents>(
    event: K,
    callback: ConsoleEvents[K],
    options?: DataService.Options,
  ) {
    return ctx.effect(() => {
      console.addListener(event, callback, {
        ...options,
        authority: CONSOLE_MIN_AUTHORITY,
      })
      const registration = console.listeners[event]
      return () => {
        if (console.listeners[event] === registration) {
          delete console.listeners[event]
        }
      }
    })
  }
}
