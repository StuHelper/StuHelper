import createClient from 'openapi-fetch'
import type { paths } from '../types/api.gen'

type OpenApiClient = ReturnType<typeof createClient<paths>>

export interface ApiClientOptions {
  baseUrl: string
  credentials?: RequestCredentials
  fetch?: typeof fetch
}

/**
 * Shared API client surface used by all frontends.
 *
 * Keep this structural instead of exporting the concrete openapi-fetch return
 * type directly, so non-browser runtimes (admin wrappers, uniappx transport)
 * can implement the same contract and still reuse the generated shared API
 * factories.
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
