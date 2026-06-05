import type { ChatCompletionRequest, ChatCompletionResponse, ChatMessage } from '../../types'
import { isRecord, toLogSnippet, unknownErrorMessage } from './ai-log-format'
import type { AIModule } from './ai.module'

export interface OpenAIRequestInput {
  readonly messages: ChatMessage[]
  readonly model: string
  readonly temperature: number
  readonly maxTokens: number
  readonly apiKey: string
  readonly apiUrl: string
}

export async function callOpenAI(
  host: AIModule,
  input: OpenAIRequestInput,
): Promise<ChatCompletionResponse> {
  const endpoint = `${input.apiUrl.endsWith('/') ? input.apiUrl : input.apiUrl + '/'}chat/completions`
  const requestBody: ChatCompletionRequest = {
    model: input.model,
    messages: input.messages,
    temperature: input.temperature,
    max_tokens: input.maxTokens,
  }

  try {
    host.data.writeLog(`[ai] 调用 API: ${endpoint}, model: ${input.model}`)
    const response = await host.ctx.http.post(endpoint, requestBody, {
      headers: {
        'Content-Type': 'application/json',
        'Authorization': `Bearer ${input.apiKey}`,
      },
    })
    host.data.writeLog(`[ai] API 响应: ${toLogSnippet(response, 200)}`)
    return response as ChatCompletionResponse
  } catch (error: unknown) {
    const errorDetail = httpErrorDetail(error)
    host.data.writeLog(`[ai] OpenAI API 调用出错: ${errorDetail}`)
    throw error
  }
}

function httpErrorDetail(error: unknown): string {
  if (isRecord(error) && isRecord(error.response) && error.response.data !== undefined) {
    return toLogSnippet(error.response.data, 500)
  }
  return unknownErrorMessage(error)
}
