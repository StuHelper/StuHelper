import type { Course, CourseCategory, Department, TeacherStats, Term } from '@stuhelper/shared/course'

export interface TeacherSummaryPayload {
  teacherID: number
  teacherName: string
  departmentName?: string
  avgRating?: number | null
  reviewCount: number
  courseCount: number
}

export interface GroupedCoursePayload {
  departmentName?: string
  courses: Course[]
}

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

function readBoolean(record: Record<string, unknown>, key: string, message: string): boolean {
  const value = record[key]
  if (typeof value !== 'boolean') {
    throw new Error(message)
  }
  return value
}

function readOptionalNullableNumber(
  record: Record<string, unknown>,
  key: string,
  message: string,
): number | null | undefined {
  const value = record[key]
  if (value === undefined || value === null) {
    return value
  }
  if (typeof value !== 'number' || !Number.isFinite(value)) {
    throw new Error(message)
  }
  return value
}

function readOptionalStringArray(
  record: Record<string, unknown>,
  key: string,
  message: string,
): string[] | undefined {
  const value = record[key]
  if (value === undefined) {
    return undefined
  }
  if (!Array.isArray(value) || value.some(item => typeof item !== 'string')) {
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

function readPage<T>(
  payload: unknown,
  message: string,
  reader: (item: unknown, message: string) => T,
): { list: T[]; total: number } {
  if (!isRecord(payload) || !Array.isArray(payload.list)) {
    throw new Error(message)
  }

  const total = readInteger(payload, 'total', message)
  if (total < 0) {
    throw new Error(message)
  }

  return {
    list: payload.list.map(item => reader(item, message)),
    total,
  }
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

export function readCoursePagePayload(
  payload: unknown,
  message = 'Invalid courses response',
): { list: Course[]; total: number } {
  return readPage(payload, message, readCoursePayload)
}

export function readGroupedCourseListPayload(
  payload: unknown,
  message = 'Invalid grouped courses response',
): GroupedCoursePayload[] {
  if (!isRecord(payload) || !Array.isArray(payload.groups)) {
    throw new Error(message)
  }

  return payload.groups.map((group) => {
    if (!isRecord(group) || !Array.isArray(group.courses)) {
      throw new Error(message)
    }

    return {
      departmentName: readOptionalString(group, 'departmentName', message),
      courses: group.courses.map(course => readCoursePayload(course, message)),
    }
  })
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

export function readTermPayload(payload: unknown, message = 'Invalid terms response'): Term {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  return {
    id: readString(payload, 'id', message),
    schoolID: readOptionalInteger(payload, 'schoolID', message),
    name: readString(payload, 'name', message),
    isCurrent: readBoolean(payload, 'isCurrent', message),
  }
}

export function readTermArrayPayload(
  payload: unknown,
  message = 'Invalid terms response',
): Term[] {
  return readArray(payload, message, readTermPayload)
}

export function readTeacherStatsPayload(
  payload: unknown,
  message = 'Invalid course teachers response',
): TeacherStats {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const teacherID = readInteger(payload, 'teacherID', message)
  const avgRating = readOptionalNullableNumber(payload, 'avgRating', message)
  const courseCount = readInteger(payload, 'courseCount', message)
  const reviewCount = readInteger(payload, 'reviewCount', message)
  if (
    teacherID <= 0 ||
    (typeof avgRating === 'number' && avgRating < 0) ||
    courseCount < 0 ||
    reviewCount < 0
  ) {
    throw new Error(message)
  }

  return {
    teacherID,
    teacherName: readString(payload, 'teacherName', message),
    departmentName: readString(payload, 'departmentName', message),
    avgRating,
    courseCount,
    reviewCount,
    tags: readOptionalStringArray(payload, 'tags', message),
  }
}

export function readTeacherStatsArrayPayload(
  payload: unknown,
  message = 'Invalid course teachers response',
): TeacherStats[] {
  return readArray(payload, message, readTeacherStatsPayload)
}

export function readTeacherSummaryPayload(
  payload: unknown,
  message = 'Invalid teachers response',
): TeacherSummaryPayload {
  if (!isRecord(payload)) {
    throw new Error(message)
  }

  const teacherID = readInteger(payload, 'teacherID', message)
  const avgRating = readOptionalNullableNumber(payload, 'avgRating', message)
  const reviewCount = readInteger(payload, 'reviewCount', message)
  const courseCount = readInteger(payload, 'courseCount', message)
  if (
    teacherID <= 0 ||
    (typeof avgRating === 'number' && avgRating < 0) ||
    reviewCount < 0 ||
    courseCount < 0
  ) {
    throw new Error(message)
  }

  return {
    teacherID,
    teacherName: readString(payload, 'teacherName', message),
    departmentName: readOptionalString(payload, 'departmentName', message),
    avgRating,
    reviewCount,
    courseCount,
  }
}

export function readTeacherSummaryListPayload(
  payload: unknown,
  message = 'Invalid teachers response',
): TeacherSummaryPayload[] {
  return readList(payload, message, readTeacherSummaryPayload)
}

export function readTeacherSummaryPagePayload(
  payload: unknown,
  message = 'Invalid teachers response',
): { list: TeacherSummaryPayload[]; total: number } {
  return readPage(payload, message, readTeacherSummaryPayload)
}
