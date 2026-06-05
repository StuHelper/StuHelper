import assert from 'node:assert/strict'
import test from 'node:test'
import type { Session } from 'koishi'

import {
  clearCommandExecutionState,
  commandExecutionDuration,
  getCommandExecutionState,
  markCommandExecutionFailed,
  markCommandExecutionStarted,
} from './command-execution-state'

test('command execution state tracks duration without mutating argv', () => {
  const argv: Record<string, unknown> = {}

  markCommandExecutionStarted(argv, 1000)

  assert.equal(commandExecutionDuration(argv, 1250), 250)
  assert.deepEqual(argv, {})
})

test('command execution state tracks failures without mutating session', () => {
  const session = {} as Session

  markCommandExecutionFailed(session, 'permission denied')

  assert.deepEqual(getCommandExecutionState(session), {
    failed: true,
    error: 'permission denied',
  })
  assert.equal('_commandFailed' in (session as unknown as Record<string, unknown>), false)
  assert.equal('_commandError' in (session as unknown as Record<string, unknown>), false)

  clearCommandExecutionState(session)

  assert.deepEqual(getCommandExecutionState(session), { failed: false })
})
