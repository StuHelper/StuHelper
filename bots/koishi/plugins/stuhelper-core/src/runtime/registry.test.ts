import assert from 'node:assert/strict'
import test from 'node:test'

import { getRuntimeModules } from './registry'

test('runtime registry keeps P3 module order', () => {
  const ids = getRuntimeModules().map(module => module.id)

  assert.deepEqual(ids, [
    'warn',
    'keyword',
    'manage-member',
    'manage-message',
    'manage-order',
    'antirepeat',
    'welcome',
    'repeat',
    'dice',
    'banme',
    'antirecall',
    'ai',
    'config',
    'log',
    'subscription',
    'help',
    'report',
    'getauth',
    'auth',
    'event',
    'status',
    'manage-cross-group',
  ])
})
