import assert from 'node:assert/strict'
import test from 'node:test'

import type { ChatCompletionResponse, ChatMessage, UserContext } from '../../types'
import type { AIModule } from './ai.module'
import { callAiModeration, processAiMessage, translateAiText } from './ai-processing.ts'

type TestAIModule = AIModule & {
  readonly logs: string[]
}

function createHost(
  callOpenAI: () => Promise<ChatCompletionResponse>,
): TestAIModule {
  const logs: string[] = []
  const userContexts = new Map<string, UserContext>()

  return {
    logs,
    userContexts,
    config: {
      openai: {
        enabled: true,
        apiKey: 'test-key',
        model: 'test-model',
        systemPrompt: 'system prompt',
      },
    },
    data: {
      writeLog: (message: string) => logs.push(message),
    },
    getGroupConfig: () => undefined,
    addMessageToContext(input: {
      readonly userId: string
      readonly message: ChatMessage
      readonly systemPrompt: string
      readonly contextLimit: number
    }): void {
      let context = userContexts.get(input.userId)
      if (!context) {
        context = {
          userId: input.userId,
          messages: [{ role: 'system', content: input.systemPrompt }],
          lastTimestamp: Date.now(),
        }
        userContexts.set(input.userId, context)
      }
      context.messages.push(input.message)
      context.lastTimestamp = Date.now()
    },
    callOpenAI,
  } as unknown as TestAIModule
}

function completion(content: string, role: ChatMessage['role'] = 'assistant'): ChatCompletionResponse {
  return {
    id: 'completion-1',
    object: 'chat.completion',
    created: 1,
    model: 'test-model',
    choices: [{
      index: 0,
      message: { role, content },
      finish_reason: 'stop',
    }],
    usage: {
      prompt_tokens: 1,
      completion_tokens: 1,
      total_tokens: 2,
    },
  }
}

function circularResponse(): ChatCompletionResponse {
  const response: Record<string, unknown> = {}
  response.self = response
  return response as unknown as ChatCompletionResponse
}

test('processAiMessage normalizes assistant response before saving context', async () => {
  const host = createHost(async () => completion('hello', 'function'))

  const result = await processAiMessage(host, {
    userId: 'user-1',
    content: 'hi',
  })

  assert.equal(result, 'hello')
  assert.deepEqual(host.userContexts.get('user-1')?.messages.at(-1), {
    role: 'assistant',
    content: 'hello',
  })
})

test('processAiMessage reports malformed circular responses without throwing again', async () => {
  const host = createHost(async () => circularResponse())

  const result = await processAiMessage(host, {
    userId: 'user-1',
    content: 'hi',
  })

  assert.equal(result, '处理消息时出错: API 响应格式异常，缺少 choices 字段')
  assert.match(host.logs[0], /^\[ai\] API 响应格式异常: \[unserializable value:/)
  assert.equal(host.logs[1], '[ai] AI处理消息失败: API 响应格式异常，缺少 choices 字段')
})

test('translateAiText reports string failures without losing the cause', async () => {
  const host = createHost(async () => {
    throw 'network down'
  })

  const result = await translateAiText(host, {
    userId: 'user-1',
    text: 'hello',
  })

  assert.equal(result, '翻译出错: network down')
  assert.equal(host.logs.at(-1), '[ai] AI翻译失败: network down')
})

test('callAiModeration rejects malformed choices with a stable message', async () => {
  const host = createHost(async () => ({
    ...completion('unused'),
    choices: [{}],
  } as unknown as ChatCompletionResponse))

  await assert.rejects(
    () => callAiModeration(host, 'review this'),
    { message: '内容审核失败: API 响应缺少 message.content' },
  )
  assert.equal(host.logs[0], '[ai] API 响应缺少内容: {}')
  assert.equal(host.logs[1], '[ai] AI内容审核失败: API 响应缺少 message.content')
})
