import assert from 'node:assert/strict'
import test from 'node:test'

import type { ChatCompletionResponse } from '../../types'
import type { AIModule } from './ai.module'
import { callOpenAI, type OpenAIRequestInput } from './ai-openai.ts'

type TestAIModule = AIModule & {
  readonly logs: string[]
}

function createHost(post: () => Promise<unknown>): TestAIModule {
  const logs: string[] = []
  return {
    logs,
    ctx: {
      http: {
        post,
      },
    },
    data: {
      writeLog: (message: string) => logs.push(message),
    },
  } as unknown as TestAIModule
}

function requestInput(): OpenAIRequestInput {
  return {
    messages: [{ role: 'user', content: 'hello' }],
    model: 'test-model',
    temperature: 0.1,
    maxTokens: 16,
    apiKey: 'test-key',
    apiUrl: 'https://api.example.test/v1',
  }
}

function circularObject(): Record<string, unknown> {
  const value: Record<string, unknown> = {}
  value.self = value
  return value
}

test('callOpenAI logs circular API responses without throwing again', async () => {
  const response = circularObject()
  const host = createHost(async () => response)

  const result = await callOpenAI(host, requestInput())

  assert.equal(result, response as unknown as ChatCompletionResponse)
  assert.equal(host.logs[0], '[ai] 调用 API: https://api.example.test/v1/chat/completions, model: test-model')
  assert.equal(host.logs[1], '[ai] API 响应: {"self":"[Circular]"}')
})

test('callOpenAI logs structured HTTP error details safely', async () => {
  const thrown = {
    response: {
      data: circularObject(),
    },
  }
  const host = createHost(async () => {
    throw thrown
  })

  await assert.rejects(
    () => callOpenAI(host, requestInput()),
    (error) => error === thrown,
  )
  assert.equal(host.logs[0], '[ai] 调用 API: https://api.example.test/v1/chat/completions, model: test-model')
  assert.equal(host.logs[1], '[ai] OpenAI API 调用出错: {"self":"[Circular]"}')
})

test('callOpenAI redacts secrets from structured HTTP error logs', async () => {
  const thrown = {
    response: {
      data: {
        error: {
          message: 'Authorization: Bearer openai_error_secret',
          apiKey: 'sk-openai-secret',
          headers: {
            cookie: 'sid=openai_cookie_secret',
          },
        },
      },
    },
  }
  const host = createHost(async () => {
    throw thrown
  })

  await assert.rejects(
    () => callOpenAI(host, requestInput()),
    (error) => error === thrown,
  )
  const payload = host.logs.join('\n')

  assert.doesNotMatch(payload, /openai_error_secret/)
  assert.doesNotMatch(payload, /sk-openai-secret/)
  assert.doesNotMatch(payload, /openai_cookie_secret/)
  assert.match(payload, /\[REDACTED\]/)
})
