import type { Course, CourseCategory, Department } from '@stuhelper/shared/course'

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

function readInteger(record: Record<string, unknown>, key: string, message: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readOptionalInteger(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'number' || !Number.isInteger(value)) {
    throw new Error(message)
  }
  return value
}

function readNumber(record: Record<string, unknown>, key: string, message: string): number {
  const value = record[key]
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(message)
  }
  return value
}

function readOptionalBoolean(
  record: Record<string, unknown>,
  key: string,
  message: string,
): boolean | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (typeof value !== 'boolean') {
    throw new Error(message)
  }
  return value
}

function readArray<T>(
  payload: unknown,
  message: string,
  reader: (item: unknown, message: string) => T,
): T[] {
  if (!Array.isArray(payload)) {
    throw new Error(message)
  }
  return payload.map(item => reader(item, message))
}

function readList<T>(
  payload: unknown,
  message: string,
  reader: (item: unknown, message: string) => T,
): T[] {
  if (!isRecord(payload) || !Array.isArray(payload.list)) {
    throw new Error(message)
  }
  return payload.list.map(item => reader(item, message))
}

export function readCoursePayload(payload: unknown, message = 'Invalid course response'): Course {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const id = readInteger(payload, 'id', message)
  const departmentID = readInteger(payload, 'departmentID', message)
  const credits = readNumber(payload, 'credits', message)
  const reviewCount = readInteger(payload, 'reviewCount', message)
  if (id <= 0 || departmentID <= 0 || credits < 0 || reviewCount < 0) {
    throw new Error(message)
  }

  return {
    id,
    schoolID: readOptionalInteger(payload, 'schoolID', message),
    departmentID,
    departmentName: readOptionalString(payload, 'departmentName', message),
    code: readOptionalString(payload, 'code', message),
    name: readString(payload, 'name', message),
    credits,
    category: readOptionalString(payload, 'category', message),
    reviewCount,
    isFavorited: readOptionalBoolean(payload, 'isFavorited', message),
  }
}

export function readCourseListPayload(
  payload: unknown,
  message = 'Invalid courses response',
): Course[] {
  return readList(payload, message, readCoursePayload)
}

export function readDepartmentPayload(
  payload: unknown,
  message = 'Invalid departments response',
): Department {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const id = readInteger(payload, 'id', message)
  if (id <= 0) {
    throw new Error(message)
  }

  return {
    id,
    schoolID: readOptionalInteger(payload, 'schoolID', message),
    name: readString(payload, 'name', message),
    shortName: readOptionalString(payload, 'shortName', message),
    category: readString(payload, 'category', message),
    sortOrder: readOptionalInteger(payload, 'sortOrder', message),
  }
}

export function readDepartmentArrayPayload(
  payload: unknown,
  message = 'Invalid departments response',
): Department[] {
  return readArray(payload, message, readDepartmentPayload)
}

export function readCourseCategoryPayload(
  payload: unknown,
  message = 'Invalid course categories response',
): CourseCategory {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const id = readInteger(payload, 'id', message)
  if (id <= 0) {
    throw new Error(message)
  }

  return {
    id,
    schoolID: readOptionalInteger(payload, 'schoolID', message),
    name: readString(payload, 'name', message),
    sortOrder: readOptionalInteger(payload, 'sortOrder', message),
  }
}

export function readCourseCategoryArrayPayload(
  payload: unknown,
  message = 'Invalid course categories response',
): CourseCategory[] {
  return readArray(payload, message, readCourseCategoryPayload)
}
