import { h } from 'koishi'

import type { AIModule } from './ai.module'

export function registerAiMiddleware(host: AIModule): void {
  host.ctx.middleware(async (session, next) => {
    if (!isMentioningBot(session) || session.content?.startsWith('/')) {
      return next()
    }
    if (!host.config.openai?.enabled) return next()

    try {
      const content = stripBotMention(session.content, session.selfId)
      if (!content) return next()

      const response = await host.processMessage(session.userId, content, session.guildId)
      return `${h.quote(session.messageId)}${h.at(session.userId)} ${response}`
    } catch (error) {
      host.data.writeLog(`[ai] AI中间件处理失败: ${error}`)
      return next()
    }
  })
}

function isMentioningBot(session: any): boolean {
  return session.elements?.some((el: any) => el.type === 'at' && el.attrs?.id === session.selfId)
}

function stripBotMention(content: string | undefined, selfId: string): string | undefined {
  return content
    ?.replace(new RegExp(`<at id="${selfId}"/>`, 'g'), '')
    .replace(new RegExp(`@${selfId}`, 'g'), '')
    .trim()
}
