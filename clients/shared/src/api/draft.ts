import type { ApiClient } from './client'
import type { components } from '../types'

type SaveDraftRequest = components['schemas']['SaveDraftRequest']

export const createDraftApi = (client: ApiClient) => ({
  getDraft: (courseID: number) =>
    client.GET('/api/v1/course/review/drafts/{courseID}', { params: { path: { courseID } } }),

  saveDraft: (data: SaveDraftRequest) =>
    client.POST('/api/v1/course/review/drafts', { body: data }),

  deleteDraft: (courseID: number) =>
    client.DELETE('/api/v1/course/review/drafts/{courseID}', { params: { path: { courseID } } })
})
