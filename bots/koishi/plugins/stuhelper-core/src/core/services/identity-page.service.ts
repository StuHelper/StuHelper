import type { GuardMemberRecord, QQVerificationStatus } from '@stuhelper/koishi-shared'

import { serializeGuardMember } from './page-serializers'
import type {
  IdentityLookupError,
  IdentityMemberSnapshot,
  IdentityPageData,
} from './page-types'

export interface IdentityPageBuilderInput {
  generatedAt: string
  guardRecords: GuardMemberRecord[]
  verificationProfiles: QQVerificationStatus[]
  lookupErrors: IdentityLookupError[]
}

export function buildIdentityPageData(input: IdentityPageBuilderInput): IdentityPageData {
  const profiles = new Map(input.verificationProfiles.map((profile) => [profile.qqID, profile]))
  const activeMembers = input.guardRecords.filter((record) => !record.releasedAt && !record.kickedAt)
  const releasedMembers = input.guardRecords.filter((record) => Boolean(record.releasedAt))

  return {
    generatedAt: input.generatedAt,
    summary: {
      pendingMembers: activeMembers.length,
      verifiedMembers: input.verificationProfiles.filter((item) => item.verificationState === 'verified').length,
      boundUnverifiedMembers: input.verificationProfiles.filter((item) => item.verificationState === 'bound_unverified').length,
      unboundMembers: input.verificationProfiles.filter((item) => item.verificationState === 'unbound').length,
      releasedMembers: releasedMembers.length,
    },
    groups: buildGroupSummaries(input.guardRecords),
    members: activeMembers
      .sort((left, right) => left.deadlineAt.getTime() - right.deadlineAt.getTime())
      .map((record) => joinProfile(record, profiles.get(record.memberId))),
    recentReleases: releasedMembers
      .sort((left, right) => (right.releasedAt?.getTime() || 0) - (left.releasedAt?.getTime() || 0))
      .slice(0, 12)
      .map((record) => joinProfile(record, profiles.get(record.memberId))),
    lookupErrors: [...input.lookupErrors],
  }
}

export interface IdentityPageServiceDeps {
  loadGuardRecords: () => Promise<GuardMemberRecord[]>
  lookupVerificationProfiles: (memberIds: string[]) => Promise<{
    profiles: QQVerificationStatus[]
    errors: IdentityLookupError[]
  }>
}

export class IdentityPageService {
  constructor(private readonly deps: IdentityPageServiceDeps) {}

  async getPageData() {
    const guardRecords = await this.deps.loadGuardRecords()
    const memberIds = [...new Set(guardRecords.map((record) => record.memberId))]
    const { profiles, errors } = await this.deps.lookupVerificationProfiles(memberIds)

    return buildIdentityPageData({
      generatedAt: new Date().toISOString(),
      guardRecords,
      verificationProfiles: profiles,
      lookupErrors: errors,
    })
  }
}

function buildGroupSummaries(records: readonly GuardMemberRecord[]) {
  const map = new Map<string, { memberCount: number; pendingCount: number; releasedCount: number }>()

  for (const record of records) {
    const current = map.get(record.guildId) ?? { memberCount: 0, pendingCount: 0, releasedCount: 0 }
    current.memberCount += 1
    if (record.releasedAt) {
      current.releasedCount += 1
    } else if (!record.kickedAt) {
      current.pendingCount += 1
    }
    map.set(record.guildId, current)
  }

  return [...map.entries()]
    .map(([guildId, counts]) => ({ guildId, ...counts }))
    .sort((left, right) => right.memberCount - left.memberCount || left.guildId.localeCompare(right.guildId))
}

function joinProfile(record: GuardMemberRecord, profile: QQVerificationStatus | undefined): IdentityMemberSnapshot {
  return {
    ...serializeGuardMember(record),
    profile: profile ?? null,
  }
}
