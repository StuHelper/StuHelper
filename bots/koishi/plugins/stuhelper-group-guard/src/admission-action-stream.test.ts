import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import type { PlatformClient } from '@stuhelper/koishi-shared'

import {
  admissionActionReconnectDelayMs,
  registerAdmissionActionStreams,
  shouldLogReconnectAttempt,
} from './admission-action-stream'
import type { MemberGuardService } from './member-guard'

test('admission action stream can be opened and closed by runtime setting without restart', async () => {
  let enabled = false
  const opened: unknown[] = []
  const closed: unknown[] = []
  const controller = registerAdmissionActionStreams(fakeContext(), {
    platform: {
      streamAdmissionActions(input) {
        opened.push(input)
        return {
          close() {
            closed.push(input)
          },
        }
      },
    } as unknown as PlatformClient,
    memberGuard: {} as MemberGuardService,
    logger: fakeLogger(),
    config: {
      reconnectDelaySeconds: 1,
    },
    isEnabled: () => enabled,
  })

  await controller.refresh()
  assert.equal(opened.length, 0)
  assert.equal(closed.length, 0)

  enabled = true
  await controller.refresh()
  assert.deepEqual(opened, [{
    platform: 'qq',
    botSelfID: '2118785781',
    limit: 50,
  }])
  assert.equal(closed.length, 0)

  await controller.refresh()
  assert.equal(opened.length, 1)

  enabled = false
  await controller.refresh()
  assert.deepEqual(closed, [{
    platform: 'qq',
    botSelfID: '2118785781',
    limit: 50,
  }])

  controller.close()
})

test('admission action stream stays closed when runtime setting is disabled', async () => {
  const opened: unknown[] = []
  const controller = registerAdmissionActionStreams(fakeContext(), {
    platform: {
      streamAdmissionActions(input) {
        opened.push(input)
        return {
          close() {},
        }
      },
    } as unknown as PlatformClient,
    memberGuard: {} as MemberGuardService,
    logger: fakeLogger(),
    config: {
      reconnectDelaySeconds: 1,
    },
    isEnabled: () => false,
  })

  await controller.refresh()
  assert.equal(opened.length, 0)

  controller.close()
})

test('admission action stream reconnects after stream disconnects', async () => {
  const opened: unknown[] = []
  let onError: ((error: unknown) => void) | undefined
  const controller = registerAdmissionActionStreams(fakeContext(), {
    platform: {
      streamAdmissionActions(input, handlers) {
        opened.push(input)
        onError = handlers.onError
        return {
          close() {},
        }
      },
    } as unknown as PlatformClient,
    memberGuard: {} as MemberGuardService,
    logger: fakeLogger(),
    config: {
      reconnectDelaySeconds: 1,
    },
  })

  await controller.refresh()
  assert.equal(opened.length, 1)

  onError?.(new Error('stream closed'))
  onError?.(new Error('duplicate stream error'))
  await waitFor(() => opened.length > 1)

  assert.equal(opened.length, 2)

  controller.close()
})

test('admission action stream retries after a transient runtime setting failure', async () => {
  const opened: unknown[] = []
  let settingsReads = 0
  const controller = registerAdmissionActionStreams(fakeContext(), {
    platform: {
      streamAdmissionActions(input) {
        opened.push(input)
        return {
          close() {},
        }
      },
    } as unknown as PlatformClient,
    memberGuard: {} as MemberGuardService,
    logger: fakeLogger(),
    config: {
      reconnectDelaySeconds: 1,
    },
    isEnabled: () => {
      settingsReads += 1
      if (settingsReads === 1) {
        throw new Error('database unavailable')
      }
      return true
    },
  })

  await controller.refresh()
  assert.equal(opened.length, 0)
  await waitFor(() => opened.length > 0)

  assert.equal(opened.length, 1)
  assert.equal(settingsReads, 2)
  controller.close()
})

test('admission action stream reconnect delay is bounded exponential backoff with jitter', () => {
  assert.equal(admissionActionReconnectDelayMs(5, 0, () => 0), 5_000)
  assert.equal(admissionActionReconnectDelayMs(5, 0, () => 1), 6_000)
  assert.equal(admissionActionReconnectDelayMs(5, 1, () => 0), 8_000)
  assert.equal(admissionActionReconnectDelayMs(5, 1, () => 0.5), 10_000)
  assert.equal(admissionActionReconnectDelayMs(5, 100, () => 1), 300_000)
})

test('admission action stream samples reconnect warnings without resetting on an action', () => {
  assert.equal(shouldLogReconnectAttempt(1), true)
  assert.equal(shouldLogReconnectAttempt(2), true)
  assert.equal(shouldLogReconnectAttempt(3), false)
  assert.equal(shouldLogReconnectAttempt(4), true)
  assert.equal(shouldLogReconnectAttempt(15), false)
  assert.equal(shouldLogReconnectAttempt(16), true)
})

function fakeContext() {
  return {
    bots: [{
      platform: 'onebot',
      selfId: '2118785781',
    }],
    on() {},
  } as unknown as Context
}

function fakeLogger() {
  return {
    info() {},
    warn() {},
  }
}

async function waitFor(condition: () => boolean | Promise<boolean>, timeoutMs = 1500) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (await condition()) {
      return
    }
    await new Promise((resolve) => setTimeout(resolve, 10))
  }
  assert.fail('condition was not met before timeout')
}
