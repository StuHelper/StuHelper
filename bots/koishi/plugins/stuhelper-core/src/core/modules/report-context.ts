import type { Session } from 'koishi'

import type { ReportGuildConfig } from '../../types'
import type { ReportModule } from './report.module'

const DEFAULT_CONTEXT_SIZE = 5
const CONTEXT_BUFFER_FACTOR = 2

export function registerReportMessageListener(host: ReportModule): void {
  host.ctx.on('message', (session) => recordReportContext(host, session))
}

export function buildReportPrompt(input: {
  readonly host: ReportModule
  readonly guildId: string
  readonly content: string
  readonly guildConfig: ReportGuildConfig | null
}): string {
  const { host, guildId, content, guildConfig } = input
  if (!guildConfig?.includeContext) {
    return host.getDefaultPrompt().replace('{content}', content)
  }

  return host.getContextPrompt()
    .replace('{context}', formatContextMessages(host, guildId, guildConfig))
    .replace('{content}', content)
}

function recordReportContext(host: ReportModule, session: Session): void {
  if (!session.guildId || !session.content) return

  const guildConfig = host.getReportGuildConfig(session.guildId)
  if (!guildConfig?.includeContext) return

  const guildId = session.guildId
  host.guildMessages[guildId] = host.guildMessages[guildId] || []
  host.guildMessages[guildId].push({
    userId: session.userId,
    content: session.content,
    timestamp: Date.now(),
  })

  trimContextMessages(host, guildId, guildConfig.contextSize || DEFAULT_CONTEXT_SIZE)
}

function trimContextMessages(host: ReportModule, guildId: string, contextSize: number): void {
  const maxMessages = contextSize * CONTEXT_BUFFER_FACTOR
  if (host.guildMessages[guildId].length <= maxMessages) return
  host.guildMessages[guildId] = host.guildMessages[guildId].slice(-maxMessages)
}

function formatContextMessages(
  host: ReportModule,
  guildId: string,
  guildConfig: ReportGuildConfig,
): string {
  const contextSize = guildConfig.contextSize || DEFAULT_CONTEXT_SIZE
  const contextMessages = host.guildMessages[guildId] || []
  return [...contextMessages]
    .sort((left, right) => left.timestamp - right.timestamp)
    .slice(-contextSize)
    .map((message, index) => `消息${index + 1} [用户${message.userId}]: ${message.content}`)
    .join('\n')
}
