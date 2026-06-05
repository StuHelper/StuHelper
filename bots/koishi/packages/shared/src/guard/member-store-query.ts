export interface GuardMemberVersionRef {
  readonly id: string
  readonly updatedAt: Date
}

export interface ClaimedGuardMemberVersionRef {
  readonly guardId: string
  readonly claimedAt: Date
}

export function activeGuardMemberQuery(record: GuardMemberVersionRef) {
  return {
    id: record.id,
    updatedAt: record.updatedAt,
    releasedAt: null,
    kickedAt: null,
  }
}

export function claimedGuardMemberQuery(input: ClaimedGuardMemberVersionRef) {
  return {
    id: input.guardId,
    updatedAt: input.claimedAt,
    releasedAt: null,
    kickedAt: null,
  }
}
