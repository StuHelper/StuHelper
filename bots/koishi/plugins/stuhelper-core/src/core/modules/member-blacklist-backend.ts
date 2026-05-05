import type {
  MemberBlacklistEntry,
  MemberBlacklistListResult,
  MemberBlacklistScopeType,
  PlatformClient,
} from '@stuhelper/koishi-shared'

export const MEMBER_BLACKLIST_PAGE_SIZE = 200

export type MemberBlacklistBackend = Pick<
  PlatformClient,
  'createMemberBlacklist' | 'listMemberBlacklist' | 'releaseMemberBlacklistBySubject'
>

export interface MemberBlacklistCommandInput {
  readonly platform: string
  readonly guildID: string
  readonly subjectID: string
  readonly operatorQQID: string
  readonly rawCommand: string
  readonly global?: boolean
}

export async function listVisibleMemberBlacklists(
  backend: MemberBlacklistBackend,
  platform: string,
  guildID: string,
): Promise<readonly MemberBlacklistEntry[]> {
  const [globalItems, guildItems] = await Promise.all([
    listMemberBlacklistPage(backend, { platform, scopeType: 'global' }),
    listMemberBlacklistPage(backend, { platform, scopeType: 'guild', guildID }),
  ])
  return [...globalItems.list, ...guildItems.list]
}

export function createManualMemberBlacklist(
  backend: MemberBlacklistBackend,
  input: MemberBlacklistCommandInput,
) {
  return backend.createMemberBlacklist({
    ...memberBlacklistSubject(input),
    source: 'manual_admin',
    reasonCode: 'manual_blacklist',
    reasonText: 'manual blacklist from Koishi command',
    metadata: memberBlacklistMetadata(input, 'qq_command'),
  })
}

export function releaseManualMemberBlacklist(
  backend: MemberBlacklistBackend,
  input: MemberBlacklistCommandInput,
) {
  return backend.releaseMemberBlacklistBySubject({
    ...memberBlacklistSubject(input),
    releaseReasonCode: 'manual_pardon',
    releaseReason: 'manual release from Koishi command',
    operatorQQID: input.operatorQQID,
  })
}

export function createKickMemberBlacklist(
  backend: Pick<PlatformClient, 'createMemberBlacklist'>,
  input: MemberBlacklistCommandInput,
) {
  return backend.createMemberBlacklist({
    ...memberBlacklistSubject(input),
    source: 'kick_blacklist',
    reasonCode: 'manual_kick_blacklist',
    reasonText: 'manual kick with blacklist from Koishi command',
    metadata: memberBlacklistMetadata(input, 'qq_command'),
  })
}

function listMemberBlacklistPage(
  backend: MemberBlacklistBackend,
  input: {
    readonly platform: string
    readonly scopeType: MemberBlacklistScopeType
    readonly guildID?: string
  },
): Promise<MemberBlacklistListResult> {
  return backend.listMemberBlacklist({
    ...input,
    subjectType: 'qq_user',
    status: 'active',
    pageSize: MEMBER_BLACKLIST_PAGE_SIZE,
  })
}

function memberBlacklistSubject(input: MemberBlacklistCommandInput) {
  const base = {
    platform: input.platform,
    subjectType: 'qq_user' as const,
    subjectID: input.subjectID,
    scopeType: memberBlacklistScope(input),
  }
  if (base.scopeType === 'global') return base
  return { ...base, guildID: input.guildID }
}

function memberBlacklistScope(input: MemberBlacklistCommandInput): MemberBlacklistScopeType {
  return input.global ? 'global' : 'guild'
}

function memberBlacklistMetadata(
  input: MemberBlacklistCommandInput,
  createdFrom: 'qq_command',
): Record<string, unknown> {
  return {
    operatorQQID: input.operatorQQID,
    guildID: input.guildID,
    rawCommand: input.rawCommand,
    createdFrom,
  }
}
