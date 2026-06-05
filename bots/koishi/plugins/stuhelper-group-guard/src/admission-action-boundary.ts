import type {
  AdmissionPendingAction,
  GuardMemberRecord,
  GuardPolicyStore,
} from '@stuhelper/koishi-shared'

import {
  requireAdmissionSubjectPlatform,
  resolveAdmissionSubjectPlatform,
} from './admission-subject-platform'
import type { GuardBotRuntime } from './member-guard'

interface AdmissionActionBoundaryInput {
  readonly bot: GuardBotRuntime
  readonly action: AdmissionPendingAction
  readonly record: GuardMemberRecord | null
  readonly policyStore: GuardPolicyStore
}

export function requireAdmissionActionPlatform(bot: GuardBotRuntime) {
  return requireAdmissionSubjectPlatform(bot.platform)
}

export function isAdmissionActionPlatform(bot: GuardBotRuntime) {
  return Boolean(resolveAdmissionSubjectPlatform(bot.platform))
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
