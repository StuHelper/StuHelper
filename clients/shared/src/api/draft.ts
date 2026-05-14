import type { ApiClient } from './client'
import type { components } from '../types'
import type { SaveDraftParams } from '../types/business/draft'
import { normalizeReviewGrade } from '../constants/review'

type SaveDraftRequest = components['schemas']['SaveDraftRequest']

function toSaveDraftRequest(data: SaveDraftParams): SaveDraftRequest {
  const grade = normalizeReviewGrade(data.grade)

  return {
    ...(data.courseID !== undefined && { courseID: data.courseID }),
    ...(data.teacherID !== undefined && { teacherID: data.teacherID }),
    ...(data.termID !== undefined && { termID: data.termID }),
    ...(data.title !== undefined && { title: data.title }),
    ...(data.content !== undefined && { content: data.content }),
    ...(grade !== undefined && { grade }),
    ...(data.ratings !== undefined && { ratings: data.ratings }),
  }
}

export const createDraftApi = (client: ApiClient) => ({
  getDraft: () =>
    client.GET('/api/v1/course/review/drafts'),

  saveDraft: (data: SaveDraftParams) =>
    client.POST('/api/v1/course/review/drafts', { body: toSaveDraftRequest(data) }),

  deleteDraft: () =>
    client.DELETE('/api/v1/course/review/drafts')
})
