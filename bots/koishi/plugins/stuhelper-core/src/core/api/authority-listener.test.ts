import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import { createAuthority4ListenerRegistrar } from './authority-listener'

test('authority listener disposal preserves a newer registration for the same event', () => {
  const listeners: Record<string, {
    callback: () => string
    authority?: number
  }> = Object.create(null)
  const ctx = {
    console: {
      listeners,
      addListener(
        event: string,
        callback: () => string,
        options?: { authority?: number },
      ) {
        listeners[event] = { callback, ...options }
      },
    },
    effect(register: () => () => void) {
      return register()
    },
  } as unknown as Context

  const addAuthorityListener = createAuthority4ListenerRegistrar(ctx)
  const disposeFirst = addAuthorityListener('ping', () => 'first', { authority: 1 })
  const firstRegistration = listeners.ping
  assert.equal(firstRegistration.authority, 4)

  const disposeSecond = addAuthorityListener('ping', () => 'second')
  const secondRegistration = listeners.ping
  assert.notEqual(secondRegistration, firstRegistration)
  assert.equal(secondRegistration.callback(), 'second')

  disposeFirst()
  assert.equal(listeners.ping, secondRegistration)

  disposeSecond()
  assert.equal('ping' in listeners, false)
})
