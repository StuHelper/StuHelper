import assert from 'node:assert/strict'
import test from 'node:test'

import { ReportService } from './report-service'

test('handleReport marks AI review as failed when the request times out', async () => {
  const originalFetch = globalThis.fetch
  const originalTimeout = AbortSignal.timeout
  const updates: Array<{ id: string; status: string; severity: string; summary: string | null }> = []
  const warnings: Array<{ message: string; meta: Record<string, unknown> | undefined }> = []

  globalThis.fetch = async (_input, init) => {
    const signal = init?.signal
    if (!(signal instanceof AbortSignal)) {
      return new Response(JSON.stringify({
        severity: 'low',
        summary: 'no-timeout',
      }), {
        status: 200,
        headers: {
          'content-type': 'application/json',
        },
      })
    }
    return await new Promise((_resolve, reject) => {
      signal.addEventListener('abort', () => {
        reject(signal.reason)
      }, { once: true })
    })
  }
  AbortSignal.timeout = () => {
    const controller = new AbortController()
    queueMicrotask(() => controller.abort(new Error('timeout')))
    return controller.signal
  }

  const service = new ReportService({
    store: {
      createReport: async () => ({ id: 'rp-1' }),
      appendEvent: async () => {},
      updateReportAIResult: async (input: {
        readonly id: string
        readonly aiStatus: string
        readonly aiSeverity: string
        readonly aiSummary: string | null
      }) => {
        const { id, aiStatus: status, aiSeverity: severity, aiSummary: summary } = input
        updates.push({ id, status, severity, summary })
      },
      createReview: async () => {
        throw new Error('createReview should not be called when AI times out')
      },
    } as any,
    actions: {
      warnMember: async () => {
        throw new Error('warnMember should not be called when AI times out')
      },
      muteMember: async () => {
        throw new Error('muteMember should not be called when AI times out')
      },
    } as any,
    logger: {
      warn(message: string, meta?: Record<string, unknown>) {
        warnings.push({ message, meta })
      },
    } as any,
    aiSettings: async () => ({
      enabled: true,
      endpoint: 'https://example.test/review',
      apiKey: 'test-key',
      model: 'gpt-test',
    }),
  })

  try {
    const message = await service.handleReport(createSession(), '10002', '恶意刷屏')
    assert.equal(message, '举报已记录，但 AI 审核失败，事件已保留供人工处理。')
    assert.deepEqual(updates, [{
      id: 'rp-1',
      status: 'failed',
      severity: 'none',
      summary: null,
    }])
    assert.equal(warnings[0]?.message, 'report ai review failed')
  } finally {
    globalThis.fetch = originalFetch
    AbortSignal.timeout = originalTimeout
  }
})

test('handleReport uses runtime AI settings and records disabled status without fetch', async () => {
  const originalFetch = globalThis.fetch
  const reports: Array<{ aiStatus: string }> = []
  let fetched = false

  globalThis.fetch = async () => {
    fetched = true
    return new Response('{}', { status: 200 })
  }

  const service = new ReportService({
    store: {
      createReport: async (input: { aiStatus: string }) => {
        reports.push(input)
        return { id: 'rp-disabled' }
      },
      appendEvent: async () => {},
      updateReportAIResult: async () => {
        throw new Error('updateReportAIResult should not be called when AI is disabled')
      },
      createReview: async () => {
        throw new Error('createReview should not be called when AI is disabled')
      },
    } as any,
    actions: {
      warnMember: async () => {
        throw new Error('warnMember should not be called when AI is disabled')
      },
      muteMember: async () => {
        throw new Error('muteMember should not be called when AI is disabled')
      },
    } as any,
    logger: { warn: () => {} } as any,
    aiSettings: async () => ({
      enabled: false,
      endpoint: 'https://example.test/review',
      apiKey: 'test-key',
      model: 'gpt-test',
    }),
  })

  try {
    const message = await service.handleReport(createSession(), '10002', '恶意刷屏')
    assert.equal(message, '举报已记录。当前未启用 AI 审核，事件已进入人工处理范围。')
    assert.equal(reports[0]?.aiStatus, 'disabled')
    assert.equal(fetched, false)
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('handleReport 把畸形 AI 响应按失败处理而不是默认 none', async () => {
  const originalFetch = globalThis.fetch
  const malformedPayloads = [
    JSON.stringify({ verdict: 'ok' }),
    JSON.stringify({ severity: 'catastrophic', summary: '未知等级' }),
    JSON.stringify('not-an-object'),
  ]
  let reportIndex = 0
  let payloadIndex = 0
  const updates: Array<{ id: string; status: string; severity: string; summary: string | null }> = []
  const warnings: string[] = []

  globalThis.fetch = async () => new Response(malformedPayloads[payloadIndex++], {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })

  const service = new ReportService({
    store: {
      createReport: async () => ({ id: `rp-${++reportIndex}` }),
      appendEvent: async () => {},
      updateReportAIResult: async (input: {
        readonly id: string
        readonly aiStatus: string
        readonly aiSeverity: string
        readonly aiSummary: string | null
      }) => {
        updates.push({
          id: input.id,
          status: input.aiStatus,
          severity: input.aiSeverity,
          summary: input.aiSummary,
        })
      },
      createReview: async () => {
        throw new Error('createReview should not be called for malformed AI responses')
      },
    } as any,
    actions: {
      warnMember: async () => {
        throw new Error('warnMember should not be called for malformed AI responses')
      },
      muteMember: async () => {
        throw new Error('muteMember should not be called for malformed AI responses')
      },
    } as any,
    logger: {
      warn(message: string) {
        warnings.push(message)
      },
    } as any,
    aiSettings: async () => ({
      enabled: true,
      endpoint: 'https://example.test/review',
      apiKey: 'test-key',
      model: 'gpt-test',
    }),
  })

  try {
    for (let index = 0; index < malformedPayloads.length; index += 1) {
      const message = await service.handleReport(createSession(), '10002', '恶意刷屏')
      assert.equal(message, '举报已记录，但 AI 审核失败，事件已保留供人工处理。')
    }
    assert.deepEqual(updates, [
      { id: 'rp-1', status: 'failed', severity: 'none', summary: null },
      { id: 'rp-2', status: 'failed', severity: 'none', summary: null },
      { id: 'rp-3', status: 'failed', severity: 'none', summary: null },
    ])
    assert.equal(updates.some((item) => item.status === 'completed'), false)
    assert.equal(warnings.length, malformedPayloads.length)
  } finally {
    globalThis.fetch = originalFetch
  }
})

test('handleReport 接受合法 AI 响应并按 severity 处理', async () => {
  const originalFetch = globalThis.fetch
  const updates: Array<{ status: string; severity: string; summary: string | null }> = []

  globalThis.fetch = async () => new Response(JSON.stringify({
    severity: 'none',
    summary: '内容无违规',
    extraProviderField: 'ignored',
  }), {
    status: 200,
    headers: { 'content-type': 'application/json' },
  })

  const service = new ReportService({
    store: {
      createReport: async () => ({ id: 'rp-ok' }),
      appendEvent: async () => {},
      updateReportAIResult: async (input: {
        readonly aiStatus: string
        readonly aiSeverity: string
        readonly aiSummary: string | null
      }) => {
        updates.push({ status: input.aiStatus, severity: input.aiSeverity, summary: input.aiSummary })
      },
      createReview: async () => {
        throw new Error('createReview should not be called for severity none')
      },
    } as any,
    actions: {
      warnMember: async () => {
        throw new Error('warnMember should not be called for severity none')
      },
      muteMember: async () => {
        throw new Error('muteMember should not be called for severity none')
      },
    } as any,
    logger: { warn: () => {} } as any,
    aiSettings: async () => ({
      enabled: true,
      endpoint: 'https://example.test/review',
      apiKey: 'test-key',
      model: 'gpt-test',
    }),
  })

  try {
    const message = await service.handleReport(createSession(), '10002', '恶意刷屏')
    assert.equal(message, '举报已提交，AI 未判定出可执行违规动作。')
    assert.deepEqual(updates, [{ status: 'completed', severity: 'none', summary: '内容无违规' }])
  } finally {
    globalThis.fetch = originalFetch
  }
})

function createSession() {
  return {
    guildId: 'group-1',
    channelId: 'group-1',
    platform: 'onebot',
    selfId: '514',
    userId: '10001',
    bot: {
      platform: 'onebot',
      selfId: '514',
    },
  } as any
}
