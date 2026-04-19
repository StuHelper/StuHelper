import createClient from 'openapi-fetch'
import type { paths } from '../types/api.gen'

type OpenApiClient = ReturnType<typeof createClient<paths>>

export interface ApiClientOptions {
  baseUrl: string
  credentials?: RequestCredentials
  fetch?: typeof fetch
}

/**
 * 所有前端共用的 API client 结构。
 *
 * 这里暴露结构类型，而不是直接暴露 openapi-fetch 的具体返回类型，
 * 这样 admin 包装层、uniappx 传输层等非浏览器运行时也能实现同一契约，
 * 并复用共享的 API 工厂。
 */
export interface ApiClient {
  GET: OpenApiClient['GET']
  PUT: OpenApiClient['PUT']
  PATCH: OpenApiClient['PATCH']
  POST: OpenApiClient['POST']
  DELETE: OpenApiClient['DELETE']
}

export const createApiClient = (options: ApiClientOptions): OpenApiClient => {
  return createClient<paths>({
    baseUrl: options.baseUrl,
    credentials: options.credentials,
    fetch: options.fetch
  })
}
