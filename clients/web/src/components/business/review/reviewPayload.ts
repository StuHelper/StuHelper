import { isReviewGrade } from '@stuhelper/shared/constants'
import type { PostReviewRequest, ReviewRatings } from '@stuhelper/shared/review'

export interface CreateReviewPayloadInput {
  courseID: number
  teacherID?: number
  termID: string
  title: string
  content: string
  grade?: string
  ratings: ReviewRatings
}

export function buildCreateReviewPayload(input: CreateReviewPayloadInput): PostReviewRequest {
  const termID = input.termID?.trim()
  if (!termID) {
    throw new Error('termID is required')
  }

  const grade = input.grade?.trim()
  const gradeFields = grade && isReviewGrade(grade) ? { grade } : {}

  return {
    courseID: input.courseID,
    ...(input.teacherID !== undefined && { teacherID: input.teacherID }),
    termID,
    title: input.title,
    content: input.content,
    ...gradeFields,
    ratings: input.ratings,
  }
}
