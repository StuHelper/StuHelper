import {
  safeGetSessionStorageItem,
  safeRemoveSessionStorageItem,
  safeSetSessionStorageItem,
} from '@/utils/browserStorage'

const PRESELECT_COURSE_KEY = 'review_post_preselect_course_id'

export function rememberReviewPostCourse(courseID: number) {
  if (!Number.isFinite(courseID) || courseID <= 0) return
  safeSetSessionStorageItem(PRESELECT_COURSE_KEY, String(courseID))
}

export function consumeReviewPostCourseID(): number | null {
  const raw = safeGetSessionStorageItem(PRESELECT_COURSE_KEY)
  safeRemoveSessionStorageItem(PRESELECT_COURSE_KEY)

  const courseID = Number(raw)
  return Number.isFinite(courseID) && courseID > 0 ? courseID : null
}
