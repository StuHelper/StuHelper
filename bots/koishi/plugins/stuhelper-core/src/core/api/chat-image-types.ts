import type { ConsoleGuildScope } from './console-guild-scope'

export interface ChatImageFetchParams {
  url?: string
  file?: string
}

export interface ChatImageFetchRequest {
  url: string
  file: string
}

export interface ChatImageFetchResult {
  dataUrl: string
  hash: string
  mimeType: string
  source: string
}

export interface ChatImageAccessRegistry {
  remember(elements: unknown, guildId: string | undefined): void
  assertAllowed(params: ChatImageFetchParams, scope: ConsoleGuildScope): ChatImageFetchRequest
}

export interface ChatImageLogger {
  warn(format: string, ...args: unknown[]): void
}
