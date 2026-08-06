import assert from 'node:assert/strict'
import { createServer as createNetServer } from 'node:net'
import test from 'node:test'

import Server from '@koishijs/plugin-server'
import MockBot from '@koishijs/plugin-mock'
import type { AdmissionPolicyTarget } from '@stuhelper/koishi-shared'
import { createKoishiTestRuntime } from '../../test-utils/runtime.ts'

import {
  AlertmanagerDeliveryService,
  handleAlertmanagerHTTPRequest,
  registerAlertmanagerWebhook,
  resolveUniqueManagementGuildID,
  validateAlertmanagerWebhookConfig,
  type AlertDeliveryBot,
} from './alertmanager-webhook'

const TEST_TOKEN = 'alertmanager-test-token-0123456789abcdef'
const MANAGEMENT_GUILD_ID = '123456789'

test('Alertmanager delivery uses the unique backend management guild and neutralizes message injection', async () => {
  const deliveries: Array<{ channelID: string, content: string }> = []
  const payload = alertPayload('firing')
  Object.assign(payload, { guildID: '999999999' })
  payload.commonAnnotations.summary = '[CQ:at,qq=all] @all <at id="all"> & urgent'
  payload.alerts[0].annotations.summary = payload.commonAnnotations.summary
  payload.alerts[0].labels.alertname = '<DangerAlert>'

  const service = deliveryService({
    bots: [deliveryBot(async (channelID, content) => {
      deliveries.push({ channelID, content: String(content) })
      return ['message-1']
    })],
  })

  const result = await service.deliver(payload)

  assert.equal(result.deduplicated, false)
  assert.equal(deliveries.length, 1)
  assert.equal(deliveries[0].channelID, MANAGEMENT_GUILD_ID)
  assert.match(deliveries[0].content, /FIRING/)
  assert.match(deliveries[0].content, /＜DangerAlert＞/)
  assert.match(deliveries[0].content, /［CQ:at,qq=all］ ＠all ＜at id="all"＞ ＆ urgent/)
  assert.doesNotMatch(deliveries[0].content, /999999999|\[CQ:|@all|<at/)
})

test('Alertmanager delivery deduplicates retries but delivers the resolved transition', async () => {
  const messages: string[] = []
  const service = deliveryService({
    bots: [deliveryBot(async (_channelID, content) => {
      messages.push(String(content))
      return [`message-${messages.length}`]
    })],
  })

  const first = await service.deliver(alertPayload('firing'))
  const retry = await service.deliver(alertPayload('firing'))
  const resolved = await service.deliver(alertPayload('resolved'))

  assert.equal(first.deduplicated, false)
  assert.equal(retry.deduplicated, true)
  assert.equal(resolved.deduplicated, false)
  assert.equal(messages.length, 2)
  assert.match(messages[0], /FIRING/)
  assert.match(messages[1], /RESOLVED/)
})

test('concurrent Alertmanager retries share one QQ delivery', async () => {
  let release: (() => void) | undefined
  const blocked = new Promise<void>((resolve) => {
    release = resolve
  })
  let deliveryCount = 0
  const service = deliveryService({
    bots: [deliveryBot(async () => {
      deliveryCount += 1
      await blocked
      return ['message-concurrent']
    })],
  })

  const first = service.deliver(alertPayload('firing'))
  const second = service.deliver(alertPayload('firing'))
  await new Promise((resolve) => setImmediate(resolve))
  assert.equal(deliveryCount, 1)
  release?.()

  const [firstResult, secondResult] = await Promise.all([first, second])
  assert.equal(firstResult.deduplicated, false)
  assert.equal(secondResult.deduplicated, true)
})

test('HTTP boundary authenticates before reading and enforces content type and body size', async () => {
  let bodyRead = false
  const trackedBody = async function* () {
    bodyRead = true
    yield Buffer.from(JSON.stringify(alertPayload('firing')))
  }
  const service = deliveryService({ bots: [deliveryBot()] })

  const unauthorized = await handleAlertmanagerHTTPRequest({
    authorization: 'Bearer wrong-token',
    contentType: 'application/json',
    contentLength: '',
    body: trackedBody(),
  }, TEST_TOKEN, service)
  assert.deepEqual(unauthorized, { status: 401, code: 'unauthorized' })
  assert.equal(bodyRead, false)

  const unsupported = await handleAlertmanagerHTTPRequest({
    authorization: `Bearer ${TEST_TOKEN}`,
    contentType: 'text/plain',
    contentLength: '',
    body: trackedBody(),
  }, TEST_TOKEN, service)
  assert.deepEqual(unsupported, { status: 415, code: 'unsupported_content_type' })
  assert.equal(bodyRead, false)

  const oversized = await handleAlertmanagerHTTPRequest({
    authorization: `Bearer ${TEST_TOKEN}`,
    contentType: 'application/json; charset=utf-8',
    contentLength: String(64 * 1024 + 1),
    body: trackedBody(),
  }, TEST_TOKEN, service)
  assert.deepEqual(oversized, { status: 413, code: 'payload_too_large' })
  assert.equal(bodyRead, false)
})

test('HTTP boundary returns retryable 503 until backend, management target, bot, and QQ delivery succeed', async () => {
  const payload = alertPayload('firing')
  const cases: Array<{
    expected: string
    service: AlertmanagerDeliveryService
  }> = [
    {
      expected: 'backend_unavailable',
      service: deliveryService({ backendError: true, bots: [deliveryBot()] }),
    },
    {
      expected: 'management_guild_configuration_invalid',
      service: deliveryService({
        targets: [policyTarget(['123456789', '987654321'])],
        bots: [deliveryBot()],
      }),
    },
    {
      expected: 'qq_bot_unavailable',
      service: deliveryService({ bots: [] }),
    },
    {
      expected: 'qq_delivery_failed',
      service: deliveryService({
        bots: [deliveryBot(async () => {
          throw new Error('raw upstream detail must not cross the boundary')
        })],
      }),
    },
  ]

  for (const item of cases) {
    const response = await postPayload(payload, item.service)
    assert.deepEqual(response, { status: 503, code: item.expected })
    assert.doesNotMatch(JSON.stringify(response), /raw upstream detail/)
  }

  const success = await postPayload(payload, deliveryService({ bots: [deliveryBot()] }))
  assert.deepEqual(success, { status: 200, code: 'delivered' })
})

test('Alertmanager payload validation rejects nonstandard and oversized alert groups', async () => {
  const invalidVersion = alertPayload('firing')
  invalidVersion.version = '3'
  const tooManyAlerts = alertPayload('firing')
  tooManyAlerts.alerts = Array.from({ length: 51 }, (_, index) => ({
    ...alertPayload('firing').alerts[0],
    fingerprint: index.toString(16).padStart(16, '0'),
  }))
  const service = deliveryService({ bots: [deliveryBot()] })

  assert.deepEqual(await postPayload(invalidVersion, service), {
    status: 400,
    code: 'invalid_alertmanager_payload',
  })
  assert.deepEqual(await postPayload(tooManyAlerts, service), {
    status: 400,
    code: 'invalid_alertmanager_payload',
  })
})

test('management target and native secret configuration fail closed', () => {
  assert.equal(resolveUniqueManagementGuildID([
    policyTarget([MANAGEMENT_GUILD_ID]),
    policyTarget([MANAGEMENT_GUILD_ID], 'policy-2'),
  ]), MANAGEMENT_GUILD_ID)
  assert.throws(
    () => resolveUniqueManagementGuildID([policyTarget([])]),
    /management_guild_configuration_invalid/,
  )
  assert.throws(
    () => validateAlertmanagerWebhookConfig({ enabled: true, bearerToken: 'short', botSelfID: '' }),
    /alertmanager_webhook_token_invalid/,
  )
  assert.throws(
    () => validateAlertmanagerWebhookConfig({ enabled: true, bearerToken: TEST_TOKEN, botSelfID: 'not-qq' }),
    /alertmanager_webhook_bot_self_id_invalid/,
  )
  assert.deepEqual(
    validateAlertmanagerWebhookConfig({ enabled: true, bearerToken: TEST_TOKEN, botSelfID: ' 2118785781 ' }),
    { enabled: true, bearerToken: TEST_TOKEN, botSelfID: '2118785781' },
  )
})

test('registered Koishi route bypasses the generic body parser and sends a real HTTP notification', async () => {
  const port = await freePort()
  const runtime = createKoishiTestRuntime()
  runtime.register(Server, { host: '127.0.0.1', port })
  runtime.register(MockBot, { selfId: '2118785781' })
  const sent: string[] = []
  const logger = { info() {}, warn() {} }

  registerAlertmanagerWebhook(runtime.root, {
    enabled: true,
    bearerToken: TEST_TOKEN,
    botSelfID: '2118785781',
  }, {
    platform: {
      async listAdmissionPolicyTargets() {
        return [policyTarget([MANAGEMENT_GUILD_ID])]
      },
    },
    getBots: () => runtime.root.bots as unknown as AlertDeliveryBot[],
  }, logger)

  try {
    await runtime.root.start()
    const bot = runtime.root.bots[0] as unknown as AlertDeliveryBot & { platform?: string }
    bot.platform = 'onebot'
    bot.sendMessage = async (_channelID, content) => {
      sent.push(String(content))
      return ['route-message-1']
    }

    const response = await fetch(`http://127.0.0.1:${port}/stuhelper/internal/alertmanager`, {
      method: 'POST',
      headers: {
        authorization: `Bearer ${TEST_TOKEN}`,
        'content-type': 'application/json',
      },
      body: JSON.stringify(alertPayload('firing')),
    })
    assert.equal(response.status, 200)
    assert.deepEqual(await response.json(), { success: true, code: 'delivered' })
    assert.equal(sent.length, 1)
  } finally {
    runtime.dispose()
  }
})

function deliveryService(options: {
  readonly targets?: readonly AdmissionPolicyTarget[]
  readonly bots: readonly AlertDeliveryBot[]
  readonly backendError?: boolean
}) {
  return new AlertmanagerDeliveryService({
    platform: {
      async listAdmissionPolicyTargets() {
        if (options.backendError) {
          throw new Error('backend details')
        }
        return options.targets ?? [policyTarget([MANAGEMENT_GUILD_ID])]
      },
    },
    getBots: () => options.bots,
  })
}

function deliveryBot(
  sendMessage: AlertDeliveryBot['sendMessage'] = async () => ['message-1'],
): AlertDeliveryBot {
  return {
    selfId: '2118785781',
    platform: 'onebot',
    sendMessage,
  }
}

function policyTarget(
  managementGuildIDs: readonly string[],
  policyID = 'policy-1',
): AdmissionPolicyTarget {
  return {
    policyID,
    platform: 'qq',
    guildID: '178037297',
    guardEnabled: true,
    joinHandlingStrategy: 'post_join_guard',
    linkWaitSeconds: 600,
    managementGuildIDs,
  }
}

function alertPayload(status: 'firing' | 'resolved') {
  return {
    version: '4',
    groupKey: '{}:{alertname="StuHelperBackendUnavailable"}',
    truncatedAlerts: 0,
    status,
    receiver: 'webhook',
    groupLabels: {
      alertname: 'StuHelperBackendUnavailable',
    },
    commonLabels: {
      alertname: 'StuHelperBackendUnavailable',
      severity: 'critical',
      service: 'backend',
    },
    commonAnnotations: {
      summary: 'StuHelper backend is unavailable',
    },
    externalURL: 'https://grafana.example.test/alerting',
    alerts: [{
      status,
      labels: {
        alertname: 'StuHelperBackendUnavailable',
        severity: 'critical',
        service: 'backend',
        instance: 'app:8080',
      },
      annotations: {
        summary: 'StuHelper backend is unavailable',
        description: 'The application health probe is failing.',
      },
      startsAt: '2026-08-07T00:00:00Z',
      endsAt: status === 'resolved' ? '2026-08-07T00:05:00Z' : '0001-01-01T00:00:00Z',
      generatorURL: 'https://prometheus.example.test/graph',
      fingerprint: '0123456789abcdef',
    }],
  }
}

async function postPayload(payload: unknown, service: AlertmanagerDeliveryService) {
  const body = Buffer.from(JSON.stringify(payload))
  return handleAlertmanagerHTTPRequest({
    authorization: `Bearer ${TEST_TOKEN}`,
    contentType: 'application/json',
    contentLength: String(body.byteLength),
    body: singleChunk(body),
  }, TEST_TOKEN, service)
}

async function* singleChunk(body: Uint8Array) {
  yield body
}

function freePort() {
  return new Promise<number>((resolve, reject) => {
    const server = createNetServer()
    server.once('error', reject)
    server.listen(0, '127.0.0.1', () => {
      const address = server.address()
      if (!address || typeof address === 'string') {
        server.close()
        reject(new Error('free port unavailable'))
        return
      }
      server.close((error) => error ? reject(error) : resolve(address.port))
    })
  })
}
