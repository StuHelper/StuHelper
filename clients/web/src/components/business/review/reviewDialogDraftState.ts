import type { Course } from '@/types/course'
import { isValidRating } from '@/types/course'
import type { ReviewRatings } from '@/types/review'

const LOCAL_DRAFT_KEY_PREFIX = 'review_draft:'
const LOCAL_DRAFT_CLEARED_KEY_PREFIX = 'review_draft_cleared:'
const LEGACY_LOCAL_DRAFT_KEY = 'review_draft'
const LEGACY_LOCAL_DRAFT_CLEARED_KEY = 'review_draft_cleared'

export interface LocalReviewDraft {
  courseID: number
  courseName: string
  teacherID?: number | null
  teacherName?: string
  title: string
  content: string
  grade: string
  termID: string
  ratings: ReviewRatings
  updatedAt: number
}

export interface ReviewDialogServerDraft {
  teacherID?: number | null
  title?: string
  content?: string
  grade?: string
  termID?: string
  ratings?: Record<string, number>
  updatedAt: string
}

interface CreateLocalReviewDraftInput {
  course: Pick<Course, 'id' | 'name'>
  teacherID?: number | null
  teacherName?: string
  title: string
  content: string
  grade: string
  termID: string
  ratings: ReviewRatings
}

export function createLocalReviewDraft(input: CreateLocalReviewDraftInput): LocalReviewDraft {
  return {
    courseID: input.course.id,
    courseName: input.course.name,
    teacherID: input.teacherID ?? null,
    teacherName: input.teacherName,
    title: input.title,
    content: input.content,
    grade: input.grade,
    termID: input.termID,
    ratings: input.ratings,
    updatedAt: Date.now(),
  }
}

function getLocalDraftKey(courseID: number): string {
  return `${LOCAL_DRAFT_KEY_PREFIX}${courseID}`
}

function getLocalDraftClearedKey(courseID: number): string {
  return `${LOCAL_DRAFT_CLEARED_KEY_PREFIX}${courseID}`
}

function isLegacyDraftKey(key: string): boolean {
  return key === LEGACY_LOCAL_DRAFT_KEY
}

function isDraftKey(key: string): boolean {
  return key.startsWith(LOCAL_DRAFT_KEY_PREFIX) || isLegacyDraftKey(key)
}

function isClearedKey(key: string): boolean {
  return key.startsWith(LOCAL_DRAFT_CLEARED_KEY_PREFIX) || key === LEGACY_LOCAL_DRAFT_CLEARED_KEY
}

function parseDraftRecord(
  raw: string,
  validRatingKeys: ReadonlySet<string>,
): LocalReviewDraft | null {
  try {
    const parsed = JSON.parse(raw) as Record<string, unknown>
    const ratings = sanitizeDraftRatings(parsed.ratings, validRatingKeys)
    if (
      typeof parsed.courseID !== 'number' ||
      typeof parsed.courseName !== 'string' ||
      typeof parsed.title !== 'string' ||
      typeof parsed.content !== 'string' ||
      typeof parsed.grade !== 'string' ||
      typeof parsed.termID !== 'string' ||
      typeof parsed.updatedAt !== 'number' ||
      ratings === null
    ) {
      return null
    }

    const teacherID =
      parsed.teacherID === undefined || parsed.teacherID === null
        ? null
        : typeof parsed.teacherID === 'number'
          ? parsed.teacherID
          : null

    const teacherName =
      typeof parsed.teacherName === 'string' ? parsed.teacherName : undefined

    return {
      courseID: parsed.courseID,
      courseName: parsed.courseName,
      teacherID,
      teacherName,
      title: parsed.title,
      content: parsed.content,
      grade: parsed.grade,
      termID: parsed.termID,
      ratings,
      updatedAt: parsed.updatedAt,
    }
  } catch {
    return null
  }
}

export function saveLocalReviewDraft(draft: LocalReviewDraft): void {
  try {
    localStorage.setItem(getLocalDraftKey(draft.courseID), JSON.stringify(draft))
    localStorage.removeItem(getLocalDraftClearedKey(draft.courseID))
    localStorage.removeItem(LEGACY_LOCAL_DRAFT_KEY)
    localStorage.removeItem(LEGACY_LOCAL_DRAFT_CLEARED_KEY)
  } catch (error) {
    console.warn('[ReviewDialog] Failed to save local draft:', error)
  }
}

export function sanitizeDraftRatings(
  ratings: unknown,
  validRatingKeys?: ReadonlySet<string>,
): ReviewRatings | null {
  if (!ratings || typeof ratings !== 'object') return null

  const normalized: ReviewRatings = {}
  for (const [key, value] of Object.entries(ratings as Record<string, unknown>)) {
    if (validRatingKeys && !validRatingKeys.has(key)) return null
    if (!isValidRating(value)) return null
    normalized[key] = value
  }

  return normalized
}

export function loadLocalReviewDraft(
  validRatingKeys: ReadonlySet<string>,
  courseID?: number,
): LocalReviewDraft | null {
  try {
    if (typeof courseID === 'number') {
      const raw = localStorage.getItem(getLocalDraftKey(courseID))
      if (raw) {
        const parsed = parseDraftRecord(raw, validRatingKeys)
        if (parsed) return parsed
        localStorage.removeItem(getLocalDraftKey(courseID))
      }

      const legacyRaw = localStorage.getItem(LEGACY_LOCAL_DRAFT_KEY)
      if (legacyRaw) {
        const parsed = parseDraftRecord(legacyRaw, validRatingKeys)
        if (parsed && parsed.courseID === courseID) return parsed
      }
      return null
    }

    const drafts: LocalReviewDraft[] = []
    for (let i = 0; i < localStorage.length; i++) {
      const key = localStorage.key(i)
      if (!key || !isDraftKey(key)) continue
      const raw = localStorage.getItem(key)
      if (!raw) continue
      const parsed = parseDraftRecord(raw, validRatingKeys)
      if (parsed) drafts.push(parsed)
    }

    if (drafts.length === 0) return null
    drafts.sort((a, b) => b.updatedAt - a.updatedAt)
    return drafts[0]
  } catch {
    return null
  }
}

export function clearLocalReviewDraft(courseID?: number): void {
  if (typeof courseID === 'number') {
    localStorage.removeItem(getLocalDraftKey(courseID))
    localStorage.setItem(getLocalDraftClearedKey(courseID), String(Date.now()))
    localStorage.removeItem(LEGACY_LOCAL_DRAFT_KEY)
    localStorage.removeItem(LEGACY_LOCAL_DRAFT_CLEARED_KEY)
    return
  }

  const keys: string[] = []
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (key && (isDraftKey(key) || isClearedKey(key))) {
      keys.push(key)
    }
  }
  for (const key of keys) {
    localStorage.removeItem(key)
  }
}

export function getLocalReviewDraftClearedAt(courseID?: number): number {
  if (typeof courseID === 'number') {
    return Number(localStorage.getItem(getLocalDraftClearedKey(courseID))) || 0
  }

  let latest = 0
  for (let i = 0; i < localStorage.length; i++) {
    const key = localStorage.key(i)
    if (!key || !isClearedKey(key)) continue
    const value = Number(localStorage.getItem(key))
    if (Number.isFinite(value) && value > latest) {
      latest = value
    }
  }
  return latest
}

export function createDraftCourse(draft: Pick<LocalReviewDraft, 'courseID' | 'courseName'>): Course {
  return {
    id: draft.courseID,
    name: draft.courseName,
    credits: 0,
    departmentID: 0,
    reviewCount: 0,
  }
}
