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
      updateReportAIResult: async (id: string, status: string, severity: string, summary: string | null) => {
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
    config: {
      moderation: {
        defaultMuteSeconds: 180,
      },
      ai: {
        enabled: true,
        endpoint: 'https://example.test/review',
        apiKey: 'test-key',
        model: 'gpt-test',
      },
    } as any,
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
