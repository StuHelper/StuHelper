import type { ChatCompletionRequest, ChatCompletionResponse, ChatMessage } from '../../types'
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
    host.data.writeLog(`[ai] API 响应: ${JSON.stringify(response).substring(0, 200)}...`)
    return response as ChatCompletionResponse
  } catch (error: any) {
    const errorDetail = error.response?.data
      ? JSON.stringify(error.response.data).substring(0, 500)
      : error.message
    host.data.writeLog(`[ai] OpenAI API 调用出错: ${errorDetail}`)
    throw error
  }
}
