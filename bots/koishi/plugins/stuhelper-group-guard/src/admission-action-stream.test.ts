import assert from 'node:assert/strict'
import test from 'node:test'
import type { Context } from 'koishi'

import type { PlatformClient } from '@stuhelper/koishi-shared'

import { registerAdmissionActionStreams } from './admission-action-stream'
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

test('admission action stream respects static disabled default without runtime override', async () => {
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
      enabled: false,
      reconnectDelaySeconds: 1,
    },
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
  await waitFor(() => opened.length > 1)

  assert.equal(opened.length, 2)

  controller.close()
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
