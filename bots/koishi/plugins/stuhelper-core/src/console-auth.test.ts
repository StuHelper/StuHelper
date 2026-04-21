import assert from 'node:assert/strict'
import test from 'node:test'

import { validateConsoleAdminPassword } from './console-auth'

test('validateConsoleAdminPassword rejects undefined or blank passwords', () => {
  assert.throws(
    () => validateConsoleAdminPassword(undefined),
    /STUHELPER_CONSOLE_ADMIN_PASSWORD/,
  )
  assert.throws(
    () => validateConsoleAdminPassword('   '),
    /STUHELPER_CONSOLE_ADMIN_PASSWORD/,
  )
})

test('validateConsoleAdminPassword accepts non-empty passwords', () => {
  assert.equal(
    validateConsoleAdminPassword('correct-horse-battery-staple'),
    'correct-horse-battery-staple',
  )
})
