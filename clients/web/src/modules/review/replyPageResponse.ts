import type { Reply } from '@stuhelper/shared/reply'

function isRecord(value: unknown): value is Record<string, unknown> {
  return Boolean(value) && typeof value === 'object' && !Array.isArray(value)
}

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readNullableString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readInteger(record: Record<string, unknown>, key: string, message: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readBoolean(record: Record<string, unknown>, key: string, message: string): boolean {
  const value = record[key]
  if (typeof value !== 'boolean') {
    throw new Error(message)
  }
  return value
}

export function readReplyPage(payload: unknown): { list: Reply[]; total: number } {
  if (!isRecord(payload)) {
    throw new Error('Invalid reply list response')
  }

  const { list, total } = payload
  if (
    !Array.isArray(list) ||
    typeof total !== 'number' ||
    !Number.isInteger(total) ||
    total < 0
  ) {
    throw new Error('Invalid reply list response')
  }

  return {
    list: list.map(item => readReply(item, 'Invalid reply list response')),
    total,
  }
}

export function readReply(payload: unknown, message = 'Invalid reply response'): Reply {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const likeCount = readInteger(payload, 'likeCount', message)
  if (likeCount < 0) {
    throw new Error(message)
  }

  return {
    id: readString(payload, 'id', message),
    reviewID: readString(payload, 'reviewID', message),
    parentID: readNullableString(payload, 'parentID', message),
    content: readString(payload, 'content', message),
    likeCount,
    status: readString(payload, 'status', message),
    isOwner: readBoolean(payload, 'isOwner', message),
    createdAt: readString(payload, 'createdAt', message),
    updatedAt: readString(payload, 'updatedAt', message),
  }
}
