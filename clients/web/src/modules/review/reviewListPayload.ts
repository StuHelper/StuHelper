import type { Review, ReviewRatings } from '@stuhelper/shared/review'
import { isNonArrayRecord as isRecord } from '@stuhelper/shared/utils'

const REVIEW_STATUS_VALUES = new Set([
  'published',
  'pending_review',
  'hidden',
  'deleted',
])
const REVIEW_GRADE_VALUES = new Set([
  'A+',
  'A',
  'A-',
  'B+',
  'B',
  'B-',
  'C+',
  'C',
  'C-',
  'D',
  'F',
])
const CONTENT_FLAG_VALUES = new Set(['warn', 'review', 'cleared'])
const REVIEW_VOTE_VALUES = new Set(['like', 'dislike'])

function readString(record: Record<string, unknown>, key: string, message: string): string {
  const value = record[key]
  if (typeof value !== 'string') {
    throw new Error(message)
  }
  return value
}

function readOptionalString(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
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

function readNullableInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readOptionalEnum<T extends string>(
  record: Record<string, unknown>,
  key: string,
  values: Set<string>,
  message: string,
): T | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'string' || !values.has(value)) {
    throw new Error(message)
  }
  return value as T
}

function readEnum<T extends string>(
  record: Record<string, unknown>,
  key: string,
  values: Set<string>,
  message: string,
): T {
  const value = readString(record, key, message)
  if (!values.has(value)) {
    throw new Error(message)
  }
  return value as T
}

function readReviewRatings(value: unknown, message: string): ReviewRatings {
  if (!isRecord(value)) {
    throw new Error(message)
  }

  const ratings: ReviewRatings = {}
  for (const [key, rating] of Object.entries(value)) {
    if (
      typeof rating !== 'number' ||
      !Number.isFinite(rating) ||
      rating < 1 ||
      rating > 5
    ) {
      throw new Error(message)
    }
    ratings[key] = rating
  }
  return ratings
}

export function readReviewPayload(
  payload: unknown,
  message = 'Invalid review response',
): Review {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    id: readString(payload, 'id', message),
    courseID: readInteger(payload, 'courseID', message),
    courseName: readOptionalString(payload, 'courseName', message),
    teacherID: readNullableInteger(payload, 'teacherID', message),
    teacherName: readOptionalString(payload, 'teacherName', message),
    termID: readString(payload, 'termID', message),
    termName: readOptionalString(payload, 'termName', message),
    title: readString(payload, 'title', message),
    content: readString(payload, 'content', message),
    grade: readOptionalEnum<NonNullable<Review['grade']>>(
      payload,
      'grade',
      REVIEW_GRADE_VALUES,
      message,
    ),
    ratings: readReviewRatings(payload.ratings, message),
    likeCount: readInteger(payload, 'likeCount', message),
    dislikeCount: readInteger(payload, 'dislikeCount', message),
    userVote: readOptionalEnum<NonNullable<Review['userVote']>>(
      payload,
      'userVote',
      REVIEW_VOTE_VALUES,
      message,
    ),
    replyCount: readInteger(payload, 'replyCount', message),
    status: readEnum<Review['status']>(payload, 'status', REVIEW_STATUS_VALUES, message),
    contentFlag: readOptionalEnum<NonNullable<Review['contentFlag']>>(
      payload,
      'contentFlag',
      CONTENT_FLAG_VALUES,
      message,
    ),
    moderationReason: readNullableString(payload, 'moderationReason', message),
    createdAt: readString(payload, 'createdAt', message),
    updatedAt: readOptionalString(payload, 'updatedAt', message),
  }
}

export function readReviewPagePayload(
  payload: unknown,
  message = 'Invalid review list response',
): { list: Review[]; total: number } {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const { list, total } = payload
  if (
    !Array.isArray(list) ||
    typeof total !== 'number' ||
    !Number.isInteger(total) ||
    total < 0
  ) {
    throw new Error(message)
  }

  return {
    list: list.map(item => readReviewPayload(item, message)),
    total,
  }
}
