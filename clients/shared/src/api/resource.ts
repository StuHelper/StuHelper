import type { ApiClient } from './client'
import type { components, operations } from '../types/api.gen'

export type ResourceListParams = operations['listResources']['parameters']['query']
export type ResourceItem = components['schemas']['ResourceItem']
export type CreateResourceRequest = components['schemas']['CreateResourceRequest']
export type UpdateResourceRequest = components['schemas']['UpdateResourceRequest']
export type ResourceDownloadURL = components['schemas']['ResourceDownloadURL']

export const createResourceApi = (client: ApiClient) => ({
  listResources: (params?: ResourceListParams) =>
    client.GET('/api/v1/resources', { params: { query: params } }),

  getResource: (resourceID: number) =>
    client.GET('/api/v1/resources/{resourceID}', {
      params: { path: { resourceID } },
    }),

  createResource: (data: CreateResourceRequest) =>
    client.POST('/api/v1/resources', { body: data }),

  updateResource: (resourceID: number, data: UpdateResourceRequest) =>
    client.PATCH('/api/v1/resources/{resourceID}', {
      params: { path: { resourceID } },
      body: data,
    }),

  deleteResource: (resourceID: number) =>
    client.DELETE('/api/v1/resources/{resourceID}', {
      params: { path: { resourceID } },
    }),

  getDownloadURL: (resourceID: number) =>
    client.GET('/api/v1/resources/{resourceID}/download-url', {
      params: { path: { resourceID } },
    }),
})
