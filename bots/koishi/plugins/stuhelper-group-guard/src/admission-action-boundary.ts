import type {
  AdmissionPendingAction,
  GuardPolicyStore,
} from '@stuhelper/koishi-shared'

import type { GuardBotRuntime } from './member-guard'
import type { GuardMemberRecord } from './model'

const ADMISSION_ACTION_PLATFORM = 'qq'

interface AdmissionActionBoundaryInput {
  readonly bot: GuardBotRuntime
  readonly action: AdmissionPendingAction
  readonly record: GuardMemberRecord | null
  readonly policyStore: GuardPolicyStore
}

export function requireAdmissionActionPlatform(bot: GuardBotRuntime) {
  if (bot.platform !== ADMISSION_ACTION_PLATFORM) {
    throw new Error(`admission action worker requires platform ${ADMISSION_ACTION_PLATFORM}`)
  }
  return bot.platform
}

export async function assertAdmissionActionBoundary(input: AdmissionActionBoundaryInput) {
  const platform = requireAdmissionActionPlatform(input.bot)
  assertActionBot(input.action, input.bot, platform)
  const guildID = resolveActionGuildID(input.action, input.record)
  assertRecordMatchesAction(input.record, input.action, input.bot, platform, guildID)
  const policy = await input.policyStore.resolvePolicy(platform, guildID)
  if (!policy) {
    throw new Error(`admission action ${input.action.sessionID} targets unmanaged guild ${guildID}`)
  }
}

function assertActionBot(action: AdmissionPendingAction, bot: GuardBotRuntime, platform: string) {
  if (action.platform && action.platform !== platform) {
    throw new Error(`admission action ${action.sessionID} platform mismatch`)
  }
  if (action.botSelfID && action.botSelfID !== bot.selfId) {
    throw new Error(`admission action ${action.sessionID} botSelfID mismatch`)
  }
}

function resolveActionGuildID(action: AdmissionPendingAction, record: GuardMemberRecord | null) {
  const guildID = action.guildID ?? record?.guildId
  if (!guildID) {
    throw new Error(`admission action ${action.sessionID} missing guildID for local policy check`)
  }
  return guildID
}

function assertRecordMatchesAction(
  record: GuardMemberRecord | null,
  action: AdmissionPendingAction,
  bot: GuardBotRuntime,
  platform: string,
  guildID: string,
) {
  if (!record) return
  if (record.platform !== platform) {
    throw new Error(`admission action ${action.sessionID} record platform mismatch`)
  }
  if (record.botSelfId !== bot.selfId) {
    throw new Error(`admission action ${action.sessionID} record botSelfID mismatch`)
  }
  if (record.guildId !== guildID) {
    throw new Error(`admission action ${action.sessionID} record guildID mismatch`)
  }
}
