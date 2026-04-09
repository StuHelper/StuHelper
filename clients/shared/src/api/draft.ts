import type { ApiClient } from './client'
import type { components } from '../types'
import type { SaveDraftParams } from '../types/business/draft'

type SaveDraftRequest = components['schemas']['SaveDraftRequest']
type ReviewGrade = NonNullable<SaveDraftRequest['grade']>

const REVIEW_GRADES: readonly ReviewGrade[] = ['A+', 'A', 'A-', 'B+', 'B', 'B-', 'C+', 'C', 'C-', 'D', 'F'] as const

function normalizeReviewGrade(grade?: string): ReviewGrade | undefined {
  if (!grade) return undefined
  const trimmed = grade.trim()
  return REVIEW_GRADES.includes(trimmed as ReviewGrade) ? (trimmed as ReviewGrade) : undefined
}

function toSaveDraftRequest(data: SaveDraftParams): SaveDraftRequest {
  return {
    courseID: data.courseID,
    ...(data.teacherID !== undefined && { teacherID: data.teacherID }),
    ...(data.termID !== undefined && { termID: data.termID }),
    ...(data.title !== undefined && { title: data.title }),
    ...(data.content !== undefined && { content: data.content }),
    ...(normalizeReviewGrade(data.grade) !== undefined && { grade: normalizeReviewGrade(data.grade) }),
    ...(data.ratings !== undefined && { ratings: data.ratings }),
  }
}

export const createDraftApi = (client: ApiClient) => ({
  getDraft: (courseID: number) =>
    client.GET('/api/v1/course/review/drafts/{courseID}', { params: { path: { courseID } } }),

  saveDraft: (data: SaveDraftParams) =>
    client.POST('/api/v1/course/review/drafts', { body: toSaveDraftRequest(data) }),

  deleteDraft: (courseID: number) =>
    client.DELETE('/api/v1/course/review/drafts/{courseID}', { params: { path: { courseID } } })
})
