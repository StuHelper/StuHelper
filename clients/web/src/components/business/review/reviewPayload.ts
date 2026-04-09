import type { ReviewRatings } from '@/types/review'

export interface CreateReviewPayloadInput {
  courseID: number
  teacherID?: number
  termID: string
  title: string
  content: string
  grade?: string
  ratings: ReviewRatings
}

export function buildCreateReviewPayload(input: CreateReviewPayloadInput): CreateReviewPayloadInput {
  const termID = input.termID?.trim()
  if (!termID) {
    throw new Error('termID is required')
  }

  return {
    courseID: input.courseID,
    ...(input.teacherID !== undefined && { teacherID: input.teacherID }),
    termID,
    title: input.title,
    content: input.content,
    ...(input.grade !== undefined && { grade: input.grade }),
    ratings: input.ratings,
  }
}
