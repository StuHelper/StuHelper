import type {
  CommandPolicyRecord,
  ModerationEventRecord,
  ModerationReportRecord,
  ReviewQueueRecord,
} from '@stuhelper/koishi-moderation-core'
import type {
  GuardGroupBindingRecord,
  GuardMemberRecord,
  GuardTemplateRecord,
} from '@stuhelper/koishi-shared'

import type {
  SerializedCommandPolicy,
  SerializedEvent,
  SerializedGuardBinding,
  SerializedGuardMember,
  SerializedGuardTemplate,
  SerializedReport,
  SerializedReview,
} from './page-types'

export function serializeGuardMember(record: GuardMemberRecord): SerializedGuardMember {
  return {
    ...record,
    joinedAt: record.joinedAt.toISOString(),
    deadlineAt: record.deadlineAt.toISOString(),
    mutedAt: serializeNullableDate(record.mutedAt),
    reminderSentAt: serializeNullableDate(record.reminderSentAt),
    releasedAt: serializeNullableDate(record.releasedAt),
    kickedAt: serializeNullableDate(record.kickedAt),
    createdAt: record.createdAt.toISOString(),
    updatedAt: record.updatedAt.toISOString(),
  }
}

export function serializeReview(record: ReviewQueueRecord): SerializedReview {
  return serializeRecord(record)
}

export function serializeReport(record: ModerationReportRecord): SerializedReport {
  return serializeRecord(record)
}

export function serializeEvent(record: ModerationEventRecord): SerializedEvent {
  return serializeRecord(record)
}

export function serializeGuardTemplate(record: GuardTemplateRecord): SerializedGuardTemplate {
  return serializeRecord(record)
}

export function serializeGuardBinding(record: GuardGroupBindingRecord): SerializedGuardBinding {
  return serializeRecord(record)
}

export function serializeCommandPolicy(record: CommandPolicyRecord): SerializedCommandPolicy {
  return serializeRecord(record)
}

export function serializeNullableDate(value: Date | null) {
  return value ? value.toISOString() : null
}

function serializeRecord<T extends { createdAt: Date; updatedAt: Date }>(record: T) {
  return {
    ...record,
    createdAt: record.createdAt.toISOString(),
    updatedAt: record.updatedAt.toISOString(),
  }
}
