import type { Session } from 'koishi'

export type ReportActionBot = Pick<Session['bot'], 'kickGuildMember' | 'muteGuildMember'>

export type ReportActionSession = Session & {
  readonly bot: ReportActionBot
  readonly guildId?: string
  readonly messageId?: string
  readonly platform?: string
}
