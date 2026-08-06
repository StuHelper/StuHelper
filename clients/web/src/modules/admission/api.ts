import { api } from '@/api'
import type { AdmissionMe, AdmissionSession } from '@stuhelper/shared/api'

type ApiResult<T> = {
  data?: { data?: T; error?: unknown; success?: boolean }
  error?: unknown
}

const ADMISSION_STATUS_VALUES = new Set([
  'created',
  'awaiting_account_link',
  'awaiting_requirements',
  'pending_manual_review',
  'eligible',
  'action_pending',
  'admitted',
  'released',
  'rejected',
  'cancelled',
  'expired',
])

function isRecord(value: unknown): value is Record<string, unknown> {
  return value !== null && typeof value === 'object'
}

function requireData<T>(result: ApiResult<T>, message: string): T {
  if (result.error !== undefined) throw result.error
  if (result.data?.success === false || result.data?.error !== undefined) throw result.data
  if (result.data?.data === undefined || result.data.data === null) throw new Error(message)
  return result.data.data
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') throw new Error(message)
  return value
}

function readOptionalString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) return undefined
  if (typeof value !== 'string') throw new Error(message)
  return value
}

function readOptionalID(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) return undefined
  if (typeof value === 'string') return value
  if (typeof value === 'number' && Number.isInteger(value)) return String(value)
  throw new Error(message)
}

function readNullableString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) return value
  if (typeof value !== 'string') throw new Error(message)
  return value
}

function readOptionalInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number | undefined {
  const value = record[key]
  if (value === undefined || value === null) return undefined
  if (typeof value !== 'number' || !Number.isInteger(value)) throw new Error(message)
  return value
}

function readBoolean(record: Record<string, unknown>, key: string, message: string): boolean {
  const value = record[key]
  if (typeof value !== 'boolean') throw new Error(message)
  return value
}

function readStatus(
  record: Record<string, unknown>,
  key: string,
  message: string,
): AdmissionSession['status'] {
  const value = readString(record, key, message)
  if (!ADMISSION_STATUS_VALUES.has(value)) throw new Error(message)
  return value as AdmissionSession['status']
}

function readOptionalAbsoluteURL(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = readOptionalString(record, key, message)
  if (value === undefined) return undefined
  try {
    const url = new URL(value)
    if (url.protocol !== 'http:' && url.protocol !== 'https:') throw new Error(message)
  } catch {
    throw new Error(message)
  }
  return value
}

function readAdmissionSession(payload: unknown, message: string): AdmissionSession {
  if (!isRecord(payload)) throw new Error(message)
  return {
    id: readString(payload, 'id', message),
    platform: readString(payload, 'platform', message),
    botSelfID: readOptionalString(payload, 'botSelfID', message),
    guildID: readString(payload, 'guildID', message),
    channelID: readOptionalString(payload, 'channelID', message),
    qqID: readString(payload, 'qqID', message),
    userID: readOptionalID(payload, 'userID', message),
    status: readStatus(payload, 'status', message),
    tokenExpiresAt: readString(payload, 'tokenExpiresAt', message),
    tokenConsumedAt: readNullableString(payload, 'tokenConsumedAt', message),
    linkWaitDeadlineAt: readString(payload, 'linkWaitDeadlineAt', message),
    submissionWaitDeadlineAt: readString(payload, 'submissionWaitDeadlineAt', message),
    manualReviewDeadlineAt: readNullableString(payload, 'manualReviewDeadlineAt', message),
    initialMuteUntil: readString(payload, 'initialMuteUntil', message),
    verifiedAt: readNullableString(payload, 'verifiedAt', message),
    cancelledAt: readNullableString(payload, 'cancelledAt', message),
    lastBotError: readNullableString(payload, 'lastBotError', message),
    eligibilityRevision: readOptionalInteger(payload, 'eligibilityRevision', message),
    eligibilityEvaluatedAt: readNullableString(payload, 'eligibilityEvaluatedAt', message),
    projectionPending: readBoolean(payload, 'projectionPending', message),
    failureCount: readOptionalInteger(payload, 'failureCount', message),
    remainingRetryCount: readOptionalInteger(payload, 'remainingRetryCount', message),
    willBlacklistOnTimeout: payload.willBlacklistOnTimeout === undefined
      ? undefined
      : readBoolean(payload, 'willBlacklistOnTimeout', message),
    authURL: readOptionalAbsoluteURL(payload, 'authURL', message),
  }
}

function readAdmissionMe(payload: unknown, message: string): AdmissionMe {
  if (!isRecord(payload)) throw new Error(message)
  return {
    status: readStatus(payload, 'status', message),
    projectionPending: readBoolean(payload, 'projectionPending', message),
    ...(payload.session === undefined
      ? {}
      : { session: readAdmissionSession(payload.session, message) }),
  }
}

export const admissionApi = {
  async getAdmissionSession(token: string): Promise<AdmissionSession> {
    const result = await api.admission.getAdmissionSession(token)
    return readAdmissionSession(
      requireData(result, 'Admission session response is empty'),
      'Invalid admission session response',
    )
  },

  async linkAdmissionSession(token: string): Promise<AdmissionSession> {
    const result = await api.admission.linkAdmissionSession(token)
    return readAdmissionSession(
      requireData(result, 'Admission link response is empty'),
      'Invalid admission link response',
    )
  },

  async getAdmissionMe(admissionSessionID?: string): Promise<AdmissionMe> {
    const result = admissionSessionID
      ? await api.admission.getAdmissionMe(admissionSessionID)
      : await api.admission.getAdmissionMe()
    return readAdmissionMe(
      requireData(result, 'Admission me response is empty'),
      'Invalid admission me response',
    )
  },
}
