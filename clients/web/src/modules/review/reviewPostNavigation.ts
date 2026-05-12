const PRESELECT_COURSE_KEY = 'review_post_preselect_course_id'

function hasSessionStorage() {
  return typeof window !== 'undefined' && typeof window.sessionStorage !== 'undefined'
}

export function rememberReviewPostCourse(courseID: number) {
  if (!Number.isFinite(courseID) || courseID <= 0 || !hasSessionStorage()) return
  window.sessionStorage.setItem(PRESELECT_COURSE_KEY, String(courseID))
}

export function consumeReviewPostCourseID(): number | null {
  if (!hasSessionStorage()) return null

  const raw = window.sessionStorage.getItem(PRESELECT_COURSE_KEY)
  window.sessionStorage.removeItem(PRESELECT_COURSE_KEY)

  const courseID = Number(raw)
  return Number.isFinite(courseID) && courseID > 0 ? courseID : null
}
