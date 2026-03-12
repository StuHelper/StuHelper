import createClient from 'openapi-fetch'
import type { paths } from '../types/api.gen'

export interface ApiClientOptions {
  baseUrl: string
  credentials?: RequestCredentials
  fetch?: (input: Request) => Promise<Response>
}

export const createApiClient = (options: ApiClientOptions) => {
  return createClient<paths>({
    baseUrl: options.baseUrl,
    credentials: options.credentials,
    fetch: options.fetch
  })
}

export type ApiClient = ReturnType<typeof createApiClient>
