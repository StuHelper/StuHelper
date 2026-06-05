import type { Bot } from 'koishi'

const ONEBOT_PLATFORM = 'onebot'

export interface OneBotGroupMemberInfo {
  readonly shut_up_timestamp?: unknown
}

export interface OneBotStrangerInfo {
  readonly level?: unknown
}

export interface OneBotInternalMethods {
  getImage(file: string): Promise<unknown>
  getGroupMemberInfo(guildId: string, userId: string, noCache?: boolean): Promise<OneBotGroupMemberInfo>
  getStrangerInfo(userId: string, noCache?: boolean): Promise<OneBotStrangerInfo>
  setGroupAdmin(guildId: string, userId: string, enabled: boolean): Promise<unknown>
  setGroupCard(guildId: string, userId: string, nickname?: string): Promise<unknown>
  setGroupLeave(guildId: string, dismiss?: boolean): Promise<unknown>
  setGroupSpecialTitle(guildId: string, userId: string, title: string): Promise<unknown>
}

export type OneBotInternalMethodName = keyof OneBotInternalMethods

export function getOneBotInternalMethod<Name extends OneBotInternalMethodName>(
  bot: Bot,
  methodName: Name,
): OneBotInternalMethods[Name] | null {
  if (bot.platform !== ONEBOT_PLATFORM) return null

  const internal = readInternal(bot)
  const method = internal?.[methodName]
  if (typeof method !== 'function') return null

  return method.bind(internal) as OneBotInternalMethods[Name]
}

export function requireOneBotInternalMethod<Name extends OneBotInternalMethodName>(
  bot: Bot,
  methodName: Name,
  actionName: string = methodName,
): OneBotInternalMethods[Name] {
  const method = getOneBotInternalMethod(bot, methodName)
  if (!method) {
    throw new Error(`当前适配器不支持 OneBot ${actionName}`)
  }
  return method
}

function readInternal(bot: Bot): Record<string, unknown> | null {
  const internal = (bot as unknown as { readonly internal?: unknown }).internal
  if (!isRecord(internal)) return null

  return internal
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null
}
