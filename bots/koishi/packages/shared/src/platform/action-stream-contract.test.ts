import assert from 'node:assert/strict'
import { readFileSync } from 'node:fs'
import path from 'node:path'
import test from 'node:test'

import { createPlatformClient } from './index'

// 与 server 侧 internal/modules/admission/stream_contract_test.go 共享同一份
// fixture，双侧字节级锁定 admission action SSE 协议。server 改协议会先在 Go
// 测试失败并更新 fixture，本测试随即用新字节验证解析器仍然兼容。
const CONTRACT_PATH = path.resolve(
  __dirname,
  '../../../../../../server/api/contracts/admission-action-stream.json',
)

interface AdmissionActionStreamContract {
  commentOnOpen: string
  frames: Record<string, string>
  actionPayload: Record<string, unknown>
}

const contract = JSON.parse(
  readFileSync(CONTRACT_PATH, 'utf8'),
) as AdmissionActionStreamContract

test('契约帧回放：action 完整分发，keepalive/error/注释忽略，关流触发重连错误', async (t) => {
  const originalFetch = globalThis.fetch
  const actions: unknown[] = []
  let streamError: unknown
  globalThis.fetch = async () =>
    new Response(
      new ReadableStream({
        start(controller) {
          controller.enqueue(
            new TextEncoder().encode(
              contract.commentOnOpen +
                contract.frames.keepalive +
                contract.frames.action +
                contract.frames.error,
            ),
          )
          controller.close()
        },
      }),
      {
        status: 200,
        headers: { 'content-type': 'text/event-stream' },
      },
    )
  t.after(() => {
    globalThis.fetch = originalFetch
  })

  const client = createPlatformClient({
    baseUrl: 'https://api.example.test',
    serviceToken: 'service-token',
  })

  client.streamAdmissionActions(
    { platform: 'qq', botSelfID: '2118785781' },
    {
      onAction(action) {
        actions.push(action)
      },
      onError(error) {
        streamError = error
      },
    },
  )

  await waitFor(() => actions.length > 0 && streamError !== undefined)

  assert.deepEqual(actions, [contract.actionPayload])
  // error 事件本身不分发；服务端随后关流，解析器以"流关闭"错误交给重连逻辑。
  assert.match(String(streamError), /admission action stream closed/)
})

test('契约只允许解析器已知的事件名与帧格式', () => {
  const knownEvents = new Set(['action', 'keepalive', 'error'])
  assert.ok(Object.keys(contract.frames).length > 0)
  for (const [name, frame] of Object.entries(contract.frames)) {
    assert.ok(knownEvents.has(name), `解析器不认识契约事件 ${name}`)
    assert.ok(frame.startsWith(`event:${name}\n`), `帧 ${name} 必须以 event:${name} 开头`)
    assert.ok(frame.endsWith('\n\n'), `帧 ${name} 必须以空行结束`)
  }
  assert.ok(contract.commentOnOpen.startsWith(': '), '开流注释必须是 SSE 注释行')
})

async function waitFor(predicate: () => boolean, timeoutMs = 500) {
  const deadline = Date.now() + timeoutMs
  while (Date.now() < deadline) {
    if (predicate()) return
    await new Promise((resolve) => setTimeout(resolve, 10))
  }
  assert.fail('condition not met before timeout')
}
