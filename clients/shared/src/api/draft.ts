import type { ApiClient } from './client'
import type { components } from '../types'
import type { SaveDraftParams } from '../types/business/draft'
import { isReviewGrade, type ReviewGrade } from '../constants/review'

type SaveDraftRequest = components['schemas']['SaveDraftRequest']

function normalizeReviewGrade(grade?: string): ReviewGrade | undefined {
  if (!grade) return undefined
  const trimmed = grade.trim()
  return isReviewGrade(trimmed) ? trimmed : undefined
}

function toSaveDraftRequest(data: SaveDraftParams): SaveDraftRequest {
  const grade = normalizeReviewGrade(data.grade)

  return {
    courseID: data.courseID,
    ...(data.teacherID !== undefined && { teacherID: data.teacherID }),
    ...(data.termID !== undefined && { termID: data.termID }),
    ...(data.title !== undefined && { title: data.title }),
    ...(data.content !== undefined && { content: data.content }),
    ...(grade !== undefined && { grade }),
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
