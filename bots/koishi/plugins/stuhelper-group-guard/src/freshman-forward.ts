import { h, type Universal } from 'koishi'

import type {
  FreshmanForwardItem,
  StuhelperGroupGuardMessageConfig,
} from '@stuhelper/koishi-shared'

import { resolveAdmissionSubjectPlatform } from './admission-subject-platform'
import { formatFreshmanForwardSummary } from './admission-format'
import { validateFreshmanMaterialURL } from './freshman-material-url'

export function resolveFreshmanForwardBot(
  bots: readonly FreshmanForwardBot[],
  item: FreshmanForwardItem,
) {
  if (item.botSelfID) {
    const bot = bots.find((runtime) => botMatchesForward(runtime, item))
    if (!bot) throw new Error(`freshman forward bot not found: ${item.platform || '*'}:${item.botSelfID}`)
    return bot
  }
  if (bots.length === 1) return bots[0]
  throw new Error(`freshman forward ${item.application.id} missing botSelfID for multi-bot runtime`)
}

export interface FreshmanForwardFailure {
  readonly guildID: string
  readonly error: unknown
}

export interface FreshmanForwardDeliveryResult {
  readonly deliveredGuildIDs: readonly string[]
  readonly failures: readonly FreshmanForwardFailure[]
}

export async function forwardFreshmanMaterial(
  bot: Universal.Methods,
  item: FreshmanForwardItem,
  messages?: Partial<StuhelperGroupGuardMessageConfig>,
): Promise<FreshmanForwardDeliveryResult> {
  if (!item.materialURL) {
    throw new Error(`freshman forward ${item.application.id} missing materialURL`)
  }
  if (!item.managementGuildIDs.length) {
    throw new Error(`freshman forward ${item.application.id} missing managementGuildIDs`)
  }
  const materialURL = validateFreshmanMaterialURL(item.materialURL)
  const content = [
    h.image(materialURL),
    formatFreshmanForwardSummary(item, messages),
  ].join('\n')
  const deliveredGuildIDs: string[] = []
  const failures: FreshmanForwardFailure[] = []
  for (const guildID of item.managementGuildIDs) {
    try {
      await bot.sendMessage(guildID, content)
      deliveredGuildIDs.push(guildID)
    } catch (error) {
      failures.push({ guildID, error })
    }
  }
  return { deliveredGuildIDs, failures }
}

export function formatFreshmanForwardFailures(failures: readonly FreshmanForwardFailure[]): string {
  return failures
    .map((failure) => `guild ${failure.guildID}: ${forwardErrorMessage(failure.error)}`)
    .join('；')
}

export interface FreshmanForwardBot extends Universal.Methods {
  platform?: string
  selfId: string
}

function botMatchesForward(bot: FreshmanForwardBot, item: FreshmanForwardItem) {
  const botPlatform = resolveAdmissionSubjectPlatform(bot.platform)
  const platformMatches = !item.platform || !botPlatform || botPlatform === item.platform
  return platformMatches && bot.selfId === item.botSelfID
}

function forwardErrorMessage(error: unknown) {
  return error instanceof Error ? error.message : String(error)
}
